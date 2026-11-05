#!/usr/bin/env python3
"""Validate the Claude plugin bundle under packaging/claude-plugin.

Mirrors the checks Anthropic's directory runs on a plugin bundle (manifest,
README and licence, file limits, how the local MCP server is launched) and the
repository's single-source invariants: the plugin vendors the npm launcher,
pins and onboarding skill byte-for-byte, and its version equals the npm
wrapper's. `claude plugin validate` covers manifest syntax; this covers the
directory rules and the vendoring contract.
"""
from __future__ import annotations

import json
import re
import sys
from pathlib import Path


MCPB_ROOT = Path(__file__).resolve().parents[1]
REPO_ROOT = MCPB_ROOT.parents[1]
PLUGIN_ROOT = REPO_ROOT / "packaging" / "claude-plugin"
NPM_LAUNCHER = MCPB_ROOT / "npm" / "bin" / "stackql-mcp.js"
SKILL_SOURCE = MCPB_ROOT / "skills" / "stackql-setup" / "SKILL.md"
ICON_SOURCE = MCPB_ROOT / "assets" / "icon.png"
LICENSE_SOURCE = MCPB_ROOT / "LICENSE"

LAUNCHER_REL = "server/bin/stackql-mcp.js"
SKILL_REL = "skills/stackql-setup/SKILL.md"

# Held for a reviewer or blocked by the directory validator.
FORBIDDEN_NAMES = {
    ".DS_Store", "Thumbs.db", "desktop.ini", "__MACOSX",
    "package.json", "package-lock.json", "npm-shrinkwrap.json", "bun.lock", "bun.lockb",
    "node_modules", ".npmrc", "bunfig.toml", "uv.toml",
}
IMAGE_SUFFIXES = {".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg"}
FONT_SUFFIXES = {".woff", ".woff2", ".ttf", ".otf"}
TEXT_SUFFIXES = {".md", ".json", ".js", ".txt", ".yaml", ".yml", ".sh", ".py", ""}
MAX_TEXT_BYTES = 256 * 1024
MAX_FILES = 512
HTTPS_RE = re.compile(r"^https://\S+$")
VERSION_RE = re.compile(r"^\d+\.\d+\.\d+$")
SHA256_RE = re.compile(r"^[0-9a-f]{64}$")
NAME_RE = re.compile(r"^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$")
PLATFORM_KEYS = {"linux-x64", "linux-arm64", "windows-x64", "windows-arm64", "darwin-universal"}


def load_json(path: Path) -> dict:
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as err:
        raise ValueError(f"{path}: {err}") from err


def require(condition: bool, message: str) -> None:
    if not condition:
        raise ValueError(message)


def word_count_outside_code(markdown: str) -> int:
    in_code = False
    words = 0
    for line in markdown.splitlines():
        if line.strip().startswith("```"):
            in_code = not in_code
            continue
        if not in_code:
            words += len(line.split())
    return words


def validate_manifest() -> dict:
    manifest = load_json(PLUGIN_ROOT / ".claude-plugin" / "plugin.json")
    name = manifest.get("name", "")
    require(name == "stackql", "plugin name must be stackql")
    require(NAME_RE.match(name) is not None and len(name) <= 64, "plugin name must be lowercase kebab-case")
    require(manifest.get("displayName") == "StackQL", "displayName must be StackQL")
    version = manifest.get("version", "")
    require(VERSION_RE.match(version) is not None, "manifest version is invalid")
    require(isinstance(manifest.get("description"), str) and manifest["description"].strip(),
            "manifest description is required")
    author = manifest.get("author", {})
    require(isinstance(author.get("name"), str) and author["name"], "author.name is required")
    require(manifest.get("license") == "MIT" or (PLUGIN_ROOT / "LICENSE").is_file(),
            "a license field or LICENSE file is required")
    require(HTTPS_RE.match(manifest.get("homepage", "")) is not None, "homepage must be an https URL")
    for key in ("documentationUrl", "supportUrl", "privacyPolicyUrl", "termsOfServiceUrl"):
        require(HTTPS_RE.match(manifest.get(key, "")) is not None, f"{key} must be an https URL")
    icon = manifest.get("icon", "")
    require(icon.startswith("./") and (PLUGIN_ROOT / icon).is_file(), "icon must be a ./-relative file in the plugin")
    require((PLUGIN_ROOT / icon).read_bytes() == ICON_SOURCE.read_bytes(),
            "icon differs from packaging/mcpb/assets/icon.png")

    servers = manifest.get("mcpServers")
    require(isinstance(servers, dict) and set(servers) == {"stackql"},
            "mcpServers must be an inline map with only the stackql server")
    require(servers["stackql"] == {
        "command": "node",
        "args": [f"${{CLAUDE_PLUGIN_ROOT}}/{LAUNCHER_REL}"],
    }, "the stackql server must run node ${CLAUDE_PLUGIN_ROOT}/" + LAUNCHER_REL + " with no other arguments")
    for key in ("hooks", "agents", "commands", "lspServers", "userConfig"):
        require(key not in manifest, f"manifest must not declare {key}")
    require(not (PLUGIN_ROOT / ".mcp.json").exists(),
            ".mcp.json must not exist; the server is declared inline in plugin.json")
    return manifest


def validate_payload(version: str) -> None:
    launcher = PLUGIN_ROOT / LAUNCHER_REL
    require(launcher.is_file(), f"{LAUNCHER_REL} is missing (run: make claude-plugin-manifest VERSION={version})")
    require(launcher.read_bytes() == NPM_LAUNCHER.read_bytes(),
            f"{LAUNCHER_REL} differs from packaging/mcpb/npm/bin/stackql-mcp.js (the npm launcher is the single source)")
    text = launcher.read_text(encoding="utf-8")
    modules = re.findall(r'require\("([^"]*)"\)', text)
    require(all(m in ("child_process", "crypto", "fs", "os", "path", "zlib", "../platforms.json") for m in modules),
            f"launcher must only load Node.js built-ins and its pins, found {modules}")

    pins = load_json(PLUGIN_ROOT / "server" / "platforms.json")
    require(pins.get("version") == version, "server/platforms.json version differs from the manifest")
    require(set(pins.get("platforms", {})) == PLATFORM_KEYS, "server/platforms.json must pin all five platforms")
    for key, info in pins["platforms"].items():
        require(info.get("bundle") == f"stackql-mcp-{key}.mcpb", f"platforms.json bundle name for {key} is wrong")
        require(SHA256_RE.match(info.get("sha256", "")) is not None, f"platforms.json sha256 for {key} is invalid")
    require(pins.get("baseUrl", "").startswith("https://"), "platforms.json baseUrl must be https")

    npm_manifest = load_json(MCPB_ROOT / "npm" / "package.json")
    require(npm_manifest.get("version") == version,
            "plugin version and packaging/mcpb/npm/package.json version differ")

    skill = PLUGIN_ROOT / SKILL_REL
    require(skill.is_file(), f"{SKILL_REL} is missing")
    require(skill.read_bytes() == SKILL_SOURCE.read_bytes(),
            f"{SKILL_REL} differs from packaging/mcpb/skills/stackql-setup/SKILL.md")
    skill_text = skill.read_text(encoding="utf-8")
    require(skill_text.startswith("---\n") and "\nname: stackql-setup\n" in skill_text
            and "\ndescription: " in skill_text, "SKILL.md front matter needs name and description")

    require((PLUGIN_ROOT / "LICENSE").read_bytes() == LICENSE_SOURCE.read_bytes(),
            "plugin LICENSE differs from packaging/mcpb/LICENSE")
    readme = PLUGIN_ROOT / "README.md"
    require(readme.is_file(), "README.md is missing; the directory shows it as the listing description")
    require(word_count_outside_code(readme.read_text(encoding="utf-8")) >= 40,
            "README.md needs at least 40 words outside code blocks")


def validate_tree() -> None:
    require(not (PLUGIN_ROOT / "bin").exists(),
            "a top-level bin/ directory stops claude.ai and Cowork from installing the plugin")
    files = [p for p in PLUGIN_ROOT.rglob("*") if p.is_file()]
    require(len(files) <= MAX_FILES, f"plugin holds {len(files)} files; the directory limit is {MAX_FILES}")
    for path in files:
        rel = path.relative_to(PLUGIN_ROOT)
        require(not FORBIDDEN_NAMES.intersection(rel.parts), f"{rel} cannot ship in the plugin")
        require(not path.is_symlink(), f"{rel} is a symbolic link")
        suffix = path.suffix.lower()
        if suffix in IMAGE_SUFFIXES or suffix in FONT_SUFFIXES:
            continue
        require(suffix in TEXT_SUFFIXES, f"{rel}: only text files, images and fonts may ship")
        require(path.stat().st_size <= MAX_TEXT_BYTES, f"{rel} exceeds 256 KiB")
        require(suffix not in (".mcpb", ".dxt"), f"{rel}: bundles are not accepted")


def validate() -> None:
    manifest = validate_manifest()
    validate_payload(manifest["version"])
    validate_tree()


if __name__ == "__main__":
    try:
        validate()
    except ValueError as err:
        print(f"validation failed: {err}", file=sys.stderr)
        raise SystemExit(1) from err
    print("Claude plugin validation passed")
