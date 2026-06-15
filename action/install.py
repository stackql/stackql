#!/usr/bin/env python3
"""
Installer for the Setup StackQL MCP Server action. Stdlib only.

Downloads the platform's .mcpb bundle from the stackql GitHub release (or
uses a local bundle via STACKQL_SETUP_BUNDLE), verifies the sha256 against
the published .sha256 asset, extracts the stackql binary, and writes:

  GITHUB_OUTPUT: binary-path, mcp-config (mcpServers JSON, single line)
  GITHUB_ENV:    STACKQL_MCP_BIN (so the npm/pypi wrappers skip downloading)
  GITHUB_PATH:   the install directory

Runs outside Actions too (for local testing): set RUNNER_OS/RUNNER_ARCH or
let it fall back to platform detection; outputs print to stdout when the
GITHUB_* files are absent.
"""
from __future__ import annotations

import hashlib
import json
import os
import platform
import sys
import urllib.request
import zipfile
from io import BytesIO
from pathlib import Path

RELEASE_BASE = "https://github.com/stackql/stackql/releases"


def log(msg: str) -> None:
    print(f"setup-stackql-mcp: {msg}", flush=True)


def fail(msg: str) -> None:
    print(f"::error::setup-stackql-mcp: {msg}", flush=True)
    sys.exit(1)


def platform_key() -> str:
    os_name = os.environ.get("RUNNER_OS", "").lower() or sys.platform
    arch = os.environ.get("RUNNER_ARCH", "").lower() or platform.machine().lower()
    is_arm = arch in ("arm64", "aarch64")
    if os_name.startswith("linux"):
        return "linux-arm64" if is_arm else "linux-x64"
    if os_name.startswith(("windows", "win32")):
        return "windows-x64"
    if os_name.startswith(("macos", "darwin")):
        return "darwin-universal"
    fail(f"unsupported platform: {os_name}/{arch}")
    raise SystemExit  # unreachable


def fetch(url: str) -> bytes:
    req = urllib.request.Request(url, headers={"User-Agent": "setup-stackql-mcp"})
    with urllib.request.urlopen(req) as resp:
        return resp.read()


def write_kv(env_file_var: str, lines: list[str]) -> None:
    path = os.environ.get(env_file_var)
    if path:
        with open(path, "a", encoding="utf-8") as fh:
            fh.write("\n".join(lines) + "\n")
    else:
        print(f"[{env_file_var}]")
        print("\n".join(lines))


def main() -> None:
    version = os.environ.get("STACKQL_SETUP_VERSION", "latest").lstrip("v") or "latest"
    mode = os.environ.get("STACKQL_SETUP_MODE", "read_only")
    auth = os.environ.get("STACKQL_SETUP_AUTH", "")
    local_bundle = os.environ.get("STACKQL_SETUP_BUNDLE", "")

    key = platform_key()
    bin_name = "stackql.exe" if key == "windows-x64" else "stackql"
    bundle_name = f"stackql-mcp-{key}.mcpb"

    if local_bundle:
        log(f"installing from local bundle {local_bundle}")
        bundle = Path(local_bundle).read_bytes()
    else:
        base = (
            f"{RELEASE_BASE}/latest/download"
            if version == "latest"
            else f"{RELEASE_BASE}/download/v{version}"
        )
        log(f"downloading {base}/{bundle_name}")
        bundle = fetch(f"{base}/{bundle_name}")
        expected = fetch(f"{base}/{bundle_name}.sha256").decode().split()[0]
        digest = hashlib.sha256(bundle).hexdigest()
        if digest != expected:
            fail(f"sha256 mismatch for {bundle_name}: expected {expected}, got {digest}")
        log(f"sha256 verified: {digest}")

    install_dir = Path(
        os.environ.get("RUNNER_TEMP") or Path.home() / ".stackql" / "action"
    ) / "stackql-mcp-bin"
    install_dir.mkdir(parents=True, exist_ok=True)
    bin_path = install_dir / bin_name
    with zipfile.ZipFile(BytesIO(bundle)) as zf:
        bin_path.write_bytes(zf.read(f"server/{bin_name}"))
    bin_path.chmod(0o755)
    log(f"installed {bin_path}")

    approot = str(Path.home() / ".stackql")
    args = [
        "mcp",
        "--mcp.server.type=stdio",
        "--approot", approot,
        "--mcp.config", json.dumps({"server": {"mode": mode, "audit": {"disabled": True}}}),
    ]
    if auth:
        args += ["--auth", auth]
    mcp_config = json.dumps(
        {"mcpServers": {"stackql": {"command": str(bin_path), "args": args}}}
    )

    write_kv("GITHUB_OUTPUT", [
        f"binary-path={bin_path}",
        f"mcp-config={mcp_config}",
    ])
    write_kv("GITHUB_ENV", [f"STACKQL_MCP_BIN={bin_path}"])
    write_kv("GITHUB_PATH", [str(install_dir)])


if __name__ == "__main__":
    main()
