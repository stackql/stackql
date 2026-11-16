"""Real-binary MCP limits with an isolated registry and gated loopback provider."""
import concurrent.futures
import contextlib
import json
import os
import queue
import select
import shutil
import socket
import subprocess
import threading
import time
import uuid
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import parse_qs, urlsplit
from urllib.request import Request, urlopen


class _StdioClient:
    def __init__(self, process):
        self.process = process
        self.lock = threading.Lock()
        self.pending = {}
        self.identifier = 0
        self.reader = threading.Thread(target=self._read, daemon=True)
        self.reader.start()

    def _read(self):
        try:
            for line in self.process.stdout:
                message = json.loads(line)
                with self.lock:
                    target = self.pending.get(message.get("id"))
                if target is not None:
                    target.put(message)
        finally:
            with self.lock:
                for target in self.pending.values():
                    target.put({"error": "stdio process closed before replying"})

    def call(self, method, params=None, notification=False):
        with self.lock:
            self.identifier += 1
            identifier = self.identifier
            target = queue.Queue()
            message = {"jsonrpc": "2.0", "method": method, "params": params or {}}
            if not notification:
                message["id"] = identifier
                self.pending[identifier] = target
            self.process.stdin.write((json.dumps(message) + "\n").encode())
            self.process.stdin.flush()
        if notification:
            return {}
        try:
            return target.get(timeout=15)
        finally:
            with self.lock:
                self.pending.pop(identifier, None)


class _HTTPClient:
    def __init__(self, endpoint, token):
        self.endpoint = endpoint
        self.token = token
        self.session = ""
        self.identifier = 0
        self.lock = threading.Lock()

    def call(self, method, params=None, notification=False):
        with self.lock:
            self.identifier += 1
            identifier = self.identifier
        message = {"jsonrpc": "2.0", "method": method, "params": params or {}}
        if not notification:
            message["id"] = identifier
        headers = {"Content-Type": "application/json", "Accept": "application/json, text/event-stream",
                   "Authorization": "Bearer " + self.token}
        if self.session:
            headers["Mcp-Session-Id"] = self.session
        request = Request(self.endpoint, json.dumps(message).encode(), headers)
        with urlopen(request, timeout=15) as response:
            self.session = response.headers.get("Mcp-Session-Id", self.session)
            raw = response.read().decode()
            if notification:
                return {}
            if response.headers.get_content_type() == "application/json":
                return json.loads(raw)
            for line in raw.splitlines():
                if line.startswith("data:"):
                    message = json.loads(line[5:].strip())
                    if message.get("id") == identifier:
                        return message
        raise AssertionError("HTTP MCP response did not contain the requested JSON-RPC result: " + raw)


class MCPLimitsLibrary:
    ROBOT_LIBRARY_SCOPE = "SUITE"

    def prepare_mcp_limits(self, binary):
        candidate = Path(binary).resolve()
        if os.name == "nt" and candidate.with_suffix(".exe").is_file():
            candidate = candidate.with_suffix(".exe")
        assert candidate.is_file(), "Build StackQL first, or set STACKQL_EXE to a freshly built binary"
        self.binary = str(candidate)
        self.root = Path("build") / (".mcp-limits-" + uuid.uuid4().hex)
        self.root.mkdir(parents=True)
        self.root = self.root.resolve()
        self.condition = threading.Condition()
        self.calls = []
        self.cancelled = []
        self.release = threading.Event()
        self.closing = threading.Event()
        owner = self

        class API(BaseHTTPRequestHandler):
            def log_message(self, *_):
                pass

            def do_GET(self):
                region = parse_qs(urlsplit(self.path).query).get("region", [""])[0]
                with owner.condition:
                    owner.calls.append(region)
                    owner.condition.notify_all()
                deadline = time.monotonic() + 12
                while not owner.release.is_set() and not owner.closing.is_set():
                    readable, _, _ = select.select([self.connection], [], [], 0.02)
                    if readable and not self.connection.recv(1, socket.MSG_PEEK):
                        with owner.condition:
                            owner.cancelled.append(region)
                            owner.condition.notify_all()
                        return
                    if time.monotonic() > deadline:
                        return
                payload = json.dumps([{"id": region}]).encode()
                try:
                    self.send_response(200)
                    self.send_header("Content-Type", "application/json")
                    self.send_header("Content-Length", str(len(payload)))
                    self.end_headers()
                    self.wfile.write(payload)
                except (BrokenPipeError, ConnectionResetError):
                    pass

        self.provider = ThreadingHTTPServer(("127.0.0.1", 0), API)
        self.provider.daemon_threads = True
        self.worker = threading.Thread(target=self.provider.serve_forever, daemon=True)
        self.worker.start()
        endpoint = "http://127.0.0.1:" + str(self.provider.server_port)
        registry = self.root / "registry"
        version = registry / "src" / "mcp_limits" / "v00.00.00000"
        (version / "services").mkdir(parents=True)
        provider = {"id": "mcp_limits", "name": "mcp_limits", "version": "v00.00.00000",
                    "providerServices": {"testing": {"id": "testing:v00.00.00000", "name": "testing",
                        "preferred": True, "service": {"$ref": "mcp_limits/v00.00.00000/services/testing.yaml"}}},
                    "config": {"auth": {"type": "null_auth"}}}
        doc = {"openapi": "3.0.0", "info": {"title": "testing", "version": "v1"},
               "servers": [{"url": endpoint}], "paths": {"/items": {"get": {
                   "operationId": "list_items",
                   "parameters": [{"name": "region", "in": "query", "required": True, "schema": {"type": "string"}}],
                   "responses": {"200": {"content": {"application/json": {"schema": {
                       "type": "array", "items": {"type": "object", "properties": {"id": {"type": "string"}}}}}}}}}}},
               "components": {"x-stackQL-resources": {"items": {
                   "id": "mcp_limits.testing.items", "name": "items",
                   "methods": {"list": {"operation": {"$ref": "#/paths/~1items/get"},
                       "response": {"mediaType": "application/json", "openAPIDocKey": "200"}}},
                   "sqlVerbs": {"select": [{"$ref": "#/components/x-stackQL-resources/items/methods/list"}]}}}}}
        (version / "provider.yaml").write_text(json.dumps(provider), encoding="utf-8")
        (version / "services" / "testing.yaml").write_text(json.dumps(doc), encoding="utf-8")
        self.registry = json.dumps({"localDocRoot": str(registry), "verifyConfig": {"nopVerify": True}})

    def stop_mcp_limits(self):
        if hasattr(self, "closing"):
            self.closing.set()
            self.release.set()
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

    @contextlib.contextmanager
    def _server(self, transport, concurrent, timeout):
        run = self.root / uuid.uuid4().hex
        run.mkdir()
        token = uuid.uuid4().hex
        server = {"transport": transport, "mode": "read_only", "audit": {"disabled": True},
                  "max_concurrent_requests": concurrent, "request_timeout": timeout}
        if transport == "http":
            with socket.socket() as reservation:
                reservation.bind(("127.0.0.1", 0))
                address = "127.0.0.1:" + str(reservation.getsockname()[1])
            server.update(address=address, auth_token_env_var="STACKQL_MCP_LIMITS_TOKEN")
        env = dict(os.environ, GOMAXPROCS="2", STACKQL_MCP_LIMITS_TOKEN=token)
        command = [self.binary, "mcp", "--mcp.server.type=" + transport,
                   "--mcp.config", json.dumps({"server": server, "enabled_tools": ["server_info", "run_select_query"]}),
                   "--approot", str(run / "app"), "--registry", self.registry,
                   "--auth", json.dumps({"mcp_limits": {"type": "null_auth"}}),
                   "--sqlBackend", json.dumps({"dsn": str(run / "staging.db")})]
        process = subprocess.Popen(command, cwd=run, env=env, stdin=subprocess.PIPE,
                                   stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        diagnostics = []
        stderr = threading.Thread(target=lambda: diagnostics.extend(
            line.decode(errors="replace") for line in process.stderr), daemon=True)
        stderr.start()
        try:
            if transport == "stdio":
                client = _StdioClient(process)
            else:
                threading.Thread(target=lambda: process.stdout.read(), daemon=True).start()
                client = _HTTPClient("http://" + address, token)
                deadline = time.monotonic() + 10
                while True:
                    try:
                        with socket.create_connection(tuple([address.split(":")[0], int(address.split(":")[1])]), timeout=0.2):
                            break
                    except OSError:
                        assert process.poll() is None, "".join(diagnostics)
                        assert time.monotonic() < deadline, "HTTP listener did not start: " + "".join(diagnostics)
                        time.sleep(0.02)
            initialized = client.call("initialize", {
                "protocolVersion": "2025-06-18", "capabilities": {},
                "clientInfo": {"name": "mcp-limits-robot", "version": "1"}})
            assert "result" in initialized, initialized
            client.call("notifications/initialized", notification=True)
            yield client
        finally:
            self.release.set()
            if process.stdin and not process.stdin.closed:
                process.stdin.close()
            try:
                process.wait(timeout=3)
            except subprocess.TimeoutExpired:
                process.terminate()
                try:
                    process.wait(timeout=3)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait(timeout=3)
            stderr.join(timeout=3)
            for stream in (process.stdout, process.stderr):
                stream.close()

    def _wait_for(self, predicate, message):
        with self.condition:
            assert self.condition.wait_for(predicate, timeout=5), message + ": " + repr(self.calls)

    @staticmethod
    def _query(client, region):
        return client.call("tools/call", {"name": "run_select_query", "arguments": {
            "sql": "SELECT id FROM mcp_limits.testing.items WHERE region = '" + region + "';", "format": "json"}})

    @staticmethod
    def _success(response, region):
        assert "error" not in response and not response.get("result", {}).get("isError"), response
        assert region in json.dumps(response["result"]), response

    def configured_concurrency_bounds_provider_requests(self, transport):
        for limit in (1, 2):
            self.release.clear()
            first, second = (uuid.uuid4().hex for _ in range(2))
            with self._server(transport, limit, "10s") as client:
                with concurrent.futures.ThreadPoolExecutor(max_workers=2) as executor:
                    holder = executor.submit(self._query, client, first)
                    self._wait_for(lambda: first in self.calls, "first call did not reach provider")
                    queued = executor.submit(self._query, client, second)
                    try:
                        if limit == 1:
                            with self.condition:
                                assert not self.condition.wait_for(lambda: second in self.calls, timeout=0.3), (
                                    "max_concurrent_requests=1 admitted two blocked provider calls", self.calls)
                        else:
                            self._wait_for(lambda: second in self.calls, "max_concurrent_requests=2 did not admit both calls")
                        info = client.call("tools/call", {"name": "server_info"})
                        assert "error" not in info and not info["result"].get("isError"), info
                    finally:
                        self.release.set()
                    self._success(holder.result(timeout=12), first)
                    self._success(queued.result(timeout=12), second)

    def configured_deadline_cancels_provider_and_session_survives(self, transport):
        self.release.clear()
        slow, followup = (uuid.uuid4().hex for _ in range(2))
        with self._server(transport, 1, "1s") as client:
            with concurrent.futures.ThreadPoolExecutor(max_workers=1) as executor:
                started = time.monotonic()
                pending = executor.submit(self._query, client, slow)
                self._wait_for(lambda: slow in self.calls, "deadline call did not reach provider")
                response = pending.result(timeout=4)
            assert time.monotonic() - started < 4, "request_timeout did not bound the provider call"
            assert response.get("result", {}).get("isError"), response
            assert "server.request_timeout" in json.dumps(response), response
            self._wait_for(lambda: slow in self.cancelled, "deadline did not cancel the provider HTTP connection")
            self.release.set()
            self._success(self._query(client, followup), followup)
            assert "result" in client.call("ping"), "session died after the timeout"
