#!/usr/bin/env python3
from __future__ import annotations

import json
import re
import sys
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[3]
PACKAGE_ROOT = REPO_ROOT / "packaging" / "openai-plugin"
PLUGIN_ROOT = PACKAGE_ROOT / "plugins" / "stackql"


def load_json(path: Path) -> dict:
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as err:
        raise ValueError(f"{path}: {err}") from err


def require(condition: bool, message: str) -> None:
    if not condition:
        raise ValueError(message)


def validate() -> None:
    marketplace = load_json(REPO_ROOT / ".agents" / "plugins" / "marketplace.json")
    require(marketplace.get("name") == "stackql", "marketplace name must be stackql")
    entries = marketplace.get("plugins", [])
    require(len(entries) == 1, "marketplace must contain one plugin")
    entry = entries[0]
    require(entry.get("name") == "stackql", "marketplace plugin name must be stackql")
    require(entry.get("policy") == {
        "installation": "AVAILABLE",
        "authentication": "ON_INSTALL",
    }, "marketplace policy is invalid")
    source = entry.get("source", {})
    require(source.get("source") == "local", "plugin source must be local")
    source_path = (REPO_ROOT / source.get("path", "")).resolve()
    require(source_path == PLUGIN_ROOT.resolve(), "marketplace source path is invalid")

    manifest = load_json(PLUGIN_ROOT / ".codex-plugin" / "plugin.json")
    require(manifest.get("name") == PLUGIN_ROOT.name, "manifest name must match its directory")
    version = manifest.get("version", "")
    require(re.fullmatch(r"\d+\.\d+\.\d+", version) is not None, "manifest version is invalid")
    require(manifest.get("mcpServers") == "./.mcp.json", "manifest MCP path is invalid")
    require(manifest.get("interface", {}).get("category") == entry.get("category"),
            "manifest and marketplace categories differ")

    mcp = load_json(PLUGIN_ROOT / ".mcp.json")
    servers = mcp.get("mcpServers", {})
    require(set(servers) == {"stackql"}, "MCP config must contain only the stackql server")
    server = servers["stackql"]
    require(server == {
        "command": "node",
        "args": ["./bin/stackql-mcp.js"],
        "cwd": ".",
    }, "MCP config must launch the bundled stdio entrypoint")

    launcher = (PLUGIN_ROOT / "bin" / "stackql-mcp.js").read_text(encoding="utf-8")
    match = re.search(r'"@stackql/mcp-server@(\d+\.\d+\.\d+)"', launcher)
    require(match is not None, "launcher must pin @stackql/mcp-server")
    require(match.group(1) == version, "launcher and manifest versions differ")
    require('"--approot"' in launcher, "launcher must configure the application root")
    require('"--env.file"' in launcher, "launcher must configure the credential env file")
    require('"--configfile"' in launcher, "launcher must configure .stackqlrc")
    require('"--auth"' not in launcher, "launcher must use provider default authentication")


if __name__ == "__main__":
    try:
        validate()
    except ValueError as err:
        print(f"validation failed: {err}", file=sys.stderr)
        raise SystemExit(1) from err
    print("OpenAI stdio plugin validation passed")
