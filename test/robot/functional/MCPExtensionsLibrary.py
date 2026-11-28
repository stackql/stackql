"""MCP extensions (skills, tasks, file uploads) through the real native binary with raw JSON-RPC."""
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
from pathlib import Path
from urllib.error import HTTPError
from urllib.request import Request, urlopen

NEW = "2026-07-28"
LEGACY = "2025-11-25"
SKILLS = "io.modelcontextprotocol/skills"
TASKS = "io.modelcontextprotocol/tasks"
FILES = "io.modelcontextprotocol/files"
# The capabilities main advertises with no extensions configured.
BASELINE_CAPABILITIES = {
    "logging": {},
    "prompts": {"listChanged": True},
    "resources": {"listChanged": True},
    "tools": {"listChanged": True},
}
EXTENSION_ONLY_TOOLS = {"attach_policy_document"}


class _StdioClient:
    """One stackql mcp process over stdio. 2026-07-28 requests carry _meta;
    legacy requests follow an initialize handshake."""

    def __init__(self, command, cwd, env):
        self.process = subprocess.Popen(command, cwd=cwd, env=env, stdin=subprocess.PIPE,
                                        stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        self.responses = queue.Queue()
        self.diagnostics = []
        self.identifier = 0
        self.legacy = False

        def read_responses():
            try:
                for line in self.process.stdout:
                    message = json.loads(line)
                    if "id" in message:
                        self.responses.put(message)
            finally:
                self.responses.put(None)

        self.reader = threading.Thread(target=read_responses, daemon=True)
        self.stderr = threading.Thread(target=lambda: self.diagnostics.extend(
            line.decode(errors="replace") for line in self.process.stderr), daemon=True)
        self.reader.start()
        self.stderr.start()

    def _send(self, message):
        self.process.stdin.write((json.dumps(message) + "\n").encode())
        self.process.stdin.flush()

    def initialize(self, capabilities=None):
        """Legacy handshake; later requests omit _meta."""
        response = self.request("initialize", {
            "protocolVersion": LEGACY, "capabilities": capabilities or {},
            "clientInfo": {"name": "extensions-robot", "version": "1"},
        }, raw=True)
        self._send({"jsonrpc": "2.0", "method": "notifications/initialized"})
        self.legacy = True
        return response

    def request(self, method, params=None, capabilities=None, raw=False, timeout=30):
        self.identifier += 1
        params = dict(params or {})
        if not raw and not self.legacy:
            params["_meta"] = {
                "io.modelcontextprotocol/protocolVersion": NEW,
                "io.modelcontextprotocol/clientCapabilities": capabilities or {},
                "io.modelcontextprotocol/clientInfo": {"name": "extensions-robot", "version": "1"},
                **params.get("_meta", {}),
            }
        self._send({"jsonrpc": "2.0", "id": self.identifier, "method": method, "params": params})
        try:
            response = self.responses.get(timeout=timeout)
        except queue.Empty:
            raise AssertionError("MCP call timed out: " + "".join(self.diagnostics)) from None
        assert response is not None, "MCP process closed: " + "".join(self.diagnostics)
        assert response.get("id") == self.identifier, response
        return response

    def close(self):
        self.process.stdin.close()
        try:
            self.process.wait(timeout=10)
        except subprocess.TimeoutExpired:
            self.process.kill()
            self.process.wait(timeout=5)
        self.reader.join(timeout=5)
        self.stderr.join(timeout=5)
        self.process.stdout.close()
        self.process.stderr.close()


class _HTTPClient:
    """Sessionless Streamable HTTP at 2026-07-28."""

    def __init__(self, endpoint, token):
        self.endpoint = endpoint
        self.token = token
        self.identifier = 0
        self.lock = threading.Lock()

    def request(self, method, params=None, capabilities=None, token=None, timeout=30):
        with self.lock:
            self.identifier += 1
            request_id = self.identifier
        params = dict(params or {})
        params["_meta"] = {
            "io.modelcontextprotocol/protocolVersion": NEW,
            "io.modelcontextprotocol/clientCapabilities": capabilities or {},
            "io.modelcontextprotocol/clientInfo": {"name": "extensions-robot", "version": "1"},
        }
        headers = {"Content-Type": "application/json", "Accept": "application/json, text/event-stream",
                   "MCP-Protocol-Version": NEW, "Mcp-Method": method, "Connection": "close",
                   "Authorization": "Bearer " + (token or self.token)}
        name = {"tools/call": params.get("name"), "resources/read": params.get("uri"),
                "prompts/get": params.get("name")}.get(method)
        if name is not None:
            headers["Mcp-Name"] = name
        message = {"jsonrpc": "2.0", "id": request_id, "method": method, "params": params}
        request = Request(self.endpoint, json.dumps(message).encode(), headers)
        try:
            with urlopen(request, timeout=timeout) as response:
                assert not response.headers.get("Mcp-Session-Id"), "expected sessionless HTTP"
                raw = response.read().decode()
                if response.headers.get_content_type() == "application/json":
                    result = json.loads(raw)
                else:
                    events = [json.loads(line[5:].strip()) for line in raw.splitlines() if line.startswith("data:")]
                    result = next(event for event in events if event.get("id") == request_id)
        except HTTPError as error:
            with error:
                body = error.read().decode(errors="replace")
            # At 2026-07-28 a JSON-RPC error travels with a 4xx status.
            try:
                result = json.loads(body)
            except ValueError:
                result = None
            if not isinstance(result, dict) or result.get("id") != request_id or "error" not in result:
                raise AssertionError("MCP HTTP " + str(error.code) + ": " + body) from None
        assert result.get("id") == request_id, result
        return result


class MCPExtensionsLibrary:
    ROBOT_LIBRARY_SCOPE = "TEST"

    def prepare_mcp_extensions(self, binary):
        candidate = Path(binary).resolve()
        if os.name == "nt" and candidate.with_suffix(".exe").is_file():
            candidate = candidate.with_suffix(".exe")
        assert candidate.is_file(), "STACKQL_EXE must identify a freshly built native binary"
        self.binary = str(candidate)
        self.root = (Path("build") / (".mcp-extensions-" + uuid.uuid4().hex)).resolve()
        (self.root / "registry" / "src").mkdir(parents=True)
        self.auth = {}
        self.env = dict(os.environ, GOMAXPROCS="2")

    def stop_mcp_extensions(self):
        if hasattr(self, "provider"):
            self.provider.shutdown()
            self.provider.server_close()
            self.provider_worker.join(timeout=5)
        if hasattr(self, "root"):
            for attempt in range(50):
                try:
                    shutil.rmtree(self.root)
                    break
                except PermissionError:
                    if os.name != "nt" or attempt == 49:
                        raise
                    time.sleep(0.1)

    # --- harness -------------------------------------------------------------

    def _command(self, transport, server, run, enabled_tools=None, preview=False):
        config = {"server": {"transport": transport, "mode": "full_access", "audit": {"disabled": True}, **server}}
        if enabled_tools:
            config["enabled_tools"] = enabled_tools
        command = [
            self.binary, "mcp", "--mcp.server.type=" + transport, "--mcp.config", json.dumps(config),
            "--approot", str(run / "app"),
            "--registry", json.dumps({"localDocRoot": str(self.root / "registry"), "verifyConfig": {"nopVerify": True}}),
            "--auth", json.dumps(self.auth),
            "--sqlBackend", json.dumps({"dsn": str(run / "staging.db")}),
        ]
        if preview:
            command.extend(["--preview", json.dumps({"unstable": True, "iacStateRoot": str(run / "ledger")})])
        return command

    @contextlib.contextmanager
    def _stdio(self, server=None, **kwargs):
        run = self.root / uuid.uuid4().hex
        run.mkdir()
        client = _StdioClient(self._command("stdio", server or {}, run, **kwargs), run, self.env)
        try:
            yield client
        finally:
            client.close()

    @contextlib.contextmanager
    def _http(self, server=None, env=None, **kwargs):
        run = self.root / uuid.uuid4().hex
        run.mkdir()
        token = uuid.uuid4().hex
        with socket.socket() as reservation:
            reservation.bind(("127.0.0.1", 0))
            port = reservation.getsockname()[1]
        settings = {"address": "127.0.0.1:" + str(port), "stateless": True,
                    "auth_token_env_var": "STACKQL_MCP_EXTENSIONS_TOKEN", **(server or {})}
        process = subprocess.Popen(self._command("http", settings, run, **kwargs), cwd=run,
                                   env=dict(self.env, STACKQL_MCP_EXTENSIONS_TOKEN=token, **(env or {})),
                                   stdout=subprocess.DEVNULL, stderr=subprocess.PIPE)
        diagnostics = []
        reader = threading.Thread(target=lambda: diagnostics.extend(
            line.decode(errors="replace") for line in process.stderr), daemon=True)
        reader.start()
        try:
            deadline = time.monotonic() + 15
            while True:
                try:
                    with socket.create_connection(("127.0.0.1", port), timeout=0.2):
                        break
                except OSError:
                    assert process.poll() is None, "".join(diagnostics)
                    assert time.monotonic() < deadline, "HTTP MCP listener did not start"
                    time.sleep(0.05)
            yield _HTTPClient("http://127.0.0.1:" + str(port), token)
        finally:
            if process.poll() is None:
                process.terminate()
            try:
                process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait(timeout=5)
            reader.join(timeout=5)
            process.stderr.close()

    @staticmethod
    def _result(response):
        assert "error" not in response, response
        return response["result"]

    @staticmethod
    def _error_code(response):
        assert "error" in response, response
        return response["error"]["code"]

    def _surface(self, server):
        """Capabilities from server/discover (2026-07-28) and initialize
        (2025-11-25), plus every listed name."""
        with self._stdio(server) as client:
            discover = self._result(client.request("server/discover"))["capabilities"]
            names = {
                "tools": [t["name"] for t in self._result(client.request("tools/list"))["tools"]],
                "prompts": [p["name"] for p in self._result(client.request("prompts/list"))["prompts"]],
                "resources": [r["uri"] for r in self._result(client.request("resources/list"))["resources"]],
                "templates": [r["uriTemplate"] for r in
                              self._result(client.request("resources/templates/list"))["resourceTemplates"]],
            }
        with self._stdio(server) as client:
            initialize = self._result(client.initialize())["capabilities"]
        return discover, initialize, names

    # --- Group A: plumbing ---------------------------------------------------

    def discover_capabilities_unchanged_with_extensions_disabled(self):
        baseline = self._surface({})
        disabled = self._surface({"extensions": {
            "skills": {"enabled": False}, "tasks": {"enabled": False}, "files": {"enabled": False}}})
        assert baseline == disabled, (baseline, disabled)
        discover, initialize, names = disabled
        assert discover == BASELINE_CAPABILITIES, discover
        assert initialize == BASELINE_CAPABILITIES, initialize
        assert not EXTENSION_ONLY_TOOLS & set(names["tools"]), names["tools"]
        assert names["templates"] == [], names["templates"]
        assert not [uri for uri in names["resources"] if uri.startswith(("skill://", "stackql://policies/"))], names

    def discover_advertises_enabled_extensions(self):
        expected = {
            "skills": {SKILLS: {}},
            "tasks": {TASKS: {}},
            "files": {FILES: {"maxInlineBytes": 1048576}},
        }
        for name, extensions in expected.items():
            discover, initialize, _ = self._surface({"extensions": {name: {"enabled": True}}})
            for capabilities in (discover, initialize):
                assert capabilities.get("extensions") == extensions, (name, capabilities)
                assert {key: value for key, value in capabilities.items() if key != "extensions"} == \
                    BASELINE_CAPABILITIES, (name, capabilities)
        discover, _, _ = self._surface({"extensions": {"files": {"enabled": True, "max_inline_bytes": 2048}}})
        assert discover["extensions"] == {FILES: {"maxInlineBytes": 2048}}, discover

    # --- Group B: skills -----------------------------------------------------

    SKILL_MIME_TYPES = {".md": "text/markdown", ".yml": "application/yaml", ".iql": "text/plain"}

    @contextlib.contextmanager
    def _transport(self, transport, server, **kwargs):
        opener = self._stdio if transport == "stdio" else self._http
        with opener(server, **kwargs) as client:
            yield client

    def _check_skill_entry(self, client, entry):
        import hashlib
        name = entry["frontmatter"]["name"]
        assert entry["uri"] == "skill://" + name + "/SKILL.md", entry["uri"]
        assert entry["frontmatter"]["description"], entry
        uris = [resource["uri"] for resource in entry["resources"]]
        assert entry["uri"] in uris, uris
        assert uris == sorted(uris) and len(uris) == len(set(uris)), uris
        for resource in entry["resources"]:
            assert set(resource) == {"uri", "digest", "size"}, resource
            assert resource["uri"].startswith("skill://" + name + "/"), resource
            read = self._result(client.request("resources/read", {"uri": resource["uri"]}))
            content = read["contents"][0]
            data = content["text"].encode("utf-8")
            assert resource["digest"] == "sha256:" + hashlib.sha256(data).hexdigest(), resource
            assert resource["size"] == len(data), resource
            suffix = "." + resource["uri"].rsplit(".", 1)[-1]
            assert content["mimeType"] == self.SKILL_MIME_TYPES[suffix], content
            assert content["uri"] == resource["uri"], content

    def skills_list_returns_complete_entries_with_valid_digests(self):
        server = {"extensions": {"skills": {"enabled": True}}}
        for transport in ("stdio", "http"):
            with self._transport(transport, server) as client:
                listed = self._result(client.request("skills/list"))
                assert listed["resultType"] == "complete", listed
                assert listed["ttlMs"] == 30000 and listed["cacheScope"] == "private", listed
                assert "nextCursor" not in listed, listed
                names = sorted(entry["frontmatter"]["name"] for entry in listed["skills"])
                assert names == ["stackql-deploy", "stackql-iac"], names
                for entry in listed["skills"]:
                    self._check_skill_entry(client, entry)
                    got = self._result(client.request("skills/get", {"uri": entry["uri"]}))
                    assert got["skill"] == entry, (got, entry)
                    assert got["resultType"] == "complete" and got["ttlMs"] == 30000, got
        with self._stdio(server) as client:
            client.initialize()
            legacy = self._result(client.request("skills/list"))
            assert not {"resultType", "ttlMs", "cacheScope"} & set(legacy), legacy
            assert len(legacy["skills"]) == 2, legacy

    def skills_get_unknown_uri_is_invalid_params(self):
        server = {"extensions": {"skills": {"enabled": True}}}
        for transport in ("stdio", "http"):
            with self._transport(transport, server) as client:
                for uri in ("skill://no-such-skill/SKILL.md", "skill://stackql-iac/references/walkthrough.md", ""):
                    assert self._error_code(client.request("skills/get", {"uri": uri})) == -32602, uri
                assert self._error_code(client.request("skills/list", {"cursor": "next"})) == -32602
                assert "error" in client.request("resources/read", {"uri": "skill://stackql-iac/missing.md"})

    def skills_absent_when_disabled(self):
        for transport in ("stdio", "http"):
            with self._transport(transport, {}) as client:
                assert self._error_code(client.request("skills/list")) == -32601
                assert self._error_code(client.request("skills/get", {"uri": "skill://stackql-iac/SKILL.md"})) == -32601
                discover = self._result(client.request("server/discover"))["capabilities"]
                assert "extensions" not in discover, discover
                disabled = self._result(client.request("resources/list"))["resources"]
                templates = self._result(client.request("resources/templates/list"))["resourceTemplates"]
                assert templates == [], templates
            with self._transport(transport, {"extensions": {"skills": {"enabled": True}}}) as client:
                enabled = self._result(client.request("resources/list"))["resources"]
                templates = self._result(client.request("resources/templates/list"))["resourceTemplates"]
                assert [t["uriTemplate"] for t in templates] == ["skill://{+path}"], templates
            assert enabled == disabled, (enabled, disabled)

    def skills_allowlist_filters_entries(self):
        server = {"extensions": {"skills": {"enabled": True, "enabled_skills": ["stackql-iac"]}}}
        for transport in ("stdio", "http"):
            with self._transport(transport, server) as client:
                listed = self._result(client.request("skills/list"))["skills"]
                assert [entry["frontmatter"]["name"] for entry in listed] == ["stackql-iac"], listed
                self._check_skill_entry(client, listed[0])
                code = self._error_code(client.request("skills/get", {"uri": "skill://stackql-deploy/SKILL.md"}))
                assert code == -32602, code
                assert "error" in client.request("resources/read", {"uri": "skill://stackql-deploy/SKILL.md"})
