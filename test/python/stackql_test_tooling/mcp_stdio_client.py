"""Minimal stdio MCP harness used by robot scenarios.

Drives a `stackql mcp --mcp.server.type=stdio` child over raw byte pipes in
binary mode with a configurable line terminator (issue #668 CRLF framing).
Also hosts the issue #688 credential reload roundtrip against the
`--env.file` dotenv file, the issue #701 malformed-frame resilience
roundtrip, and the issue #729 protocol revision conformance roundtrips
(2025-06-18 handshake client and 2026-07-28 stateless client, stdio and
Streamable HTTP).
"""

import json
import os
import subprocess
import threading

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


def run_stdio_credential_reload_roundtrip(
    stackql_exe,
    registry_cfg,
    auth_cfg,
    env_file_path,
    secret_env_var,
    secret_value,
    select_sql,
    timeout_seconds=120,
):
    """Issue #688 end-to-end: credential (re)sourcing over a stdio session.

    Spawns the server WITHOUT `secret_env_var`, `--env.file` pointing at a
    not-yet-existing file; runs `select_sql` (expects a credential error),
    writes the env file, calls `reload_credentials`, re-runs the query
    (expects rows).  Returns the three flattened tool results plus streams.
    """
    if os.path.exists(env_file_path):
        os.remove(env_file_path)
    child_env = {
        k: v for k, v in os.environ.items()
        if k.upper() != secret_env_var.upper()
    }
    argv = [
        stackql_exe,
        "mcp",
        "--mcp.server.type=stdio",
        "--mcp.config",
        '{"server": {"audit": {"disabled": true}} }',
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
    select_before = reload_response = select_after = None
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
        send({
            "jsonrpc": "2.0",
            "id": 2,
            "method": "tools/call",
            "params": {
                "name": "run_select_query",
                "arguments": {"sql": select_sql},
            },
        })
        select_before = _await_response(proc, 2, stdout_lines)
        with open(env_file_path, "w") as f:
            f.write("# written mid-session by the robot harness\n")
            f.write(f"{secret_env_var}={secret_value}\n")
        send({
            "jsonrpc": "2.0",
            "id": 3,
            "method": "tools/call",
            "params": {"name": "reload_credentials", "arguments": {}},
        })
        reload_response = _await_response(proc, 3, stdout_lines)
        send({
            "jsonrpc": "2.0",
            "id": 4,
            "method": "tools/call",
            "params": {
                "name": "run_select_query",
                "arguments": {"sql": select_sql},
            },
        })
        select_after = _await_response(proc, 4, stdout_lines)
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
        "select_before": _tool_result_text(select_before),
        "reload": _tool_result_text(reload_response),
        "select_after": _tool_result_text(select_after),
        "stdout": b"".join(stdout_lines).decode("utf-8", errors="replace"),
        "stderr": stderr.decode("utf-8", errors="replace"),
        "returncode": proc.returncode,
    }


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
    timeout_seconds=90,
):
    """Drives a 2026-07-28 client over stdio: no initialize handshake, the
    protocol version and client capabilities ride in params._meta on every
    request. Runs tools/list, then a gated mutation whose approval comes back
    as an input_required result and is answered on the retry via
    inputResponses. Returns the tool names, the first mutation result shape,
    the retried mutation text and the raw streams.
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
    return {
        "tools": tools,
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
        return err.code, {k.lower(): v for k, v in err.headers.items()}, {"http_error": err.read().decode("utf-8", errors="replace")}
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


def run_http_legacy_roundtrip(url):
    """2025-06-18 client over Streamable HTTP: initialize (session id issued),
    initialized, then tools/list on that session. Returns the negotiated
    version, whether a session id was issued and the tool names."""
    status, headers, init = _http_post(url, {
        "jsonrpc": "2.0",
        "id": 1,
        "method": "initialize",
        "params": {
            "protocolVersion": "2025-06-18",
            "capabilities": {},
            "clientInfo": {"name": "robot-legacy-http", "version": "0.1.0"},
        },
    })
    session_id = headers.get("mcp-session-id", "")
    session_headers = {"Mcp-Session-Id": session_id} if session_id else {}
    _http_post(url, {"jsonrpc": "2.0", "method": "notifications/initialized"}, session_headers)
    _, _, listed = _http_post(url, {"jsonrpc": "2.0", "id": 2, "method": "tools/list"}, session_headers)
    return {
        "status": status,
        "negotiated": (init.get("result") or {}).get("protocolVersion", ""),
        "session_issued": bool(session_id),
        "tools": [t.get("name") for t in (listed.get("result") or {}).get("tools") or []],
    }


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
    header; the revision travels in the Mcp-Protocol-Version header and in
    _meta on tools/list and a server_info call."""
    _, headers, listed = _http_post_stateless(url, {
        "jsonrpc": "2.0", "id": 1, "method": "tools/list", "params": {"_meta": _STATELESS_META},
    })
    _, _, info = _http_post_stateless(url, {
        "jsonrpc": "2.0", "id": 2, "method": "tools/call",
        "params": {"_meta": _STATELESS_META, "name": "server_info", "arguments": {}},
    })
    return {
        "session_issued": bool(headers.get("mcp-session-id", "")),
        "tools": [t.get("name") for t in (listed.get("result") or {}).get("tools") or []],
        "server_info": _tool_result_text(info),
    }


def run_http_stateless_gated_write(url, mutation_sql, approval_action="accept"):
    """2026-07-28 client over Streamable HTTP against a safe-mode server: the
    gated tools/call comes back input_required with an elicitation/create
    input request, and the retry carries `approval_action` in inputResponses.
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
