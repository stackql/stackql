#!/usr/bin/env python3
"""
Optional Gemini-Flash-driven smoke test for a built .mcpb bundle.

Spawns the embedded stackql MCP server and exposes its tools to Gemini via
function calling. Asks the model to pull the github provider and list its
services, executes whatever tools the model decides to call, and verifies that
github services come back.

This is a SOFT check - it requires GEMINI_API_KEY and will exit 0 with a notice
if the key is not set. Failures here should not block a release; they indicate
the agent integration regressed, not the bundle itself.

Uses only stdlib (urllib) so it runs without pip install in CI.

Usage:
  GEMINI_API_KEY=... python scripts/gemini-smoke.py <path-to-bundle.mcpb>

Tunables (env):
  GEMINI_MODEL   default: gemini-2.0-flash
  GEMINI_TURNS   default: 6 (max tool-call rounds before giving up)
"""
from __future__ import annotations

import json
import os
import subprocess
import sys
import tempfile
import threading
import time
import urllib.error
import urllib.request
import zipfile
from pathlib import Path

GITHUB_AUTH = json.dumps({"github": {"type": "null_auth"}})
INIT_TIMEOUT_S = 30
CALL_TIMEOUT_S = 90
HTTP_TIMEOUT_S = 60

MODEL = os.environ.get("GEMINI_MODEL", "gemini-2.0-flash")
MAX_TURNS = int(os.environ.get("GEMINI_TURNS", "6"))

USER_PROMPT = (
    "Use the available tools to pull the 'github' provider into the local "
    "stackql cache, then list 5 services available under github. Respond with "
    "the names of the services you found, one per line, after the word DONE on "
    "its own line."
)


def log(msg: str) -> None:
    print(f"[gemini] {msg}", flush=True)


def soft_skip(msg: str) -> "Never":  # type: ignore[name-defined]
    print(f"[gemini] SKIP: {msg}", flush=True)
    sys.exit(0)


def soft_fail(msg: str) -> "Never":  # type: ignore[name-defined]
    # Soft check - print as a warning and exit 0 so CI doesn't fail.
    print(f"[gemini] WARN: {msg}", flush=True)
    sys.exit(0)


def extract_bundle(bundle: Path, dest: Path) -> Path:
    with zipfile.ZipFile(bundle) as zf:
        zf.extractall(dest)
    manifest = json.loads((dest / "manifest.json").read_text())
    binary = dest / manifest["server"]["entry_point"]
    if not binary.exists():
        soft_fail(f"entry_point not found in bundle")
    if os.name != "nt":
        binary.chmod(0o755)
    return binary


class JsonRpcClient:
    def __init__(self, proc: subprocess.Popen) -> None:
        self.proc = proc
        self._lock = threading.Lock()
        self._responses: dict[int, dict] = {}
        self._next_id = 100
        threading.Thread(target=self._read_loop, daemon=True).start()

    def _read_loop(self) -> None:
        assert self.proc.stdout is not None
        for line in self.proc.stdout:
            line = line.strip()
            if not line:
                continue
            try:
                msg = json.loads(line)
            except json.JSONDecodeError:
                continue
            if isinstance(msg, dict) and "id" in msg:
                with self._lock:
                    self._responses[msg["id"]] = msg

    def call(self, method: str, params: dict | None, timeout: float) -> dict:
        with self._lock:
            id_ = self._next_id
            self._next_id += 1
        msg = {"jsonrpc": "2.0", "id": id_, "method": method}
        if params is not None:
            msg["params"] = params
        assert self.proc.stdin is not None
        self.proc.stdin.write(json.dumps(msg) + "\n")
        self.proc.stdin.flush()
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            with self._lock:
                if id_ in self._responses:
                    return self._responses.pop(id_)
            if self.proc.poll() is not None:
                soft_fail(f"server exited (rc={self.proc.returncode}) waiting on {method}")
            time.sleep(0.05)
        soft_fail(f"timed out waiting for {method}")

    def notify(self, method: str, params: dict | None = None) -> None:
        msg = {"jsonrpc": "2.0", "method": method}
        if params is not None:
            msg["params"] = params
        assert self.proc.stdin is not None
        self.proc.stdin.write(json.dumps(msg) + "\n")
        self.proc.stdin.flush()


def mcp_schema_to_gemini(schema: dict) -> dict:
    """
    Convert an MCP tool's JSON Schema to Gemini's function-declaration schema.
    Gemini accepts a strict subset of OpenAPI 3 schema. We strip unsupported
    keys ('additionalProperties', '$schema', etc.) and pass through the basics.
    """
    if not isinstance(schema, dict):
        return {"type": "object", "properties": {}}
    out: dict = {}
    if "type" in schema:
        out["type"] = schema["type"].upper() if schema["type"] != "object" else "OBJECT"
        # Gemini wants type names uppercased (STRING/OBJECT/ARRAY/INTEGER/...).
        out["type"] = schema["type"].upper()
    else:
        out["type"] = "OBJECT"
    if "properties" in schema:
        out["properties"] = {
            k: mcp_schema_to_gemini(v) for k, v in schema["properties"].items()
        }
    if "items" in schema:
        out["items"] = mcp_schema_to_gemini(schema["items"])
    if "required" in schema:
        out["required"] = schema["required"]
    if "description" in schema:
        out["description"] = schema["description"]
    return out


def gemini_call(api_key: str, body: dict) -> dict:
    url = (
        f"https://generativelanguage.googleapis.com/v1beta/models/"
        f"{MODEL}:generateContent?key={api_key}"
    )
    data = json.dumps(body).encode("utf-8")
    req = urllib.request.Request(
        url, data=data, headers={"Content-Type": "application/json"}
    )
    try:
        with urllib.request.urlopen(req, timeout=HTTP_TIMEOUT_S) as resp:
            return json.loads(resp.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        body_txt = e.read().decode("utf-8", errors="replace")
        soft_fail(f"Gemini HTTP {e.code}: {body_txt[:400]}")
    except urllib.error.URLError as e:
        soft_fail(f"Gemini URL error: {e}")


def run(bundle_path: Path) -> None:
    api_key = os.environ.get("GEMINI_API_KEY", "").strip()
    if not api_key:
        soft_skip("GEMINI_API_KEY not set, skipping agent smoke test")
    if not bundle_path.exists():
        soft_fail(f"bundle not found: {bundle_path}")

    with tempfile.TemporaryDirectory(prefix="mcpb-gemini-") as tmp:
        binary = extract_bundle(bundle_path, Path(tmp))
        cmd = [str(binary), "mcp", "--mcp.server.type=stdio", f"--auth={GITHUB_AUTH}"]
        log(f"spawning: {binary.name} mcp --mcp.server.type=stdio")
        proc = subprocess.Popen(
            cmd,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            bufsize=1,
        )
        try:
            client = JsonRpcClient(proc)

            init = client.call(
                "initialize",
                {
                    "protocolVersion": "2024-11-05",
                    "capabilities": {},
                    "clientInfo": {"name": "stackql-mcpb-gemini-smoke", "version": "1"},
                },
                INIT_TIMEOUT_S,
            )
            if "result" not in init:
                soft_fail(f"initialize failed: {init}")
            client.notify("notifications/initialized", {})

            tools_resp = client.call("tools/list", {}, CALL_TIMEOUT_S)
            mcp_tools = tools_resp.get("result", {}).get("tools", [])
            if not mcp_tools:
                soft_fail("no tools listed by MCP server")
            log(f"exposing {len(mcp_tools)} MCP tools to Gemini")

            function_declarations = [
                {
                    "name": t["name"],
                    "description": t.get("description", ""),
                    "parameters": mcp_schema_to_gemini(t.get("inputSchema", {})),
                }
                for t in mcp_tools
            ]

            contents: list[dict] = [
                {"role": "user", "parts": [{"text": USER_PROMPT}]}
            ]
            body_template = {
                "tools": [{"functionDeclarations": function_declarations}],
                "toolConfig": {"functionCallingConfig": {"mode": "AUTO"}},
            }

            transcript_text = ""
            agent_called_pull = False
            agent_called_list = False

            for turn in range(MAX_TURNS):
                body = dict(body_template)
                body["contents"] = contents
                resp = gemini_call(api_key, body)
                candidates = resp.get("candidates", [])
                if not candidates:
                    soft_fail(f"Gemini returned no candidates: {json.dumps(resp)[:400]}")
                parts = candidates[0].get("content", {}).get("parts", [])
                if not parts:
                    log("Gemini returned empty parts; stopping")
                    break

                contents.append({"role": "model", "parts": parts})

                function_responses: list[dict] = []
                text_this_turn = []
                for part in parts:
                    if "functionCall" in part:
                        fc = part["functionCall"]
                        name = fc.get("name", "")
                        args = fc.get("args", {}) or {}
                        log(f"  turn {turn}: model calls {name}({json.dumps(args)})")
                        if name == "pull_provider":
                            agent_called_pull = True
                        if name == "list_services":
                            agent_called_list = True
                        tool_resp = client.call(
                            "tools/call",
                            {"name": name, "arguments": args},
                            CALL_TIMEOUT_S,
                        )
                        result = tool_resp.get("result", tool_resp.get("error", {}))
                        # Pass MCP result back as a function response.
                        function_responses.append(
                            {
                                "functionResponse": {
                                    "name": name,
                                    "response": {"result": json.dumps(result)[:8000]},
                                }
                            }
                        )
                    elif "text" in part:
                        text_this_turn.append(part["text"])

                if text_this_turn:
                    transcript_text = "\n".join(text_this_turn)

                if not function_responses:
                    # Model produced text only - we're done.
                    break

                contents.append({"role": "user", "parts": function_responses})

            if not agent_called_pull and not agent_called_list:
                soft_fail("Gemini did not call any stackql tools")

            log("--- model final response ---")
            for line in transcript_text.splitlines():
                log(f"  {line}")

            if "DONE" not in transcript_text:
                soft_fail("model did not emit DONE sentinel")

            # Be lenient about exactly which services come back; require at
            # least one well-known github service name in the model's reply.
            known = {"actions", "repos", "issues", "pulls", "orgs", "users", "apps", "search"}
            mentioned = {s for s in known if s in transcript_text.lower()}
            if not mentioned:
                soft_fail(f"model response did not mention any known github services")
            log(f"model mentioned github services: {sorted(mentioned)}")
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

    log("gemini agent smoke test passed")


if __name__ == "__main__":
    if len(sys.argv) != 2:
        print(f"usage: {sys.argv[0]} <path-to-bundle.mcpb>", file=sys.stderr)
        sys.exit(2)
    run(Path(sys.argv[1]))
