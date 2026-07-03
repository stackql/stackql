"""Minimal stdio MCP harness used by robot scenarios.

Drives a `stackql mcp --mcp.server.type=stdio` child process over raw byte
pipes with a configurable line terminator, so CRLF framing (issue #668) can
be exercised end to end.  Deliberately binary-mode (no `text=True`) so the
requested terminator reaches the server byte-for-byte on every platform.
"""

import json
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
