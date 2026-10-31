#!/usr/bin/env python3
"""Run a local caller-access walkthrough. Requires StackQL, Python 3 and OpenSSL."""
import argparse
import base64
import csv
import hashlib
import json
import os
from pathlib import Path
import secrets
import shutil
import socket
import sqlite3
import ssl
import subprocess
import tempfile
import time
import urllib.error
import urllib.request


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--mode", choices=["jwt", "bearer", "mtls", "local_os"], required=True)
    parser.add_argument("--directory", type=Path)
    parser.add_argument("--stackql", default="stackql")
    args = parser.parse_args()
    root = (args.directory or Path(tempfile.mkdtemp(prefix="stackql-access-demo-"))).resolve()
    root.mkdir(mode=0o700, parents=True, exist_ok=True)
    if any(root.iterdir()):
        parser.error("choose an empty directory; existing credentials are never overwritten")
    exe = str(Path(shutil.which(args.stackql) or args.stackql).resolve())

    def write(name, value):
        path = root / name
        path.write_text(value)
        path.chmod(0o600)
        return path

    def openssl(*command):
        result = subprocess.run(["openssl", *map(str, command)], check=True, capture_output=True)
        return result.stdout

    def private(name):
        path = root / (name + ".key")
        openssl("genpkey", "-algorithm", "RSA", "-pkeyopt", "rsa_keygen_bits:2048", "-out", path)
        path.chmod(0o600)
        return path

    config = {"type": args.mode, "issuer": "access-demo", "max_session_seconds": 300,
              "rules": [{"subject": "group:readers", "resource": "host/sql/read", "action": "execute"}]}
    env = os.environ.copy()
    env.pop("STACKQL_CALLER_TOKEN", None)
    subject = "reader"
    if args.mode == "jwt":
        key = private("signing")
        config.update(audience="stackql", keys=[{"kid": "demo", "algorithm": "RS256", "public_key_pem": openssl("pkey", "-in", key, "-pubout").decode()}])
        encode = lambda value: base64.urlsafe_b64encode(value).rstrip(b"=")
        claims = {"iss": "access-demo", "sub": subject, "aud": "stackql", "iat": int(time.time()), "exp": int(time.time()) + 300}
        unsigned = encode(json.dumps({"alg": "RS256", "kid": "demo"}).encode()) + b"." + encode(json.dumps(claims).encode())
        message = write("unsigned.jwt", unsigned.decode())
        signature = openssl("dgst", "-sha256", "-sign", key, message)
        env["STACKQL_CALLER_TOKEN"] = (unsigned + b"." + encode(signature)).decode()
    elif args.mode == "bearer":
        env["STACKQL_CALLER_TOKEN"] = secrets.token_urlsafe(32)
        config["tokens"] = [{"subject": subject, "sha256": hashlib.sha256(env["STACKQL_CALLER_TOKEN"].encode()).hexdigest()}]
    elif args.mode == "local_os":
        subject = next(csv.reader([subprocess.check_output(["whoami", "/user", "/fo", "csv", "/nh"], text=True).strip()]))[1] if os.name == "nt" else str(os.geteuid())
    else:
        subject = "spiffe://access-demo/reader"
        ca_key = private("ca")
        ca = root / "ca.crt"
        ca_config = write("ca.cnf", "[req]\ndistinguished_name=dn\nx509_extensions=ca\n[dn]\n[ca]\nbasicConstraints=critical,CA:TRUE\nkeyUsage=critical,keyCertSign,cRLSign\nsubjectKeyIdentifier=hash\n")
        openssl("req", "-new", "-x509", "-sha256", "-key", ca_key, "-subj", "/CN=Access Demo CA", "-days", "1", "-config", ca_config, "-out", ca)
        for name, extension in [("server", "subjectAltName=IP:127.0.0.1\nextendedKeyUsage=serverAuth\n"), ("client", "subjectAltName=URI:" + subject + "\nextendedKeyUsage=clientAuth\n")]:
            key = private(name)
            csr = root / (name + ".csr")
            openssl("req", "-new", "-sha256", "-key", key, "-subj", "/CN=" + name, "-out", csr)
            extensions = write(name + ".ext", extension + "basicConstraints=critical,CA:FALSE\nkeyUsage=critical,digitalSignature\nsubjectKeyIdentifier=hash\nauthorityKeyIdentifier=keyid,issuer\n")
            openssl("x509", "-req", "-sha256", "-in", csr, "-CA", ca, "-CAkey", ca_key, "-CAcreateserial", "-days", "1", "-extfile", extensions, "-out", root / (name + ".crt"))
        config.update(client_ca_pem=ca.read_text(), certificate_identity="uri_san")

    database = root / "principals.db"
    with sqlite3.connect(database) as db:
        db.executescript("""
CREATE TABLE stackql_access_principals (id TEXT PRIMARY KEY, attributes TEXT NOT NULL DEFAULT '{}');
CREATE TABLE stackql_access_groups (id TEXT PRIMARY KEY);
CREATE TABLE stackql_access_identities (issuer TEXT NOT NULL, subject TEXT NOT NULL,
 principal_id TEXT NOT NULL REFERENCES stackql_access_principals(id), PRIMARY KEY (issuer, subject));
CREATE TABLE stackql_access_memberships (principal_id TEXT NOT NULL REFERENCES stackql_access_principals(id),
 group_id TEXT NOT NULL REFERENCES stackql_access_groups(id), PRIMARY KEY (principal_id, group_id));
INSERT INTO stackql_access_principals VALUES ('reader','{}');
INSERT INTO stackql_access_groups VALUES ('readers');
INSERT INTO stackql_access_memberships VALUES ('reader','readers');
""")
        db.execute("INSERT INTO stackql_access_identities VALUES (?,?,?)", ("access-demo", subject, "reader"))
    database.chmod(0o600)
    access = write("access.json", json.dumps(config, indent=2))
    backend = write("backend.json", json.dumps({"dsn": str(database)}))
    command = [exe, "--approot", str(root), "--access.config", json.dumps({"path": str(access)}), "--sqlBackend", backend.read_text(), "--output", "json"]
    query = "SELECT whoami() AS principal, groups() AS memberships;"
    if args.mode != "mtls":
        result = subprocess.run(command + ["exec", query], env=env, cwd=root, capture_output=True, text=True)
        if result.returncode:
            raise RuntimeError("protected query failed: " + result.stderr)
        output = result.stdout
    else:
        with socket.socket() as listener:
            listener.bind(("127.0.0.1", 0))
            port = listener.getsockname()[1]
        mcp = {"server": {"transport": "http", "address": "127.0.0.1:" + str(port), "protocol_version": "2026-07-28", "audit": {"disabled": True}, "tls_cert_file": str(root / "server.crt"), "tls_key_file": str(root / "server.key")}, "enabled_tools": ["run_select_query"]}
        write("mcp.json", json.dumps(mcp, indent=2))
        context = ssl.create_default_context(cafile=str(root / "ca.crt"))
        context.load_cert_chain(str(root / "client.crt"), str(root / "client.key"))
        payload = {"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": {"_meta": {"io.modelcontextprotocol/protocolVersion": "2026-07-28", "io.modelcontextprotocol/clientCapabilities": {}, "io.modelcontextprotocol/clientInfo": {"name": "access-demo", "version": "1"}}, "name": "run_select_query", "arguments": {"sql": query}}}
        headers = {"Content-Type": "application/json", "Accept": "application/json, text/event-stream", "MCP-Protocol-Version": "2026-07-28", "Mcp-Method": "tools/call", "Mcp-Name": "run_select_query"}
        with (root / "server.log").open("w") as log:
            server = subprocess.Popen(command + ["--mcp.config", json.dumps(mcp), "mcp"], env=env, cwd=root, stdout=log, stderr=log)
            try:
                for attempt in range(100):
                    if server.poll() is not None:
                        raise RuntimeError("MCP startup failed: " + (root / "server.log").read_text())
                    try:
                        request = urllib.request.Request("https://127.0.0.1:" + str(port), data=json.dumps(payload).encode(), headers=headers)
                        with urllib.request.urlopen(request, context=context, timeout=5) as response:
                            output = response.read().decode()
                        break
                    except urllib.error.URLError:
                        if attempt == 99:
                            raise
                        time.sleep(0.1)
            finally:
                server.terminate()
                server.wait(timeout=10)
    if args.mode == "mtls":
        message = json.loads(output) if output.lstrip().startswith("{") else next(json.loads(line[5:]) for line in output.splitlines() if line.startswith("data:"))
        result = message.get("result", {})
        if message.get("error") or result.get("isError"):
            raise RuntimeError("protected MCP query failed: " + output)
        rows = result.get("structuredContent", {}).get("rows", [])
    else:
        rows = json.loads(output)
    if len(rows) != 1 or rows[0].get("principal") != "reader":
        raise RuntimeError("expected the provisioned principal: " + output)
    groups = rows[0].get("memberships")
    if isinstance(groups, str):
        groups = json.loads(groups)
    if groups != ["readers"]:
        raise RuntimeError("expected exactly the directory readers group: " + output)
    print(output.strip())
    print("Verified " + args.mode + ": principal reader; directory group readers.")
    print("Configuration: " + str(access))


if __name__ == "__main__":
    main()
