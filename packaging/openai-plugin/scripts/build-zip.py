#!/usr/bin/env python3
"""Build (or check) the ChatGPT plugin directory ZIP for the stdio plugin.

The archive holds the plugin folder's contents at the ZIP root (plugin.json,
mcp.json, skills/, assets/, server/ and the Codex overlay), with OS litter left
out and the entries the directory rejects (hooks, app references, reviewer
credentials, package installs) refused outright. Entry order and timestamps are
fixed so the same tree always produces the same bytes.

  build-zip.py --version X.Y.Z [--out DIR]   write DIR/stackql-plugin-X.Y.Z.zip
  build-zip.py --check PATH                  verify an existing ZIP
"""
from __future__ import annotations

import argparse
import json
import sys
import zipfile
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[3]
PACKAGE_ROOT = REPO_ROOT / "packaging" / "openai-plugin"
PLUGIN_ROOT = PACKAGE_ROOT / "plugins" / "stackql"
DEFAULT_OUT = PACKAGE_ROOT / "dist"

OS_LITTER = {".DS_Store", "Thumbs.db", "desktop.ini", "__MACOSX", "__pycache__"}
REJECTED = {"hooks", ".app.json", "test_credentials", "package.json", "package-lock.json", "node_modules"}
REQUIRED = (
    "plugin.json",
    "mcp.json",
    ".codex-plugin/plugin.json",
    ".mcp.json",
    "README.md",
    "LICENSE",
    "assets/icon.png",
    "skills/stackql-setup/SKILL.md",
    "server/bin/stackql-mcp.js",
    "server/platforms.json",
)
ALLOWED_TOP_LEVEL = {"plugin.json", "mcp.json", ".codex-plugin", ".mcp.json", "README.md", "LICENSE",
                     "assets", "skills", "server"}
MAX_ZIP_BYTES = 100 * 1024 * 1024
FIXED_TIMESTAMP = (2026, 1, 1, 0, 0, 0)


def fail(message: str) -> None:
    print(f"error: {message}", file=sys.stderr)
    raise SystemExit(1)


def plugin_files() -> list[Path]:
    files = []
    for path in sorted(PLUGIN_ROOT.rglob("*")):
        rel = path.relative_to(PLUGIN_ROOT)
        if OS_LITTER.intersection(rel.parts) or path.suffix == ".pyc":
            continue
        if REJECTED.intersection(rel.parts):
            fail(f"{rel} cannot be submitted to the plugin directory")
        if path.is_file():
            files.append(path)
    return files


def build(version: str, out_dir: Path) -> Path:
    manifest = json.loads((PLUGIN_ROOT / "plugin.json").read_text(encoding="utf-8"))
    if manifest.get("version") != version:
        fail(f"plugin.json version is {manifest.get('version')}, not {version}")
    out_dir.mkdir(parents=True, exist_ok=True)
    target = out_dir / f"stackql-plugin-{version}.zip"
    with zipfile.ZipFile(target, "w", compression=zipfile.ZIP_DEFLATED) as archive:
        for path in plugin_files():
            info = zipfile.ZipInfo(path.relative_to(PLUGIN_ROOT).as_posix(), date_time=FIXED_TIMESTAMP)
            info.compress_type = zipfile.ZIP_DEFLATED
            info.external_attr = 0o644 << 16
            archive.writestr(info, path.read_bytes())
    check(target)
    print(f"wrote {target} ({target.stat().st_size} bytes)")
    return target


def check(target: Path) -> None:
    if not target.is_file():
        fail(f"{target} does not exist")
    size = target.stat().st_size
    if size > MAX_ZIP_BYTES:
        fail(f"{target} is {size} bytes; the directory cap is {MAX_ZIP_BYTES}")
    with zipfile.ZipFile(target) as archive:
        names = archive.namelist()
        for name in names:
            parts = name.split("/")
            if parts[0] not in ALLOWED_TOP_LEVEL:
                fail(f"unexpected entry at the ZIP root: {name}")
            if REJECTED.intersection(parts) or OS_LITTER.intersection(parts):
                fail(f"{name} must not be in the ZIP")
        for required in REQUIRED:
            if required not in names:
                fail(f"{required} is missing from the ZIP")
        manifest = json.loads(archive.read("plugin.json"))
        if not manifest.get("name") == "stackql":
            fail("plugin.json inside the ZIP is not the stackql plugin")
    print(f"{target.name}: {len(names)} entries, {size} bytes, layout ok")


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("--version", help="plugin version the ZIP must carry (X.Y.Z)")
    parser.add_argument("--out", type=Path, default=DEFAULT_OUT, help="output directory")
    parser.add_argument("--check", type=Path, help="verify an existing ZIP instead of building")
    args = parser.parse_args()
    if args.check:
        check(args.check)
        return
    if not args.version:
        parser.error("--version is required to build")
    build(args.version, args.out)


if __name__ == "__main__":
    main()
