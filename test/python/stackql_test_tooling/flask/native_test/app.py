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
