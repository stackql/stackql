#!/usr/bin/env python3
from __future__ import annotations

import importlib.util
import os
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path


REPO_ROOT = Path(__file__).resolve().parents[3]
PLUGIN_ROOT = REPO_ROOT / "packaging" / "openai-plugin" / "plugins" / "stackql"
SHARED_SMOKE = REPO_ROOT / "packaging" / "mcpb" / "scripts" / "smoke-test.py"


def load_shared_smoke():
    spec = importlib.util.spec_from_file_location("stackql_mcp_smoke", SHARED_SMOKE)
    if spec is None or spec.loader is None:
        raise RuntimeError(f"could not load {SHARED_SMOKE}")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


def exercise_plugin(smoke, node: str, launcher: Path) -> None:
    proc = subprocess.Popen(
        [node, str(launcher)],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    failed = False
    try:
        client = smoke.JsonRpcClient(proc)
        client.send(
            "initialize",
            {
                "protocolVersion": "2025-06-18",
                "capabilities": {},
                "clientInfo": {"name": "stackql-openai-plugin-smoke", "version": "1"},
            },
            id_=1,
        )
        if "result" not in client.wait(1, smoke.INIT_TIMEOUT_S):
            raise RuntimeError("initialize failed")
        client.send("notifications/initialized", {})

        client.send("tools/list", {}, id_=2)
        tools = client.wait(2, smoke.CALL_TIMEOUT_S).get("result", {}).get("tools", [])
        names = {tool["name"] for tool in tools}
        required = {"reload_credentials", "pull_provider", "list_services"}
        if not required.issubset(names):
            raise RuntimeError(f"missing tools: {sorted(required - names)}")

        client.send(
            "tools/call",
            {"name": "pull_provider", "arguments": {"provider": "github"}},
            id_=3,
        )
        if "error" in client.wait(3, smoke.CALL_TIMEOUT_S):
            raise RuntimeError("pull_provider failed")

        client.send(
            "tools/call",
            {"name": "list_services", "arguments": {"provider": "github", "row_limit": 5}},
            id_=4,
        )
        services = client.wait(4, smoke.CALL_TIMEOUT_S)
        if "error" in services:
            raise RuntimeError(f"list_services failed: {services['error']}")
        text = "\n".join(
            block.get("text", "")
            for block in services.get("result", {}).get("content", [])
            if isinstance(block, dict)
        )
        if "actions" not in text and "apps" not in text:
            raise RuntimeError("list_services returned no expected GitHub service")

        client.send(
            "tools/call",
            {"name": "pull_provider", "arguments": {"provider": "aws"}},
            id_=5,
        )
        if "error" in client.wait(5, smoke.CALL_TIMEOUT_S):
            raise RuntimeError("pull_provider aws failed")

        client.send(
            "tools/call",
            {"name": "list_services", "arguments": {"provider": "aws", "row_limit": 1}},
            id_=6,
        )
        if "error" in client.wait(6, smoke.CALL_TIMEOUT_S):
            raise RuntimeError("list_services aws failed")

        client.send(
            "tools/call",
            {"name": "reload_credentials", "arguments": {}},
            id_=7,
        )
        reload_response = client.wait(7, smoke.CALL_TIMEOUT_S)
        if "error" in reload_response:
            raise RuntimeError(f"reload_credentials failed: {reload_response['error']}")
        structured = reload_response.get("result", {}).get("structuredContent", {})
        providers = structured.get("providers", [])
        aws = next((item for item in providers if item.get("provider") == "aws"), None)
        expected_vars = {"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"}
        if not structured.get("env_file_sourced"):
            raise RuntimeError(f"reload_credentials did not source the env file: {structured}")
        if not expected_vars.issubset(structured.get("sourced_vars", [])):
            raise RuntimeError(f"reload_credentials missed AWS variables: {structured}")
        if aws is None or aws.get("status") != "ok":
            raise RuntimeError(f"unexpected AWS credential status: {aws}")
    except BaseException:
        failed = True
        raise
    finally:
        if proc.stdin and not proc.stdin.closed:
            proc.stdin.close()
        try:
            proc.terminate()
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            proc.kill()
        if failed and proc.stderr:
            stderr = proc.stderr.read().decode("utf-8", errors="replace").strip()
            if stderr:
                print(stderr, file=sys.stderr)


def main() -> None:
    node = shutil.which("node")
    if node is None:
        raise RuntimeError("node is required")
    launcher = PLUGIN_ROOT / "bin" / "stackql-mcp.js"
    smoke = load_shared_smoke()

    config_dir_var = "STACKQL_PLUGIN_CONFIG_DIR"
    original_config_dir = os.environ.get(config_dir_var)
    try:
        with tempfile.TemporaryDirectory(prefix="stackql-openai-plugin-") as temp:
            stackql_root = Path(temp) / ".stackql"
            stackql_root.mkdir()
            (stackql_root / ".env").write_text(
                "AWS_ACCESS_KEY_ID=plugin-smoke-key\n"
                "AWS_SECRET_ACCESS_KEY=plugin-smoke-secret\n",
                encoding="utf-8",
            )
            os.environ[config_dir_var] = str(stackql_root)
            exercise_plugin(smoke, node, launcher)
    finally:
        if original_config_dir is None:
            os.environ.pop(config_dir_var, None)
        else:
            os.environ[config_dir_var] = original_config_dir

    print("OpenAI stdio plugin smoke test passed")


if __name__ == "__main__":
    main()
