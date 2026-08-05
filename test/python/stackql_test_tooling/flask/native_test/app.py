"""Flask mock backing the stackql_native_test provider.

Serves two concerns on a single port (the registry_rewrite default, 1070):

* /casing/echo  - echoes the wire query string / request body so the casing
  suite can assert that snake_case SQL keys were reverse-resolved to the
  PascalCase wire form declared by request.nativeCasing.
* /xml/...      - returns canned XML payloads per AWS protocol archetype
  (ec2 / query / rest-xml) so the schema_driven_xml walker suite can assert
  per-row, schema-typed projection.

Run with:
    flask --app=test/python/stackql_test_tooling/flask/native_test/app run --host 0.0.0.0 --port 1070
"""

import base64
import hashlib
import re

from flask import Flask, Response, jsonify, request


_OCI_POST_SIGNED_HEADERS = "date (request-target) host content-length content-type x-content-sha256"


def _oci_auth_fields(req) -> dict:
    # Crack the draft-cavage Authorization header into columns the suite can
    # assert on; the signature itself varies per request (date) so it is not echoed.
    auth = req.headers.get("Authorization", "")
    fields = dict(re.findall(r'(\w+)="([^"]*)"', auth))
    return {
        "auth_scheme": auth.split(" ")[0] if auth else "",
        "auth_version": fields.get("version", ""),
        "auth_algorithm": fields.get("algorithm", ""),
        "auth_key_id": fields.get("keyId", ""),
        "auth_signed_headers": fields.get("headers", ""),
    }


def create_app() -> Flask:
    app = Flask(__name__)

    # ---- casing -------------------------------------------------------------

    @app.get("/casing/echo")
    def casing_echo_get():
        return jsonify(
            {
                "VpcId": request.args.get("VpcId"),
                "SubnetId": request.args.get("SubnetId"),
                "echoed_query": request.query_string.decode("utf-8"),
            }
        )

    @app.post("/casing/echo")
    def casing_echo_post():
        return jsonify(
            {
                "echoed_body": request.get_data(as_text=True),
                "ok": True,
            }
        )

    @app.get("/casing/echo_strict")
    def casing_echo_strict_get():
        # Same echo semantics as /casing/echo; the spec-side difference is that
        # VpcId is a REQUIRED wire parameter, so reaching this endpoint at all
        # proves the router satisfied the requirement (via the snake alias when
        # the SQL used vpc_id).
        return jsonify(
            {
                "VpcId": request.args.get("VpcId"),
                "SubnetId": request.args.get("SubnetId"),
                "echoed_query": request.query_string.decode("utf-8"),
            }
        )

    # ---- OCI request signing (oci_signing_v1) target -----------------------

    @app.get("/oci/buckets")
    def oci_buckets_list():
        return jsonify({"items": [dict(name="bucket-1", **_oci_auth_fields(request))]})

    @app.post("/oci/buckets")
    def oci_buckets_create():
        # Enforce the OCI body-verb signing contract: the six-header signed set
        # and a correct x-content-sha256 digest, else the insert visibly fails.
        fields = _oci_auth_fields(request)
        if fields["auth_signed_headers"] != _OCI_POST_SIGNED_HEADERS:
            return jsonify({"error": f"unexpected signed headers: {fields['auth_signed_headers']}"}), 400
        expected_digest = base64.b64encode(hashlib.sha256(request.get_data()).digest()).decode()
        if request.headers.get("x-content-sha256") != expected_digest:
            return jsonify({"error": "x-content-sha256 mismatch"}), 400
        body = request.get_json(silent=True) or {}
        return jsonify(dict(name=body.get("name", ""), **fields))

    # ---- server variable env resolution target -----------------------------

    @app.get("/srvvar/<deployment>/items")
    def srvvar_items(deployment):
        # Echo the deployment path segment so tests can assert which server
        # variable value (env-resolved vs WHERE-supplied) reached the wire.
        return jsonify({"items": [{"name": "item-1", "deployment": deployment}]})

    # ---- OData push-down target --------------------------------------------

    @app.get("/odata/people")
    def odata_people():
        # Echo the DECODED request query (Flask url-decodes args) so tests can assert
        # which OData options stackql pushed down via any-sdk ApplyPushdown.
        echoed = " ".join(f"{k}={v}" for k, v in request.args.items())
        people = [
            {"name": "Alice", "city": "NYC", "age": 30, "echoed": echoed},
            {"name": "Acme", "city": "SF", "age": 40, "echoed": echoed},
            {"name": "Bob", "city": "LA", "age": 25, "echoed": echoed},
        ]
        # Honour $top server-side so a wrongly-pushed $top is observable as an
        # under-count (the grain-change guard test relies on this).
        top = request.args.get("$top")
        if top is not None and top.isdigit():
            people = people[: int(top)]
        # Honour $select server-side as real OData services do; makes issue
        # #682 observable (an omitted WHERE/ORDER BY column drops every row).
        select = request.args.get("$select")
        if select:
            requested = {f.strip() for f in select.split(",") if f.strip()}
            people = [
                {k: v for k, v in person.items() if k in requested}
                for person in people
            ]
        return jsonify({"value": people, "@odata.count": len(people)})

    # ---- GraphQL cursor pagination -----------------------------------------

    _things = [
        {"name": "red", "rank": 1},
        {"name": "green", "rank": 2},
        {"name": "blue", "rank": 3},
        {"name": "yellow", "rank": 4},
        {"name": "purple", "rank": 5},
    ]

    @app.post("/graphql")
    def graphql():
        body = request.get_json(silent=True) or {}
        query = body.get("query", "")
        # Reflect the wire-query page args into each node so tests can assert the
        # pushed LIMIT (first:) and the followed cursor (after:) from STDOUT - the
        # http.log stderr is not portably captured under docker.
        fm = re.search(r"first:\s*(\d+)", query)
        wire_first = int(fm.group(1)) if fm else 0
        am = re.search(r'after:\s*"c(\d+)"', query)
        wire_after = ("c" + am.group(1)) if am else ""
        # Relay cursor: edge cursor is "c<absolute index>"; "after: cN" resumes at N+1.
        start = (int(am.group(1)) + 1) if am else 0
        page = 2
        window = _things[start:start + page]
        edges = []
        for i in range(len(window)):
            idx = start + i
            node = dict(_things[idx])
            node["wire_first"] = wire_first
            node["wire_after"] = wire_after
            edges.append({"node": node, "cursor": "c" + str(idx)})
        has_next = (start + page) < len(_things)
        end_cursor = edges[-1]["cursor"] if edges else None
        return jsonify(
            {
                "data": {
                    "things": {
                        "edges": edges,
                        "pageInfo": {"hasNextPage": has_next, "endCursor": end_cursor},
                    }
                }
            }
        )

    # ---- REST page_number pagination (issue 684) ---------------------------

    _paged_items = [{"name": f"paged-item-{i}", "idx": i} for i in range(1, 7)]

    @app.get("/paged/items")
    def paged_items():
        # page_number strategy: reader requests page N+1 until page == total.
        page_raw = request.args.get("page", "1")
        page = int(page_raw) if page_raw.isdigit() else 1
        size = 2
        window = _paged_items[(page - 1) * size:(page - 1) * size + size]
        rows = [dict(item, wire_page=page) for item in window]
        return jsonify({
            "items": rows,
            "result_info": {"page": page, "total_pages": 3},
        })

    @app.get("/paged/items_unterminated")
    def paged_items_unterminated():
        # Negative case: no total_pages terminator; the reader must stop
        # after the first page, never loop forever.
        page_raw = request.args.get("page", "1")
        page = int(page_raw) if page_raw.isdigit() else 1
        window = _paged_items[(page - 1) * 2:(page - 1) * 2 + 2]
        rows = [dict(item, wire_page=page) for item in window]
        return jsonify({"items": rows, "result_info": {"page": page}})

    # ---- GraphQL pluggable cursor strategies (issue 684) --------------------

    @app.post("/graphql/keyset")
    def graphql_keyset():
        # keyset: filter comparator on the last row's sort key; empty rows terminate.
        body = request.get_json(silent=True) or {}
        query = body.get("query", "")
        m = re.search(r"rankGt:\s*(\d+)", query)
        after_rank = int(m.group(1)) if m else 0
        window = [t for t in _things if t["rank"] > after_rank][:2]
        nodes = [dict(t, wire_rank_gt=after_rank) for t in window]
        return jsonify({"data": {"kthings": {"nodes": nodes}}})

    @app.post("/graphql/offset")
    def graphql_offset():
        # offset: synthesised running row count; empty rows terminate.
        body = request.get_json(silent=True) or {}
        query = body.get("query", "")
        m = re.search(r"offset:\s*(\d+)", query)
        offset = int(m.group(1)) if m else 0
        window = _things[offset:offset + 2]
        nodes = [dict(t, wire_offset=offset) for t in window]
        return jsonify({"data": {"othings": {"nodes": nodes}}})

    @app.post("/graphql/pageinfo")
    def graphql_pageinfo():
        # page_info: Relay-strict - endCursor stays non-empty on the final
        # page, so only pageInfo.hasNextPage may terminate.
        body = request.get_json(silent=True) or {}
        query = body.get("query", "")
        m = re.search(r'after:\s*"c(\d+)"', query)
        start = (int(m.group(1)) + 1) if m else 0
        window = _things[start:start + 2]
        edges = []
        for i in range(len(window)):
            idx = start + i
            node = dict(_things[idx])
            node["wire_after"] = ("c" + m.group(1)) if m else ""
            edges.append({"node": node, "cursor": "c" + str(idx)})
        has_next = (start + 2) < len(_things)
        end_cursor = edges[-1]["cursor"] if edges else "c-terminal"
        return jsonify(
            {
                "data": {
                    "pthings": {
                        "edges": edges,
                        "pageInfo": {"hasNextPage": has_next, "endCursor": end_cursor},
                    }
                }
            }
        )

    # ---- schema_driven_xml archetypes --------------------------------------

    @app.get("/xml/ec2/volumes")
    def xml_ec2_volumes():
        # `state` is single-word (snake alias == wire) so its value projects; the
        # multi-word `volumeId`/`cidrBlock` exercise snake column-NAME aliasing.
        body = (
            "<DescribeVolumesResponse>"
            "<requestId>req-ec2-1</requestId>"
            "<volumeSet>"
            "<item><volumeId>vol-1</volumeId><size>8</size>"
            "<encrypted>true</encrypted><state>available</state>"
            "<cidrBlock>10.0.0.0/24</cidrBlock></item>"
            "<item><volumeId>vol-2</volumeId><size>16</size>"
            "<encrypted>false</encrypted><state>in-use</state>"
            "<cidrBlock>10.0.1.0/24</cidrBlock></item>"
            "</volumeSet>"
            "</DescribeVolumesResponse>"
        )
        return Response(body, mimetype="text/xml")

    @app.get("/xml/ec2/volumes_alias")
    def xml_ec2_volumes_alias():
        # Wire elements use EC2 locationName casing (volumeId, attachmentSet);
        # the spec's schema keys are the botocore member names (VolumeId,
        # Attachments) with xml: name overrides - the AWS provider shape. The
        # nested <attachmentSet> content exercises JSON stringification of a
        # complex value under a string-typed column.
        body = (
            "<DescribeVolumesResponse>"
            "<requestId>req-ec2-alias-1</requestId>"
            "<volumeSet>"
            "<item><volumeId>vol-a1</volumeId><size>8</size><state>available</state>"
            "<attachmentSet><item><instanceId>i-1</instanceId>"
            "<device>/dev/sda1</device></item></attachmentSet></item>"
            "<item><volumeId>vol-a2</volumeId><size>16</size><state>in-use</state>"
            "<attachmentSet/></item>"
            "</volumeSet>"
            "</DescribeVolumesResponse>"
        )
        return Response(body, mimetype="text/xml")

    @app.get("/xml/ec2/vpc")
    def xml_ec2_vpc():
        # CreateVpc-style singleton: the row lives under a named wrapper member
        # one level below the response root (walker singleton-unwrap regime).
        body = (
            "<CreateVpcResponse>"
            "<requestId>req-ec2-vpc-1</requestId>"
            "<vpc><vpcId>vpc-fixture-1</vpcId>"
            "<cidrBlock>10.99.0.0/16</cidrBlock><state>pending</state></vpc>"
            "</CreateVpcResponse>"
        )
        return Response(body, mimetype="text/xml")

    @app.get("/xml/ec2/volumes_empty_body")
    def xml_ec2_volumes_empty_body():
        # S3 CreateBucket-style: 200 with an empty body. The walker must yield
        # zero rows rather than an mxj EOF error.
        return Response("", mimetype="text/xml")

    @app.get("/xml/ec2/volumes_paged")
    def xml_ec2_volumes_paged():
        # Two-page fixture: page 1 carries a <nextToken> scalar sibling that the
        # schema_driven_xml transform must pass through for pagination to
        # traverse; page 2 is terminal (no token).
        if request.args.get("NextToken") == "xmltok-2":
            body = (
                "<DescribeVolumesResponse>"
                "<requestId>req-ec2-paged-2</requestId>"
                "<volumeSet>"
                "<item><volumeId>vol-p3</volumeId><size>32</size>"
                "<encrypted>true</encrypted><state>available</state>"
                "<cidrBlock>10.7.2.0/24</cidrBlock></item>"
                "</volumeSet>"
                "</DescribeVolumesResponse>"
            )
        else:
            body = (
                "<DescribeVolumesResponse>"
                "<requestId>req-ec2-paged-1</requestId>"
                "<volumeSet>"
                "<item><volumeId>vol-p1</volumeId><size>8</size>"
                "<encrypted>true</encrypted><state>available</state>"
                "<cidrBlock>10.7.0.0/24</cidrBlock></item>"
                "<item><volumeId>vol-p2</volumeId><size>16</size>"
                "<encrypted>false</encrypted><state>in-use</state>"
                "<cidrBlock>10.7.1.0/24</cidrBlock></item>"
                "</volumeSet>"
                "<nextToken>xmltok-2</nextToken>"
                "</DescribeVolumesResponse>"
            )
        return Response(body, mimetype="text/xml")

    @app.get("/xml/query/stacks")
    def xml_query_stacks():
        # `id`/`region` are single-word (value projects); `stackName` is multi-word
        # (column-NAME aliasing only, value null under the known any-sdk gap).
        body = (
            "<DescribeStacksResponse><DescribeStacksResult><Stacks>"
            "<member><id>s1</id><region>us-east-1</region><stackName>prod</stackName></member>"
            "<member><id>s2</id><region>us-west-2</region><stackName>dev</stackName></member>"
            "</Stacks></DescribeStacksResult></DescribeStacksResponse>"
        )
        return Response(body, mimetype="text/xml")

    @app.get("/xml/query/stacks_empty")
    def xml_query_stacks_empty():
        body = (
            "<DescribeStacksResponse><DescribeStacksResult>"
            "<Stacks/>"
            "</DescribeStacksResult></DescribeStacksResponse>"
        )
        return Response(body, mimetype="text/xml")

    @app.get("/xml/restxml/hostedzone")
    def xml_restxml_hostedzone():
        # `id`/`name` are single-word (value projects); `callerReference` is
        # multi-word (column-NAME aliasing only, value null under the known gap).
        body = (
            "<GetHostedZoneResponse>"
            "<id>Z1</id>"
            "<name>example.com</name>"
            "<callerReference>ref-1</callerReference>"
            "</GetHostedZoneResponse>"
        )
        return Response(body, mimetype="text/xml")

    return app


app = create_app()


if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser()
    parser.add_argument("--port", type=int, default=1070)
    parser.add_argument("--host", default="0.0.0.0")
    args = parser.parse_args()
    app.run(host=args.host, port=args.port)
