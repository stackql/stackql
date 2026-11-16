"""Nested EXEC policy through a real MCP process and a counted loopback POST."""
import contextlib
import json
import os
import queue
import shutil
import subprocess
import threading
import time
import uuid
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path


class MCPEffectsLibrary:
    ROBOT_LIBRARY_SCOPE = "TEST"
    SQL = "select status from (exec mcp_effects.testing.activation.activate @id = 'x')"

    def prepare_mcp_effects(self, binary):
        candidate = Path(binary).resolve()
        if os.name == "nt" and candidate.with_suffix(".exe").is_file():
            candidate = candidate.with_suffix(".exe")
        assert candidate.is_file(), "STACKQL_EXE must identify a freshly built native binary"
        self.binary = str(candidate)
        self.root = (Path("build") / (".mcp-effects-" + uuid.uuid4().hex)).resolve()
        self.root.mkdir(parents=True)
        self.calls = []
        self.lock = threading.Lock()
        owner = self

        class API(BaseHTTPRequestHandler):
            def log_message(self, *_):
                pass

            def do_POST(self):
                self.rfile.read(int(self.headers.get("Content-Length", "0")))
                with owner.lock:
                    owner.calls.append(self.path)
                payload = json.dumps({"status": "ACTIVE", "id": "x"}).encode()
                self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(payload)))
                self.end_headers()
                self.wfile.write(payload)

        self.provider = ThreadingHTTPServer(("127.0.0.1", 0), API)
        self.provider.daemon_threads = True
        self.worker = threading.Thread(target=self.provider.serve_forever, daemon=True)
        self.worker.start()
        version = self.root / "registry" / "src" / "mcp_effects" / "v00.00.00000"
        (version / "services").mkdir(parents=True)
        provider = {
            "id": "mcp_effects", "name": "mcp_effects", "version": "v00.00.00000",
            "providerServices": {"testing": {
                "id": "testing:v00.00.00000", "name": "testing", "preferred": True,
                "service": {"$ref": "mcp_effects/v00.00.00000/services/testing.yaml"},
            }},
            "config": {"auth": {"type": "null_auth"}},
        }
        document = {
            "openapi": "3.0.0", "info": {"title": "MCP effects fixture", "version": "v1"},
            "servers": [{"url": "http://127.0.0.1:" + str(self.provider.server_port)}],
            "paths": {"/activate/{id}": {"post": {
                "operationId": "activate",
                "parameters": [{"name": "id", "in": "path", "required": True,
                                "schema": {"type": "string"}}],
                "responses": {"200": {"description": "Activated", "content": {
                    "application/json": {"schema": {"type": "object", "properties": {
                        "status": {"type": "string"}, "id": {"type": "string"},
                    }}},
                }}},
            }}},
            "components": {"x-stackQL-resources": {"activation": {
                "id": "mcp_effects.testing.activation", "name": "activation",
                "methods": {"activate": {
                    "operation": {"$ref": "#/paths/~1activate~1{id}/post"},
                    "response": {"mediaType": "application/json", "openAPIDocKey": "200"},
                }},
                "sqlVerbs": {"exec": [{"$ref": "#/components/x-stackQL-resources/activation/methods/activate"}]},
            }}},
        }
        (version / "provider.yaml").write_text(json.dumps(provider), encoding="utf-8")
        (version / "services" / "testing.yaml").write_text(json.dumps(document), encoding="utf-8")

    def stop_mcp_effects(self):
        if hasattr(self, "provider"):
            self.provider.shutdown()
            self.provider.server_close()
            self.worker.join(timeout=5)
        if hasattr(self, "root"):
            for attempt in range(50):
                try:
                    shutil.rmtree(self.root)
                    break
                except PermissionError:
                    if os.name != "nt" or attempt == 49:
                        raise
                    time.sleep(0.1)

    def _count(self):
        with self.lock:
            return len(self.calls)

    @contextlib.contextmanager
    def _server(self, mode, preview=False):
        run = self.root / uuid.uuid4().hex
        run.mkdir()
        command = [
            self.binary, "mcp", "--mcp.server.type=stdio",
            "--mcp.config", json.dumps({
                "server": {"transport": "stdio", "mode": mode, "audit": {"disabled": True}},
                "enabled_tools": ["server_info", "run_select_query", "run_mutation_query", "run_lifecycle_operation"],
            }),
            "--approot", str(run / "app"),
            "--registry", json.dumps({"localDocRoot": str(self.root / "registry"),
                                     "verifyConfig": {"nopVerify": True}}),
            "--auth", json.dumps({"mcp_effects": {"type": "null_auth"}}),
            "--sqlBackend", json.dumps({"dsn": str(run / "staging.db")}),
        ]
        if preview:
            command.extend(["--preview", json.dumps({
                "unstable": True, "iacStateRoot": str(self.root / "ledger"),
            })])
        process = subprocess.Popen(command, cwd=run, env=dict(os.environ, GOMAXPROCS="2"),
                                   stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        responses = queue.Queue()
        diagnostics = []

        def read_responses():
            try:
                for line in process.stdout:
                    responses.put(json.loads(line))
            finally:
                responses.put(None)

        reader = threading.Thread(target=read_responses, daemon=True)
        stderr = threading.Thread(target=lambda: diagnostics.extend(
            line.decode(errors="replace") for line in process.stderr), daemon=True)
        reader.start()
        stderr.start()
        identifier = 0

        def call(tool, approval=True, retry=None, sql=None):
            nonlocal identifier
            identifier += 1
            capabilities = {"elicitation": {"form": {}}} if approval else {}
            params = {
                "_meta": {"io.modelcontextprotocol/protocolVersion": "2026-07-28",
                          "io.modelcontextprotocol/clientCapabilities": capabilities},
                "name": tool, "arguments": {} if tool == "server_info" else {"sql": sql or self.SQL},
            }
            if retry:
                prompt, action = retry
                params["inputResponses"] = {key: {"action": action} for key in prompt["inputRequests"]}
                params["requestState"] = prompt["requestState"]
            message = {"jsonrpc": "2.0", "id": identifier, "method": "tools/call", "params": params}
            process.stdin.write((json.dumps(message) + "\n").encode())
            process.stdin.flush()
            try:
                response = responses.get(timeout=15)
            except queue.Empty:
                raise AssertionError("MCP call timed out: " + "".join(diagnostics)) from None
            assert response is not None, "MCP process closed: " + "".join(diagnostics)
            assert response.get("id") == identifier, response
            return response

        try:
            self._success(call("server_info"))
            yield call
        finally:
            process.stdin.close()
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait(timeout=5)
            reader.join(timeout=5)
            stderr.join(timeout=5)
            process.stdout.close()
            process.stderr.close()

    @staticmethod
    def _success(response):
        assert "error" not in response and not response["result"].get("isError"), response
        assert not response["result"].get("inputRequests"), response

    @staticmethod
    def _refused(response, message):
        result = response.get("result", {})
        assert "error" in response or result.get("isError"), response
        assert message in json.dumps(response), response
        assert not result.get("inputRequests"), response

    def nested_exec_read_only_has_zero_operations(self):
        with self._server("read_only") as call:
            for tool in ["run_select_query", "run_mutation_query", "run_lifecycle_operation"]:
                self._refused(call(tool), "read_only")
                assert self._count() == 0, self.calls
            self._success(call("run_select_query", sql="select 'EXEC is literal text' as description"))
            assert self._count() == 0, self.calls

    def iac_read_only_does_not_write_ledger(self):
        ledger = self.root / "ledger"
        with self._server("read_only", preview=True) as call:
            metadata = call("run_select_query", sql="SHOW RESOURCES IN stackql_preview.iac")
            self._success(metadata)
            assert "stackql_preview.iac.recover" in json.dumps(metadata), metadata
            assert not ledger.exists(), "Metadata discovery created IaC state"
            predicates = (" where collection = 'policy-test' and state = 'recovery' "
                          "and resource_key = 'item' and version = 'observed' and outcome = 'absent'")
            for relation in ["stackql_preview.iac.run", "stackql_preview.iac.destroy",
                             "stackql_preview.iac.recover", "iac.recover"]:
                sql = "select * from " + relation + predicates
                for tool in ["run_select_query", "run_mutation_query", "run_lifecycle_operation"]:
                    self._refused(call(tool, sql=sql), "read_only")
                    assert not ledger.exists(), "Read-only IaC query wrote local state"
                    assert self._count() == 0, self.calls

    def nested_exec_requires_approval(self, mode):
        assert mode in ("safe", "delete_safe"), mode
        with self._server(mode) as call:
            self._refused(call("run_select_query"), "lifecycle statements run through")
            assert self._count() == 0, self.calls
            for expected_before, tool in enumerate(["run_mutation_query", "run_lifecycle_operation"]):
                self._refused(call(tool, approval=False), "does not support elicitation")
                before = self._count()
                assert before == expected_before, self.calls
                for action, refusal in [("decline", "declined approval"), ("cancel", "dismissed"), ("accept", None)]:
                    response = call(tool)
                    assert "error" not in response and not response["result"].get("isError"), response
                    prompt = response["result"]
                    assert prompt.get("requestState") and prompt.get("inputRequests"), response
                    assert "lifecycle" in json.dumps(prompt["inputRequests"]), response
                    assert self._count() == before, self.calls
                    result = call(tool, retry=(prompt, action))
                    if refusal:
                        self._refused(result, refusal)
                        assert self._count() == before, self.calls
                    else:
                        self._success(result)
                        assert self._count() == before + 1, self.calls
            assert self._count() == 2, self.calls

    def nested_exec_full_access_executes(self):
        with self._server("full_access") as call:
            self._refused(call("run_select_query"), "lifecycle statements run through")
            assert self._count() == 0, self.calls
            for expected, tool in enumerate(["run_mutation_query", "run_lifecycle_operation"], start=1):
                self._success(call(tool, approval=False))
                assert self._count() == expected, self.calls
            assert self.calls == ["/activate/x", "/activate/x"], self.calls

    def approval_state_tampering_and_replay_prevent_operations(self, mode):
        assert mode in ("safe", "delete_safe"), mode
        with self._server(mode) as call:
            for before, tool in enumerate(["run_mutation_query", "run_lifecycle_operation"]):
                response = call(tool)
                assert "error" not in response and not response["result"].get("isError"), response
                prompt = response["result"]
                assert prompt.get("inputRequests") and prompt.get("requestState"), response
                assert self._count() == before, self.calls

                state = prompt["requestState"]
                replacement = "BB" if state.endswith("AA") else "AA"
                tampered = dict(prompt, requestState=state[:-2] + replacement)
                assert tampered["requestState"] != state
                self._refused(call(tool, retry=(tampered, "accept")), "not issued by this server")
                assert self._count() == before, self.calls

                self._success(call(tool, retry=(prompt, "accept")))
                assert self._count() == before + 1, self.calls
                self._refused(call(tool, retry=(prompt, "accept")), "already used")
                assert self._count() == before + 1, self.calls
            assert self.calls == ["/activate/x", "/activate/x"], self.calls
