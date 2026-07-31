"""Mock server for the StackQL query library URL contract.

Serves the same paths the published library will expose at
https://stackql.io/docs/query-library so the MCP query library tools can be
exercised against a live HTTP endpoint before part 1 (the real site) exists:

    /docs/query-library/manifest.json
    /docs/query-library/index.json
    /docs/query-library/index.md
    /docs/query-library/queries/<id>.json
    /docs/query-library/queries/<id>.md

Run standalone:

    flask --app=test/python/stackql_test_tooling/flask/query_library/app run --host 0.0.0.0 --port 1091

then point the MCP server at it:

    STACKQL_QUERY_LIBRARY_BASE_URL=http://127.0.0.1:1091/docs/query-library stackql mcp ...
"""

import argparse
import json

from flask import Flask, Response

BUILD_ID = "mock-build-001"

QUERIES = {
    "aws/ec2/regions-enabled": {
        "id": "aws/ec2/regions-enabled",
        "title": "Enabled AWS regions",
        "description": "Lists AWS regions with their opt-in status.",
        "mutation": False,
        "status": "stable",
        "providers": ["aws"],
        "services": ["ec2"],
        "params": [
            {
                "name": "seed_region",
                "type": "identifier",
                "required": False,
                "default": "us-east-1",
                "description": "Routing region for the describe-regions call",
                "example": "us-east-1",
            }
        ],
        "template": "SELECT regionName, optInStatus FROM aws.ec2_native.regions WHERE region = '{{seed_region}}';",
        "notes": "region is a routing parameter, not a filter.",
        "doc_url": "https://stackql.io/docs/query-library/queries/aws/ec2/regions-enabled",
    },
    "aws/s3/buckets-list": {
        "id": "aws/s3/buckets-list",
        "title": "S3 buckets cheap enumeration",
        "description": "Enumerates S3 bucket names and regions via the list-only resource.",
        "mutation": False,
        "status": "stable",
        "providers": ["aws"],
        "services": ["s3"],
        "params": [
            {
                "name": "region",
                "type": "identifier",
                "required": True,
                "description": "Routing region",
                "example": "us-east-1",
            }
        ],
        "template": "SELECT bucket_name, region FROM aws.s3.buckets_list_only WHERE region = '{{region}}';",
        "notes": "S3 bucket listing is account-global; the region routes the request.",
        "doc_url": "https://stackql.io/docs/query-library/queries/aws/s3/buckets-list",
    },
}


def _index_entry(doc: dict) -> dict:
    return {
        "id": doc["id"],
        "title": doc["title"],
        "description": doc["description"],
        "providers": doc["providers"],
        "services": doc["services"],
        "tags": doc["providers"] + doc["services"],
        "keywords": [],
        "intent_keywords": {
            "aws/ec2/regions-enabled": ["list enabled aws regions", "enumerate aws regions"],
            "aws/s3/buckets-list": ["list all s3 buckets", "bucket inventory"],
        }.get(doc["id"], []),
        "mutation": doc["mutation"],
        "status": doc["status"],
        "required_params": [p["name"] for p in doc["params"] if p["required"]],
    }


def create_app() -> Flask:
    app = Flask(__name__)

    def _json(payload: dict) -> Response:
        return Response(json.dumps(payload), mimetype="application/json")

    @app.route("/docs/query-library/manifest.json")
    def manifest() -> Response:
        return _json(
            {
                "build_id": BUILD_ID,
                "generated_at": "2026-07-26T00:00:00Z",
                "library_commit": "mock",
                "entry_count": len(QUERIES),
            }
        )

    @app.route("/docs/query-library/index.json")
    def index_json() -> Response:
        return _json({"build_id": BUILD_ID, "entries": [_index_entry(q) for q in QUERIES.values()]})

    @app.route("/docs/query-library/index.md")
    def index_md() -> Response:
        lines = ["# Query library index", ""]
        for q in QUERIES.values():
            lines.append(f"- `{q['id']}` - {q['description']}")
        return Response("\n".join(lines), mimetype="text/markdown")

    @app.route("/docs/query-library/queries/<path:query_id>.json")
    def query_json(query_id: str) -> Response:
        doc = QUERIES.get(query_id)
        if doc is None:
            return Response("not found", status=404)
        return _json(doc)

    @app.route("/docs/query-library/queries/<path:query_id>.md")
    def query_md(query_id: str) -> Response:
        doc = QUERIES.get(query_id)
        if doc is None:
            return Response("not found", status=404)
        body = f"# {doc['title']}\n\n## Query\n\n```sql\n{doc['template']}\n```\n\n## Notes\n\n{doc['notes']}\n"
        return Response(body, mimetype="text/markdown")

    return app


app = create_app()

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--host", default="0.0.0.0")
    parser.add_argument("--port", type=int, default=1091)
    args = parser.parse_args()
    app.run(host=args.host, port=args.port)
