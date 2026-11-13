"""Session configuration through the CLI and a loopback-only provider."""
import json
import os
import shutil
import shlex
import subprocess
import tempfile
import threading
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import parse_qs, urlparse, urlsplit, urlunsplit, parse_qsl, urlencode


class ProviderSessionLibrary:
    ROBOT_LIBRARY_SCOPE = "SUITE"

    def prepare_session_provider(self, binary, execution_platform="native", backend="{}"):
        self.prefix = [shutil.which(binary) or str(Path(binary).resolve())]
        self.root = Path(tempfile.mkdtemp(prefix="stackql-session-"))
        self.calls = []
        self.aws_calls = []
        self.concurrent_gate = None
        self.env = os.environ.copy()
        if os.name != "nt":
            self.prepare_os_secret_mock()
        owner = self

        class API(BaseHTTPRequestHandler):
            def log_message(self, *_):
                pass

            def do_GET(self):
                token = self.headers.get("Authorization", "")
                if urlparse(self.path).path == "/buckets":
                    import re
                    match = re.search(r"Credential=([^/]+)/", token)
                    key = match.group(1) if match else ""
                    owner.aws_calls.append((key, parse_qs(urlparse(self.path).query).get("region", [""])[0], token))
                    account = {"TEST_RYUK_KEY": "ryuk", "TEST_STACKQL_KEY": "stackql"}.get(key)
                    if account is None or not token.startswith("AWS4-HMAC-SHA256 ") or "Signature=" not in token:
                        self.send_error(403)
                        return
                    import hashlib
                    import hmac
                    scope = re.search(r"Credential=[^/]+/([^, ]+)", token).group(1)
                    signed_headers = re.search(r"SignedHeaders=([^, ]+)", token).group(1)
                    headers = "".join(name + ":" + " ".join(self.headers[name].split()) + "\n" for name in signed_headers.split(";"))
                    url = urlsplit(self.path)
                    payload_hash = self.headers.get("X-Amz-Content-Sha256", hashlib.sha256(b"").hexdigest())
                    canonical = "\n".join(["GET", url.path, url.query, headers, signed_headers, payload_hash])
                    to_sign = "\n".join(["AWS4-HMAC-SHA256", self.headers["X-Amz-Date"], scope, hashlib.sha256(canonical.encode()).hexdigest()])
                    secret = {"ryuk": "dummy-ryuk-secret", "stackql": "dummy-stackql-secret"}[account]
                    signing_key = ("AWS4" + secret).encode()
                    for component in scope.split("/"):
                        signing_key = hmac.new(signing_key, component.encode(), hashlib.sha256).digest()
                    expected = hmac.new(signing_key, to_sign.encode(), hashlib.sha256).hexdigest()
                    if not hmac.compare_digest(expected, token.split("Signature=")[1]):
                        self.send_error(403, "incorrect account signing secret")
                        return
                    payload = json.dumps([{"name": account + "-bucket"}]).encode()
                    self.send_response(200)
                    self.send_header("Content-Type", "application/json")
                    self.send_header("Content-Length", str(len(payload)))
                    self.end_headers()
                    self.wfile.write(payload)
                    return
                region = parse_qs(urlparse(self.path).query).get("region", [""])[0]
                if owner.concurrent_gate is not None:
                    owner.concurrent_gate.wait(timeout=10)
                owner.calls.append((token, region))
                payload = json.dumps([{"id": region or "account", "active": True, "count": 42}]).encode()
                self.send_response(200)
                self.send_header("Content-Type", "application/json")
                self.send_header("Content-Length", str(len(payload)))
                self.end_headers()
                self.wfile.write(payload)

        self.server = ThreadingHTTPServer(("0.0.0.0" if execution_platform == "docker" else "127.0.0.1", 0), API)
        self.worker = threading.Thread(target=self.server.serve_forever, daemon=True)
        self.worker.start()
        endpoint = f"http://127.0.0.1:{self.server.server_port}"
        if execution_platform == "docker":
            self.prefix = ["docker", "--log-level", "error", "compose", "--progress", "quiet", "-p", "execrun", "run", "--rm", "--no-deps", "-T",
                           "--user", f"{os.getuid()}:{os.getgid()}", "--workdir", str(self.root),
                           "-v", str(self.root) + ":" + str(self.root), "--entrypoint", "/bin/sh", "stackqlsrv",
                           "-c", 'PATH="$1:$PATH"; export PATH; shift; exec stackql "$@"',
                           "session-provider", str(self.root / "os-bin")]
            endpoint = endpoint.replace("127.0.0.1", "host.docker.internal")
        provider = {"id": "widgets", "name": "widgets", "version": "v00.00.00000",
                    "providerServices": {"things": {"id": "things:v00.00.00000", "name": "things", "preferred": True,
                                                     "service": {"$ref": "widgets/v00.00.00000/services/things.yaml"}}},
                    "config": {"auth": {"type": "bearer"}}}
        doc = {"openapi": "3.0.0", "info": {"title": "things", "version": "v1"}, "servers": [{"url": endpoint}],
               "paths": {"/widgets": {"get": {"operationId": "list_widgets",
                         "parameters": [{"name": "region", "in": "query", "required": True, "schema": {"type": "string"}}],
                         "responses": {"200": {"content": {"application/json": {"schema": {"type": "array", "items": {"type": "object", "properties": {"id": {"type": "string"}, "active": {"type": "boolean"}, "count": {"type": "integer"}}}}}}}}}}},
               "components": {"x-stackQL-resources": {"widgets": {"id": "widgets.things.widgets", "name": "widgets",
                              "methods": {"list": {"operation": {"$ref": "#/paths/~1widgets/get"}, "response": {"mediaType": "application/json", "openAPIDocKey": "200"}}},
                              "sqlVerbs": {"select": [{"$ref": "#/components/x-stackQL-resources/widgets/methods/list"}]}}}}}
        registry = self.root / "registry"
        version = registry / "src/widgets/v00.00.00000"
        (version / "services").mkdir(parents=True)
        (version / "provider.yaml").write_text(json.dumps(provider))
        (version / "services/things.yaml").write_text(json.dumps(doc))
        self.backend = json.loads(backend)
        self.postgres = self.backend.get("dbEngine") == "postgres_tcp"
        if self.postgres:
            import psycopg
            import time
            import uuid
            dsn = self.backend["dsn"]
            if execution_platform == "docker":
                subprocess.run(["docker", "--log-level", "error", "compose", "--progress", "quiet", "-p", "execrun", "up", "-d", "postgres_stackql"], capture_output=True, check=True, timeout=120)
                published = subprocess.check_output(["docker", "--log-level", "error", "compose", "--progress", "quiet", "-p", "execrun", "port", "postgres_stackql", "5432"], text=True).strip().splitlines()[0].rsplit(":", 1)[1]
                url = urlsplit(dsn)
                dsn = urlunsplit((url.scheme, url.netloc.rsplit("@", 1)[0] + "@127.0.0.1:" + published, url.path, url.query, url.fragment))
            for attempt in range(100):
                try:
                    self.database = psycopg.connect(dsn, autocommit=True)
                    break
                except psycopg.OperationalError:
                    if attempt == 99:
                        raise
                    time.sleep(0.1)
            self.database_schema = 'session_test_' + uuid.uuid4().hex
            self.database.execute(psycopg.sql.SQL('CREATE SCHEMA {}').format(psycopg.sql.Identifier(self.database_schema)))
            url = urlsplit(self.backend['dsn'])
            options = dict(parse_qsl(url.query))
            options['options'] = '-csearch_path=' + self.database_schema + ',public'
            self.backend['dsn'] = urlunsplit((url.scheme, url.netloc, url.path, urlencode(options), url.fragment))
        else:
            self.backend = {"dsn": str(self.root / "session.db")}
        self.flags = ["--approot", str(self.root / "app"), "--registry", json.dumps({"localDocRoot": str(registry), "verifyConfig": {"nopVerify": True}}),
                      "--preview", json.dumps({"unstable": True, "endpoint": endpoint}), "--output", "json", "--sqlBackend", json.dumps(self.backend)]

    @staticmethod
    def setting(path, value, scope=""):
        raw = value if isinstance(value, str) else json.dumps(value)
        return f'SET {scope} "$.{path}" = ' + "'" + raw.replace("'", "''") + "';"

    def run_sql(self, sql, expected_calls, extra_flags=(), completion_messages=()):
        before = len(self.calls)
        result = subprocess.run(self.prefix + ["exec", sql] + self.flags + list(extra_flags), text=True, capture_output=True, timeout=30, env=self.env)
        assert result.returncode == 0, result.stderr
        assert all(line in completion_messages for line in result.stderr.splitlines()), result.stderr
        assert self.calls[before:] == expected_calls, (self.calls[before:], result.stdout, result.stderr)
        assert "east-token" not in result.stdout and "west-token" not in result.stdout, "credential leaked"
        return result

    def provider_aliases_survive_session_edits(self):
        sql = ""
        for name in ["east", "west"]:
            sql += self.setting("providers." + name, {"provider": "widgets", "tenancy": name, "auth": {"type": "bearer", "credentials": name + "-token"}, "params": {"region": name}, "tuning": {"max_concurrent": 1}})
        sql += "SHOW SERVICES IN east; SHOW RESOURCES IN east.things; SHOW METHODS IN east.things.widgets; DESCRIBE east.things.widgets; SELECT * FROM east.things.widgets; SELECT id FROM west.things.widgets;"
        sql += self.setting("providers.east.auth.credentials", "changed-token")
        sql += "SELECT id FROM east.things.widgets WHERE region = 'override';"
        self.run_sql(sql, [("Bearer east-token", "east"), ("Bearer west-token", "west"), ("Bearer changed-token", "override")])

    def aws_aliases_union_buckets_with_independent_signing_credentials(self):
        registry = self.root / "registry/src"
        provider = json.loads((registry / "widgets/v00.00.00000/provider.yaml").read_text())
        doc = json.loads((registry / "widgets/v00.00.00000/services/things.yaml").read_text())
        provider = json.loads(json.dumps(provider).replace("widgets", "aws").replace("things", "s3"))
        provider["config"]["auth"] = {"type": "aws_signing_v4"}
        doc = json.loads(json.dumps(doc).replace("widgets.things.widgets", "aws.s3.buckets").replace("widgets", "buckets"))
        doc["paths"]["/buckets"]["get"]["responses"]["200"]["content"]["application/json"]["schema"]["items"]["properties"] = {"name": {"type": "string"}}
        version = registry / "aws/v00.00.00000"
        (version / "services").mkdir(parents=True)
        (version / "provider.yaml").write_text(json.dumps(provider))
        (version / "services/s3.yaml").write_text(json.dumps(doc))
        env_names = ["TEST_RYUK_ACCESS_KEY", "TEST_RYUK_SECRET_KEY", "TEST_STACKQL_ACCESS_KEY", "TEST_STACKQL_SECRET_KEY"]
        values = ["TEST_RYUK_KEY", "dummy-ryuk-secret", "TEST_STACKQL_KEY", "dummy-stackql-secret"]
        previous = {name: self.env.get(name) for name in env_names}
        self.env.update(zip(env_names, values))
        prefix = self.prefix
        if prefix[0] == "docker":
            self.prefix = list(prefix)
            position = self.prefix.index("--entrypoint")
            self.prefix[position:position] = [part for name, value in zip(env_names, values) for part in ("-e", name + "=" + value)]
        before = len(self.aws_calls)
        try:
            sql = ""
            for account, tuning in [("ryuk", {"max_concurrent": 2}), ("stackql", {"max_concurrent": 4, "max_per_host": 4})]:
                env_prefix = "TEST_" + account.upper()
                sql += self.setting("providers.aws_" + account, {
                    "provider": "aws", "tenancy": account + "-account",
                    "auth": {"type": "aws_signing_v4", "access_key_id_env_var": env_prefix + "_ACCESS_KEY", "secret_access_key_env_var": env_prefix + "_SECRET_KEY"},
                    "params": {"region": "ap-southeast-2"}, "tuning": tuning})
            sql += "SELECT name, 'ryuk' AS tenant FROM aws_ryuk.s3.buckets UNION ALL SELECT name, 'stackql' AS tenant FROM aws_stackql.s3.buckets;"
            result = self.run_sql(sql, [])
            assert sorted(json.loads(result.stdout), key=lambda row: row["tenant"]) == [
                {"name": "ryuk-bucket", "tenant": "ryuk"}, {"name": "stackql-bucket", "tenant": "stackql"}], result.stdout
            calls = self.aws_calls[before:]
            assert sorted((key, region) for key, region, _ in calls) == [
                ("TEST_RYUK_KEY", "ap-southeast-2"), ("TEST_STACKQL_KEY", "ap-southeast-2")], calls
            assert calls[0][2].split("Signature=")[1] != calls[1][2].split("Signature=")[1], calls
            assert all(value not in result.stdout + result.stderr for value in values), "credential leaked"
        finally:
            self.prefix = prefix
            for name, value in previous.items():
                if value is None:
                    self.env.pop(name, None)
                else:
                    self.env[name] = value

    def secret_credentials_rotate_and_reset(self):
        sql = self.setting("secret_store", {"backend": "os", "namespace": "stackql-test-" + self.root.name})
        sql += self.setting("secrets.account", json.dumps({"type": "bearer", "credentials": "first-token"}))
        sql += self.setting("providers.secure", {"provider": "widgets", "tenancy": "secure-account", "auth_secret": "account", "params": {"region": "secure"}})
        sql += "SELECT id FROM secure.things.widgets;"
        sql += self.setting("secrets.account", json.dumps({"type": "bearer", "credentials": "rotated-token"}))
        sql += "SELECT id FROM secure.things.widgets; UNSET \"$.providers.secure\"; UNSET \"$.secrets.account\";"
        self.run_sql(sql, [("Bearer first-token", "secure"), ("Bearer rotated-token", "secure")], ["--allow-persistent-secrets"])
        missing = self.setting("secret_store", {"backend": "os", "namespace": "stackql-test-" + self.root.name})
        missing += self.setting("providers.secure", {"provider": "widgets", "tenancy": "secure", "auth_secret": "account", "params": {"region": "secure"}})
        missing += "SELECT id FROM secure.things.widgets;"
        before = len(self.calls)
        result = subprocess.run(self.prefix + ["exec", missing, "--allow-persistent-secrets"] + self.flags,
                                text=True, capture_output=True, timeout=30, env=self.env)
        assert "secret" in result.stderr.lower() and len(self.calls) == before, (result.stdout, result.stderr, self.calls[before:])

    def auth_dto_changes_and_unsets(self):
        sql = self.setting("auth.widgets", {"type": "api_key", "credentialsenvvar": "STACKQL_UNUSED_CREDENTIAL"})
        sql += 'UNSET "$.auth.widgets.credentialsenvvar";'
        sql += self.setting("auth.widgets", {"type": "null_auth"})
        sql += "SELECT id FROM widgets.things.widgets WHERE region = 'plain';"
        self.run_sql(sql, [("", "plain")])

    def unset_alias_auth_fails_before_request(self):
        sql = self.setting("providers.disabled", {"provider": "widgets", "tenancy": "disabled-account", "auth": {"type": "bearer", "credentials": "explicit-token"}, "params": {"region": "disabled"}})
        sql += 'UNSET "$.providers.disabled.auth"; SELECT id FROM disabled.things.widgets;'
        before = len(self.calls)
        result = subprocess.run(self.prefix + ["exec", sql] + self.flags, text=True, capture_output=True, timeout=30, env=self.env)
        assert "requires resolved explicit credentials" in result.stderr, result.stderr
        assert len(self.calls) == before, "missing auth reached the provider"

    def stop_session_provider(self):
        if hasattr(self, 'database'):
            if hasattr(self, 'database_schema'):
                import psycopg
                self.database.execute(psycopg.sql.SQL('DROP SCHEMA {} CASCADE').format(psycopg.sql.Identifier(self.database_schema)))
            self.database.close()
        if hasattr(self, "server"):
            self.server.shutdown()
            self.server.server_close()
            self.worker.join(timeout=5)
        if hasattr(self, "root"):
            shutil.rmtree(self.root)

    def explicit_session_scope(self):
        binding = {"provider":"widgets", "tenancy":"scoped", "auth":{"type":"bearer","credentials":"scoped-token"}, "params":{"region":"scoped"}}
        sql = self.setting("providers.scoped", binding, "SESSION")
        sql += "SELECT id FROM scoped.things.widgets; UNSET SESSION \"$.providers.scoped\", \"$.auth.widgets.password\";"
        self.run_sql(sql, [("Bearer scoped-token", "scoped")])

    def unsupported_configuration_scope(self):
        for statement in ['SET GLOBAL "auth.widgets.password" = \'value\';', 'UNSET GLOBAL "auth.widgets.password";']:
            before = len(self.calls)
            result = subprocess.run(self.prefix + ["exec", statement] + self.flags, text=True, capture_output=True, timeout=30, env=self.env)
            assert result.returncode != 0 and "not supported" in result.stderr, result.stderr
            assert len(self.calls) == before, "unsupported scope reached provider"

    def dollar_targets_configure_runtime_and_plain_targets_store_variables(self):
        binding = json.dumps({"provider": "widgets", "tenancy": "plain", "auth": {"type": "bearer", "credentials": "plain-token"}, "params": {"region": "plain"}})
        sql = 'SET "providers.plain" = ' + "'" + binding + "';"
        sql += 'SELECT "$providers.plain" AS saved;'
        result = self.run_sql(sql, [])
        assert json.loads(result.stdout)[0]["saved"] == binding, result.stdout
        sql += 'SET "$.providers.plain" = "$providers.plain"; SELECT id FROM plain.things.widgets;'
        sql += 'UNSET "providers.plain"; SELECT id FROM plain.things.widgets;'
        sql += 'UNSET "$.providers.plain";'
        self.run_sql(sql, [("Bearer plain-token", "plain"), ("Bearer plain-token", "plain")])
        self.run_sql("SET region = 'east'; SELECT $region AS value;", [])

    def scalar_variables_preserve_sql_expressions(self):
        sql = '''SET "variables.word" = 'hello'; SET "variables.number" = 9007199254740993;
                 SET "variables.yes" = true; SET "variables.empty" = null; SET "variables.negative" = -17;
                 SELECT upper($word) || '-world' AS label, $number AS exact_number,
                        $yes = true AS truth, $empty IS NULL AS is_null, $negative AS negative;
                 SET "variables.word" = 'changed'; SELECT upper($word) AS changed;'''
        result = self.run_sql(sql, [])
        for expected in ['HELLO-world', '9007199254740993', 'CHANGED', '-17']:
            assert expected in result.stdout, result.stdout

    def variable_predicate_resolves_before_provider_dispatch(self):
        sql = self.setting("providers.variable", {"provider": "widgets", "tenancy": "variable", "auth": {"type": "bearer", "credentials": "variable-token"}, "params": {"region": "default"}})
        sql += '''SET "variables.region" = 'east'; SELECT id FROM variable.things.widgets WHERE region = upper($region);'''
        self.run_sql(sql, [("Bearer variable-token", "EAST")])

    def variable_errors_prevent_provider_dispatch(self):
        for expression in ['$missing', '"$providers.missing"', '"$auth.widgets.password"']:
            before = len(self.calls)
            result = subprocess.run(self.prefix + ["exec", 'SELECT ' + expression + ';'] + self.flags, text=True, capture_output=True, timeout=30, env=self.env)
            assert result.returncode != 0 and result.stderr.strip(), (result.stdout, result.stderr)
            assert len(self.calls) == before, "invalid variable reached provider"
        sql = '''SET "variables.removed" = 'old'; UNSET "variables.removed"; SELECT $removed;'''
        result = subprocess.run(self.prefix + ["exec", sql] + self.flags, text=True, capture_output=True, timeout=30, env=self.env)
        assert result.returncode != 0 and 'not defined' in result.stderr, result.stderr

    def secret_variables_require_runtime_opt_in(self):
        sql = self.setting("secret_store", {"backend": "os", "namespace": "stackql-deref-" + self.root.name})
        sql += self.setting("secrets.demo", "private-demo-value")
        sql += 'SELECT upper("$secrets.demo") AS value;'
        before = len(self.calls)
        denied = subprocess.run(self.prefix + ["exec", sql, "--allow-persistent-secrets"] + self.flags, text=True, capture_output=True, timeout=30, env=self.env)
        assert denied.returncode != 0 and 'secret variable dereferencing is disabled' in denied.stderr, denied.stderr
        assert 'private-demo-value' not in denied.stdout + denied.stderr, "secret leaked without opt-in"
        allowed = subprocess.run(self.prefix + ["exec", sql, "--allow-secret-dereference", "--allow-persistent-secrets"] + self.flags, text=True, capture_output=True, timeout=30, env=self.env)
        assert allowed.returncode == 0 and 'PRIVATE-DEMO-VALUE' in allowed.stdout, allowed.stdout + allowed.stderr
        assert len(self.calls) == before, "secret expression reached provider"
        cleanup = self.setting("secret_store", {"backend": "os", "namespace": "stackql-deref-" + self.root.name}) + 'UNSET "$.secrets.demo";'
        self.run_sql(cleanup, [], ["--allow-persistent-secrets"])

    def persistent_secret_stores_require_operator_opt_in(self):
        for config in [{"path": (self.root / "denied-secrets.json").as_posix()},
                       {"backend": "os", "namespace": "stackql-test-denied"}]:
            sql = self.setting("secret_store", config)
            result = subprocess.run(self.prefix + ["exec", sql] + self.flags,
                                    text=True, capture_output=True, timeout=30, env=self.env)
            assert result.returncode != 0 and "persistent secret stores are disabled" in result.stderr, result.stderr
        assert not (self.root / "denied-secrets.json").exists(), "disabled store created a file"

    def flat_file_secret_stores_are_rejected_even_with_opt_in(self):
        for config in [{"path": (self.root / "rejected.json").as_posix()},
                       {"backend": "file", "path": (self.root / "rejected.json").as_posix()}]:
            result = subprocess.run(self.prefix + ["exec", self.setting("secret_store", config),
                                    "--allow-persistent-secrets"] + self.flags,
                                    text=True, capture_output=True, timeout=30, env=self.env)
            assert result.returncode != 0 and "flat-file secrets are not supported" in result.stderr, result.stderr
        assert not (self.root / "rejected.json").exists(), "rejected backend created a file"

    def os_secrets_survive_new_sessions_and_namespaces_are_isolated(self):
        primary = {"backend": "os", "namespace": "stackql-cross-" + self.root.name}
        other = {"backend": "os", "namespace": "stackql-other-" + self.root.name}
        flags = ["--allow-persistent-secrets", "--allow-secret-dereference"]
        try:
            self.run_sql(self.setting("secret_store", primary) + self.setting("secrets.demo", "persistent-value"), [], flags)
            self.run_sql(self.setting("secret_store", other) + self.setting("secrets.demo", "isolated-value"), [], flags)
            for config, expected in [(primary, "PERSISTENT-VALUE"), (other, "ISOLATED-VALUE")]:
                result = self.run_sql(self.setting("secret_store", config) + 'SELECT upper("$secrets.demo") AS value;', [], flags)
                assert expected in result.stdout, result.stdout
        finally:
            for config in [primary, other]:
                self.run_sql(self.setting("secret_store", config) + 'UNSET "$.secrets.demo";', [], flags)

    def in_memory_auth_values_support_sql_with_explicit_opt_in(self):
        sql = self.setting("auth.widgets", {"type": "basic", "password": "memory-only-value"})
        sql += 'SELECT upper("$auth.widgets.password") AS value;'
        result = self.run_sql(sql, [], ["--allow-secret-dereference"])
        assert "MEMORY-ONLY-VALUE" in result.stdout, result.stdout
        sql = self.setting("providers.memory", {"provider": "widgets", "tenancy": "memory", "auth": {"type": "bearer", "credentials": "alias-only-value"}})
        sql += 'SELECT upper("$providers.memory.auth.credentials") AS value;'
        result = self.run_sql(sql, [], ["--allow-secret-dereference"])
        assert "ALIAS-ONLY-VALUE" in result.stdout, result.stdout

    def structured_os_secret_fields_support_sql_with_explicit_opt_in(self):
        config = {"backend": "os", "namespace": "stackql-json-" + self.root.name}
        flags = ["--allow-persistent-secrets", "--allow-secret-dereference"]
        sql = self.setting("secret_store", config)
        sql += self.setting("secrets.account", json.dumps({"credentials": "json-value", "enabled": True, "count": 3}))
        sql += '''SELECT upper("$secrets.account.credentials") AS credential,
                         "$secrets.account.enabled" = true AS enabled,
                         "$secrets.account.count" + 1 AS count;'''
        try:
            result = self.run_sql(sql, [], flags)
            rows = json.loads(result.stdout)
            assert rows == [{"credential": "JSON-VALUE", "enabled": "true" if self.postgres else "1", "count": "4"}], rows
        finally:
            self.run_sql(self.setting("secret_store", config) + 'UNSET "$.secrets.account";', [], flags)

    def prepare_os_secret_mock(self):
        """Mock Unix OS command boundaries; Windows exercises Credential Manager directly."""
        bins = self.root / "os-bin"
        bins.mkdir()
        backing = self.root / "mock-os-credentials"
        backing.mkdir()
        script = "#!/bin/sh\nroot=" + shlex.quote(str(backing)) + "\n" + r'''
operation=$1
if [ "$operation" = "-i" ]; then
    IFS= read -r line
    set -- $line
    operation=$1
fi
while [ "$#" -gt 0 ]; do
    case "$1" in
        -s|namespace) shift; namespace=$1 ;;
        -a|key) shift; key=$1 ;;
        -w) shift; value=$1 ;;
    esac
    shift
done
item="$root/$namespace-$key"
case "$operation" in
    store) cat > "$item" ;;
    add-generic-password) printf '%s' "$value" | tr -d '"' > "$item" ;;
    lookup|find-generic-password)
        if [ ! -f "$item" ]; then
            if [ "$operation" = "find-generic-password" ]; then exit 44; else exit 1; fi
        fi
        cat "$item" ;;
    clear|delete-generic-password) rm -f "$item" ;;
    *) exit 2 ;;
esac
'''
        for name in ["security", "secret-tool"]:
            command = bins / name
            command.write_text(script)
            command.chmod(0o700)
        self.env["PATH"] = str(bins) + os.pathsep + self.env.get("PATH", "")

    def scalar_variables_support_conversion_concatenation_and_digest_chains(self):
        import hashlib
        sql = '''SET some_id = 42; SET suffix = 'East'; SET digits = '007'; SET absent = null;
                 SELECT CAST($some_id AS TEXT) AS cast_text,
                        CONVERT($some_id, TEXT) AS convert_text,
                        CONVERT($digits, INTEGER) + 1 AS converted_integer,
                        CAST($some_id AS TEXT) || '-' || upper($suffix) AS concatenated,
                        md5(lower(CONVERT($some_id, TEXT) || '-' || $suffix)) AS digest,
                        CAST($absent AS TEXT) IS NULL AS null_preserved,
                        '$some_id' AS quoted_literal;'''
        result = self.run_sql(sql, [])
        row = json.loads(result.stdout)[0]
        assert row['cast_text'] == '42' and row['convert_text'] == '42', row
        assert row['converted_integer'] == '8' and row['concatenated'] == '42-EAST', row
        assert row['digest'] == hashlib.md5(b'42-east').hexdigest(), row
        assert row['null_preserved'] in ['1', 'true'] and row['quoted_literal'] == '$some_id', row

    def conversion_and_digest_chains_bind_provider_parameters(self):
        import hashlib
        binding = {"provider": "widgets", "tenancy": "conversion", "auth": {"type": "bearer", "credentials": "conversion-token"}}
        for expression, expected in [("CAST($some_id AS TEXT)", "42"),
                                     ("CONVERT($some_id, TEXT) || '-' || lower($suffix)", "42-east"),
                                     ("md5(lower(CONVERT($some_id, TEXT) || '-' || $suffix))", hashlib.md5(b'42-east').hexdigest())]:
            sql = self.setting('providers.conversion', binding)
            sql += "SET some_id = 42; SET suffix = 'East';"
            sql += 'SELECT id FROM conversion.things.widgets WHERE region = ' + expression + ';'
            result = self.run_sql(sql, [('Bearer conversion-token', expected)])
            assert json.loads(result.stdout) == [{'id': expected}], result.stdout

    def variables_resolve_relation_column_alias_order_and_limit_identifiers(self):
        sql = self.setting('providers.objects', {"provider": "widgets", "tenancy": "objects", "auth": {"type": "bearer", "credentials": "objects-token"}})
        sql += '''SET relation_name = 'objects.things.widgets'; SET column_name = 'id';
                  SET output_name = 'bucket'; SET table_alias = 'w'; SET region = 'east'; SET row_count = 1;
                  SELECT IDENTIFIER($column_name) AS "$output_name" FROM $relation_name AS "$table_alias"
                  WHERE region = $region ORDER BY IDENTIFIER($column_name) LIMIT $row_count;'''
        result = self.run_sql(sql, [('Bearer objects-token', 'east')])
        assert json.loads(result.stdout) == [{'bucket': 'east'}], result.stdout
        sql = self.setting('providers.parts', {"provider": "widgets", "tenancy": "parts", "auth": {"type": "bearer", "credentials": "parts-token"}})
        sql += '''SET provider_name = 'parts'; SET service_name = 'things'; SET resource_name = 'widgets';
                  SELECT id FROM $provider_name.$service_name.$resource_name WHERE region = 'parts';'''
        self.run_sql(sql, [('Bearer parts-token', 'parts')])

    def quoted_variables_and_identifier_values_cannot_inject_sql(self):
        value = "O'Reilly; DROP TABLE imaginary; --"
        sql = "SET text_value = '" + value.replace("'", "''") + "'; SELECT $text_value AS value, '$text_value' AS literal;"
        result = self.run_sql(sql, [])
        assert json.loads(result.stdout) == [{'value': value, 'literal': '$text_value'}], result.stdout
        label = 'quoted" label -- literal'
        sql = "SET output_name = '" + label + "'; SELECT 1 AS \"$output_name\";"
        result = self.run_sql(sql, [])
        assert json.loads(result.stdout) == [{label: '1'}], result.stdout
        for setup, query in [("SET object_name = 'widgets; DROP TABLE imaginary';", 'SELECT * FROM $object_name;'),
                             ('SET object_name = 17;', 'SELECT * FROM $object_name;'),
                             ("SET object_name = '';", 'SELECT IDENTIFIER($object_name);'),
                             ('', 'SELECT * FROM $missing_object;')]:
            before = len(self.calls)
            result = subprocess.run(self.prefix + ['exec', setup + query] + self.flags,
                                    text=True, capture_output=True, timeout=30, env=self.env)
            assert result.returncode != 0 and result.stderr.strip(), (result.stdout, result.stderr)
            assert len(self.calls) == before, 'invalid identifier reached provider'

    def memory_secrets_rotate_delete_and_expire_with_session(self):
        binding = {"provider": "widgets", "tenancy": "memory-secret", "auth_secret": "account", "params": {"region": "memory"}}
        sql = self.setting('secrets.account', json.dumps({"type": "bearer", "credentials": "memory-first-token"}))
        sql += self.setting('providers.memory_secret', binding) + 'SELECT id FROM memory_secret.things.widgets;'
        sql += self.setting('secrets.account', json.dumps({"type": "bearer", "credentials": "memory-second-token"}))
        sql += 'SELECT id FROM memory_secret.things.widgets; UNSET "$.secret_store";'
        sql += self.setting('secrets.account', json.dumps({"type": "bearer", "credentials": "memory-reset-token"}))
        sql += 'SELECT id FROM memory_secret.things.widgets; UNSET "$.secrets.account";'
        self.run_sql(sql, [('Bearer memory-first-token', 'memory'), ('Bearer memory-second-token', 'memory'), ('Bearer memory-reset-token', 'memory')])
        before = len(self.calls)
        result = subprocess.run(self.prefix + ['exec', self.setting('providers.memory_secret', binding) + 'SELECT id FROM memory_secret.things.widgets;'] + self.flags,
                                text=True, capture_output=True, timeout=30, env=self.env)
        assert 'secret' in result.stderr.lower() and len(self.calls) == before, (result.stdout, result.stderr)
        if self.postgres:
            import psycopg
            tables = self.database.execute('SELECT tablename FROM pg_tables WHERE schemaname = %s', [self.database_schema]).fetchall()
            for (table,) in tables:
                rows = self.database.execute(psycopg.sql.SQL('SELECT row_to_json(t)::text FROM {}.{} t').format(
                    psycopg.sql.Identifier(self.database_schema), psycopg.sql.Identifier(table))).fetchall()
                assert all('memory-first-token' not in row[0] and 'memory-second-token' not in row[0] for row in rows), 'credential persisted in PostgreSQL'
        for candidate in self.root.rglob('*'):
            if candidate.is_file() and (candidate.suffix in ['.db', '.sqlite', '.sqlite3'] or candidate.name.endswith(('-wal', '-journal'))):
                assert b'memory-first-token' not in candidate.read_bytes() and b'memory-second-token' not in candidate.read_bytes(), 'credential persisted in RDBMS'

    def independent_sql_sessions_isolate_variables_secrets_and_runtime_config(self):
        import socket
        import time
        import uuid
        from concurrent.futures import ThreadPoolExecutor
        import psycopg
        with socket.socket() as sock:
            sock.bind(('127.0.0.1', 0))
            port = sock.getsockname()[1]
        prefix = list(self.prefix)
        container = 'stackql-session-test-' + uuid.uuid4().hex[:10]
        docker = prefix[0] == 'docker'
        if docker:
            position = prefix.index('--entrypoint')
            prefix[position:position] = ['--name', container, '-p', f'127.0.0.1:{port}:{port}']
        log = open(self.root / 'session-server.log', 'w')
        process = subprocess.Popen(prefix + ['srv', '--pgsrv.address=0.0.0.0', f'--pgsrv.port={port}', '--querycachesize=4096'] + self.flags,
                                   stdout=log, stderr=log, env=self.env)
        connections = []
        try:
            for attempt in range(100):
                try:
                    first = psycopg.connect(host='127.0.0.1', port=port, user='stackql', dbname='stackql', sslmode='disable', autocommit=True, connect_timeout=1)
                    connections.append(first)
                    break
                except psycopg.OperationalError:
                    if process.poll() is not None or attempt == 99:
                        log.flush()
                        raise AssertionError((self.root / 'session-server.log').read_text())
                    time.sleep(0.1)
            second = psycopg.connect(host='127.0.0.1', port=port, user='stackql', dbname='stackql', sslmode='disable', autocommit=True)
            connections.append(second)
            for connection, name in [(first, 'east'), (second, 'west')]:
                connection.execute("SET region = '" + name + "';")
                connection.execute(self.setting('secrets.account', json.dumps({'type': 'bearer', 'credentials': name + '-session-token'})))
                connection.execute(self.setting('providers.shared', {'provider': 'widgets', 'tenancy': name, 'auth_secret': 'account', 'params': {'region': name}}))
            before = len(self.calls)
            def read(connection):
                return connection.execute('SELECT id FROM shared.things.widgets WHERE region = $region;').fetchall()
            self.concurrent_gate = threading.Barrier(2)
            try:
                with ThreadPoolExecutor(max_workers=2) as pool:
                    results = list(pool.map(read, [first, second]))
            finally:
                self.concurrent_gate = None
            assert results == [[('east',)], [('west',)]], results
            assert sorted(self.calls[before:]) == [('Bearer east-session-token', 'east'), ('Bearer west-session-token', 'west')], self.calls[before:]
            for _ in range(2):
                assert first.execute('SELECT id FROM shared.things.widgets;').fetchall() == [('east',)]
                assert second.execute('SELECT id FROM shared.things.widgets;').fetchall() == [('west',)]
            first.execute('UNSET region; UNSET "$.secrets.account";')
            try:
                first.execute('SELECT $region;')
                raise AssertionError('unset variable remained visible')
            except psycopg.Error:
                pass
            assert second.execute('SELECT $region;').fetchall() == [('west',)]
            assert read(second) == [('west',)]
            first.close()
            third = psycopg.connect(host='127.0.0.1', port=port, user='stackql', dbname='stackql', sslmode='disable', autocommit=True)
            connections.append(third)
            for query in ['SELECT $region;', 'SELECT "$secrets.account";']:
                try:
                    third.execute(query)
                    raise AssertionError('new session accessed another session value')
                except psycopg.Error:
                    pass
            third.execute(self.setting('providers.shared', {'provider': 'widgets', 'tenancy': 'third', 'auth_secret': 'account', 'params': {'region': 'third'}}))
            third.execute("SET region = 'third';")
            before = len(self.calls)
            try:
                read(third)
                raise AssertionError('new session reused another session secret')
            except psycopg.Error:
                pass
            assert len(self.calls) == before, 'missing secret reached provider'
        finally:
            for connection in connections:
                connection.close()
            process.terminate()
            try:
                process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                process.kill()
                process.wait(timeout=10)
            if docker:
                subprocess.run(['docker', 'rm', '-f', container], capture_output=True, timeout=30)
            log.close()

    def variable_identifiers_work_in_ddl_and_table_mutations(self):
        import uuid
        name = 'session_variable_' + uuid.uuid4().hex[:10]
        setup = "SET object_name = '" + name + "'; SET field_name = 'id'; SET some_id = 42;"
        sql = setup + '''CREATE TABLE $object_name ("$field_name" TEXT);
                         INSERT INTO $object_name ("$field_name") VALUES (CAST($some_id AS TEXT));
                         SELECT IDENTIFIER($field_name) AS value FROM $object_name WHERE id = CONVERT($some_id, TEXT);'''
        try:
            result = self.run_sql(sql, [], completion_messages=["DDL Execution Completed", "DML Execution Completed", "insert into table completed"])
            assert '42' in result.stdout, result.stdout
            result = self.run_sql(setup + '''UPDATE $object_name SET "$field_name" = '43' WHERE id = CAST($some_id AS TEXT);
                                            SELECT id FROM $object_name;''', [], completion_messages=["DML Execution Completed", "exec completed"])
            assert '43' in result.stdout, result.stdout
            self.run_sql(setup + 'DELETE FROM $object_name; DROP TABLE $object_name;', [], completion_messages=["DML Execution Completed", "DDL Execution Completed", "delete from table completed", "exec completed"])
        finally:
            result = subprocess.run(self.prefix + ['exec', setup + 'DROP TABLE IF EXISTS $object_name;'] + self.flags,
                                    text=True, capture_output=True, timeout=30, env=self.env)
            assert all(line == "DDL Execution Completed" for line in result.stderr.splitlines()), result.stderr

    def variable_identifiers_work_in_ctes_joins_and_subqueries(self):
        sql = self.setting('providers.nested', {'provider': 'widgets', 'tenancy': 'nested', 'auth': {'type': 'bearer', 'credentials': 'nested-token'}})
        sql += "SET object_name = 'nested.things.widgets'; SET field_name = 'id'; SET region = 'east';"
        for query, calls in [
            ('WITH selected AS (SELECT id FROM $object_name WHERE region = $region) SELECT IDENTIFIER($field_name) FROM selected;', [('Bearer nested-token', 'east')]),
            ('SELECT IDENTIFIER($field_name) FROM (SELECT id FROM $object_name WHERE region = $region) selected;', [('Bearer nested-token', 'east')]),
            ('SELECT a.id FROM $object_name a JOIN $object_name b ON a.id = b.id WHERE a.region = $region AND b.region = $region;', [('Bearer nested-token', 'east'), ('Bearer nested-token', 'east')]),
        ]:
            result = self.run_sql(sql + query, calls)
            assert json.loads(result.stdout) == [{'id': 'east'}], result.stdout

    def string_boolean_integer_and_null_variables_filter_provider_rows(self):
        sql = self.setting('providers.typed', {'provider': 'widgets', 'tenancy': 'typed', 'auth': {'type': 'bearer', 'credentials': 'typed-token'}})
        sql += '''SET some_id = 42; SET enabled = true; SET minimum_count = 40; SET optional_value = null;
                   SELECT id FROM typed.things.widgets WHERE region = CAST($some_id AS TEXT)
                   AND id = CONVERT($some_id, TEXT) AND active = $enabled AND count > $minimum_count
                   AND $optional_value IS NULL;'''
        result = self.run_sql(sql, [('Bearer typed-token', '42')])
        assert json.loads(result.stdout) == [{'id': '42'}], result.stdout
        sql += "SET enabled = false; SELECT id FROM typed.things.widgets WHERE region = '42' AND active = $enabled;"
        result = self.run_sql(sql, [('Bearer typed-token', '42'), ('Bearer typed-token', '42')])
        assert result.stdout.rstrip().endswith('null'), result.stdout

    def document_merge_inline_file_and_reset(self):
        binding = self.setting("providers.merged", {"provider": "widgets", "tenancy": "merge-test", "auth": {"type": "bearer", "credentials": "merge-token"}, "params": {"region": "merge-region"}})
        config = {"patches": [{"service": "stackql_unstable_widgets.things", "merge": {"components": {"x-stackQL-resources": {"widgets": {"methods": {"list": {"response": {"transform": {"type": "golang_template_json_v0.3.0", "body": '[{"id":"patched"}]'}}}}}}}}}]}
        query = "SELECT id FROM merged.things.widgets;"
        patch_file = self.root / "merge.json"
        patch_file.write_text(json.dumps(config))
        base_doc = self.root / "registry/src/widgets/v00.00.00000/services/things.yaml"
        original = base_doc.read_bytes()
        for value in [json.dumps(config).replace("\\", "\\\\"), "@" + patch_file.as_posix()]:
            sql = binding + self.setting("document_merge", value) + query
            sql += 'UNSET "$.document_merge";' + query
            result = self.run_sql(sql, [("Bearer merge-token", "merge-region")] * 2)
            assert 'patched' in result.stdout and 'merge-region' in result.stdout, result.stdout
            assert base_doc.read_bytes() == original, "base document changed"
        result = self.run_sql(binding + query, [("Bearer merge-token", "merge-region")])
        assert 'patched' not in result.stdout, "merge leaked into a new session"

    def iac_inputs_file_errors_precede_provider_dispatch(self):
        missing = str(self.root / "missing-inputs.json")
        sql = "SELECT * FROM stackql_preview.iac.run WHERE collection = 'file-check' AND resources = '[]' AND inputs_file = '" + missing + "';"
        before = len(self.calls)
        result = subprocess.run(self.prefix + ["exec", sql] + self.flags, text=True, capture_output=True, timeout=30, env=self.env)
        assert result.returncode != 0 and "inputs_file:" in result.stderr, result.stderr
        assert len(self.calls) == before, "provider called before inputs loaded"

    def ledger_query_is_read_only_and_collection_scoped(self):
        import base64
        def encoded(value):
            return base64.b64encode(value.encode()).decode()
        root = self.root / "app/iac/ledger"
        for collection in ["demo", "demo-other"]:
            directory = root / collection
            directory.mkdir(parents=True, exist_ok=True)
            (directory / "vm.v1.json").write_text(json.dumps({"phase": 1, "identity": encoded("i-ledger-test"), "about": encoded(json.dumps({"provider":"aws","address":"ec2.web_instance","params":{"secret":"hidden"}}))}))
        before = {str(p.relative_to(root)): p.read_bytes() for p in root.rglob("*") if p.is_file()}
        result = self.run_sql("SELECT * FROM stackql_preview.iac.ledger WHERE collection = 'demo' AND resource_key = 'vm';", [])
        assert "i-ledger-test" in result.stdout and 'live' in result.stdout, result.stdout
        assert "demo-other" not in result.stdout and "hidden" not in result.stdout, result.stdout
        after = {str(p.relative_to(root)): p.read_bytes() for p in root.rglob("*") if p.is_file()}
        assert before == after, "ledger changed during inspection"
        assert not (self.root / "app/iac/journal").exists(), "reader created a journal"

    def recover_verified_absent_create_without_provider_calls(self):
        import base64
        root = self.root / "app/iac/ledger/recover-test"
        root.mkdir(parents=True, exist_ok=True)
        record = {"phase":0,"proposed":base64.b64encode(b"{}").decode()}
        (root / "pending.v1.json").write_text(json.dumps(record))
        sql = "SELECT * FROM stackql_preview.iac.recover WHERE collection = 'recover-test' AND resource_key = 'pending' AND outcome = 'absent' AND version = '1';"
        result = self.run_sql(sql, [])
        assert 'recovered_absent' in result.stdout, result.stdout
        assert json.loads((root / 'pending.v2.json').read_text())['tombstone'], "recovery did not clear verified absent create"
        evidence = [json.loads(line) for path in (self.root / 'app/iac/journal').glob('*.diagnostics.jsonl') for line in path.read_text().splitlines()]
        assert any(row.get('diagnostic', {}).get('event') == 'recovery_assertion' for row in evidence), 'recovery decision not durable'
