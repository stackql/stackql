"""Minimal stdio MCP harness used by robot scenarios.

Drives a `stackql mcp --mcp.server.type=stdio` child over raw byte pipes in
binary mode with a configurable line terminator (issue #668 CRLF framing).
Also hosts the issue #688 scripted credential reload scenarios against the
`--env.file` dotenv file, the issue #701 malformed-frame resilience
roundtrip, and the issue #729 protocol revision conformance roundtrips
(2025-06-18 handshake client and 2026-07-28 stateless client, stdio and
Streamable HTTP).  The issue #784 scenarios reuse those clients against
servers pinned via server.protocol_version / --mcp.protocol.version.
"""

import json
import os
import subprocess
import tempfile
import threading
import time

_LINE_ENDINGS = {
    "lf": b"\n",
    "crlf": b"\r\n",
}


def _frame_messages(messages, terminator):
    return b"".join(
        json.dumps(m, separators=(",", ":")).encode("utf-8") + terminator
        for m in messages
    )


def run_stdio_initialize_roundtrip(
    stackql_exe,
    registry_cfg,
    auth_cfg,
    line_ending="lf",
    timeout_seconds=90,
):
    """Runs initialize -> initialized -> tools/list over stdio.

    Returns {stdout, stderr, returncode}; responses for ids 1 and 2 are
    awaited before stdin closes, so slow handling cannot race shutdown.
    """
    terminator = _LINE_ENDINGS[line_ending]
    messages = [
        {
            "jsonrpc": "2.0",
            "id": 1,
            "method": "initialize",
            "params": {
                "protocolVersion": "2025-06-18",
                "capabilities": {},
                "clientInfo": {"name": "robot-stdio-harness", "version": "0.1.0"},
            },
        },
        {"jsonrpc": "2.0", "method": "notifications/initialized"},
        {"jsonrpc": "2.0", "id": 2, "method": "tools/list"},
    ]
    argv = [
        stackql_exe,
        "mcp",
        "--mcp.server.type=stdio",
        "--mcp.config",
        '{"server": {"audit": {"disabled": true}} }',
        "--registry",
        registry_cfg,
        "--auth",
        auth_cfg,
        "--tls.allowInsecure",
    ]
    proc = subprocess.Popen(
        argv,
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    watchdog = threading.Timer(timeout_seconds, proc.kill)
    watchdog.start()
    stdout_lines = []
    stderr = b""
    try:
        try:
            proc.stdin.write(_frame_messages(messages, terminator))
            proc.stdin.flush()
        except OSError:
            # A server that dies on the first (CRLF-terminated) line breaks
            # the pipe; fall through so assertions see the empty output.
            pass
        awaited_ids = {1, 2}
        while awaited_ids:
            line = proc.stdout.readline()
            if not line:
                # Server exited (or was killed by the watchdog) before
                # responding; the caller's assertions surface the failure.
                break
            stdout_lines.append(line)
            try:
                decoded = json.loads(line)
            except ValueError:
                continue
            if isinstance(decoded, dict):
                awaited_ids.discard(decoded.get("id"))
        try:
            proc.stdin.close()
        except OSError:
            pass
        # stdin is closed above, so the server sees EOF and exits; drain the
        # remaining output directly (communicate() would re-flush the closed
        # stdin and raise).
        stdout_lines.append(proc.stdout.read())
        stderr = proc.stderr.read()
        proc.wait(timeout=timeout_seconds)
    finally:
        watchdog.cancel()
    return {
        "stdout": b"".join(stdout_lines).decode("utf-8", errors="replace"),
        "stderr": stderr.decode("utf-8", errors="replace"),
        "returncode": proc.returncode,
    }


def run_stdio_malformed_frame_roundtrip(
    stackql_exe,
    registry_cfg,
    auth_cfg,
    malformed_frame,
    timeout_seconds=90,
):
    """Issue #701: a malformed frame must be answered with a JSON-RPC error
    (id null) and must NOT terminate the stdio session.

    Runs initialize -> initialized -> <malformed_frame> -> ping, then closes
    stdin.  Returns a dict of pre-digested assertion inputs plus raw streams:
    error_code / error_id_is_null (from the first error response after the
    malformed frame), ping_ok (ping answered with an empty result),
    still_running_after_ping, and returncode (expected 0: clean EOF exit).
    """
    argv = [
        stackql_exe,
        "mcp",
        "--mcp.server.type=stdio",
        "--mcp.config",
        '{"server": {"audit": {"disabled": true}} }',
        "--registry",
        registry_cfg,
        "--auth",
        auth_cfg,
        "--tls.allowInsecure",
    ]
    proc = subprocess.Popen(
        argv,
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    watchdog = threading.Timer(timeout_seconds, proc.kill)
    watchdog.start()
    stdout_lines = []
    stderr = b""
    error_code = None
    error_id_is_null = False
    ping_ok = False
    still_running = False
    try:
        def send_raw(payload):
            try:
                proc.stdin.write(payload)
                proc.stdin.flush()
            except OSError:
                # A server that died mid-sequence breaks the pipe; fall
                # through so the assertions see the failure.
                pass

        send_raw(_frame_messages([{
            "jsonrpc": "2.0",
            "id": 1,
            "method": "initialize",
            "params": {
                "protocolVersion": "2025-06-18",
                "capabilities": {},
                "clientInfo": {"name": "robot-stdio-harness", "version": "0.1.0"},
            },
        }], b"\n"))
        _await_response(proc, 1, stdout_lines)
        send_raw(_frame_messages(
            [{"jsonrpc": "2.0", "method": "notifications/initialized"}], b"\n"))
        send_raw(malformed_frame.encode("utf-8") + b"\n")
        while True:
            line = proc.stdout.readline()
            if not line:
                break
            stdout_lines.append(line)
            try:
                decoded = json.loads(line)
            except ValueError:
                continue
            if isinstance(decoded, dict) and decoded.get("error"):
                error_code = decoded["error"].get("code")
                error_id_is_null = "id" in decoded and decoded["id"] is None
                break
        send_raw(_frame_messages(
            [{"jsonrpc": "2.0", "id": 2, "method": "ping"}], b"\n"))
        ping_response = _await_response(proc, 2, stdout_lines)
        ping_ok = (
            isinstance(ping_response, dict)
            and ping_response.get("result") == {}
            and "error" not in ping_response
        )
        still_running = proc.poll() is None
        try:
            proc.stdin.close()
        except OSError:
            pass
        stdout_lines.append(proc.stdout.read())
        stderr = proc.stderr.read()
        proc.wait(timeout=timeout_seconds)
    finally:
        watchdog.cancel()
        if proc.poll() is None:
            proc.kill()
    return {
        "error_code": error_code,
        "error_id_is_null": error_id_is_null,
        "ping_ok": ping_ok,
        "still_running_after_ping": still_running,
        "stdout": b"".join(stdout_lines).decode("utf-8", errors="replace"),
        "stderr": stderr.decode("utf-8", errors="replace"),
        "returncode": proc.returncode,
    }


def _await_response(proc, request_id, collected_lines):
    """Reads stdout until the response for `request_id` arrives (None if the
    server exited first); raw lines are appended to `collected_lines`."""
    while True:
        line = proc.stdout.readline()
        if not line:
            return None
        collected_lines.append(line)
        try:
            decoded = json.loads(line)
        except ValueError:
            continue
        if isinstance(decoded, dict) and decoded.get("id") == request_id:
            return decoded


def _tool_result_text(response):
    """Flattens a tools/call response (text blocks, structured content, any
    error object) to a single string for assertions."""
    if response is None:
        return ""
    parts = []
    error = response.get("error")
    if error:
        parts.append(json.dumps(error))
    result = response.get("result") or {}
    for block in result.get("content") or []:
        if isinstance(block, dict) and block.get("text"):
            parts.append(block["text"])
    structured = result.get("structuredContent")
    if structured is not None:
        parts.append(json.dumps(structured))
    if result.get("isError"):
        parts.append("isError=true")
    return "\n".join(parts)


def run_stdio_startup_script(stackql_exe, argv, calls, timeout_seconds=90):
    """Exercise startup arguments, tool restrictions and an idle stdio session."""
    with tempfile.TemporaryFile() as stderr_file:
        proc = subprocess.Popen(
            [stackql_exe, *argv], stdin=subprocess.PIPE,
            stdout=subprocess.PIPE, stderr=stderr_file,
        )
        watchdog = threading.Timer(timeout_seconds, proc.kill)
        watchdog.start()
        lines, results = [], {}
        try:
            def send(message):
                proc.stdin.write(_frame_messages([message], b"\n"))
                proc.stdin.flush()

            send({
                "jsonrpc": "2.0", "id": 1, "method": "initialize",
                "params": {
                    "protocolVersion": "2025-06-18", "capabilities": {},
                    "clientInfo": {"name": "robot-startup", "version": "1"},
                },
            })
            results["initialize"] = _await_response(proc, 1, lines)
            send({"jsonrpc": "2.0", "method": "notifications/initialized"})
            send({"jsonrpc": "2.0", "id": 2, "method": "tools/list"})
            listed = _await_response(proc, 2, lines) or {}
            results["tools"] = [t["name"] for t in listed.get("result", {}).get("tools", [])]
            for request_id, call in enumerate(calls, start=3):
                send({"jsonrpc": "2.0", "id": request_id, "method": "tools/call", "params": call})
                results[call["name"]] = _tool_result_text(_await_response(proc, request_id, lines))
            time.sleep(1)
            send({"jsonrpc": "2.0", "id": 100, "method": "ping"})
            results["ping"] = _await_response(proc, 100, lines)
            results["alive_after_idle"] = proc.poll() is None
            proc.stdin.close()
            lines.append(proc.stdout.read())
            proc.wait(timeout=timeout_seconds)
        finally:
            watchdog.cancel()
            if proc.poll() is None:
                proc.kill()
                proc.wait()
        stderr_file.seek(0)
        results["stderr"] = stderr_file.read().decode("utf-8", errors="replace")
    results["stdout"] = b"".join(lines).decode("utf-8", errors="replace")
    results["returncode"] = proc.returncode
    return results


def run_stdio_credential_script(
    stackql_exe,
    registry_cfg,
    auth_cfg,
    env_file_path,
    child_env_overrides,
    steps,
    timeout_seconds=120,
):
    """Drives a scripted reload_credentials scenario over one stdio session.

    `child_env_overrides` maps var names to values for the child env (None
    removes the var).  `steps` is an ordered list of dicts, one of:
      {"call": <tool>, "args": {...}, "as": <label>}  tool call, flattened
                                                        text stored under label
      {"write_env": {"VAR": "value", ...}}             (re)write the env file
      {"remove_env": true}                             delete the env file
    Returns {label: text, ..., stderr, returncode}.
    """
    if os.path.exists(env_file_path):
        os.remove(env_file_path)
    child_env = {
        k: v for k, v in os.environ.items()
        if k.upper() not in {o.upper() for o in child_env_overrides}
    }
    for k, v in child_env_overrides.items():
        if v is not None:
            child_env[k] = v
    argv = [
        stackql_exe,
        "mcp",
        "--mcp.server.type=stdio",
        "--mcp.config",
        '{"server": {"mode": "full_access", "audit": {"disabled": true}} }',
        f"--env.file={env_file_path}",
        "--registry",
        registry_cfg,
        "--auth",
        auth_cfg,
        "--tls.allowInsecure",
    ]
    proc = subprocess.Popen(
        argv,
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        env=child_env,
    )
    watchdog = threading.Timer(timeout_seconds, proc.kill)
    watchdog.start()
    stdout_lines = []
    stderr = b""
    results = {}
    try:
        def send(message):
            proc.stdin.write(_frame_messages([message], b"\n"))
            proc.stdin.flush()

        send({
            "jsonrpc": "2.0",
            "id": 1,
            "method": "initialize",
            "params": {
                "protocolVersion": "2025-06-18",
                "capabilities": {},
                "clientInfo": {"name": "robot-stdio-harness", "version": "0.1.0"},
            },
        })
        _await_response(proc, 1, stdout_lines)
        send({"jsonrpc": "2.0", "method": "notifications/initialized"})
        next_id = 2
        for step in steps:
            if "write_env" in step:
                with open(env_file_path, "w") as f:
                    for k, v in step["write_env"].items():
                        f.write(f"{k}={v}\n")
                continue
            if step.get("remove_env"):
                os.remove(env_file_path)
                continue
            send({
                "jsonrpc": "2.0",
                "id": next_id,
                "method": "tools/call",
                "params": {"name": step["call"], "arguments": step.get("args", {})},
            })
            results[step["as"]] = _tool_result_text(_await_response(proc, next_id, stdout_lines))
            next_id += 1
        try:
            proc.stdin.close()
        except OSError:
            pass
        stdout_lines.append(proc.stdout.read())
        stderr = proc.stderr.read()
        proc.wait(timeout=timeout_seconds)
    finally:
        watchdog.cancel()
        if proc.poll() is None:
            proc.kill()
    results["stderr"] = stderr.decode("utf-8", errors="replace")
    results["returncode"] = proc.returncode
    return results


# ---- protocol revision 2026-07-28 conformance (issue #729) ----------------

_STATELESS_META = {
    "io.modelcontextprotocol/protocolVersion": "2026-07-28",
    "io.modelcontextprotocol/clientCapabilities": {"elicitation": {"form": {}}},
    "io.modelcontextprotocol/clientInfo": {"name": "robot-stateless-harness", "version": "0.1.0"},
}


def _await_message(proc, collected_lines, predicate):
    """Reads stdout until a decoded frame satisfies `predicate` (None if the
    server exited first)."""
    while True:
        line = proc.stdout.readline()
        if not line:
            return None
        collected_lines.append(line)
        try:
            decoded = json.loads(line)
        except ValueError:
            continue
        if isinstance(decoded, dict) and predicate(decoded):
            return decoded


def run_stdio_stateless_roundtrip(
    stackql_exe,
    registry_cfg,
    auth_cfg,
    mutation_sql,
    approval_action="accept",
    mode="safe",
    protocol_version=None,
    timeout_seconds=90,
):
    """Drives a 2026-07-28 client over stdio: no initialize handshake, the
    protocol version and client capabilities ride in params._meta on every
    request. Runs tools/list, then a gated mutation whose approval comes back
    as an input_required result and is answered on the retry via
    inputResponses, echoing the signed requestState. Returns the tool names,
    the first mutation result shape, the retried mutation text and the raw
    streams.  `protocol_version` pins the server (issue #784); a pin below
    2026-07-28 answers tools/list with the -32022 downgrade error surfaced as
    list_error_code / _supported.
    """
    server_cfg = {"mode": mode, "audit": {"disabled": True}}
    if protocol_version:
        server_cfg["protocol_version"] = protocol_version
    argv = [
        stackql_exe,
        "mcp",
        "--mcp.server.type=stdio",
        "--mcp.config",
        json.dumps({"server": server_cfg}),
        "--registry",
        registry_cfg,
        "--auth",
        auth_cfg,
        "--tls.allowInsecure",
    ]
    proc = subprocess.Popen(
        argv,
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    watchdog = threading.Timer(timeout_seconds, proc.kill)
    watchdog.start()
    stdout_lines = []
    stderr = b""
    tools = []
    first = {}
    retry_response = None
    try:
        def send(message):
            proc.stdin.write(_frame_messages([message], b"\n"))
            proc.stdin.flush()

        send({"jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": {"_meta": _STATELESS_META}})
        listed = _await_response(proc, 1, stdout_lines) or {}
        tools = [t.get("name") for t in (listed.get("result") or {}).get("tools") or []]
        call_params = {
            "_meta": _STATELESS_META,
            "name": "run_mutation_query",
            "arguments": {"sql": mutation_sql},
        }
        send({"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": call_params})
        first_response = _await_response(proc, 2, stdout_lines) or {}
        first = first_response.get("result") or {}
        input_requests = first.get("inputRequests") or {}
        if input_requests:
            retry_params = dict(call_params)
            retry_params["inputResponses"] = {
                key: {"action": approval_action} for key in input_requests
            }
            # The server binds the answer to this prompt with a signed
            # requestState, which the retry must echo unchanged.
            if first.get("requestState"):
                retry_params["requestState"] = first["requestState"]
            send({"jsonrpc": "2.0", "id": 3, "method": "tools/call", "params": retry_params})
            retry_response = _await_response(proc, 3, stdout_lines)
        try:
            proc.stdin.close()
        except OSError:
            pass
        stdout_lines.append(proc.stdout.read())
        stderr = proc.stderr.read()
        proc.wait(timeout=timeout_seconds)
    finally:
        watchdog.cancel()
        if proc.poll() is None:
            proc.kill()
    list_error = listed.get("error") or {}
    return {
        "tools": tools,
        "tool_cache_ttl_ms": (listed.get("result") or {}).get("ttlMs"),
        "tool_cache_scope": (listed.get("result") or {}).get("cacheScope"),
        "list_error_code": list_error.get("code"),
        "list_error_supported": (list_error.get("data") or {}).get("supported") or [],
        "first_result_type": first.get("resultType", ""),
        "input_request_methods": sorted(
            (v or {}).get("method", "") for v in (first.get("inputRequests") or {}).values()
        ),
        "first_text": _tool_result_text({"result": first}),
        "retry": _tool_result_text(retry_response),
        "stdout": b"".join(stdout_lines).decode("utf-8", errors="replace"),
        "stderr": stderr.decode("utf-8", errors="replace"),
        "returncode": proc.returncode,
    }


def run_stdio_legacy_approval_roundtrip(
    stackql_exe,
    registry_cfg,
    auth_cfg,
    mutation_sql,
    approval_action="accept",
    mode="safe",
    timeout_seconds=90,
):
    """Drives a 2025-06-18 client over stdio that advertises elicitation:
    initialize -> initialized -> gated tools/call. The server issues an
    elicitation/create request mid-flight, which is answered with
    `approval_action`; returns the elicitation prompt and the final call text.
    """
    argv = [
        stackql_exe,
        "mcp",
        "--mcp.server.type=stdio",
        "--mcp.config",
        json.dumps({"server": {"mode": mode, "audit": {"disabled": True}}}),
        "--registry",
        registry_cfg,
        "--auth",
        auth_cfg,
        "--tls.allowInsecure",
    ]
    proc = subprocess.Popen(
        argv,
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    watchdog = threading.Timer(timeout_seconds, proc.kill)
    watchdog.start()
    stdout_lines = []
    stderr = b""
    negotiated = ""
    elicitation = None
    call_response = None
    try:
        def send(message):
            proc.stdin.write(_frame_messages([message], b"\n"))
            proc.stdin.flush()

        send({
            "jsonrpc": "2.0",
            "id": 1,
            "method": "initialize",
            "params": {
                "protocolVersion": "2025-06-18",
                "capabilities": {"elicitation": {}},
                "clientInfo": {"name": "robot-legacy-harness", "version": "0.1.0"},
            },
        })
        init = _await_response(proc, 1, stdout_lines) or {}
        negotiated = (init.get("result") or {}).get("protocolVersion", "")
        send({"jsonrpc": "2.0", "method": "notifications/initialized"})
        send({
            "jsonrpc": "2.0",
            "id": 2,
            "method": "tools/call",
            "params": {"name": "run_mutation_query", "arguments": {"sql": mutation_sql}},
        })
        elicitation = _await_message(
            proc, stdout_lines,
            lambda m: m.get("method") == "elicitation/create" or m.get("id") == 2,
        ) or {}
        if elicitation.get("method") == "elicitation/create":
            send({
                "jsonrpc": "2.0",
                "id": elicitation.get("id"),
                "result": {"action": approval_action},
            })
            call_response = _await_response(proc, 2, stdout_lines)
        else:
            call_response = elicitation
        try:
            proc.stdin.close()
        except OSError:
            pass
        stdout_lines.append(proc.stdout.read())
        stderr = proc.stderr.read()
        proc.wait(timeout=timeout_seconds)
    finally:
        watchdog.cancel()
        if proc.poll() is None:
            proc.kill()
    return {
        "negotiated": negotiated,
        "elicitation_message": ((elicitation or {}).get("params") or {}).get("message", ""),
        "call": _tool_result_text(call_response),
        "stdout": b"".join(stdout_lines).decode("utf-8", errors="replace"),
        "stderr": stderr.decode("utf-8", errors="replace"),
        "returncode": proc.returncode,
    }


def _http_post(url, payload, headers=None):
    """POSTs one JSON-RPC message; returns (status, headers, first JSON body
    decoded from either a JSON or an SSE response)."""
    import urllib.request
    import urllib.error

    body = json.dumps(payload).encode("utf-8")
    request = urllib.request.Request(url, data=body, method="POST")
    request.add_header("Content-Type", "application/json")
    request.add_header("Accept", "application/json, text/event-stream")
    for key, value in (headers or {}).items():
        request.add_header(key, value)
    try:
        with urllib.request.urlopen(request, timeout=60) as response:
            status = response.status
            response_headers = {k.lower(): v for k, v in response.headers.items()}
            raw = response.read().decode("utf-8", errors="replace")
    except urllib.error.HTTPError as err:
        # SDK v1.8.0 answers a rejected revision with a JSON-RPC error body
        # on a 400 status; keep it decodable for the downgrade assertions.
        error_headers = {k.lower(): v for k, v in err.headers.items()}
        raw = err.read().decode("utf-8", errors="replace")
        decoded = {"http_error": raw}
        if error_headers.get("content-type", "").startswith("application/json"):
            try:
                decoded = json.loads(raw)
            except ValueError:
                pass
        return err.code, error_headers, decoded
    decoded = {}
    if raw.strip():
        if response_headers.get("content-type", "").startswith("text/event-stream"):
            for line in raw.splitlines():
                if line.startswith("data:"):
                    try:
                        decoded = json.loads(line[len("data:"):].strip())
                    except ValueError:
                        continue
                    if isinstance(decoded, dict) and "id" in decoded:
                        break
        else:
            decoded = json.loads(raw)
    return status, response_headers, decoded


def _http_delete(url, headers=None):
    """Sends a DELETE with no session header and returns the HTTP status."""
    import urllib.request
    import urllib.error

    request = urllib.request.Request(url, method="DELETE")
    for key, value in (headers or {}).items():
        request.add_header(key, value)
    try:
        with urllib.request.urlopen(request, timeout=60) as response:
            return response.status
    except urllib.error.HTTPError as err:
        return err.code


def run_http_legacy_roundtrip(url, extra_headers=None):
    """2025-06-18 client over Streamable HTTP: initialize (session id issued),
    initialized, then tools/list on that session. Returns the negotiated
    version, whether a session id was issued and the tool names.
    extra_headers (eg Authorization) are sent on every request."""
    status, headers, init = _http_post(url, {
        "jsonrpc": "2.0",
        "id": 1,
        "method": "initialize",
        "params": {
            "protocolVersion": "2025-06-18",
            "capabilities": {},
            "clientInfo": {"name": "robot-legacy-http", "version": "0.1.0"},
        },
    }, extra_headers)
    session_id = headers.get("mcp-session-id", "")
    session_headers = dict(extra_headers or {})
    if session_id:
        session_headers["Mcp-Session-Id"] = session_id
    _http_post(url, {"jsonrpc": "2.0", "method": "notifications/initialized"}, session_headers)
    _, _, listed = _http_post(url, {"jsonrpc": "2.0", "id": 2, "method": "tools/list"}, session_headers)
    return {
        "status": status,
        "negotiated": (init.get("result") or {}).get("protocolVersion", ""),
        "session_issued": bool(session_id),
        "tools": [t.get("name") for t in (listed.get("result") or {}).get("tools") or []],
    }


def run_http_oauth_resource_server(
    stackql_exe,
    registry_cfg,
    auth_cfg,
    tls_cert_file,
    tls_key_file,
):
    """Runs OAuth acceptance/rejection requests against a real TLS StackQL
    process and a loopback TLS authorization server, acquiring a JWT through
    client_credentials before using it at MCP. No credentials or tokens are
    included in the returned result or written to files.
    """
    import base64
    import socket
    import ssl
    import http.client
    import hmac
    import secrets
    from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
    from urllib.error import HTTPError, URLError
    from urllib.request import Request, urlopen
    from urllib.parse import parse_qs, urlencode

    from cryptography.hazmat.primitives.asymmetric.ed25519 import Ed25519PrivateKey
    from cryptography.hazmat.primitives.serialization import Encoding, PublicFormat

    issuer = "https://issuer.example.test/"
    audience = "https://mcp.example.test/api"
    metadata_url = "https://mcp.example.test/.well-known/oauth-protected-resource"
    key_id = "robot-ed25519"
    private_key = Ed25519PrivateKey.generate()
    public_key = private_key.public_key().public_bytes(Encoding.Raw, PublicFormat.Raw)
    client_id = "robot-service"
    client_secret = secrets.token_urlsafe(32)
    client_authorization = "Basic " + base64.b64encode(
        (client_id + ":" + client_secret).encode()
    ).decode()

    def b64url(raw):
        return base64.urlsafe_b64encode(raw).rstrip(b"=").decode("ascii")

    def make_token(claims, kid=key_id, algorithm="EdDSA", signer=private_key):
        header = {"alg": algorithm, "kid": kid, "typ": "JWT"}
        first = b64url(json.dumps(header, separators=(",", ":"), sort_keys=True).encode())
        second = b64url(json.dumps(claims, separators=(",", ":"), sort_keys=True).encode())
        signing_input = (first + "." + second).encode("ascii")
        signature = signer.sign(signing_input)
        return first + "." + second + "." + b64url(signature)

    def valid_claims():
        return {
            "iss": issuer,
            "aud": audience,
            "exp": int(time.time()) + 300,
            "scope": "tools:read tools:write",
            "sub": "robot-service",
        }

    class JWKSHandler(BaseHTTPRequestHandler):
        def do_POST(self):
            if self.path != "/token":
                self.send_error(404)
                return
            length = int(self.headers.get("Content-Length", "0"))
            if not 0 < length <= 4096:
                self.send_error(400)
                return
            form = parse_qs(self.rfile.read(length).decode())
            if not hmac.compare_digest(self.headers.get("Authorization", ""), client_authorization):
                self._json_response(401, {"error": "invalid_client"})
                return
            if form.get("grant_type") != ["client_credentials"]:
                self._json_response(400, {"error": "unsupported_grant_type"})
                return
            scopes = form.get("scope", [""])[0].split()
            if not scopes or not set(scopes) <= {"tools:read", "tools:write"}:
                self._json_response(400, {"error": "invalid_scope"})
                return
            claims = valid_claims()
            claims["scope"] = " ".join(scopes)
            self._json_response(200, {
                "access_token": make_token(claims), "token_type": "Bearer",
                "expires_in": 300, "scope": claims["scope"],
            })

        def _json_response(self, status, payload):
            body = json.dumps(payload, separators=(",", ":")).encode()
            self.send_response(status)
            self.send_header("Content-Type", "application/json")
            self.send_header("Cache-Control", "no-store")
            self.send_header("Pragma", "no-cache")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

        def do_GET(self):
            if self.path != "/jwks":
                self.send_error(404)
                return
            body = json.dumps({
                "keys": [{
                    "kty": "OKP",
                    "crv": "Ed25519",
                    "x": b64url(public_key),
                    "kid": key_id,
                    "use": "sig",
                    "alg": "EdDSA",
                }],
            }, separators=(",", ":")).encode()
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

        def log_message(self, _format, *_args):
            return

    issuer_server = ThreadingHTTPServer(("127.0.0.1", 0), JWKSHandler)
    issuer_server.daemon_threads = True
    issuer_tls = ssl.SSLContext(ssl.PROTOCOL_TLS_SERVER)
    issuer_tls.minimum_version = ssl.TLSVersion.TLSv1_2
    issuer_tls.load_cert_chain(tls_cert_file, tls_key_file)
    issuer_server.socket = issuer_tls.wrap_socket(issuer_server.socket, server_side=True)
    issuer_thread = threading.Thread(target=issuer_server.serve_forever, daemon=True)
    issuer_thread.start()
    issuer = "https://127.0.0.1:" + str(issuer_server.server_port) + "/"

    with socket.socket() as reservation:
        reservation.bind(("127.0.0.1", 0))
        server_port = reservation.getsockname()[1]
    server_url = "https://127.0.0.1:" + str(server_port)
    server_config = {
        "server": {
            "transport": "http",
            "address": "127.0.0.1:" + str(server_port),
            "tls_cert_file": tls_cert_file,
            "tls_key_file": tls_key_file,
            "mode": "full_access",
            "audit": {"disabled": True},
            "oauth": {
                "issuer": issuer,
                "audience": audience,
                "resource": audience,
                "jwks_url": "https://127.0.0.1:" + str(issuer_server.server_port) + "/jwks",
                "jwks_ca_file": tls_cert_file,
                "metadata_url": metadata_url,
                "scopes": ["tools:read", "tools:write"],
                "algorithms": ["EdDSA"],
            },
        },
    }
    argv = [
        stackql_exe,
        "mcp",
        "--mcp.server.type=http",
        "--mcp.config",
        json.dumps(server_config, separators=(",", ":")),
        "--registry",
        registry_cfg,
        "--auth",
        auth_cfg,
        "--tls.allowInsecure",
    ]
    repo_root = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", "..", ".."))
    process = subprocess.Popen(
        argv,
        cwd=repo_root,
        stdin=subprocess.DEVNULL,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    client_tls = ssl.create_default_context(cafile=tls_cert_file)
    all_tokens = []

    def acquire_token(scope, authorization=client_authorization, grant_type="client_credentials"):
        req = Request(
            issuer + "token",
            data=urlencode({"grant_type": grant_type, "scope": scope}).encode(),
            headers={"Content-Type": "application/x-www-form-urlencoded", "Authorization": authorization},
            method="POST",
        )
        try:
            with urlopen(req, timeout=3, context=client_tls) as response:
                payload = json.loads(response.read())
                if response.status == 200:
                    all_tokens.append(payload["access_token"])
                return response.status, payload
        except HTTPError as error:
            with error:
                return error.code, json.loads(error.read())

    def request(url, bearer=None):
        headers = {}
        if bearer:
            headers["Authorization"] = "Bearer " + bearer
        if url.endswith("/.well-known/oauth-protected-resource"):
            req = Request(url, headers=headers, method="GET")
        else:
            payload = {
                "jsonrpc": "2.0",
                "id": 1,
                "method": "initialize",
                "params": {
                    "protocolVersion": "2025-03-26",
                    "capabilities": {},
                    "clientInfo": {"name": "oauth-robot", "version": "1"},
                },
            }
            req = Request(
                url,
                data=json.dumps(payload, separators=(",", ":")).encode(),
                headers={"Content-Type": "application/json", "Accept": "application/json, text/event-stream", **headers},
                method="POST",
            )
        try:
            with urlopen(req, timeout=3, context=client_tls) as response:
                body = response.read()
                return response.status, {key.lower(): value for key, value in response.headers.items()}, body
        except HTTPError as error:
            body = error.read()
            return error.code, {key.lower(): value for key, value in error.headers.items()}, body
        except (URLError, OSError, http.client.HTTPException) as error:
            return 0, {}, str(error).encode("utf-8", errors="replace")

    result = {"startup_error": ""}
    try:
        deadline = time.monotonic() + 15
        metadata = None
        while time.monotonic() < deadline:
            if process.poll() is not None:
                break
            try:
                metadata = request(server_url + "/.well-known/oauth-protected-resource")
                if metadata[0] == 200:
                    break
            except (OSError, URLError):
                time.sleep(0.05)
        if metadata is None or metadata[0] != 200:
            result["startup_error"] = "OAuth HTTP server did not become ready"
            return result
        result["metadata_status"] = metadata[0]
        result["metadata_resource"] = json.loads(metadata[2]).get("resource", "")
        acquired_status, acquired = acquire_token("tools:read tools:write")
        result["token_endpoint_status"] = acquired_status
        result["token_endpoint_bad_client_status"], _ = acquire_token("tools:read", authorization="Basic invalid")
        result["token_endpoint_bad_grant_status"], _ = acquire_token("tools:read", grant_type="password")
        result["token_endpoint_bad_scope_status"], _ = acquire_token("admin")
        if acquired_status != 200 or acquired.get("token_type") != "Bearer" or acquired.get("expires_in") != 300:
            result["startup_error"] = "Client Credentials acquisition failed"
            return result
        limited_status, limited = acquire_token("tools:read")
        if limited_status != 200:
            result["startup_error"] = "Limited-scope Client Credentials acquisition failed"
            return result

        cases = {}
        cases["missing"] = (None, 401)
        cases["valid"] = (acquired["access_token"], 200)
        cases["acquired_scope"] = (limited["access_token"], 403)
        invalid_issuer = valid_claims()
        invalid_issuer["iss"] = "https://wrong-issuer.example.test/"
        cases["issuer"] = (make_token(invalid_issuer), 401)
        invalid_audience = valid_claims()
        invalid_audience["aud"] = "https://wrong-audience.example.test/"
        cases["audience"] = (make_token(invalid_audience), 401)
        expired = valid_claims()
        expired["exp"] = 1
        cases["expired"] = (make_token(expired), 401)
        missing_expiration = valid_claims()
        del missing_expiration["exp"]
        cases["missing_expiration"] = (make_token(missing_expiration), 401)
        insufficient_scope = valid_claims()
        insufficient_scope["scope"] = "tools:read"
        cases["scope"] = (make_token(insufficient_scope), 403)
        unknown_key = make_token(valid_claims(), kid="unknown-key")
        cases["unknown_key"] = (unknown_key, 401)
        wrong_algorithm = make_token(valid_claims(), algorithm="HS256")
        cases["algorithm"] = (wrong_algorithm, 401)
        invalid_signature = make_token(valid_claims())
        first, second, signature = invalid_signature.split(".")
        changed_signature = ("A" if signature[:1] != "A" else "B") + signature[1:]
        cases["signature"] = (first + "." + second + "." + changed_signature, 401)

        statuses = {}
        challenge = ""
        valid_has_session = False
        for name, (token, expected) in cases.items():
            if token:
                all_tokens.append(token)
            status, headers, _ = request(server_url + "/", token)
            statuses[name] = status
            if name == "missing":
                challenge = headers.get("www-authenticate", "")
            if name == "valid":
                valid_has_session = bool(headers.get("mcp-session-id"))
            if status != expected:
                result["startup_error"] = "OAuth case returned an unexpected HTTP status"
                result["statuses"] = statuses
                result["failure_detail"] = _.decode("utf-8", errors="replace") if isinstance(_, bytes) else ""
                return result
        result["statuses"] = statuses
        result["challenge"] = challenge
        result["valid_has_session"] = valid_has_session
        result["logs_clean"] = True
    finally:
        if process.poll() is None:
            process.terminate()
        try:
            stdout, stderr = process.communicate(timeout=5)
        except subprocess.TimeoutExpired:
            process.kill()
            stdout, stderr = process.communicate(timeout=5)
        issuer_server.shutdown()
        issuer_server.server_close()
        issuer_thread.join(timeout=5)
        logs = (stdout or b"") + (stderr or b"")
        result["logs_clean"] = result.get("logs_clean", True) and all(
            token.encode() not in logs for token in all_tokens
        ) and client_secret.encode() not in logs and client_authorization.encode() not in logs \
            and b"invalid token" not in logs.lower() and b"jwks unavailable" not in logs.lower()
        if process.returncode not in (0, -15):
            result["process_returncode"] = process.returncode
    return result


def _stateless_headers(payload):
    """The 2026-07-28 Streamable HTTP binding carries the revision, the method
    and (for tools/call) the tool name as headers alongside the body."""
    headers = {
        "Mcp-Protocol-Version": "2026-07-28",
        "Mcp-Method": payload.get("method", ""),
    }
    name = (payload.get("params") or {}).get("name")
    if payload.get("method") == "tools/call" and name:
        headers["Mcp-Name"] = name
    return headers


def _http_post_stateless(url, payload):
    return _http_post(url, payload, _stateless_headers(payload))


def run_http_stateless_roundtrip(url):
    """2026-07-28 client over Streamable HTTP: no handshake, no session
    header; server/discover (SEP-2575) replaces initialize as the one-shot
    carrier of versions, capabilities and instructions, then the revision
    travels in the Mcp-Protocol-Version header and in _meta on tools/list and
    a server_info call."""
    _, _, discovered = _http_post_stateless(url, {
        "jsonrpc": "2.0", "id": 0, "method": "server/discover", "params": {"_meta": _STATELESS_META},
    })
    _, headers, listed = _http_post_stateless(url, {
        "jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": {"_meta": _STATELESS_META},
    })
    _, _, info = _http_post_stateless(url, {
        "jsonrpc": "2.0", "id": 2, "method": "tools/call",
        "params": {"_meta": _STATELESS_META, "name": "server_info", "arguments": {}},
    })
    discover_result = discovered.get("result") or {}
    list_error = listed.get("error") or {}
    return {
        "discover_versions": discover_result.get("supportedVersions") or [],
        "discover_instructions": discover_result.get("instructions") or "",
        "session_issued": bool(headers.get("mcp-session-id", "")),
        "tools": [t.get("name") for t in (listed.get("result") or {}).get("tools") or []],
        "list_error_code": list_error.get("code"),
        "list_error_supported": (list_error.get("data") or {}).get("supported") or [],
        # Sessionless servers have nothing to tear down: DELETE is 405.
        "delete_status": _http_delete(url, {"Mcp-Protocol-Version": "2026-07-28"}),
        "server_info": _tool_result_text(info),
    }


def run_http_stateless_gated_write(url, mutation_sql, approval_action="accept"):
    """2026-07-28 client over Streamable HTTP against a safe-mode server: the
    gated tools/call comes back input_required with an elicitation/create
    input request, and the retry carries `approval_action` in inputResponses
    together with the signed requestState echoed from the first result.
    """
    call_params = {
        "_meta": _STATELESS_META,
        "name": "run_mutation_query",
        "arguments": {"sql": mutation_sql},
    }
    _, _, first_response = _http_post_stateless(url, {
        "jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": call_params,
    })
    first = first_response.get("result") or {}
    input_requests = first.get("inputRequests") or {}
    retry_response = None
    if input_requests:
        retry_params = dict(call_params)
        retry_params["inputResponses"] = {
            key: {"action": approval_action} for key in input_requests
        }
        # Echo the signed requestState that binds the answer to this prompt.
        if first.get("requestState"):
            retry_params["requestState"] = first["requestState"]
        _, _, retry_response = _http_post_stateless(url, {
            "jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": retry_params,
        })
    return {
        "first_result_type": first.get("resultType", ""),
        "input_request_methods": sorted(
            (v or {}).get("method", "") for v in input_requests.values()
        ),
        "first_text": _tool_result_text({"result": first}),
        "retry": _tool_result_text(retry_response),
    }
