#!/usr/bin/env python3
"""Attach the unaltered Linux build artifacts to a stackql GitHub release.

Replaces the manual "download the artifacts from the build, upload them to the
release" steps of docs/stackql_release_process.md. For release vX.Y.Z it:

  1. Finds the successful run of the build workflow on main whose run number
     is Z (the release build number) and checks that the tag points at the
     commit that run built.
  2. Downloads the artifact zips exactly as GitHub stores them (the same bytes
     the Actions UI download gives): stackql_linux_amd64, stackql_linux_arm64,
     amd64-artifact-deb and arm64-artifact-deb. The unsigned Windows zips are
     downloaded too, for the Windows installer, but never uploaded.
  3. Writes a <asset>.sha256 sidecar for each asset (sha256sum format, the same
     as the .mcpb.sha256 sidecars).
  4. Uploads every asset and sidecar that is not already on the release. An
     attached asset is never overwritten: the digest GitHub recorded for it
     must match the downloaded bytes, otherwise the script fails and names it.
  5. Re-reads the release and verifies the recorded digest of every asset
     against the local bytes.

Requires the gh CLI, authenticated (gh auth login).

Usage (from the repo root):
  python cicd/python/attach_build_artifacts.py 0.12.778
  python cicd/python/attach_build_artifacts.py v0.12.778 --dry-run
  python cicd/python/attach_build_artifacts.py 0.12.778 --skip-deb
  python cicd/python/attach_build_artifacts.py 0.12.778 --run-id 38169009222
"""

import argparse
import hashlib
import json
import os
import re
import subprocess
import sys
import zipfile
from typing import Dict, List, Optional, Tuple

DEFAULT_REPO = 'stackql/stackql'
DEFAULT_WORKFLOW = 'build.yml'
DEFAULT_BRANCH = 'main'
RUN_LOOKUP_LIMIT = '500'

# Each artifact is attached as <artifact>.zip, the zip exactly as GitHub stores it.
BINARY_ARTIFACTS = ['stackql_linux_amd64', 'stackql_linux_arm64']
DEB_ARTIFACTS = ['amd64-artifact-deb', 'arm64-artifact-deb']
# Downloaded for the Windows installer (step 4 of the release process), never uploaded.
WINDOWS_ARTIFACTS = ['stackql_windows_amd64', 'stackql_windows_arm64']


class ReleaseError(Exception):
    pass


def parse_version(raw: str) -> str:
    version = raw[1:] if raw.startswith('v') else raw
    if not re.fullmatch(r'\d+\.\d+\.\d+', version):
        raise ReleaseError(
            f"version must be major.minor.build_number, optionally prefixed with v, got '{raw}'")
    return version


def build_number(version: str) -> int:
    return int(version.rsplit('.', 1)[1])


def sidecar_line(digest_hex: str, filename: str) -> str:
    # sha256sum format: lowercase hex, two spaces, bare filename, LF
    return f'{digest_hex}  {filename}\n'


def plan_asset(local_hex: str, remote_digest: Optional[str]) -> str:
    """Decide what to do with an asset from the digest GitHub recorded for the
    attached bytes: None when the asset is not on the release, '' when it is
    attached but has no recorded digest, otherwise 'sha256:<hex>'."""
    if remote_digest is None:
        return 'upload'
    if remote_digest == f'sha256:{local_hex}':
        return 'skip'
    if remote_digest == '':
        raise ReleaseError(
            'already on the release but GitHub reports no digest for it, so it cannot be '
            'verified against the downloaded bytes; delete it from the release and re-run to replace it')
    raise ReleaseError(
        f'already on the release with digest {remote_digest} but the downloaded artifact is '
        f'sha256:{local_hex}; the release holds different bytes than this build, delete the asset '
        '(and its .sha256) from the release manually if it needs to be replaced')


def sha256_file(path: str) -> str:
    digest = hashlib.sha256()
    with open(path, 'rb') as handle:
        for chunk in iter(lambda: handle.read(1 << 20), b''):
            digest.update(chunk)
    return digest.hexdigest()


def gh(*args: str) -> str:
    completed = subprocess.run(['gh', *args], capture_output=True, text=True)
    if completed.returncode != 0:
        detail = completed.stderr.strip() or completed.stdout.strip()
        raise ReleaseError(f"gh {' '.join(args)} failed: {detail}")
    return completed.stdout


def gh_json(*args: str):
    return json.loads(gh(*args))


def release_assets(repo: str, tag: str) -> Dict[str, str]:
    """name -> sha256 digest GitHub recorded for the attached bytes ('' if none)."""
    try:
        release = gh_json('api', f'repos/{repo}/releases/tags/{tag}')
    except ReleaseError as err:
        raise ReleaseError(f'release {tag} could not be read from {repo} (create it from the tag first): {err}')
    return {asset['name']: (asset.get('digest') or '') for asset in release.get('assets', [])}


def find_run(repo: str, workflow: str, branch: str, number: int) -> dict:
    runs = gh_json('run', 'list', '-R', repo, '-w', workflow, '-b', branch, '-L', RUN_LOOKUP_LIMIT,
                   '--json', 'databaseId,number,headSha,headBranch,conclusion')
    for run in runs:
        if run['number'] == number:
            return run
    raise ReleaseError(
        f'build {number} not found in the last {RUN_LOOKUP_LIMIT} runs of {workflow} on {branch} '
        f'in {repo}; pass --run-id to name the run explicitly')


def get_run(repo: str, run_id: int) -> dict:
    run = gh_json('api', f'repos/{repo}/actions/runs/{run_id}')
    return {
        'databaseId': run['id'],
        'number': run['run_number'],
        'headSha': run['head_sha'],
        'headBranch': run['head_branch'],
        'conclusion': run['conclusion'],
    }


def tag_commit(repo: str, tag: str) -> str:
    return gh_json('api', f'repos/{repo}/commits/{tag}')['sha']


def run_artifacts(repo: str, run_id: int) -> Dict[str, dict]:
    artifacts: Dict[str, dict] = {}
    page = 1
    while True:
        batch = gh_json('api', f'repos/{repo}/actions/runs/{run_id}/artifacts?per_page=100&page={page}')
        found = batch.get('artifacts', [])
        for artifact in found:
            artifacts[artifact['name']] = artifact
        if len(found) < 100:
            return artifacts
        page += 1


def download_artifact(repo: str, artifact: dict, dest: str) -> None:
    name = artifact['name']
    if artifact.get('expired'):
        raise ReleaseError(f"artifact {name} (id {artifact['id']}) has expired, re-run the build workflow")
    with open(dest, 'wb') as handle:
        completed = subprocess.run(
            ['gh', 'api', f"repos/{repo}/actions/artifacts/{artifact['id']}/zip"],
            stdout=handle, stderr=subprocess.PIPE)
    if completed.returncode != 0:
        raise ReleaseError(
            f"could not download artifact {name}: {completed.stderr.decode(errors='replace').strip()}")
    size = os.path.getsize(dest)
    if size != artifact['size_in_bytes']:
        raise ReleaseError(
            f"{dest} is {size} bytes but GitHub reports {artifact['size_in_bytes']} bytes for artifact {name}")
    with zipfile.ZipFile(dest) as archive:
        corrupt = archive.testzip()
    if corrupt is not None:
        raise ReleaseError(f'{dest} failed the zip integrity test at {corrupt}')


def remote_sidecar_hex(repo: str, tag: str, name: str) -> str:
    content = gh('release', 'download', tag, '-R', repo, '-p', name, '-O', '-')
    fields = content.split()
    return fields[0] if fields else ''


def repo_root() -> str:
    return os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))


def attach(args: argparse.Namespace) -> int:
    version = parse_version(args.version)
    tag = f'v{version}'
    number = build_number(version)
    repo = args.repo
    workdir = args.workdir or os.path.join(repo_root(), 'dist', 'release-assets', tag)
    os.makedirs(workdir, exist_ok=True)

    print(f'release         : {repo} {tag}')
    attached = release_assets(repo, tag)
    print(f'attached assets : {", ".join(sorted(attached)) or "(none)"}')

    run = get_run(repo, args.run_id) if args.run_id else find_run(repo, args.workflow, args.branch, number)
    print(f"build {run['number']:<10}: run {run['databaseId']} (branch {run['headBranch']}, "
          f"commit {run['headSha'][:7]}, conclusion {run['conclusion']})")
    if run['number'] != number:
        raise ReleaseError(f"run {run['databaseId']} is build {run['number']}, but {tag} is build {number}")
    if run['conclusion'] != 'success' and not args.run_id:
        raise ReleaseError(
            f"run {run['databaseId']} concluded '{run['conclusion']}', not success; pass --run-id to use it anyway")

    commit = tag_commit(repo, tag)
    if commit != run['headSha']:
        message = f"tag {tag} points at {commit[:7]} but build {run['number']} built {run['headSha'][:7]}"
        if not args.allow_tag_mismatch:
            raise ReleaseError(f'{message}; pass --allow-tag-mismatch to continue anyway')
        print(f'WARNING: {message}')

    artifacts = run_artifacts(repo, run['databaseId'])
    upload_names: List[str] = BINARY_ARTIFACTS + ([] if args.skip_deb else DEB_ARTIFACTS)
    stage_names: List[str] = [] if args.skip_windows else WINDOWS_ARTIFACTS
    missing = [name for name in upload_names + stage_names if name not in artifacts]
    if missing:
        raise ReleaseError(f"artifacts missing from run {run['databaseId']}: {', '.join(missing)}")

    print()
    print(f'downloading artifact zips as GitHub stores them to {workdir}')
    digests: Dict[str, str] = {}
    for name in upload_names + stage_names:
        asset = f'{name}.zip'
        download_artifact(repo, artifacts[name], os.path.join(workdir, asset))
        digests[asset] = sha256_file(os.path.join(workdir, asset))
        print(f'  {digests[asset]}  {asset}')

    print()
    print('plan')
    plan: List[Tuple[str, str]] = []
    for name in upload_names:
        asset = f'{name}.zip'
        sidecar = f'{asset}.sha256'
        local_hex = digests[asset]
        with open(os.path.join(workdir, sidecar), 'w', encoding='utf-8', newline='\n') as handle:
            handle.write(sidecar_line(local_hex, asset))
        try:
            plan.append((asset, plan_asset(local_hex, attached.get(asset))))
        except ReleaseError as err:
            raise ReleaseError(f'{asset}: {err}')
        if sidecar in attached:
            remote_hex = remote_sidecar_hex(repo, tag, sidecar)
            if remote_hex != local_hex:
                raise ReleaseError(
                    f'{sidecar} is already on the release and records {remote_hex}, '
                    f'but the downloaded artifact is {local_hex}')
            plan.append((sidecar, 'skip'))
        else:
            plan.append((sidecar, 'upload'))
    for asset, action in plan:
        print(f'  {action:<7} {asset}')
    for name in stage_names:
        print(f'  staged  {name}.zip (for the Windows installer, not uploaded)')

    if args.dry_run:
        print()
        print('dry run: nothing uploaded')
        return 0

    uploads = [asset for asset, action in plan if action == 'upload']
    print()
    if uploads:
        print('uploading')
        for asset in uploads:
            print(f'  {asset}')
            gh('release', 'upload', tag, os.path.join(workdir, asset), '-R', repo)
    else:
        print('nothing to upload: every asset and sidecar is already on the release')

    print()
    print('verifying the digests GitHub recorded for the attached bytes')
    attached = release_assets(repo, tag)
    for name in upload_names:
        asset = f'{name}.zip'
        remote = attached.get(asset)
        if remote != f'sha256:{digests[asset]}':
            raise ReleaseError(f'{asset}: release holds {remote!r}, the local bytes are sha256:{digests[asset]}')
        if f'{asset}.sha256' not in attached:
            raise ReleaseError(f'{asset}.sha256 is not on the release after upload')
        print(f'  OK  {digests[asset]}  {asset}')
    print()
    print(f'assets and sidecars for {tag} are attached and verified; local copies are in {workdir}')
    return 0


def main(argv: Optional[List[str]] = None) -> int:
    parser = argparse.ArgumentParser(
        description='Attach the Linux build artifacts and their .sha256 sidecars to a stackql release.')
    parser.add_argument('version', help='release version, for example 0.12.778 or v0.12.778')
    parser.add_argument('--run-id', type=int,
                        help='build workflow run id to use instead of resolving it from the build number')
    parser.add_argument('--skip-deb', action='store_true', help='do not attach the Debian package zips')
    parser.add_argument('--skip-windows', action='store_true',
                        help='do not download the unsigned Windows zips for the Windows installer')
    parser.add_argument('--dry-run', action='store_true', help='download, digest and plan, but upload nothing')
    parser.add_argument('--allow-tag-mismatch', action='store_true',
                        help='continue when the tag does not point at the commit the build run built')
    parser.add_argument('--workdir', help='download directory (default: dist/release-assets/v<version> under the repo root)')
    parser.add_argument('--repo', default=DEFAULT_REPO, help=f'GitHub repository (default: {DEFAULT_REPO})')
    parser.add_argument('--workflow', default=DEFAULT_WORKFLOW, help=f'build workflow file (default: {DEFAULT_WORKFLOW})')
    parser.add_argument('--branch', default=DEFAULT_BRANCH, help=f'branch the build ran on (default: {DEFAULT_BRANCH})')
    args = parser.parse_args(argv)
    try:
        return attach(args)
    except ReleaseError as err:
        print(f'error: {err}', file=sys.stderr)
        return 1


if __name__ == '__main__':
    sys.exit(main())
