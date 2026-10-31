"""Black-box caller access fixtures. Tokens stay in subprocess environments, never Robot results."""
import base64
import hashlib
import secrets
import csv
import yaml
import concurrent.futures
import json
import datetime
import ipaddress
import socket
import shutil
import uuid
from urllib.parse import urlsplit, urlunsplit, parse_qsl, urlencode
import ssl
import urllib.request
import urllib.error
from cryptography import x509
from cryptography.x509.oid import NameOID
import os
import pathlib
import sqlite3
import subprocess
import tempfile
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

from cryptography.hazmat.primitives import serialization
from cryptography.hazmat.primitives.asymmetric import ed25519, rsa, ec
from cryptography.hazmat.primitives import hashes
from cryptography.hazmat.primitives.asymmetric import padding
from cryptography.hazmat.primitives.asymmetric.utils import decode_dss_signature


class _DockerSQLite:
    """Commit fixture writes in the same kernel as the application's SQLite readers.

    Host SQLite locks and page-cache invalidation do not cross a Docker Desktop VM.
    The sidecar uses only Python's standard library; the tested image needs no Python.
    """

    def __init__(self, path):
        self._path = pathlib.Path(path)
        self._pending = []

    def execute(self, statement):
        self._pending.append(statement)

    def commit(self):
        if not self._pending:
            return
        program = (
            "import json, sqlite3, sys\n"
            "with sqlite3.connect(sys.argv[1]) as db:\n"
            "    for statement in json.load(sys.stdin):\n"
            "        db.execute(statement)\n"
        )
        subprocess.run([
            "docker", "run", "--rm", "--interactive", "--network", "none",
            "--user", f"{os.getuid()}:{os.getgid()}",
            "--volume", str(self._path.parent) + ":" + str(self._path.parent),
            "--entrypoint", "python3", "python:3.13-slim-trixie",
            "-c", program, str(self._path),
        ], input=json.dumps(self._pending), capture_output=True, text=True, check=True, timeout=120)
        self._pending.clear()

    def close(self):
        self._pending.clear()


class AccessManagementKeywords:
    ROBOT_LIBRARY_SCOPE = "SUITE"

    def verify_caller_access(self, executable, backend="{}", execution_platform="native", scenario="iac"):
        with tempfile.TemporaryDirectory(prefix="stackql-access-") as directory:
            root = pathlib.Path(directory)
            docker = execution_platform == "docker"
            cfg = json.loads(backend)
            postgres = cfg.get("dbEngine") == "postgres_tcp"
            directory_schema = None
            if postgres:
                import psycopg
                dsn = cfg["dsn"]
                if docker:
                    subprocess.run(["docker", "compose", "-p", "execrun", "up", "-d", "postgres_stackql"], capture_output=True, check=True, timeout=120)
                    result = subprocess.run(["docker", "compose", "-p", "execrun", "port", "postgres_stackql", "5432"], capture_output=True, text=True, check=True, timeout=30)
                    published = result.stdout.strip().splitlines()[0].rsplit(":", 1)[1]
                    url = urlsplit(dsn)
                    credentials = url.netloc.rsplit("@", 1)[0]
                    dsn = urlunsplit((url.scheme, credentials + "@127.0.0.1:" + published, url.path, url.query, url.fragment))
                for attempt in range(50):
                    try:
                        db = psycopg.connect(dsn, autocommit=True)
                        break
                    except psycopg.OperationalError:
                        if not docker or attempt == 49:
                            raise
                        time.sleep(0.2)
                directory_schema = "stackql_access_test_" + uuid.uuid4().hex
                db.execute(psycopg.sql.SQL("CREATE SCHEMA {}").format(psycopg.sql.Identifier(directory_schema)))
                db.execute(psycopg.sql.SQL("SET search_path TO {}, public").format(psycopg.sql.Identifier(directory_schema)))
                url = urlsplit(cfg["dsn"])
                options = dict(parse_qsl(url.query))
                options["options"] = "-csearch_path=" + directory_schema + ",public"
                cfg["dsn"] = urlunsplit((url.scheme, url.netloc, url.path, urlencode(options), url.fragment))
            else:
                cfg = {"dsn": str(root / "access.db")}
                db = _DockerSQLite(cfg["dsn"]) if docker else sqlite3.connect(cfg["dsn"], check_same_thread=False)
            key = ed25519.Ed25519PrivateKey.generate()
            public = key.public_key().public_bytes(serialization.Encoding.PEM, serialization.PublicFormat.SubjectPublicKeyInfo).decode()
            config = {"issuer": "access-test", "audience": "stackql", "keys": [{"kid": "local", "algorithm": "EdDSA", "public_key_pem": public}], "rules": [
                {"subject": "group:readers", "resource": "operation/accessdemo/list/listItems", "action": "execute"},
                {"subject": "group:readers", "resource": "credential/accessdemo", "action": "use"},
                {"subject": "group:operators", "resource": "**", "action": "*"},
            ]}
            config_file = root / "access.json"
            config_file.write_text(json.dumps(config))
            ddl = [
                "CREATE TABLE IF NOT EXISTS stackql_access_principals (id TEXT PRIMARY KEY, attributes TEXT NOT NULL DEFAULT '{}')",
                "CREATE TABLE IF NOT EXISTS stackql_access_groups (id TEXT PRIMARY KEY)",
                "CREATE TABLE IF NOT EXISTS stackql_access_identities (issuer TEXT NOT NULL, subject TEXT NOT NULL, principal_id TEXT NOT NULL REFERENCES stackql_access_principals(id), PRIMARY KEY (issuer,subject))",
                "CREATE TABLE IF NOT EXISTS stackql_access_memberships (principal_id TEXT NOT NULL REFERENCES stackql_access_principals(id), group_id TEXT NOT NULL REFERENCES stackql_access_groups(id), PRIMARY KEY (principal_id,group_id))",
            ]
            for statement in ddl:
                db.execute(statement)
            for table in ["memberships", "identities", "principals", "groups"]:
                db.execute("DELETE FROM stackql_access_" + table)
            for statement in [
                "INSERT INTO stackql_access_principals VALUES ('reader','{}'),('operator','{}'),('denied','{}')",
                "INSERT INTO stackql_access_groups VALUES ('readers'),('operators')",
                "INSERT INTO stackql_access_identities VALUES ('access-test','reader','reader'),('access-test','operator','operator'),('access-test','denied','denied')",
                "INSERT INTO stackql_access_memberships VALUES ('reader','readers'),('operator','operators')",
            ]:
                db.execute(statement)
            if not postgres:
                db.commit()
            counts = {"read": 0, "mutation": 0}
            managed = {}

            class API(BaseHTTPRequestHandler):
                def log_message(self, *_):
                    pass

                def do_GET(self):
                    if scenario.startswith("query_deadline"):
                        time.sleep(2)
                    counts["read"] += 1
                    self.send_response(200)
                    self.send_header("Content-Type", "application/json")
                    self.end_headers()
                    identity = self.path.rsplit("/", 1)[-1]
                    self.wfile.write(json.dumps(managed.get(identity, {}) if self.path.startswith("/items/") else [{"id":"known-item"}]).encode())

                def do_POST(self):
                    counts["mutation"] += 1
                    body = json.loads(self.rfile.read(int(self.headers.get("Content-Length", "0"))) or b"{}")
                    if body.get("name") == "upstream-reject":
                        self.send_response(400)
                        self.send_header("Content-Type", "application/json")
                        self.end_headers()
                        self.wfile.write(b'{"error":"test rejection"}')
                        return
                    self.send_response(200)
                    self.send_header("Content-Type", "application/json")
                    self.end_headers()
                    identity = "managed-" + str(counts["mutation"])
                    managed[identity] = dict(body, id=identity)
                    self.wfile.write(json.dumps(managed[identity]).encode())

                def do_PATCH(self):
                    counts["mutation"] += 1
                    identity = self.path.rsplit("/", 1)[-1]
                    body = json.loads(self.rfile.read(int(self.headers.get("Content-Length", "0"))) or b"{}")
                    managed[identity].update(body)
                    self.send_response(200)
                    self.send_header("Content-Type", "application/json")
                    self.end_headers()
                    self.wfile.write(json.dumps(managed[identity]).encode())

                def do_DELETE(self):
                    counts["mutation"] += 1
                    managed.pop(self.path.rsplit("/", 1)[-1], None)
                    self.send_response(204)
                    self.end_headers()

            server = ThreadingHTTPServer(("0.0.0.0" if docker else "127.0.0.1", 0), API)
            thread = threading.Thread(target=server.serve_forever, daemon=True)
            thread.start()
            doc_dir = root / "src" / "accessdemo" / "v0.0.1"
            doc_dir.mkdir(parents=True)
            (doc_dir / "provider.yaml").write_text("name: accessdemo\nproviderServices:\n  service:\n    service: {$ref: service.yaml}\n")
            (doc_dir / "service.yaml").write_text("""openapi: 3.0.0
servers: [{url: 'http://127.0.0.1:PORT'}]
paths:
  /items:
    get:
      operationId: listItems
      responses:
        '200':
          content:
            application/json:
              schema: {type: array, items: {type: object, properties: {id: {type: string}}}}
    post:
      operationId: createItem
      responses:
        '200':
          content:
            application/json:
              schema: {type: object, properties: {id: {type: string}}}
components:
  x-stackQL-resources:
    items:
      methods:
        list:
          operation: {$ref: '#/paths/~1items/get'}
          response: {mediaType: application/json}
        create:
          operation: {$ref: '#/paths/~1items/post'}
          response: {mediaType: application/json}
      sqlVerbs:
        select: [{$ref: '#/components/x-stackQL-resources/items/methods/list'}]
        insert: [{$ref: '#/components/x-stackQL-resources/items/methods/create'}]
""".replace("PORT", str(server.server_port)))

            service_file = doc_dir / "service.yaml"
            service = yaml.safe_load(service_file.read_text())
            if docker:
                service["servers"][0]["url"] = service["servers"][0]["url"].replace("127.0.0.1", "host.docker.internal")
            schema = {"type": "object", "properties": {"id": {"type": "string"}, "name": {"type": "string"}}}
            response = {"200": {"content": {"application/json": {"schema": schema}}}}
            body = {"required": True, "content": {"application/json": {"schema": schema}}}
            service["paths"]["/items"]["post"]["requestBody"] = body
            service["paths"]["/items/{id}"] = {
                "get": {"operationId": "getItem", "parameters": [{"name": "id", "in": "path", "required": True, "schema": {"type": "string"}}], "responses": response},
                "patch": {"operationId": "patchItem", "parameters": [{"name": "id", "in": "path", "required": True, "schema": {"type": "string"}}], "requestBody": body, "responses": response},
                "delete": {"operationId": "deleteItem", "parameters": [{"name": "id", "in": "path", "required": True, "schema": {"type": "string"}}], "responses": {"204": {"description": "deleted"}}},
            }
            resource = service["components"]["x-stackQL-resources"]["items"]
            for name, verb in [("get", "get"), ("update", "patch"), ("delete", "delete")]:
                resource["methods"][name] = {"operation": {"$ref": "#/paths/~1items~1{id}/" + verb}, "response": {"mediaType": "application/json"}}
            resource["sqlVerbs"]["select"].append({"$ref": "#/components/x-stackQL-resources/items/methods/get"})
            resource["sqlVerbs"]["update"] = [{"$ref": "#/components/x-stackQL-resources/items/methods/update"}]
            resource["sqlVerbs"]["delete"] = [{"$ref": "#/components/x-stackQL-resources/items/methods/delete"}]
            service_file.write_text(yaml.safe_dump(service))

            def token(subject="reader", updates=None, remove=(), signer=None, kid="local"):
                claims = {"iss": "access-test", "sub": subject, "aud": "stackql", "iat": int(time.time()), "exp": int(time.time()) + 300, "groups": ["operators"]}
                claims.update(updates or {})
                for field in remove:
                    claims.pop(field, None)
                encode = lambda value: base64.urlsafe_b64encode(json.dumps(value, separators=(",", ":")).encode()).rstrip(b"=")
                message = encode({"alg": "EdDSA", "kid": kid}) + b"." + encode(claims)
                signature = (signer or key).sign(message)
                return (message + b"." + base64.urlsafe_b64encode(signature).rstrip(b"=")).decode()

            prefix = (["docker", "compose", "-p", "execrun", "run", "--rm", "--no-deps", "-T", "--user", f"{os.getuid()}:{os.getgid()}", "--workdir", str(root), "-v", str(root) + ":" + str(root), "-e", "STACKQL_CALLER_TOKEN", "--entrypoint", "stackql", "stackqlsrv"]
                      if docker else [shutil.which(executable) or str(pathlib.Path(executable).resolve())])
            base = prefix + ["--approot", str(root), "--sqlBackend", json.dumps(cfg), "--registry", json.dumps({"localDocRoot": str(root), "verifyConfig": {"nopVerify": True}}), "--preview", '{"unstable":true}', "--output", "json"]

            def run(sql, credential=None, protected=True):
                env = os.environ.copy()
                env.pop("STACKQL_CALLER_TOKEN", None)
                if credential is not None:
                    env["STACKQL_CALLER_TOKEN"] = credential
                command = base + (["--access.config", json.dumps({"path": str(config_file)})] if protected else []) + ["exec", sql]
                result = subprocess.run(command, capture_output=True, text=True, env=env, timeout=45)
                # Separate CLI processes initialize the shared PostgreSQL host catalog.
                # Retry only this fixture's known read when concurrent initialization deadlocks.
                for _ in range(2):
                    if sql != "SELECT id FROM stackql_unstable_accessdemo.service.items;" or "40P01" not in result.stdout + result.stderr:
                        break
                    result = subprocess.run(command, capture_output=True, text=True, env=env, timeout=45)
                output = result.stdout + result.stderr
                if credential and credential in output:
                    raise AssertionError("caller token leaked into output")
                return result.returncode, output

            def success(sql, credential, expected):
                code, output = run(sql, credential)
                assert code == 0 and expected in output, "authorized query failed: " + output

            def denied(sql, credential=None, concurrent=False):
                before = counts.copy()
                code, output = run(sql, credential)
                assert code != 0, "protected query unexpectedly succeeded: " + sql + " output: " + output
                assert counts["mutation"] == before["mutation"] and (concurrent or counts["read"] == before["read"]), "denied query reached provider: " + sql

            try:
                query = "SELECT id FROM stackql_unstable_accessdemo.service.items;"
                reader = token()
                invalid = {
                    "missing": None,
                    "signature": token(signer=ed25519.Ed25519PrivateKey.generate()),
                    "issuer": token(updates={"iss": "other"}),
                    "audience": token(updates={"aud": "other"}),
                    "expiry_required": token(remove=["exp"]),
                    "expired": token(updates={"exp": int(time.time()) - 5}),
                    "not_before": token(updates={"nbf": int(time.time()) + 600}),
                    "issued_at": token(updates={"iat": int(time.time()) + 600}),
                    "key_id": token(kid="unknown"),
                    "identity": token(subject="unknown"),
                }
                if scenario.startswith("bearer"):
                    tokens = {name: secrets.token_urlsafe(32) for name in ["reader", "operator", "denied"]}
                    config.pop("keys")
                    config.pop("audience")
                    config.update(type="bearer", tokens=[{"subject": name, "sha256": hashlib.sha256(value.encode()).hexdigest()} for name, value in tokens.items()])
                    config_file.write_text(json.dumps(config))
                    token = lambda subject="reader": tokens[subject]
                    reader = token()
                    invalid = {"missing": None, "wrong": secrets.token_urlsafe(32)}
                if scenario.startswith("local_os"):
                    config = {"type": "local_os", "issuer": "access-test", "rules": config["rules"]}
                    config_file.write_text(json.dumps(config))
                    if os.name == "nt":
                        identity = next(csv.reader([subprocess.check_output(["whoami", "/user", "/fo", "csv", "/nh"], text=True).strip()]))[1]
                    else:
                        identity = str(os.geteuid())
                    db.execute("UPDATE stackql_access_identities SET subject = '" + identity.replace("'", "''") + "' WHERE principal_id = 'reader'")
                    if not postgres:
                        db.commit()
                    if scenario == "local_os_http_refused":
                        result = subprocess.run(base + ["--access.config", json.dumps({"path": str(config_file)}), "--mcp.config", json.dumps({"server": {"transport": "http", "tls_cert_file": "unused.crt", "tls_key_file": "unused.key"}}), "mcp"], capture_output=True, text=True, timeout=30)
                        assert "local OS authentication is unavailable over HTTP" in result.stdout + result.stderr
                        assert counts == {"read": 0, "mutation": 0}
                        return
                    if scenario.endswith("shell") or scenario.endswith("stdio"):
                        self._verify_local_transport(base, config_file, None, query, scenario.rsplit("_", 1)[-1])
                        return
                    success("SELECT whoami() AS principal, groups() AS memberships;", None, "readers")
                    success(query, None, "known-item")
                    success("SELECT whoami() AS principal;", "untrusted-environment-token", "reader")
                    denied("DELETE FROM stackql_unstable_accessdemo.service.items;")
                    db.execute("DELETE FROM stackql_access_identities WHERE principal_id = 'reader'")
                    if not postgres:
                        db.commit()
                    denied(query)
                    return
                if scenario in ("mtls_https", "mtls_https_dns"):
                    identity_kind = "dns_san" if scenario.endswith("_dns") else "uri_san"
                    roots, clients = self._client_certificates(root, identity_kind)
                    config = {"type": "mtls", "issuer": "access-test", "client_ca_pem": roots, "certificate_identity": identity_kind, "rules": config["rules"]}
                    config_file.write_text(json.dumps(config))
                    for name in ["reader", "operator", "denied"]:
                        subject = name + ".example" if identity_kind == "dns_san" else "spiffe://access-test/" + name
                        db.execute("UPDATE stackql_access_identities SET subject = '" + subject + "' WHERE principal_id = '" + name + "'")
                    if not postgres:
                        db.commit()
                    self._verify_https(base, config_file, root, key, token, [], query, db, postgres, counts, clients)
                    denied(query, reader)
                    return
                if scenario.startswith("query_deadline"):
                    config["max_session_seconds"] = 1
                    config["enforce_session_deadline"] = scenario == "query_deadline_opt_in"
                    config_file.write_text(json.dumps(config))
                    code, output = run(query, reader)
                    if config["enforce_session_deadline"]:
                        assert code != 0 and "deadline" in output.lower(), "opt-in deadline did not cancel query (rc=" + str(code) + "): " + output
                    else:
                        assert code == 0 and "known-item" in output, "default deadline interrupted query: " + output
                    return
                if scenario == "config":
                    self._verify_configuration(base, config_file, config, reader, query, counts)
                    return
                if scenario.startswith("jwt_invalid:"):
                    denied(query, invalid[scenario.split(":", 1)[1]])
                    return
                if scenario == "jwt_algorithms":
                    self._verify_algorithms(config, config_file, token, run, query, counts)
                    return
                if scenario.endswith("shell") or scenario.endswith("stdio"):
                    self._verify_local_transport(base, config_file, reader, query, scenario.rsplit("_", 1)[-1])
                    self._verify_local_transport(base, config_file, None, query, scenario.rsplit("_", 1)[-1], allowed=False)
                    return
                if scenario.endswith("https"):
                    self._verify_https(base, config_file, root, key, token, list(invalid.values()), query, db, postgres, counts)
                    return
                if scenario in ("jwt_cli", "bearer_cli"):
                    success("SELECT whoami() AS principal, groups() AS memberships;", reader, "readers")
                    success(query, reader, "known-item")
                    success("SELECT 1 AS ordinary_integer;", reader, "ordinary_integer")
                    for credential in invalid.values():
                        denied(query, credential)
                    denied(query, token(subject="denied"))
                    return
                if scenario == "compatibility":
                    code, output = run(query, protected=False)
                    assert code == 0 and "known-item" in output, "unprotected compatibility failed: " + output
                    return
                if scenario == "policy":
                    for sql in ["DELETE FROM stackql_unstable_accessdemo.service.items;", "INSERT INTO stackql_unstable_accessdemo.service.items (id) VALUES ('new');", "EXEC stackql_unstable_accessdemo.service.items.create;", "SELECT * FROM stackql_preview.iac.run WHERE collection = 'test' AND resources = '[]';", "SELECT * FROM stackql_preview.iac.destroy WHERE collection = 'test';", "SELECT * FROM stackql_access_principals;", "SELECT pg_read_file('/etc/passwd');"]:
                        denied(sql, reader)
                    denied(query, token(subject="denied"))
                    with concurrent.futures.ThreadPoolExecutor(max_workers=2) as executor:
                        futures = [executor.submit(success, query, reader, "known-item"), executor.submit(denied, "SELECT 1", token(subject="denied"), True)]
                        for future in futures:
                            future.result()
                    return
                if scenario == "revocation":
                    db.execute("DELETE FROM stackql_access_memberships WHERE principal_id = 'reader'")
                    if not postgres:
                        db.commit()
                    denied(query, reader)
                    db.execute("UPDATE stackql_access_principals SET attributes = 'null' WHERE id = 'operator'")
                    if not postgres:
                        db.commit()
                    denied(query, token(subject="operator"))
                    assert counts["mutation"] == 0, "reader performed provider mutation"
                    db.execute("UPDATE stackql_access_principals SET attributes = '{}' WHERE id = 'operator'")
                    if not postgres:
                        db.commit()
                    return
                assert scenario == "iac", "unknown access scenario: " + scenario
                operator = token(subject="operator")
                def converge(label, keys):
                    resources = [{"key": name, "provider": "accessdemo", "address": "service.items", "identity": "id", "addressed_by": "id", "desired": {"name": {"from": "input", "attribute": "label"}}} for name in keys]
                    spec = json.dumps(resources).replace("'", "''")
                    inputs = json.dumps({"label": label})
                    sql = "SELECT * FROM stackql_preview.iac.run WHERE collection = 'acceptance' AND state = 'audit' AND run_id = '" + label + "' AND resources = '" + spec + "' AND inputs = '" + inputs + "';"
                    success(sql, operator, "applied")
                    assert all(item.get("name") == label for item in managed.values()), "IaC reported applied without converging desired values"
                converge("first", ["one", "two"])
                assert len(managed) == 2, "IaC create did not retain both resources"
                db.execute("INSERT INTO stackql_access_principals VALUES ('operator2','{}')")
                db.execute("INSERT INTO stackql_access_identities VALUES ('access-test','operator2','operator2')")
                db.execute("INSERT INTO stackql_access_memberships VALUES ('operator2','operators')")
                if not postgres:
                    db.commit()
                denied("SELECT * FROM stackql_preview.iac.destroy WHERE collection = 'acceptance' AND state = 'audit' AND keys = '[\"one\"]';", token(subject="operator2"))
                denied("SELECT * FROM stackql_preview.iac.destroy WHERE collection = 'acceptance' AND state = '../../escape';", operator)
                converge("second", ["one", "two"])
                converge("third", ["one", "two"])
                # Reusing the original create ID must not unwind its old creates after an update.
                snapshot = json.loads(json.dumps(managed))
                before = counts.copy()
                replay_spec = json.dumps([{"key": name, "provider": "accessdemo", "address": "service.items", "identity": "id", "addressed_by": "id", "desired": {"name": "upstream-reject"}} for name in ["one", "two"]])
                replay = "SELECT * FROM stackql_preview.iac.run WHERE collection = 'acceptance' AND state = 'audit' AND run_id = 'first' AND resources = '" + replay_spec + "';"
                code, output = run(replay, operator)
                assert code != 0 and "already exists" in output, "reused run ID accepted: " + output
                assert managed == snapshot and counts == before, "reused run changed existing resources"
                # resources_file is the user-facing path; omitted run IDs must remain safe on repeat.
                resource_file = root / "resources.json"
                resource_file.write_text(json.dumps({"resources": [{"key": name, "provider": "accessdemo", "address": "service.items", "identity": "id", "addressed_by": "id", "desired": {"name": "third"}} for name in ["one", "two"]]}))
                file_sql = "SELECT * FROM stackql_preview.iac.run WHERE collection = 'acceptance' AND state = 'audit' AND resources_file = '" + resource_file.as_posix().replace("'", "''") + "';"
                for _ in range(2):
                    success(file_sql, operator, "applied")
                    assert managed == snapshot, "unchanged file rerun changed resources"
                denied("SELECT * FROM stackql_preview.iac.destroy WHERE collection = 'acceptance' AND state = 'audit' AND keys = '[]';", operator)
                success("SELECT * FROM stackql_preview.iac.destroy WHERE collection = 'acceptance' AND state = 'audit' AND run_id = 'prune' AND keys = '[\"two\"]';", operator, "destroyed")
                assert len(managed) == 1, "partial delete changed the retained resource set"
                converge("fourth", ["one"])
                success("SELECT * FROM stackql_preview.iac.destroy WHERE collection = 'acceptance' AND state = 'audit' AND run_id = 'teardown';", operator, "destroyed")
                assert not managed, "IaC teardown left managed resources"
                spec = json.dumps([{"key": "reject", "provider": "accessdemo", "address": "service.items", "identity": "id", "addressed_by": "id", "desired": {"name": "upstream-reject"}}])
                sql = "SELECT * FROM stackql_preview.iac.run WHERE collection = 'rejection' AND run_id = 'reject' AND resources = '" + spec + "';"
                code, output = run(sql, protected=False)
                assert code != 0 and "failed" in output.lower(), "unsuccessful IaC returned success: " + output
                assert not managed, "failed create unexpectedly retained resources"

            finally:
                server.shutdown()
                server.server_close()
                if directory_schema:
                    db.execute(psycopg.sql.SQL("DROP SCHEMA {} CASCADE").format(psycopg.sql.Identifier(directory_schema)))
                db.close()

    def _verify_https(self, base, config_file, root, key, token, invalid, query, db, postgres, counts, clients=None):
        subject = x509.Name([x509.NameAttribute(NameOID.COMMON_NAME, "localhost")])
        now = datetime.datetime.now(datetime.timezone.utc)
        certificate = (x509.CertificateBuilder().subject_name(subject).issuer_name(subject)
                       .public_key(key.public_key()).serial_number(x509.random_serial_number())
                       .not_valid_before(now - datetime.timedelta(minutes=1))
                       .not_valid_after(now + datetime.timedelta(hours=1))
                       .add_extension(x509.SubjectAlternativeName([x509.IPAddress(ipaddress.ip_address("127.0.0.1"))]), False)
                       .sign(key, None))
        cert_path, key_path = root / "server.crt", root / "server.key"
        cert_path.write_bytes(certificate.public_bytes(serialization.Encoding.PEM))
        key_path.write_bytes(key.private_bytes(serialization.Encoding.PEM, serialization.PrivateFormat.PKCS8, serialization.NoEncryption()))
        key_path.chmod(0o600)
        with socket.socket() as sock:
            sock.bind(("127.0.0.1", 0))
            port = sock.getsockname()[1]
        config = {"server": {"transport": "http", "address": "0.0.0.0:" + str(port),
                             "protocol_version": "2026-07-28", "mode": "full_access",
                             "tls_cert_file": str(cert_path), "tls_key_file": str(key_path)},
                  "enabled_tools": ["run_select_query"]}
        env = os.environ.copy()
        # Deliberately set a privileged process token: remote clients must never inherit it.
        env["STACKQL_CALLER_TOKEN"] = "server-process-token" if clients else token(subject="operator")
        container = "stackql-access-" + uuid.uuid4().hex
        server_base = list(base)
        docker = server_base[0] == "docker"
        if docker:
            index = server_base.index("stackqlsrv")
            server_base[index:index] = ["--publish", "127.0.0.1:" + str(port) + ":" + str(port), "--name", container]
        command = server_base + ["--access.config", json.dumps({"path": str(config_file)}), "--mcp.config", json.dumps(config), "mcp"]
        tls = ssl.create_default_context(cafile=str(cert_path))
        if clients:
            contexts = {}
            for name, (cert, private) in clients.items():
                context = ssl.create_default_context(cafile=str(cert_path))
                context.load_cert_chain(str(cert), str(private))
                contexts[name] = context
            token = lambda subject="reader": contexts[subject]
            invalid = [None] + [contexts[name] for name in ["untrusted", "expired", "ambiguous", "unknown"]]
        with (root / "mcp.log").open("w+") as log:
            process = subprocess.Popen(command, stdout=log, stderr=log, env=env)
            try:
                def call(credential, sql=query):
                    headers = {"Content-Type": "application/json", "Accept": "application/json, text/event-stream", "MCP-Protocol-Version": "2026-07-28", "Mcp-Method": "tools/call", "Mcp-Name": "run_select_query"}
                    if credential and not clients:
                        headers["Authorization"] = "Bearer " + credential
                    payload = {"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": {"_meta": {"io.modelcontextprotocol/protocolVersion": "2026-07-28", "io.modelcontextprotocol/clientCapabilities": {}, "io.modelcontextprotocol/clientInfo": {"name": "access-test", "version": "1"}}, "name": "run_select_query", "arguments": {"sql": sql}}}
                    request = urllib.request.Request("https://127.0.0.1:" + str(port), data=json.dumps(payload).encode(), headers=headers)
                    try:
                        with urllib.request.urlopen(request, context=credential if clients and credential else tls, timeout=20) as response:
                            return response.status, response.read().decode()
                    except urllib.error.HTTPError as error:
                        return error.code, error.read().decode()
                    except (urllib.error.URLError, ssl.SSLError, ConnectionError) as error:
                        if clients: return 0, str(error)
                        raise
                for _ in range(100):
                    if process.poll() is not None:
                        log.seek(0)
                        raise AssertionError("protected HTTPS server failed to start: " + log.read())
                    try:
                        status, response = call(token() if clients else None)
                        if clients and status == 0: raise OSError(response)
                        assert status == (200 if clients else 401), "HTTP readiness authentication failed"
                        break
                    except (OSError, urllib.error.URLError):
                        time.sleep(0.1)
                else:
                    raise AssertionError("protected HTTPS server did not start")
                reader = token()
                status, output = call(reader)
                assert status == 200 and "known-item" in output, "authorized HTTPS read failed: " + output
                for credential in invalid:
                    before = counts.copy()
                    status, _ = call(credential)
                    assert status in ((0, 401) if clients else (401,)) and counts == before, "invalid HTTP credential reached execution"
                def read():
                    status, output = call(reader)
                    assert status == 200 and "known-item" in output, "concurrent HTTPS reader failed: " + output
                def deny():
                    status, output = call(token(subject="denied"))
                    assert "known-item" not in output and (status != 200 or '"isError":true' in output.replace(" ", "")), "concurrent caller inherited access"
                with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
                    for future in [pool.submit(read), pool.submit(deny)]:
                        future.result()
                db.execute("DELETE FROM stackql_access_memberships WHERE principal_id = 'reader'")
                if not postgres:
                    db.commit()
                status, output = call(reader)
                assert "known-item" not in output and (status != 200 or '"isError":true' in output.replace(" ", "")), "HTTP membership revocation ignored: " + output
                db.execute("INSERT INTO stackql_access_memberships VALUES ('reader','readers')")
                if not postgres:
                    db.commit()
            finally:
                process.terminate()
                try:
                    process.wait(timeout=10)
                except subprocess.TimeoutExpired:
                    process.kill()
                    process.wait(timeout=10)
                if docker:
                    subprocess.run(["docker", "rm", "-f", container], capture_output=True, timeout=30)
                log.seek(0)
                text = log.read()
                assert env["STACKQL_CALLER_TOKEN"] not in text, "server token leaked into log"

    def _verify_configuration(self, base, path, config, credential, query, counts):
        env = os.environ.copy()
        env["STACKQL_CALLER_TOKEN"] = credential
        def execute(source):
            return subprocess.run(base + ["--access.config", source, "exec", query], env=env, capture_output=True, text=True, timeout=45)
        for source in [json.dumps({"path": str(path)}), str(path)]:
            result = execute(source)
            assert result.returncode == 0 and "known-item" in result.stdout, "valid configuration source failed: " + result.stderr
        before = counts.copy()
        for source in ['{}', '{"path":""}', '{"path":null}', '{"path":3}', json.dumps({"path": str(path), "unknown": True}), json.dumps({"path": str(path)}) + '{}', json.dumps({"path": str(path) + ".missing"})]:
            result = execute(source)
            assert result.returncode != 0 and counts == before, "invalid configuration source reached execution"
        for changes in [{"unknown": True}, {"type": "unknown"}, {"type": "bearer"}, {"type": "local_os"}, {"type": "mtls"}]:
            path.write_text(json.dumps(dict(config, **changes)))
            result = execute(json.dumps({"path": str(path)}))
            assert result.returncode != 0 and counts == before, "invalid authentication configuration reached execution"
        path.write_text(json.dumps(config))

    def _verify_algorithms(self, config, path, token, run, query, counts):
        key = rsa.generate_private_key(public_exponent=65537, key_size=2048)
        algorithms = [("RS" + size, key, getattr(hashes, "SHA" + size)()) for size in ["256", "384", "512"]]
        algorithms += [("PS" + size, key, getattr(hashes, "SHA" + size)()) for size in ["256", "384", "512"]]
        algorithms += [("ES256", ec.generate_private_key(ec.SECP256R1()), hashes.SHA256()), ("ES384", ec.generate_private_key(ec.SECP384R1()), hashes.SHA384()), ("ES512", ec.generate_private_key(ec.SECP521R1()), hashes.SHA512())]
        code, output = run(query, token())
        assert code == 0 and "known-item" in output, "EdDSA JWT failed: " + output
        encode = lambda value: base64.urlsafe_b64encode(value).rstrip(b"=")
        for algorithm, private, digest in algorithms:
            config["keys"] = [{"kid": "family", "algorithm": algorithm, "public_key_pem": private.public_key().public_bytes(serialization.Encoding.PEM, serialization.PublicFormat.SubjectPublicKeyInfo).decode()}]
            path.write_text(json.dumps(config))
            claims = {"iss": "access-test", "sub": "reader", "aud": "stackql", "exp": int(time.time()) + 300, "iat": int(time.time())}
            message = encode(json.dumps({"alg": algorithm, "kid": "family"}).encode()) + b"." + encode(json.dumps(claims).encode())
            if algorithm.startswith("RS"):
                signature = private.sign(message, padding.PKCS1v15(), digest)
            elif algorithm.startswith("PS"):
                signature = private.sign(message, padding.PSS(mgf=padding.MGF1(digest), salt_length=digest.digest_size), digest)
            else:
                r, value = decode_dss_signature(private.sign(message, ec.ECDSA(digest)))
                width = (private.curve.key_size + 7) // 8
                signature = r.to_bytes(width, "big") + value.to_bytes(width, "big")
            credential = (message + b"." + encode(signature)).decode()
            code, output = run(query, credential)
            assert code == 0 and "known-item" in output, algorithm + " JWT failed: " + output
            config["keys"][0]["algorithm"] = "RS256" if algorithm != "RS256" else "PS256"
            path.write_text(json.dumps(config))
            before = counts.copy()
            code, output = run(query, credential)
            assert code != 0 and "known-item" not in output and counts == before, "JWT algorithm/key binding was ignored"

    def _verify_local_transport(self, base, path, credential, query, transport, allowed=True):
        env = os.environ.copy()
        env.pop("STACKQL_CALLER_TOKEN", None)
        if credential:
            env["STACKQL_CALLER_TOKEN"] = credential
        command = base + ["--access.config", json.dumps({"path": str(path)})]
        if transport == "shell":
            result = subprocess.run(command + ["shell"], input=query + "\nexit\n", env=env, capture_output=True, text=True, timeout=30)
            output = result.stdout + result.stderr
        else:
            command += ["--mcp.config", json.dumps({"server": {"transport": "stdio", "audit": {"disabled": True}}, "enabled_tools": ["run_select_query"]}), "mcp"]
            process = subprocess.Popen(command, env=env, stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
            watchdog = threading.Timer(30, process.kill)
            watchdog.start()
            lines = []
            try:
                def request(value, response_id):
                    process.stdin.write(json.dumps(value).encode() + b"\n")
                    process.stdin.flush()
                    while True:
                        line = process.stdout.readline()
                        if not line:
                            raise AssertionError("MCP stdio closed before its response")
                        lines.append(line.decode())
                        message = json.loads(line)
                        if message.get("id") == response_id:
                            return
                request({"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {"protocolVersion": "2025-06-18", "capabilities": {}, "clientInfo": {"name": "access-test", "version": "1"}}}, 1)
                process.stdin.write(b'{"jsonrpc":"2.0","method":"notifications/initialized"}\n')
                request({"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": {"name": "run_select_query", "arguments": {"sql": query}}}, 2)
                process.stdin.close()
                process.stdin = None
                stdout, stderr = process.communicate(timeout=10)
                output = "".join(lines) + stdout.decode() + stderr.decode()
                assert process.returncode == 0, "MCP stdio failed: " + output
            finally:
                watchdog.cancel()
                if process.poll() is None:
                    process.kill()
                    process.wait()
        assert ("known-item" in output) == allowed, transport + " caller authorization failed: " + output
        if not allowed:
            assert "caller authentication failed" in output, "missing credential did not fail authentication"
        assert not credential or credential not in output, "local transport leaked a credential"

    def _client_certificates(self, root, identity_kind="uri_san"):
        now = datetime.datetime.now(datetime.timezone.utc)
        issuer = x509.Name([x509.NameAttribute(NameOID.COMMON_NAME, "access-test CA")])
        ca_key = ed25519.Ed25519PrivateKey.generate()
        ca = (x509.CertificateBuilder().subject_name(issuer).issuer_name(issuer).public_key(ca_key.public_key())
              .serial_number(x509.random_serial_number()).not_valid_before(now - datetime.timedelta(days=1))
              .not_valid_after(now + datetime.timedelta(days=1)).add_extension(x509.BasicConstraints(ca=True, path_length=0), True)
              .sign(ca_key, None))
        clients = {}
        for name in ["reader", "operator", "denied", "unknown", "untrusted", "expired", "ambiguous"]:
            private = ed25519.Ed25519PrivateKey.generate()
            san = lambda subject: x509.DNSName(subject + ".example") if identity_kind == "dns_san" else x509.UniformResourceIdentifier("spiffe://access-test/" + subject)
            uris = [san(name)]
            if name == "ambiguous":
                uris.append(san("reader"))
            certificate = (x509.CertificateBuilder().subject_name(x509.Name([x509.NameAttribute(NameOID.COMMON_NAME, name)]))
                           .issuer_name(issuer).public_key(private.public_key()).serial_number(x509.random_serial_number())
                           .not_valid_before(now - datetime.timedelta(days=2))
                           .not_valid_after(now - datetime.timedelta(seconds=1) if name == "expired" else now + datetime.timedelta(hours=1))
                           .add_extension(x509.SubjectAlternativeName(uris), False)
                           .add_extension(x509.ExtendedKeyUsage([x509.oid.ExtendedKeyUsageOID.CLIENT_AUTH]), False)
                           .sign(ed25519.Ed25519PrivateKey.generate() if name == "untrusted" else ca_key, None))
            cert_path, key_path = root / (name + ".crt"), root / (name + ".key")
            cert_path.write_bytes(certificate.public_bytes(serialization.Encoding.PEM))
            key_path.write_bytes(private.private_bytes(serialization.Encoding.PEM, serialization.PrivateFormat.PKCS8, serialization.NoEncryption()))
            key_path.chmod(0o600)
            clients[name] = (cert_path, key_path)
        return ca.public_bytes(serialization.Encoding.PEM).decode(), clients

    def verify_access_walkthrough(self, executable, mode, repository_root, execution_platform="native"):
        script = pathlib.Path(repository_root).resolve() / "docs/examples/access-management/quickstart.py"
        with tempfile.TemporaryDirectory(prefix="stackql-access-walkthrough-") as directory:
            root = pathlib.Path(directory).resolve()
            docker = execution_platform == "docker"
            container = "stackql-walkthrough-" + uuid.uuid4().hex
            env = os.environ.copy()
            env.pop("STACKQL_CALLER_TOKEN", None)
            exe = shutil.which(executable) or str(pathlib.Path(executable).resolve())
            if docker:
                # Run the published host-side walkthrough against the actual tested image.
                # This adapter forwards credentials through Docker's environment, never arguments.
                proxy = root / "stackql-proxy"
                prefix = ["docker", "run", "--rm", "--interactive", "--user", f"{os.getuid()}:{os.getgid()}",
                          "--volume", str(root) + ":" + str(root), "--workdir", str(root),
                          "--env", "STACKQL_CALLER_TOKEN", "--entrypoint", "stackql"]
                image = os.environ.get("STACKQL_IMAGE_NAME", "stackql/stackql")
                proxy.write_text("#!/usr/bin/env python3\nimport json,os,sys\n"
                    + "prefix=" + repr(prefix) + "\nargs=sys.argv[1:]\n"
                    + "if '--mcp.config' in args:\n"
                    + " i=args.index('--mcp.config')+1; cfg=json.loads(args[i]); port=cfg['server']['address'].rsplit(':',1)[1]\n"
                    + " cfg['server']['address']='0.0.0.0:'+port; args[i]=json.dumps(cfg)\n"
                    + " prefix+=['--publish','127.0.0.1:'+port+':'+port,'--name'," + repr(container) + "]\n"
                    + "os.execvp('docker',prefix+[" + repr(image) + "]+args)\n")
                proxy.chmod(0o700)
                exe = str(proxy)
            command = [shutil.which("python3") or shutil.which("python"), str(script), "--mode", mode,
                       "--directory", str(root / "demo"), "--stackql", exe if docker else os.path.relpath(exe)]
            try:
                result = subprocess.run(command, env=env, capture_output=True, text=True, timeout=60)
                assert result.returncode == 0, "documented walkthrough failed: " + result.stdout + result.stderr
                assert "Verified " + mode + ": principal reader; directory group readers." in result.stdout
                if mode == "local_os":
                    # Exercise the documented next step, not just its setup script.
                    demo = root / "demo"
                    shell = subprocess.run([exe, "--access.config", json.dumps({"path": str(demo / "access.json")}),
                        "--sqlBackend", (demo / "backend.json").read_text(), "--output", "json", "shell"],
                        input="SELECT whoami() AS principal, groups() AS memberships;\n", env=env,
                        cwd=root, capture_output=True, text=True, timeout=45)
                    assert shell.returncode == 0, "documented authenticated shell failed: " + shell.stdout + shell.stderr
                    assert '"principal":"reader"' in shell.stdout.replace(" ", ""), "shell did not resolve reader: " + shell.stdout
                    assert "readers" in shell.stdout, "shell did not resolve directory membership: " + shell.stdout
            finally:
                if docker:
                    subprocess.run(["docker", "rm", "-f", container], capture_output=True, timeout=30)
