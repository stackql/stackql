"""Minimal stdio MCP harness used by robot scenarios.

Drives a `stackql mcp --mcp.server.type=stdio` child process over raw byte
pipes with a configurable line terminator, so CRLF framing (issue #668) can
be exercised end to end.  Deliberately binary-mode (no `text=True`) so the
requested terminator reaches the server byte-for-byte on every platform.

Also hosts the issue #688 roundtrip: a stdio server spawned WITHOUT a
credential env var self-heals mid-session once the credential lands in the
`--env.file` dotenv file and the `reload_credentials` tool is called.
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

    Returns a dict with `stdout`, `stderr` and `returncode`.  Responses for
    request ids 1 (initialize) and 2 (tools/list) are awaited on stdout
    before stdin is closed, so slow request handling cannot race the
    EOF-triggered session shutdown.
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


def _await_response(proc, request_id, collected_lines):
    """Reads stdout lines until the response for `request_id` arrives.

    Returns the decoded response dict (or None if the server exited first).
    Every raw line is appended to `collected_lines` for post-mortem output.
    """
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
    """Flattens a tools/call response to a single string for assertions.

    Concatenates text content blocks, the structured content and any
    JSON-RPC error object, so callers can assert on whichever surface the
    server used to report the outcome.
    """
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

    1. Spawns a stdio MCP server whose environment deliberately LACKS
       `secret_env_var`, with `--env.file` pointing at a file that does not
       exist yet - modelling a Claude-Desktop-spawned child that never
       received the credential.
    2. Runs `select_sql`; expects a credential resolution error.
    3. Writes `secret_env_var=secret_value` to the env file (the mutable
       store), calls the `reload_credentials` tool.
    4. Re-runs `select_sql`; expects rows.

    Returns a dict with the flattened text of each of the three tool calls
    plus stderr and the exit code.
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
