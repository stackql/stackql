"""Nested EXEC policy through a real MCP process and a counted loopback POST."""
import contextlib
import json
import os
import queue
import shutil
import socket
import subprocess
import threading
import time
import uuid
from concurrent.futures import ThreadPoolExecutor
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.error import HTTPError
from urllib.request import Request, urlopen


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
        self.provider_barrier = None
        owner = self

        class API(BaseHTTPRequestHandler):
            def log_message(self, *_):
                pass

            def do_POST(self):
                self.rfile.read(int(self.headers.get("Content-Length", "0")))
                with owner.lock:
                    owner.calls.append(self.path)
                    barrier = owner.provider_barrier
                if barrier is not None:
                    try:
                        barrier.wait(timeout=10)
                    except threading.BrokenBarrierError:
                        self.send_error(503, "Concurrent provider operations did not overlap")
                        return
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

    @contextlib.contextmanager
    def _http_audit_server(self, mode):
        run = self.root / uuid.uuid4().hex
        run.mkdir()
        audit_path = run / "audit.jsonl"
        token = uuid.uuid4().hex
        with socket.socket() as reservation:
            reservation.bind(("127.0.0.1", 0))
            port = reservation.getsockname()[1]
        server = {
            "transport": "http", "address": "127.0.0.1:" + str(port), "mode": mode,
            "protocol_version": "2026-07-28", "auth_token_env_var": "STACKQL_MCP_EFFECTS_TOKEN",
            "audit": {"format": "otel", "failure_mode": "strict", "file": {"path": str(audit_path)}},
        }
        command = [
            self.binary, "mcp", "--mcp.server.type=http", "--mcp.config", json.dumps({
                "server": server, "enabled_tools": ["server_info", "run_mutation_query", "run_lifecycle_operation"],
            }),
            "--approot", str(run / "app"),
            "--registry", json.dumps({"localDocRoot": str(self.root / "registry"),
                                     "verifyConfig": {"nopVerify": True}}),
            "--auth", json.dumps({"mcp_effects": {"type": "null_auth"}}),
            "--sqlBackend", json.dumps({"dsn": str(run / "staging.db")}),
        ]
        process = subprocess.Popen(command, cwd=run,
                                   env=dict(os.environ, GOMAXPROCS="2", STACKQL_MCP_EFFECTS_TOKEN=token),
                                   stdout=subprocess.DEVNULL, stderr=subprocess.PIPE)
        diagnostics = []
        stderr = threading.Thread(target=lambda: diagnostics.extend(
            line.decode(errors="replace") for line in process.stderr), daemon=True)
        stderr.start()
        identifier = 0
        identifier_lock = threading.Lock()
        used_identifiers = set()

        def call(tool, client, traceparent, retry=None, request_id=None):
            nonlocal identifier
            with identifier_lock:
                if request_id is None:
                    request_id = identifier + 1
                assert request_id not in used_identifiers, "Duplicate HTTP request ID"
                used_identifiers.add(request_id)
                identifier = max(identifier, request_id)
            params = {
                "_meta": {"io.modelcontextprotocol/protocolVersion": "2026-07-28",
                          "io.modelcontextprotocol/clientCapabilities": {"elicitation": {"form": {}}},
                          "io.modelcontextprotocol/clientInfo": {"name": client, "version": "1"},
                          "traceparent": traceparent},
                "name": tool, "arguments": {} if tool == "server_info" else {"sql": self.SQL},
            }
            if retry:
                prompt, action = retry
                params["inputResponses"] = {key: {"action": action} for key in prompt["inputRequests"]}
                params["requestState"] = prompt["requestState"]
            message = {"jsonrpc": "2.0", "id": request_id, "method": "tools/call", "params": params}
            request = Request("http://127.0.0.1:" + str(port), json.dumps(message).encode(), {
                "Content-Type": "application/json", "Accept": "application/json, text/event-stream",
                "Authorization": "Bearer " + token, "MCP-Protocol-Version": "2026-07-28",
                "Mcp-Method": "tools/call", "Mcp-Name": tool, "Connection": "close",
            })
            try:
                with urlopen(request, timeout=15) as response:
                    assert not response.headers.get("Mcp-Session-Id"), "Expected sessionless current revision"
                    raw = response.read().decode()
                    if response.headers.get_content_type() == "application/json":
                        result = json.loads(raw)
                    else:
                        messages = [json.loads(line[5:].strip()) for line in raw.splitlines() if line.startswith("data:")]
                        result = next(message for message in messages if message.get("id") == request_id)
            except HTTPError as error:
                with error:
                    detail = json.loads(error.read())["error"]
                raise AssertionError("MCP HTTP " + str(error.code) + ": " + detail["message"]) from None
            assert result.get("id") == request_id, result
            return result

        try:
            deadline = time.monotonic() + 10
            while True:
                try:
                    with socket.create_connection(("127.0.0.1", port), timeout=0.2):
                        break
                except OSError:
                    assert process.poll() is None, "".join(diagnostics)
                    assert time.monotonic() < deadline, "HTTP MCP listener did not start"
                    time.sleep(0.02)
            self._success(call("server_info", "preflight", self._traceparent()))
            yield call, audit_path, token, diagnostics
        finally:
            if process.poll() is None:
                process.terminate()
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait(timeout=5)
            stderr.join(timeout=5)
            process.stderr.close()

    @staticmethod
    def _traceparent():
        return "00-" + uuid.uuid4().hex + "-" + uuid.uuid4().hex[:16] + "-01"

    def reconnecting_clients_keep_approval_audit_request_scoped(self, mode):
        assert mode in ("safe", "delete_safe"), mode
        expected = {}
        with self._http_audit_server(mode) as (call, audit_path, token, diagnostics):
            prompts = []
            for client, tool in [("client-a", "run_mutation_query"), ("client-b", "run_lifecycle_operation")]:
                response = call(tool, client, self._traceparent())
                assert "error" not in response and not response["result"].get("isError"), response
                prompt = response["result"]
                assert prompt.get("requestState") and prompt.get("inputRequests"), response
                prompts.append(prompt)
                assert self._count() == 0, self.calls

            trace = self._traceparent()
            self._refused(call("run_mutation_query", "client-a", trace, retry=(prompts[1], "accept")),
                          "does not match")
            expected[trace] = ("run_mutation_query", "needs_approval_declined")
            assert self._count() == 0, self.calls
            for count, client, tool, prompt in [
                (1, "client-b", "run_lifecycle_operation", prompts[1]),
                (2, "client-a", "run_mutation_query", prompts[0]),
            ]:
                trace = self._traceparent()
                self._success(call(tool, client, trace, retry=(prompt, "accept")))
                expected[trace] = (tool, "needs_approval_accepted")
                assert self._count() == count, self.calls
            trace = self._traceparent()
            self._refused(call("run_lifecycle_operation", "client-b", trace, retry=(prompts[1], "accept")),
                          "already used")
            expected[trace] = ("run_lifecycle_operation", "needs_approval_declined")
            assert self.calls == ["/activate/x", "/activate/x"], self.calls

        self._assert_scoped_audit(audit_path, token, diagnostics, prompts, expected, mode)

    def concurrent_clients_keep_approval_audit_request_scoped(self, mode):
        assert mode in ("safe", "delete_safe"), mode
        expected = {}
        clients = [("client-a", "run_mutation_query"), ("client-b", "run_lifecycle_operation")]
        with self._http_audit_server(mode) as (call, audit_path, token, diagnostics):
            prompts = []
            for client, tool in clients:
                response = call(tool, client, self._traceparent())
                assert "error" not in response and not response["result"].get("isError"), response
                prompt = response["result"]
                assert prompt.get("requestState") and prompt.get("inputRequests"), response
                prompts.append(prompt)
                assert self._count() == 0, self.calls

            overlap = threading.Event()
            barrier = threading.Barrier(2, action=overlap.set)
            with self.lock:
                self.provider_barrier = barrier
            try:
                with ThreadPoolExecutor(max_workers=2) as workers:
                    futures = []
                    for request_id, (client, tool), prompt in zip((4, 5), clients, prompts):
                        trace = self._traceparent()
                        expected[trace] = (tool, "needs_approval_accepted")
                        futures.append((request_id, workers.submit(
                            call, tool, client, trace, retry=(prompt, "accept"), request_id=request_id)))
                    for request_id, future in futures:
                        response = future.result(timeout=20)
                        assert response["id"] == request_id, response
                        self._success(response)
                assert overlap.is_set(), "Both provider operations must be in flight before either returns"
            finally:
                barrier.abort()
                with self.lock:
                    self.provider_barrier = None
            assert self.calls == ["/activate/x", "/activate/x"], self.calls

            for (client, tool), prompt in zip(clients, prompts):
                trace = self._traceparent()
                self._refused(call(tool, client, trace, retry=(prompt, "accept")), "already used")
                expected[trace] = (tool, "needs_approval_declined")
                assert self._count() == 2, self.calls

        self._assert_scoped_audit(audit_path, token, diagnostics, prompts, expected, mode)

    def _assert_scoped_audit(self, audit_path, token, diagnostics, prompts, expected, mode):
        audit_text = audit_path.read_text(encoding="utf-8")
        for secret in [token] + [prompt["requestState"] for prompt in prompts]:
            assert secret not in audit_text and secret not in "".join(diagnostics), "Credentials or approval state logged"
        records = []
        for line in audit_text.splitlines():
            for resource in json.loads(line)["resourceLogs"]:
                for scope in resource["scopeLogs"]:
                    records.extend(scope["logRecords"])
        actual = {}
        call_ids = set()
        for trace, (tool, decision) in expected.items():
            trace_id, span_id = trace.split("-")[1:3]
            matched = [record for record in records if record["traceId"] == trace_id]
            assert len(matched) == 2, (trace, matched)
            ids = set()
            methods = set()
            for record in matched:
                attrs = {attribute["key"]: attribute["value"].get("stringValue")
                         for attribute in record["attributes"]}
                assert record["spanId"] == span_id, record
                assert attrs["gen_ai.tool.name"] == tool and attrs["stackql.decision"] == decision, attrs
                assert attrs["stackql.mode"] == mode and attrs["mcp.protocol.version"] == "2026-07-28", attrs
                assert not attrs.get("mcp.session.id"), attrs
                if attrs["mcp.method.name"] == "tools/call":
                    assert attrs["stackql.query"] == self.SQL and attrs["stackql.query_class"] == "lifecycle", attrs
                ids.add(attrs["gen_ai.tool.call.id"])
                methods.add(attrs["mcp.method.name"])
            assert len(ids) == 1 and methods == {"tools/call", "elicitation/create"}, matched
            actual[trace] = matched
            call_ids.update(ids)
        assert len(call_ids) == len(expected), call_ids
        effect_records = [record for record in records if record["body"]["stringValue"] != "execute_tool server_info"]
        assert len(effect_records) == sum(len(matched) for matched in actual.values()), effect_records
