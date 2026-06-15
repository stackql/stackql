#!/usr/bin/env python3
"""
Deterministic MCP smoke test for a built .mcpb bundle.

Extracts the bundle, spawns the embedded stackql binary in MCP stdio mode,
and exercises the JSON-RPC handshake plus the github provider in no-auth mode:
initialize -> tools/list -> tools/call pull_provider -> tools/call list_services.

Exits non-zero on any failure. Designed to run on Linux, macOS, and Windows
GitHub Actions runners with only stdlib Python.

Usage:
  python scripts/smoke-test.py <path-to-bundle.mcpb>
"""
from __future__ import annotations

import json
import os
import shlex
import subprocess
import sys
import tempfile
import threading
import time
import zipfile
from pathlib import Path

GITHUB_AUTH = json.dumps({"github": {"type": "null_auth"}})
INIT_TIMEOUT_S = 30
CALL_TIMEOUT_S = 90


def log(msg: str) -> None:
    print(f"[smoke] {msg}", flush=True)


def fail(msg: str) -> "Never":  # type: ignore[name-defined]
    print(f"[smoke] FAIL: {msg}", flush=True)
    sys.exit(1)


def extract_bundle(bundle: Path, dest: Path) -> tuple[Path, dict]:
    """Unzip the .mcpb and return the server binary path and the manifest."""
    log(f"extracting {bundle.name}")
    with zipfile.ZipFile(bundle) as zf:
        zf.extractall(dest)
    manifest = json.loads((dest / "manifest.json").read_text())
    entry = manifest["server"]["entry_point"]
    binary = dest / entry
    if not binary.exists():
        fail(f"entry_point {entry} not found in bundle")
    if os.name != "nt":
        binary.chmod(0o755)
    log(f"entry_point: {entry} (version {manifest.get('version')})")
    return binary, manifest


def manifest_args(manifest: dict, extract_dir: Path, home_dir: Path) -> list[str]:
    """Resolve the manifest's mcp_config args the way an MCPB client would.

    Substituting ${HOME} with a temp dir keeps the test hermetic and proves
    the server runs without writing to its cwd (Claude Desktop launches
    extensions with cwd '/', which is read-only on macOS).
    """
    args = manifest["server"]["mcp_config"].get("args", [])
    resolved = [
        a.replace("${__dirname}", str(extract_dir)).replace("${HOME}", str(home_dir))
        for a in args
    ]
    return resolved


class JsonRpcClient:
    """Minimal line-delimited JSON-RPC client over a child process's stdio."""

    def __init__(self, proc: subprocess.Popen) -> None:
        self.proc = proc
        self._lock = threading.Lock()
        self._responses: dict[int, dict] = {}
        self._reader = threading.Thread(target=self._read_loop, daemon=True)
        self._reader.start()

    def _read_loop(self) -> None:
        assert self.proc.stdout is not None
        for raw in self.proc.stdout:
            line = raw.decode("utf-8", errors="replace").strip()
            if not line:
                continue
            try:
                msg = json.loads(line)
            except json.JSONDecodeError:
                continue
            if isinstance(msg, dict) and "id" in msg:
                with self._lock:
                    self._responses[msg["id"]] = msg

    def send(self, method: str, params: dict | None = None, *, id_: int | None = None) -> None:
        msg: dict = {"jsonrpc": "2.0", "method": method}
        if params is not None:
            msg["params"] = params
        if id_ is not None:
            msg["id"] = id_
        line = json.dumps(msg) + "\n"
        assert self.proc.stdin is not None
        self.proc.stdin.write(line.encode("utf-8"))
        self.proc.stdin.flush()

    def wait(self, id_: int, timeout: float) -> dict:
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            with self._lock:
                if id_ in self._responses:
                    return self._responses.pop(id_)
            if self.proc.poll() is not None:
                fail(f"server exited (rc={self.proc.returncode}) before responding to id={id_}")
            time.sleep(0.05)
        fail(f"timed out waiting for response id={id_} after {timeout}s")


def run(bundle_path: Path) -> None:
    if not bundle_path.exists():
        fail(f"bundle not found: {bundle_path}")

    with tempfile.TemporaryDirectory(prefix="mcpb-smoke-") as tmp:
        tmp_path = Path(tmp)
        binary, manifest = extract_bundle(bundle_path, tmp_path)

        home_dir = tmp_path / "home"
        home_dir.mkdir()
        args = manifest_args(manifest, tmp_path, home_dir)
        cmd = [str(binary), *args, f"--auth={GITHUB_AUTH}"]
        exercise(cmd)

    log("smoke test passed")


def run_command(cmd: list[str]) -> None:
    """Smoke-test an arbitrary command that speaks MCP stdio (docker image,
    npm wrapper, ...). The command must accept extra stackql flags appended
    after its own arguments (used to pass --auth for the github provider)."""
    exercise(cmd + [f"--auth={GITHUB_AUTH}"])
    log("smoke test passed")


def exercise(cmd: list[str]) -> None:
    log(f"spawning: {' '.join(cmd)}")
    # Binary pipes on purpose: text=True would translate \n to \r\n on
    # Windows stdin, and the server exits silently on the stray \r.
    proc = subprocess.Popen(
        cmd,
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    try:
        client = JsonRpcClient(proc)

        client.send(
            "initialize",
            {
                "protocolVersion": "2024-11-05",
                "capabilities": {},
                "clientInfo": {"name": "stackql-mcpb-smoke", "version": "1"},
            },
            id_=1,
        )
        init = client.wait(1, INIT_TIMEOUT_S)
        if "result" not in init:
            fail(f"initialize did not return a result: {init}")
        log(f"initialize ok: server={init['result'].get('serverInfo', {}).get('name')}")

        client.send("notifications/initialized", {})

        client.send("tools/list", {}, id_=2)
        tools = client.wait(2, CALL_TIMEOUT_S).get("result", {}).get("tools", [])
        tool_names = {t["name"] for t in tools}
        log(f"tools/list returned {len(tool_names)} tools")
        for required in ("pull_provider", "list_services", "list_providers"):
            if required not in tool_names:
                fail(f"missing required tool: {required}")

        client.send(
            "tools/call",
            {"name": "pull_provider", "arguments": {"provider": "github"}},
            id_=3,
        )
        pull = client.wait(3, CALL_TIMEOUT_S)
        if "error" in pull:
            fail(f"pull_provider failed: {pull['error']}")
        log("pull_provider github ok")

        client.send(
            "tools/call",
            {"name": "list_services", "arguments": {"provider": "github", "row_limit": 5}},
            id_=4,
        )
        services = client.wait(4, CALL_TIMEOUT_S)
        if "error" in services:
            fail(f"list_services failed: {services['error']}")
        content = services.get("result", {}).get("content", [])
        text_blocks = [c.get("text", "") for c in content if isinstance(c, dict)]
        joined = "\n".join(text_blocks)
        if "actions" not in joined and "apps" not in joined:
            fail(f"list_services did not include expected github services. content={content!r}")
        log("list_services returned github services (saw expected service names)")
    finally:
        try:
            if proc.stdin and not proc.stdin.closed:
                proc.stdin.close()
        except Exception:
            pass
        try:
            proc.terminate()
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            proc.kill()


USAGE = """usage:
  smoke-test.py <path-to-bundle.mcpb>     test an MCPB bundle (manifest-driven args)
  smoke-test.py --docker <image>          test a docker image (stdio MCP server)
  smoke-test.py --cmd "<command ...>"     test an arbitrary stdio MCP command
                                          (e.g. the npm wrapper)"""

if __name__ == "__main__":
    if len(sys.argv) == 3 and sys.argv[1] == "--docker":
        run_command(
            ["docker", "run", "-i", "--rm", sys.argv[2], "mcp", "--mcp.server.type=stdio"]
        )
    elif len(sys.argv) == 3 and sys.argv[1] == "--cmd":
        run_command(shlex.split(sys.argv[2]))
    elif len(sys.argv) == 2 and not sys.argv[1].startswith("-"):
        run(Path(sys.argv[1]))
    else:
        print(USAGE, file=sys.stderr)
        sys.exit(2)
