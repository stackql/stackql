"""Endpoint selectors using the existing native/Docker and isolated backend harness."""
import json
import subprocess
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

from ProviderSessionLibrary import ProviderSessionLibrary


class EndpointSelectorLibrary(ProviderSessionLibrary):
    def prepare_endpoint_selectors(self, binary, execution_platform="native", backend="{}"):
        self.prepare_session_provider(binary, execution_platform, backend)
        self.selector_calls, self.selector_servers = [], []
        self.selector_lock = threading.Lock()
        self.execution_platform = execution_platform
        owner = self

        def start(label):
            class API(BaseHTTPRequestHandler):
                def log_message(self, *_):
                    pass

                def do_GET(self):
                    self.reply([{"id": label}])

                def do_POST(self):
                    self.rfile.read(int(self.headers.get("Content-Length", "0")))
                    self.reply({"access_token": "selected-token", "token_type": "Bearer", "expires_in": 3600})

                def do_DELETE(self):
                    self.reply({"id": label})

                def reply(self, data):
                    if self.path != "/token" and self.headers.get("Authorization") != "Bearer selected-token":
                        self.send_error(401)
                        return
                    with owner.selector_lock:
                        owner.selector_calls.append((label, self.command, self.path))
                    raw = json.dumps(data).encode()
                    self.send_response(200)
                    self.send_header("Content-Type", "application/json")
                    self.send_header("Content-Length", str(len(raw)))
                    self.end_headers()
                    self.wfile.write(raw)

            address = "0.0.0.0" if execution_platform == "docker" else "127.0.0.1"
            server = ThreadingHTTPServer((address, 0), API)
            worker = threading.Thread(target=server.serve_forever, daemon=True)
            worker.start()
            self.selector_servers.append((server, worker))
            hostname = "host.docker.internal" if execution_platform == "docker" else "127.0.0.1"
            return f"{hostname}:{server.server_port}"

        self.selector_hosts = {name: start(name) for name in ["aws", "azure"]}
        self.selector_variable = {"default": "documentation-only.invalid", "x-stackQL-envVar": "STACKQL_SELECTOR_DIRECT",
                                  "x-stackQL-envVarSelector": {"envVar": "STACKQL_SELECTOR_CLOUD", "default": "aws", "values": self.selector_hosts}}
        version = self.root / "registry/src/widgets/v00.00.00000"
        provider = json.loads((version / "provider.yaml").read_text())
        provider["config"]["auth"] = {"type": "oauth2", "grant_type": "client_credentials", "token_url": "http://{account_host}/token",
                                        "variables": {"account_host": self.selector_variable}}
        doc = json.loads((version / "services/things.yaml").read_text())
        doc["servers"] = [{"url": "http://{account_host}", "variables": {"account_host": self.selector_variable}}]
        doc["paths"]["/widgets"]["get"]["parameters"] = []
        doc["paths"]["/widgets"]["delete"] = doc["paths"]["/widgets"]["get"].copy()
        resource = doc["components"]["x-stackQL-resources"]["widgets"]
        resource["methods"]["fetch"] = resource["methods"].pop("list")
        resource["sqlVerbs"]["select"] = [{"$ref": "#/components/x-stackQL-resources/widgets/methods/fetch"}]
        resource["methods"]["remove"] = {"operation": {"$ref": "#/paths/~1widgets/delete"}, "response": {"mediaType": "application/json", "openAPIDocKey": "200"}}
        resource["sqlVerbs"]["delete"] = [{"$ref": "#/components/x-stackQL-resources/widgets/methods/remove"}]
        (version / "provider.yaml").write_text(json.dumps(provider))
        (version / "services/things.yaml").write_text(json.dumps(doc))
        self.flags[self.flags.index("--preview") + 1] = json.dumps({"unstable": True})
        self.selector_auth = {"widgets": {"type": "client_credentials", "client_id": "dummy-client", "client_secret": "dummy-secret"}}

    def execute_selector_sql(self, sql, cloud="", direct="", auth=None, expected=()):
        env = dict(self.env, STACKQL_SELECTOR_CLOUD=cloud, STACKQL_SELECTOR_DIRECT=direct)
        prefix = list(self.prefix)
        if self.execution_platform == "docker":
            position = prefix.index("--entrypoint")
            prefix[position:position] = ["-e", "STACKQL_SELECTOR_CLOUD=" + cloud, "-e", "STACKQL_SELECTOR_DIRECT=" + direct]
        before = len(self.selector_calls)
        result = subprocess.run(prefix + ["exec", sql] + self.flags + ["--auth", json.dumps(auth or self.selector_auth)],
                                text=True, capture_output=True, env=env, timeout=30)
        assert result.returncode == 0 and all(line == "The operation was despatched successfully" for line in result.stderr.splitlines()), (result.stdout, result.stderr)
        assert self.selector_calls[before:] == list(expected), (self.selector_calls[before:], result.stdout, result.stderr)
        assert "dummy-secret" not in result.stdout, "credential leaked"
        return result

    def default_selector_matches_query_show_and_exec(self):
        result = self.execute_selector_sql("SHOW METHODS IN widgets.things.widgets;")
        methods = json.loads(result.stdout)
        assert methods and all(row["RequiredParams"] == "" for row in methods), result.stdout
        self.execute_selector_sql("SELECT id FROM widgets.things.widgets;", expected=[("aws", "POST", "/token"), ("aws", "GET", "/widgets")])
        self.execute_selector_sql("EXEC widgets.things.widgets.fetch;", expected=[("aws", "POST", "/token"), ("aws", "GET", "/widgets")])

    def selector_and_direct_override_apply_to_exec(self):
        for cloud, direct in [(" azure ", ""), ("invalid", " " + self.selector_hosts["azure"] + "/// ")]:
            self.execute_selector_sql("EXEC widgets.things.widgets.fetch;", cloud, direct,
                                      expected=[("azure", "POST", "/token"), ("azure", "GET", "/widgets")])

    def sql_and_alias_defaults_override_environment(self):
        auth = {"type": "client_credentials", "client_id": "dummy-client", "client_secret": "dummy-secret", "token_url": "http://" + self.selector_hosts["azure"] + "/token"}
        setup = self.setting("providers.tenant", {"provider": "widgets", "tenancy": "tenant-account", "auth": auth, "params": {"account_host": self.selector_hosts["azure"]}})
        result = self.execute_selector_sql(setup + "SHOW METHODS IN tenant.things.widgets;", cloud="invalid")
        assert all(row["RequiredParams"] == "" for row in json.loads(result.stdout)), result.stdout
        self.execute_selector_sql(setup + "SELECT id FROM tenant.things.widgets;", cloud="invalid",
                                  expected=[("azure", "POST", "/token"), ("azure", "GET", "/widgets")])
        self.execute_selector_sql(setup + "EXEC tenant.things.widgets.fetch;", cloud="invalid",
                                  expected=[("azure", "POST", "/token"), ("azure", "GET", "/widgets")])
        sql = setup + "EXEC tenant.things.widgets.fetch @account_host = '" + self.selector_hosts["aws"] + "';"
        self.execute_selector_sql(sql, cloud="invalid", expected=[("azure", "POST", "/token"), ("aws", "GET", "/widgets")])
        sql = setup + "SELECT id FROM tenant.things.widgets WHERE account_host = '" + self.selector_hosts["aws"] + "';"
        self.execute_selector_sql(sql, cloud="invalid", expected=[("azure", "POST", "/token"), ("aws", "GET", "/widgets")])

    def mutation_uses_the_same_selector(self):
        self.execute_selector_sql("DELETE FROM widgets.things.widgets;", cloud="azure",
                                  expected=[("azure", "POST", "/token"), ("azure", "DELETE", "/widgets")])

    def unknown_selector_fails_before_requests(self):
        env = dict(self.env, STACKQL_SELECTOR_CLOUD="typo", STACKQL_SELECTOR_DIRECT="")
        prefix = list(self.prefix)
        if self.execution_platform == "docker":
            position = prefix.index("--entrypoint")
            prefix[position:position] = ["-e", "STACKQL_SELECTOR_CLOUD=typo", "-e", "STACKQL_SELECTOR_DIRECT="]
        for sql in ["SELECT id FROM widgets.things.widgets;", "SHOW METHODS IN widgets.things.widgets;", "EXEC widgets.things.widgets.fetch;"]:
            before = len(self.selector_calls)
            result = subprocess.run(prefix + ["exec", sql] + self.flags + ["--auth", json.dumps(self.selector_auth)], text=True, capture_output=True, env=env, timeout=30)
            assert "STACKQL_SELECTOR_CLOUD" in result.stderr and "typo" in result.stderr and "aws" in result.stderr and "azure" in result.stderr, result.stderr
            assert len(self.selector_calls) == before, "invalid selector sent a provider request"

    def missing_declared_source_stays_required(self):
        version = self.root / "registry/src/widgets/v00.00.00000"
        paths = [version / "provider.yaml", version / "services/things.yaml"]
        original = {path: path.read_text() for path in paths}
        variable = {"default": "must-not-be-used.invalid", "x-stackQL-envVar": "STACKQL_SELECTOR_DIRECT"}
        try:
            provider = json.loads(original[paths[0]])
            provider["config"]["auth"]["variables"]["account_host"] = variable
            document = json.loads(original[paths[1]])
            document["servers"][0]["variables"]["account_host"] = variable
            paths[0].write_text(json.dumps(provider))
            paths[1].write_text(json.dumps(document))
            result = self.execute_selector_sql("SHOW METHODS IN widgets.things.widgets;")
            assert "account_host" in result.stdout, result.stdout
            env = dict(self.env, STACKQL_SELECTOR_CLOUD="", STACKQL_SELECTOR_DIRECT="")
            prefix = list(self.prefix)
            if self.execution_platform == "docker":
                position = prefix.index("--entrypoint")
                prefix[position:position] = ["-e", "STACKQL_SELECTOR_CLOUD=", "-e", "STACKQL_SELECTOR_DIRECT="]
            for sql in ["SELECT id FROM widgets.things.widgets;", "EXEC widgets.things.widgets.fetch;"]:
                before = len(self.selector_calls)
                result = subprocess.run(prefix + ["exec", sql] + self.flags + ["--auth", json.dumps(self.selector_auth)],
                                        text=True, capture_output=True, env=env, timeout=30)
                assert "account_host" in result.stderr and "requires an input" in result.stderr, result.stderr
                assert len(self.selector_calls) == before, "missing source sent a provider request"
        finally:
            for path, body in original.items():
                path.write_text(body)

    def stop_endpoint_selectors(self):
        for server, worker in getattr(self, "selector_servers", []):
            server.shutdown()
            server.server_close()
            worker.join(timeout=5)
        self.stop_session_provider()
