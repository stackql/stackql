"""Flask mock for the retrytestprovider used by stackql retry-policy tests.

Mirrors the upstream any-sdk retry mock so the same scenarios can be exercised
end-to-end through stackql. Counters are keyed per-route so each test can reset
state and assert exactly how many attempts the retry loop made.

Run with:
    flask --app=test/python/stackql_test_tooling/flask/retry/app run --host 0.0.0.0 --port 1199
"""

from collections import defaultdict
from threading import Lock

from flask import Flask, jsonify, request


def create_app() -> Flask:
    app = Flask(__name__)

    counters: "defaultdict[str, int]" = defaultdict(int)
    lock = Lock()

    @app.post("/reset")
    def reset():
        with lock:
            counters.clear()
        return jsonify({"ok": True})

    @app.get("/count/<key>")
    def count(key: str):
        with lock:
            return jsonify({"key": key, "attempts": counters[key]})

    @app.get("/flaky/<key>")
    def flaky(key: str):
        try:
            fail_until = int(request.args.get("fail_until", "0"))
        except ValueError:
            fail_until = 0
        with lock:
            counters[key] += 1
            attempt = counters[key]
        body = {"key": key, "attempt": attempt, "fail_until": fail_until}
        if attempt <= fail_until:
            return jsonify({**body, "ok": False}), 503
        return jsonify({**body, "ok": True})

    @app.get("/always_503")
    def always_503():
        with lock:
            counters["always_503"] += 1
            attempt = counters["always_503"]
        return jsonify({"attempt": attempt, "ok": False}), 503

    return app


app = create_app()


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser()
    parser.add_argument("--port", type=int, default=1199)
    parser.add_argument("--host", default="0.0.0.0")
    args = parser.parse_args()
    app.run(host=args.host, port=args.port)
