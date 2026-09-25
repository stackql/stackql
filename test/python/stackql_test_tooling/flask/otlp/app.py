"""Flask mock of an OTLP/HTTP logs endpoint for the --output otel push exporter.

Every export request is captured so a scenario can assert the body, headers
and batching; a keyed flaky route fails a set number of times to exercise the
exporter's retry.

Run with:
    flask --app=test/python/stackql_test_tooling/flask/otlp/app run --host 0.0.0.0 --port 1200
"""

from collections import defaultdict
from threading import Lock

from flask import Flask, jsonify, request


def create_app() -> Flask:
    app = Flask(__name__)

    received: list = []
    counters: "defaultdict[str, int]" = defaultdict(int)
    lock = Lock()

    def _record_count(body: dict) -> int:
        return sum(
            len(scope.get("logRecords", []))
            for resource in body.get("resourceLogs", [])
            for scope in resource.get("scopeLogs", [])
        )

    def _capture(key: str):
        body = request.get_json(force=True, silent=True) or {}
        with lock:
            received.append(
                {
                    "key": key,
                    "path": request.path,
                    "headers": {k.lower(): v for k, v in request.headers.items()},
                    "records": _record_count(body),
                    "body": body,
                }
            )
        return jsonify({"partialSuccess": {}})

    @app.post("/reset")
    def reset():
        with lock:
            received.clear()
            counters.clear()
        return jsonify({"ok": True})

    @app.get("/requests")
    def requests_received():
        with lock:
            return jsonify(received)

    @app.get("/count/<key>")
    def count(key: str):
        with lock:
            return jsonify({"key": key, "attempts": counters[key]})

    @app.post("/v1/logs")
    def export_logs():
        return _capture("default")

    @app.post("/flaky/<key>/v1/logs")
    def export_logs_flaky(key: str):
        try:
            fail_until = int(request.args.get("fail_until", "0"))
        except ValueError:
            fail_until = 0
        with lock:
            counters[key] += 1
            attempt = counters[key]
        if attempt <= fail_until:
            return jsonify({"attempt": attempt, "ok": False}), 503
        return _capture(key)

    return app


app = create_app()


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser()
    parser.add_argument("--port", type=int, default=1200)
    parser.add_argument("--host", default="0.0.0.0")
    args = parser.parse_args()
    app.run(host=args.host, port=args.port)
