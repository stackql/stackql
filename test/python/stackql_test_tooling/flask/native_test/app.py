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

import re

from flask import Flask, Response, jsonify, request


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
        # Honour $select server-side, as real OData services do: fields absent
        # from $select are stripped from the response. This makes issue #682
        # observable - if the pushed $select omits a WHERE/ORDER BY column, the
        # client-side re-filter sees an absent column and drops every row.
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
        # page_number strategy: the response reports the current page and the
        # total page count; the reader requests page N+1 until page == total.
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
        # Negative case: no total_pages terminator in the response. The reader
        # must stop after the first page (missing terminator == terminate),
        # never loop forever.
        page_raw = request.args.get("page", "1")
        page = int(page_raw) if page_raw.isdigit() else 1
        window = _paged_items[(page - 1) * 2:(page - 1) * 2 + 2]
        rows = [dict(item, wire_page=page) for item in window]
        return jsonify({"items": rows, "result_info": {"page": page}})

    # ---- GraphQL pluggable cursor strategies (issue 684) --------------------

    @app.post("/graphql/keyset")
    def graphql_keyset():
        # keyset: the client injects a filter comparator on the last row's
        # sort key (rankGt: N); termination is an empty row array.
        body = request.get_json(silent=True) or {}
        query = body.get("query", "")
        m = re.search(r"rankGt:\s*(\d+)", query)
        after_rank = int(m.group(1)) if m else 0
        window = [t for t in _things if t["rank"] > after_rank][:2]
        nodes = [dict(t, wire_rank_gt=after_rank) for t in window]
        return jsonify({"data": {"kthings": {"nodes": nodes}}})

    @app.post("/graphql/offset")
    def graphql_offset():
        # offset: the client synthesises a running row count (offset: N);
        # termination is an empty row array.
        body = request.get_json(silent=True) or {}
        query = body.get("query", "")
        m = re.search(r"offset:\s*(\d+)", query)
        offset = int(m.group(1)) if m else 0
        window = _things[offset:offset + 2]
        nodes = [dict(t, wire_offset=offset) for t in window]
        return jsonify({"data": {"othings": {"nodes": nodes}}})

    @app.post("/graphql/pageinfo")
    def graphql_pageinfo():
        # page_info: Relay-strict - endCursor stays NON-EMPTY on the final
        # page, so only pageInfo.hasNextPage may terminate the loop. A
        # cursor-emptiness reader would loop forever here.
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
