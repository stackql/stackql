#!/usr/bin/env python3
"""Validate the ChatGPT/Codex stdio plugin package.

Checks the OpenAI plugin directory rules the portal validator enforces (listing
field limits, review test cases, onboarding skill, ZIP content policy) and the
repository's single-source invariants: the plugin vendors the npm launcher,
pins and onboarding skill byte-for-byte, the Codex overlay is derived from the
portable manifest, and every version agrees with packaging/mcpb/npm/package.json.
"""
from __future__ import annotations

import json
import re
import sys
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[3]
PACKAGE_ROOT = REPO_ROOT / "packaging" / "openai-plugin"
PLUGIN_ROOT = PACKAGE_ROOT / "plugins" / "stackql"
MCPB_ROOT = REPO_ROOT / "packaging" / "mcpb"
NPM_LAUNCHER = MCPB_ROOT / "npm" / "bin" / "stackql-mcp.js"
SKILL_SOURCE = MCPB_ROOT / "skills" / "stackql-setup" / "SKILL.md"
ICON_SOURCE = MCPB_ROOT / "assets" / "icon.png"
LICENSE_SOURCE = MCPB_ROOT / "LICENSE"

PORTABLE_SCHEMA = "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json"
MCP_SCHEMA = "https://agent-plugins.org/schemas/1.0.0/mcp.schema.json"
LAUNCHER_REL = "server/bin/stackql-mcp.js"
SKILL_REL = "skills/stackql-setup/SKILL.md"

# Entries the directory rejects inside a submitted ZIP, plus OS litter.
FORBIDDEN_NAMES = {
    ".DS_Store", "Thumbs.db", "desktop.ini", "__MACOSX",
    "test_credentials", ".app.json", "hooks",
    "package.json", "package-lock.json", "node_modules", ".npmrc",
}
MAX_FILE_BYTES = 5 * 1024 * 1024
HTTPS_RE = re.compile(r"^https://\S+$")
VERSION_RE = re.compile(r"^\d+\.\d+\.\d+$")
SHA256_RE = re.compile(r"^[0-9a-f]{64}$")
PLATFORM_KEYS = {"linux-x64", "linux-arm64", "windows-x64", "windows-arm64", "darwin-universal"}


def load_json(path: Path) -> dict:
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as err:
        raise ValueError(f"{path}: {err}") from err


def require(condition: bool, message: str) -> None:
    if not condition:
        raise ValueError(message)


def warn(message: str) -> None:
    print(f"warning: {message}", file=sys.stderr)


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


def skill_frontmatter(path: Path) -> dict:
    text = path.read_text(encoding="utf-8")
    require(text.startswith("---\n"), f"{path}: SKILL.md must start with YAML front matter")
    end = text.find("\n---\n", 4)
    require(end > 0, f"{path}: unterminated front matter")
    fields: dict[str, str] = {}
    for line in text[4:end].splitlines():
        key, sep, value = line.partition(":")
        if sep:
            fields[key.strip()] = value.strip()
    require(fields.get("name") == path.parent.name,
            f"{path}: front matter name must match the skill directory")
    require(len(fields.get("description", "")) > 20, f"{path}: front matter needs a description")
    return fields


def validate_marketplace() -> dict:
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
    return entry


def validate_interface(interface: dict) -> None:
    limits = {
        "displayName": 30,
        "shortDescription": 30,
        "longDescription": 4000,
        "developerName": 80,
    }
    for key, limit in limits.items():
        value = interface.get(key)
        require(isinstance(value, str) and value.strip(), f"interface.{key} is required")
        require(len(value) <= limit, f"interface.{key} exceeds {limit} characters ({len(value)})")
    require(isinstance(interface.get("category"), str) and interface["category"],
            "interface.category is required")
    capabilities = interface.get("capabilities")
    require(isinstance(capabilities, list) and len(capabilities) <= 20,
            "interface.capabilities must be a list of at most 20 labels")
    for label in capabilities:
        require(isinstance(label, str) and 0 < len(label) <= 120,
                "interface.capabilities labels must be 1 to 120 characters")
    for key in ("websiteURL", "supportURL", "privacyPolicyURL", "termsOfServiceURL"):
        value = interface.get(key, "")
        require(HTTPS_RE.match(value) is not None and len(value) <= 1024,
                f"interface.{key} must be an https URL (required for review)")
    prompts = interface.get("defaultPrompt", [])
    if isinstance(prompts, str):
        prompts = [prompts]
    require(isinstance(prompts, list) and 0 < len(prompts) <= 3,
            "interface.defaultPrompt must hold 1 to 3 prompts")
    require(len(set(prompts)) == len(prompts), "interface.defaultPrompt entries must be unique")
    for prompt in prompts:
        require(isinstance(prompt, str) and 0 < len(prompt) <= 128,
                "interface.defaultPrompt entries must be 1 to 128 characters")
        require("@" not in prompt, "interface.defaultPrompt entries must not mention apps")
    for key in ("composerIcon", "logo"):
        icon_ref = interface.get(key, "")
        require(icon_ref.startswith("./"), f"interface.{key} must be a ./-relative path")
        require((PLUGIN_ROOT / icon_ref).is_file(), f"interface.{key} target is missing")
    for key in ("brandColor", "brandColorDark"):
        if key in interface:
            require(re.fullmatch(r"#[0-9A-Fa-f]{6}", interface[key]) is not None,
                    f"interface.{key} must be #RRGGBB")


def validate_review(review: dict) -> None:
    cases = review.get("test_cases")
    require(isinstance(cases, dict), "review.test_cases is required for MCP review")
    positive = cases.get("positive", [])
    negative = cases.get("negative", [])
    require(len(positive) == 5, f"review.test_cases.positive must hold exactly 5 cases ({len(positive)})")
    require(len(negative) == 3, f"review.test_cases.negative must hold exactly 3 cases ({len(negative)})")
    for case in positive:
        for key in ("description", "prompt", "tools_triggered", "expected_behavior"):
            require(isinstance(case.get(key), str) and case[key].strip(),
                    f"positive test case {case.get('description')!r} needs {key}")
        require(len(case["description"]) <= 4000, "positive test case description too long")
    for case in negative:
        for key in ("description", "prompt"):
            require(isinstance(case.get(key), str) and case[key].strip(),
                    f"negative test case {case.get('description')!r} needs {key}")
    demo = review.get("demo_recording_url")
    if demo is None:
        warn("review.demo_recording_url is not set; it is required before submitting for MCP review")
    else:
        require(HTTPS_RE.match(demo) is not None, "review.demo_recording_url must be an https URL")
    if "commerce" in review:
        require(isinstance(review["commerce"], bool), "review.commerce must be a boolean")


def validate_portable_manifest() -> dict:
    manifest = load_json(PLUGIN_ROOT / "plugin.json")
    require(manifest.get("$schema") == PORTABLE_SCHEMA, "plugin.json $schema must be the Agent Plugins schema")
    require(manifest.get("name") == PLUGIN_ROOT.name, "manifest name must match its directory")
    require(re.fullmatch(r"[a-z0-9](?:[a-z0-9-]*[a-z0-9])?", manifest["name"]) is not None
            and len(manifest["name"]) <= 64, "manifest name must be lowercase letters, digits and hyphens")
    version = manifest.get("version", "")
    require(VERSION_RE.match(version) is not None, "manifest version is invalid")
    description = manifest.get("description", "")
    require(isinstance(description, str) and 0 < len(description) <= 4000, "manifest description is invalid")
    author = manifest.get("author", {})
    require(isinstance(author.get("name"), str) and author["name"], "manifest author.name is required")
    for key, limit in (("name", 120), ("email", 320), ("url", 2048)):
        if key in author:
            require(len(author[key]) <= limit, f"author.{key} exceeds {limit} characters")
    for key in ("homepage", "repository"):
        require(HTTPS_RE.match(manifest.get(key, "")) is not None, f"manifest {key} must be an https URL")
    require(manifest.get("license") == "MIT", "manifest license must be MIT")
    for key in ("skills", "mcpServers", "interface", "review", "publication", "onboardingSkill"):
        require(key not in manifest,
                f"plugin.json must not set {key} at the root; it belongs under extensions.com.openai")

    openai = manifest.get("extensions", {}).get("com.openai")
    require(isinstance(openai, dict), "plugin.json needs extensions.com.openai")
    require(openai.get("requires_local_executor") is True,
            "extensions.com.openai.requires_local_executor must be true for a local stdio server")
    skill_ref = openai.get("onboardingSkill", "")
    require(skill_ref == f"./{SKILL_REL}", f"onboardingSkill must point at ./{SKILL_REL}")
    validate_interface(openai.get("interface", {}))
    validate_review(openai.get("review", {}))
    publication = openai.get("publication", {})
    notes = publication.get("release_notes", "")
    require(isinstance(notes, str) and notes.strip(), "publication.release_notes is required")
    for key in ("apps", "hooks"):
        require(key not in openai, f"extensions.com.openai.{key} cannot be submitted to the directory")
    return manifest


def validate_codex_overlay(portable: dict) -> None:
    """The overlay is rendered from plugin.json; prove it has not drifted."""
    overlay = load_json(PLUGIN_ROOT / ".codex-plugin" / "plugin.json")
    openai = portable["extensions"]["com.openai"]
    for key in ("name", "version", "description", "author", "homepage", "repository", "license", "keywords"):
        require(overlay.get(key) == portable.get(key), f".codex-plugin/plugin.json {key} differs from plugin.json")
    require(overlay.get("skills") == "./skills/", ".codex-plugin/plugin.json skills must be ./skills/")
    require(overlay.get("mcpServers") == "./.mcp.json", ".codex-plugin/plugin.json mcpServers must be ./.mcp.json")
    require(overlay.get("requires_local_executor") is True,
            ".codex-plugin/plugin.json requires_local_executor must be true")
    require(overlay.get("interface") == openai["interface"],
            ".codex-plugin/plugin.json interface differs from plugin.json")
    expected = {k: v for k, v in openai.items() if k not in ("interface", "requires_local_executor")}
    require(overlay.get("extensions", {}).get("com.openai") == expected,
            ".codex-plugin/plugin.json extensions.com.openai differs from plugin.json "
            "(run: make openai-plugin-manifest VERSION=<version>)")


def validate_mcp_configs() -> None:
    portable = load_json(PLUGIN_ROOT / "mcp.json")
    require(portable.get("$schema") == MCP_SCHEMA, "mcp.json $schema must be the Agent Plugins MCP schema")
    require(set(portable.get("mcpServers", {})) == {"stackql"}, "mcp.json must contain only the stackql server")
    require(portable["mcpServers"]["stackql"] == {
        "type": "stdio",
        "command": "node",
        "args": [f"${{PLUGIN_ROOT}}/{LAUNCHER_REL}"],
        "cwd": "${PLUGIN_ROOT}",
    }, "mcp.json must launch the vendored launcher over stdio with ${PLUGIN_ROOT} paths")

    codex = load_json(PLUGIN_ROOT / ".mcp.json")
    require(set(codex.get("mcpServers", {})) == {"stackql"}, ".mcp.json must contain only the stackql server")
    require(codex["mcpServers"]["stackql"] == {
        "command": "node",
        "args": [f"./{LAUNCHER_REL}"],
        "cwd": ".",
    }, ".mcp.json must launch the vendored launcher from the plugin root")


def validate_payload(version: str) -> None:
    launcher = PLUGIN_ROOT / LAUNCHER_REL
    require(launcher.is_file(), f"{LAUNCHER_REL} is missing (run: make openai-plugin-manifest VERSION={version})")
    require(launcher.read_bytes() == NPM_LAUNCHER.read_bytes(),
            f"{LAUNCHER_REL} differs from packaging/mcpb/npm/bin/stackql-mcp.js (the npm launcher is the single source)")
    text = launcher.read_text(encoding="utf-8")
    require('require("../platforms.json")' in text, "launcher must read its pins from server/platforms.json")
    require(re.search(r'require\("(?!\.\./platforms\.json)[^./][^"]*"\)', text) is None
            or all(mod in ("child_process", "crypto", "fs", "os", "path", "zlib")
                   for mod in re.findall(r'require\("([^./][^"]*)"\)', text)),
            "launcher must only load Node.js built-ins")
    for flag in ("--approot", "--env.file", "--configfile", "--mcp.server.type=stdio"):
        require(f'"{flag}"' in text, f"launcher must pass {flag}")
    require('"--auth"' not in text, "launcher must use provider default authentication")

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
    require("dependencies" not in npm_manifest, "the npm launcher must stay dependency-free")

    skill = PLUGIN_ROOT / SKILL_REL
    require(skill.is_file(), f"{SKILL_REL} is missing")
    require(skill.read_bytes() == SKILL_SOURCE.read_bytes(),
            f"{SKILL_REL} differs from packaging/mcpb/skills/stackql-setup/SKILL.md")
    skill_frontmatter(skill)

    require((PLUGIN_ROOT / "LICENSE").read_bytes() == LICENSE_SOURCE.read_bytes(),
            "plugin LICENSE differs from packaging/mcpb/LICENSE")
    require((PLUGIN_ROOT / "assets" / "icon.png").read_bytes() == ICON_SOURCE.read_bytes(),
            "plugin icon differs from packaging/mcpb/assets/icon.png")
    readme = PLUGIN_ROOT / "README.md"
    require(readme.is_file(), "plugin README.md is missing")
    require(word_count_outside_code(readme.read_text(encoding="utf-8")) >= 40,
            "plugin README.md needs at least 40 words outside code blocks")


def validate_tree() -> None:
    for path in PLUGIN_ROOT.rglob("*"):
        rel = path.relative_to(PLUGIN_ROOT)
        require(not FORBIDDEN_NAMES.intersection(rel.parts),
                f"{rel} cannot ship in the plugin package")
        if path.is_file():
            require(path.stat().st_size <= MAX_FILE_BYTES, f"{rel} exceeds 5 MiB")
    require(not (PLUGIN_ROOT / "bin").exists(),
            "the launcher lives under server/bin; a top-level bin/ directory is not allowed")


def validate() -> None:
    entry = validate_marketplace()
    portable = validate_portable_manifest()
    require(portable["extensions"]["com.openai"]["interface"]["category"] == entry.get("category"),
            "manifest and marketplace categories differ")
    validate_codex_overlay(portable)
    validate_mcp_configs()
    validate_payload(portable["version"])
    validate_tree()


if __name__ == "__main__":
    try:
        validate()
    except ValueError as err:
        print(f"validation failed: {err}", file=sys.stderr)
        raise SystemExit(1) from err
    print("OpenAI stdio plugin validation passed")
