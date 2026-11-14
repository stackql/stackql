"""Isolated STS AssumeRole + signed downstream fixture. No real AWS traffic."""
import datetime
import hashlib
import hmac
import re
import threading
import time
from urllib.parse import parse_qsl, quote
from xml.sax.saxutils import escape

from flask import Flask, Response, jsonify, request


class STSFixture:
    def __init__(self):
        self.lock = threading.Lock()
        self.requests = []
        self.issued = {}
        self.lifetime = 3600
        self.downstream_delay = 0
        self.failure = False
        self.expired = False
        self.block = threading.Event()
        self.block.set()
        self.started = threading.Event()
        self.base = ("BASEKEY", "BASESECRET", "BASETOKEN")
        self.role = "arn:aws:iam::123456789012:role/fixture"
        self.external = "fixture-external"
        self.session = "fixture-session"
        self.sessions = {self.session}
        self.region = "us-east-1"

    @staticmethod
    def verify_signature(req, expected, region, service):
        header = req.headers.get("Authorization", "")
        match = re.fullmatch(r"AWS4-HMAC-SHA256 Credential=([^/]+)/([^,]+), SignedHeaders=([^,]+), Signature=([0-9a-f]{64})", header)
        assert match, "missing SigV4 authorization"
        key, scope, signed, signature = match.groups()
        assert key == expected[0] and scope.split("/")[1:] == [region, service, "aws4_request"], "wrong signing principal/scope"
        assert req.headers.get("X-Amz-Security-Token", "") == expected[2], "wrong session token"
        assert "x-amz-security-token" in signed.split(";"), "session token was not signed"
        headers = "".join(name + ":" + req.headers[name].strip() + "\n" for name in signed.split(";"))
        query = "&".join(quote(k, safe="-_.~") + "=" + quote(v, safe="-_.~") for k, v in sorted(parse_qsl(req.query_string.decode(), keep_blank_values=True)))
        canonical = "\n".join([req.method, quote(req.path, safe="/-_.~"), query, headers, signed, hashlib.sha256(req.get_data()).hexdigest()])
        stamp = req.headers["X-Amz-Date"]
        text = "\n".join(["AWS4-HMAC-SHA256", stamp, scope, hashlib.sha256(canonical.encode()).hexdigest()])
        secret = ("AWS4" + expected[1]).encode()
        for value in [scope.split("/")[0], region, service, "aws4_request"]:
            secret = hmac.new(secret, value.encode(), hashlib.sha256).digest()
        assert hmac.compare_digest(hmac.new(secret, text.encode(), hashlib.sha256).hexdigest(), signature), "signature does not match exact request"


def create_app(state=None):
    state = state or STSFixture()
    app = Flask(__name__)
    app.config["STS_FIXTURE"] = state

    @app.post("/sts")
    def assume_role():
        state.started.set()
        try:
            STSFixture.verify_signature(request, state.base, state.region, "sts")
            form = request.form
            assert form["Action"] == "AssumeRole" and form["Version"] == "2011-06-15"
            assert form["RoleArn"] == state.role and form["ExternalId"] == state.external
            assert form["RoleSessionName"] in state.sessions and form["DurationSeconds"] == "900"
        except (AssertionError, KeyError) as exc:
            return Response(str(exc), status=400)
        state.block.wait(10)
        with state.lock:
            state.requests.append({"action": "AssumeRole", "role": form["RoleArn"], "external_id": form["ExternalId"], "session_name": form["RoleSessionName"]})
            if state.failure:
                return Response("<ErrorResponse><Error><Code>AccessDenied</Code><Message>fixture-external BASESECRET</Message></Error></ErrorResponse>", status=403)
            generation = len(state.issued) + 1
            credentials = (f"ROLEKEY{generation}", f"ROLESECRET{generation}", f"ROLETOKEN{generation}")
            state.issued[credentials[0]] = credentials
            expiration = datetime.datetime.now(datetime.timezone.utc) + datetime.timedelta(seconds=-1 if state.expired else state.lifetime)
        xml = "<AssumeRoleResponse><AssumeRoleResult><Credentials><AccessKeyId>{}</AccessKeyId><SecretAccessKey>{}</SecretAccessKey><SessionToken>{}</SessionToken><Expiration>{}</Expiration></Credentials></AssumeRoleResult></AssumeRoleResponse>"
        return Response(xml.format(*[escape(value) for value in credentials], expiration.isoformat().replace("+00:00", "Z")), mimetype="application/xml")

    @app.get("/users")
    def users():
        match = re.search(r"Credential=([^/]+)/", request.headers.get("Authorization", ""))
        key = match.group(1) if match else ""
        with state.lock:
            credentials = state.issued.get(key)
        try:
            assert credentials, "downstream request did not use assumed credentials"
            STSFixture.verify_signature(request, credentials, "ap-southeast-2", "iam")
        except AssertionError as exc:
            return Response(str(exc), status=403)
        with state.lock:
            state.requests.append({"action": "ListUsers", "key": key})
        if state.downstream_delay and not request.args.get("previous"):
            time.sleep(state.downstream_delay)
        return jsonify([{"id": "role-user"}])

    return app
