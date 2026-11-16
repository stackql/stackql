"""Isolated real-binary coverage of cache policy and OAuth opt-out compatibility."""
import json
import os
import queue
import shutil
import socket
import subprocess
import threading
import time
import uuid
from pathlib import Path
from urllib.error import HTTPError, URLError
from urllib.request import Request, urlopen


class MCPCacheAuthLibrary:
    ROBOT_LIBRARY_SCOPE = "SUITE"

    def prepare_cache_auth_fixture(self, binary):
        self.processes = []
        self.root = None
        candidate = Path(binary).resolve()
        if os.name == "nt" and candidate.with_suffix(".exe").is_file():
            candidate = candidate.with_suffix(".exe")
        assert candidate.is_file(), "Set STACKQL_EXE to a freshly built StackQL binary"
        self.binary = str(candidate)
        self.root = (Path("build") / (".mcp-cache-auth-" + uuid.uuid4().hex)).resolve()
        (self.root / "registry" / "src").mkdir(parents=True)
        self.registry = json.dumps({
            "url": (self.root / "registry").as_uri(),
            "localDocRoot": str(self.root / "registry"),
            "verifyConfig": {"nopVerify": True},
        })
        self.token = uuid.uuid4().hex
        self.meta = {
            "io.modelcontextprotocol/protocolVersion": "2026-07-28",
            "io.modelcontextprotocol/clientCapabilities": {},
            "io.modelcontextprotocol/clientInfo": {"name": "cache-auth-robot", "version": "1"},
        }

    def _start(self, transport, restricted=False, authenticated=False, legacy=False):
        instance = self.root / uuid.uuid4().hex
        instance.mkdir()
        config = {"server": {
            "transport": transport,
            "protocol_version": "2025-11-25" if legacy else "2026-07-28",
            "mode": "full_access",
            "audit": {"disabled": True},
        }}
        if restricted:
            config["enabled_tools"] = ["server_info"]
        endpoint = None
        if transport == "http":
            with socket.socket() as reservation:
                reservation.bind(("127.0.0.1", 0))
                port = reservation.getsockname()[1]
            endpoint = "http://127.0.0.1:" + str(port)
            config["server"]["address"] = "127.0.0.1:" + str(port)
            if authenticated:
                config["server"]["auth_token_env_var"] = "STACKQL_CACHE_AUTH_TEST_TOKEN"
            else:
                config["server"]["allow_unauthenticated"] = True
        env = os.environ.copy()
        env["STACKQL_CACHE_AUTH_TEST_TOKEN"] = self.token
        process = subprocess.Popen([
            self.binary, "mcp", "--mcp.server.type=" + transport,
            "--mcp.config", json.dumps(config), "--registry", self.registry,
            "--auth", "{}", "--approot", str(instance),
        ], stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE, env=env)
        output = queue.Queue()
        logs = []

        def read_stdout():
            for line in process.stdout:
                if transport == "stdio":
                    try:
                        output.put(json.loads(line))
                    except ValueError:
                        logs.append(line)
                else:
                    logs.append(line)

        def read_stderr():
            logs.extend(process.stderr.readlines())

        readers = [
            threading.Thread(target=read_stdout, daemon=True),
            threading.Thread(target=read_stderr, daemon=True),
        ]
        for reader in readers:
            reader.start()
        self.processes.append((process, readers, logs))
        client = {"process": process, "output": output, "endpoint": endpoint, "id": 0,
                  "protocol": config["server"]["protocol_version"], "session": ""}
        if endpoint:
            deadline = time.monotonic() + 15
            while time.monotonic() < deadline:
                assert process.poll() is None, "MCP process exited before becoming ready"
                try:
                    method = "initialize" if legacy else "tools/list"
                    params = {
                        "protocolVersion": "2025-11-25", "capabilities": {},
                        "clientInfo": {"name": "cache-auth-legacy", "version": "1"},
                    } if legacy else None
                    status, _ = self._call(client, method, params, token=self.token if authenticated else None)
                    if status == 200:
                        break
                except (URLError, OSError):
                    pass
                time.sleep(0.05)
            else:
                raise AssertionError("MCP HTTP endpoint did not become ready")
        return client

    def _call(self, client, method, params=None, token=None):
        client["id"] += 1
        meta = self.meta if client["protocol"] == "2026-07-28" else {}
        payload = {"jsonrpc": "2.0", "id": client["id"], "method": method,
                   "params": {"_meta": meta, **(params or {})}}
        if client["endpoint"]:
            headers = {"Content-Type": "application/json", "Accept": "application/json, text/event-stream",
                       "Mcp-Protocol-Version": client["protocol"], "Mcp-Method": method}
            if client["session"]:
                headers["Mcp-Session-Id"] = client["session"]
            if method == "resources/read":
                headers["Mcp-Name"] = params["uri"]
            if token:
                headers["Authorization"] = "Bearer " + token
            request = Request(client["endpoint"], json.dumps(payload).encode(), headers)
            try:
                with urlopen(request, timeout=10) as response:
                    raw = response.read()
                    status = response.status
                    client["session"] = response.headers.get("Mcp-Session-Id", client["session"])
            except HTTPError as error:
                error.close()
                return error.code, {}
            text = raw.decode()
            if text.startswith("data:") or "\ndata:" in text:
                frames = [json.loads(line[5:].strip()) for line in text.splitlines() if line.startswith("data:")]
                message = next(frame for frame in frames if frame.get("id") == payload["id"])
            else:
                message = json.loads(text)
        else:
            client["process"].stdin.write((json.dumps(payload) + "\n").encode())
            client["process"].stdin.flush()
            message = client["output"].get(timeout=15)
            status = 200
            assert message.get("id") == payload["id"], "Unexpected stdio reply"
        assert "error" not in message, "MCP request failed for method " + method
        assert "result" in message, "MCP reply missing result for " + method + ": " + str(sorted(message))
        return status, message.get("result", {})

    def _check_cache(self, result, ttl):
        assert result.get("cacheScope") == "private", "Cache metadata must be caller-private: " + str(sorted(result))
        assert result.get("ttlMs") == ttl, "Unexpected cache freshness policy"

    def cache_catalogues_are_private_and_reads_immediately_stale(self):
        full = self._start("stdio")
        restricted = self._start("stdio", restricted=True)
        _, full_tools = self._call(full, "tools/list")
        _, restricted_tools = self._call(restricted, "tools/list")
        full_names = [tool["name"] for tool in full_tools["tools"]]
        assert len(full_names) > 1 and full_names == sorted(full_names), "Full catalogue is missing or unordered"
        assert [tool["name"] for tool in restricted_tools["tools"]] == ["server_info"], "Restricted catalogue leaked full-server tools"
        for client in (full, restricted):
            for method in ("tools/list", "prompts/list", "resources/list", "resources/templates/list"):
                _, result = self._call(client, method)
                self._check_cache(result, 30000)
            _, resources = self._call(client, "resources/list")
            assert resources["resources"], "Expected embedded MCP resources"
            for _ in range(2):
                _, read = self._call(client, "resources/read", {"uri": resources["resources"][0]["uri"]})
                self._check_cache(read, 0)
                assert read.get("contents"), "Resource read returned no content"
                _, discover = self._call(client, "server/discover")
                self._check_cache(discover, 0)
                assert discover.get("supportedVersions"), "Discovery returned no supported versions"
        _, again = self._call(full, "tools/list")
        assert [tool["name"] for tool in again["tools"]] == full_names, "Restricted catalogue contaminated the full caller"

    def oauth_disabled_http_preserves_auth_after_cache_warming(self):
        client = self._start("http", authenticated=True)
        status, warmed = self._call(client, "tools/list", token=self.token)
        assert status == 200, "Authorized catalogue returned HTTP " + str(status)
        self._check_cache(warmed, 30000)
        for token in (None, "wrong-token"):
            status, result = self._call(client, "tools/list", token=token)
            assert status == 401 and not result, "Warm private catalogue bypassed bearer authentication"
        status, second_caller = self._call(client, "tools/list", token=self.token)
        assert status == 200 and second_caller == warmed, "Fresh authorized caller lost its catalogue"
        status, resources = self._call(client, "resources/list", token=self.token)
        assert status == 200, "Authorized resource list returned HTTP " + str(status)
        status, read = self._call(client, "resources/read", {"uri": resources["resources"][0]["uri"]}, token=self.token)
        assert status == 200, "Authorized resource read returned HTTP " + str(status)
        self._check_cache(read, 0)
        anonymous = self._start("http")
        status, anonymous_tools = self._call(anonymous, "tools/list")
        assert status == 200 and anonymous_tools.get("tools"), "Explicit unauthenticated opt-in stopped working"
        self._check_cache(anonymous_tools, 30000)
        legacy = self._start("http", authenticated=True, legacy=True)
        assert legacy["session"], "OAuth opt-out lost legacy HTTP initialization sessions"
        status, legacy_tools = self._call(legacy, "tools/list", token=self.token)
        assert status == 200 and legacy_tools.get("tools"), "Legacy authenticated session cannot list tools"
        status, _ = self._call(legacy, "tools/list")
        assert status == 401, "Legacy session ID bypassed bearer authentication"

    def stop_cache_auth_fixture(self):
        clean_logs = True
        try:
            for process, readers, logs in getattr(self, "processes", []):
                if process.poll() is None:
                    process.terminate()
                try:
                    process.wait(timeout=5)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait(timeout=5)
                for reader in readers:
                    reader.join(timeout=5)
                clean_logs = clean_logs and self.token.encode() not in b"".join(logs)
                for stream in (process.stdin, process.stdout, process.stderr):
                    stream.close()
        finally:
            if getattr(self, "root", None):
                deadline = time.monotonic() + 5
                while True:
                    try:
                        shutil.rmtree(self.root)
                        break
                    except PermissionError:
                        if time.monotonic() >= deadline:
                            raise
                        time.sleep(0.05)
        assert clean_logs, "HTTP logs exposed the test bearer token"
