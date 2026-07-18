"""
stackql-mcp-server - uvx/pip-able launcher for the StackQL MCP server.

On first run, downloads the platform's signed .mcpb bundle from the GitHub
release pinned in platforms.json, verifies its sha256, extracts the stackql
binary into ~/.stackql/mcp-server-bin/<version>/ (a cache shared with the
@stackql/mcp-server npm wrapper), then runs it as an MCP stdio server.

Extra arguments are passed through to stackql after the standard MCP args:

  uvx stackql-mcp-server --auth='{"github":{"type":"null_auth"}}'

Env overrides:
  STACKQL_MCP_BIN     path to an existing stackql binary (skips download)
  STACKQL_MCP_BUNDLE  path to a local .mcpb to extract from (CI/testing;
                      skips download and sha verification)

All diagnostics go to stderr - stdout belongs to the MCP protocol.
Stdlib only, no dependencies.
"""
from __future__ import annotations

import hashlib
import json
import os
import platform as _platform
import subprocess
import sys
import urllib.request
import zipfile
from importlib.resources import files
from io import BytesIO
from pathlib import Path


def _log(msg: str) -> None:
    print(f"stackql-mcp: {msg}", file=sys.stderr, flush=True)


def _platform_key() -> str | None:
    machine = _platform.machine().lower()
    if sys.platform.startswith("linux"):
        if machine in ("x86_64", "amd64"):
            return "linux-x64"
        if machine in ("arm64", "aarch64"):
            return "linux-arm64"
        return None
    if sys.platform == "win32":
        return "windows-x64" if machine in ("x86_64", "amd64") else None
    if sys.platform == "darwin":
        return "darwin-universal"  # universal binary covers x64 + arm64
    return None


def _load_manifest() -> dict:
    return json.loads(files(__package__).joinpath("platforms.json").read_text())


def _download(url: str, version: str) -> bytes:
    # Distinct per-vector UA so the download proxy can attribute traffic to the
    # PyPI wrapper (vs the npm one) and the version that fetched.
    user_agent = f"stackql-mcp-server-py/{version}"
    req = urllib.request.Request(url, headers={"User-Agent": user_agent})
    with urllib.request.urlopen(req) as resp:  # follows redirects
        return resp.read()


def _extract_binary(bundle: bytes, entry_name: str, dest: Path) -> None:
    with zipfile.ZipFile(BytesIO(bundle)) as zf:
        data = zf.read(entry_name)
    dest.parent.mkdir(parents=True, exist_ok=True)
    # write-then-rename so a concurrent first run cannot see a half-written binary
    tmp = dest.with_name(f"{dest.name}.tmp-{os.getpid()}")
    tmp.write_bytes(data)
    tmp.chmod(0o755)
    tmp.replace(dest)


def _ensure_binary() -> str:
    override = os.environ.get("STACKQL_MCP_BIN")
    if override:
        return override

    key = _platform_key()
    if key is None:
        _log(f"unsupported platform: {sys.platform}/{_platform.machine()}")
        raise SystemExit(1)

    manifest = _load_manifest()
    info = manifest["platforms"][key]
    bin_name = "stackql.exe" if key == "windows-x64" else "stackql"
    bin_path = (
        Path.home() / ".stackql" / "mcp-server-bin" / manifest["version"] / key / bin_name
    )
    if bin_path.exists():
        return str(bin_path)

    local_bundle = os.environ.get("STACKQL_MCP_BUNDLE")
    if local_bundle:
        bundle = Path(local_bundle).read_bytes()
    else:
        url = f"{manifest['baseUrl']}/{info['bundle']}"
        _log(f"downloading {info['bundle']} (first run only) ...")
        bundle = _download(url, manifest["version"])
        digest = hashlib.sha256(bundle).hexdigest()
        if digest != info["sha256"]:
            _log(f"sha256 mismatch for {info['bundle']}")
            _log(f"  expected {info['sha256']}")
            _log(f"  got      {digest}")
            raise SystemExit(1)
    _extract_binary(bundle, f"server/{bin_name}", bin_path)
    _log(f"installed {bin_path}")
    return str(bin_path)


def main() -> None:
    bin_path = _ensure_binary()
    # approot and the audit sink must not depend on the cwd: MCP clients may
    # launch this with cwd '/' (read-only on macOS). Later duplicate flags
    # win, so user-passed overrides still take effect.
    args = [
        bin_path,
        "mcp",
        "--mcp.server.type=stdio",
        "--approot", str(Path.home() / ".stackql"),
        "--mcp.config", json.dumps({"server": {"audit": {"disabled": True}}}),
        *sys.argv[1:],
    ]
    if os.name != "nt":
        os.execv(bin_path, args)  # replace this process; signals flow naturally
    raise SystemExit(subprocess.call(args))
