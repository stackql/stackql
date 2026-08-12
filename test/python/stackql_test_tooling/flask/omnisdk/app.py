"""Mock AWS + GCP endpoints (path-style) for omnicli robot tests.

AWS (from captured collateral):
  GET /                     -> ListBuckets   (jinja from collateral/buckets.json)
  GET /<bucket>?encryption  -> GetBucketEncryption (raw collateral/encryption/<bucket>.xml)
  GET /?Action=Create*      -> EC2 (jinja from collateral/ec2)

GCP (synthesized; async create → operation → poll DONE):
  POST /token                                   -> OAuth token
  POST /compute/v1/projects/<p>/global/networks -> Operation selfLink (op-net)
  POST /compute/v1/projects/<p>/regions/<r>/subnetworks -> Operation selfLink (op-subnet)
  GET  /compute/v1/op/<op>                      -> DONE operation (targetId/targetLink)

Run: PORT=8085 python test/mock/app.py  (or via the robot suite).
"""

import json
import time
import os
import pathlib

from flask import Flask, Response, jsonify, render_template, request

HERE = pathlib.Path(__file__).parent
COLL = HERE / "collateral"
XML = "application/xml"

app = Flask(__name__, template_folder=str(HERE / "templates"))
BUCKETS = json.loads((COLL / "buckets.json").read_text())
PROVISION = json.loads((COLL / "ec2" / "provision.json").read_text())

NOT_FOUND = (
    '<?xml version="1.0" encoding="UTF-8"?>'
    "<Error><Code>ServerSideEncryptionConfigurationNotFoundError</Code></Error>"
)


@app.get("/")
def list_buckets():
    # GET / is S3 ListBuckets.
    return Response(render_template("list_buckets.xml.j2", buckets=BUCKETS), mimetype=XML)


@app.post("/")
def ec2_query():
    # EC2 Query API: params ride the POST form body (Action, VpcId, ...).
    action = request.form.get("Action")
    if action == "CreateVpc":
        return Response(render_template("create_vpc.xml.j2", **PROVISION), mimetype=XML)
    if action == "CreateSubnet":
        # Enforce the β binding: CreateSubnet must carry the VPC id CreateVpc returned.
        if request.form.get("VpcId") != PROVISION["vpc_id"]:
            return Response("<Error><Code>InvalidVpcID.NotFound</Code></Error>", status=400, mimetype=XML)
        return Response(render_template("create_subnet.xml.j2", **PROVISION), mimetype=XML)
    return Response("<Error><Code>InvalidAction</Code></Error>", status=400, mimetype=XML)


# Local addition: an opt-in per-request delay, so a test can observe rows being
# emitted while the upstream is still producing. Zero by default, which is the
# upstream behaviour.
_DELAY_SECONDS = float(os.environ.get("OMNISDK_MOCK_DELAY_MS", "0")) / 1000.0


@app.get("/<bucket>")
def bucket_op(bucket):
    if _DELAY_SECONDS > 0:
        time.sleep(_DELAY_SECONDS)
    if "versioning" in request.args:
        return Response("<VersioningConfiguration><Status>Enabled</Status></VersioningConfiguration>", mimetype=XML)
    if "publicAccessBlock" in request.args:
        return Response(
            "<PublicAccessBlockConfiguration><RestrictPublicBuckets>true</RestrictPublicBuckets>"
            "</PublicAccessBlockConfiguration>", mimetype=XML)
    if "encryption" not in request.args:
        return Response("<Error><Code>NotImplemented</Code></Error>", status=501, mimetype=XML)
    path = COLL / "encryption" / f"{bucket}.xml"
    if not path.exists():
        return Response(NOT_FOUND, status=404, mimetype=XML)
    return Response(path.read_text(), mimetype=XML)


# --- GCP ---------------------------------------------------------------------


@app.post("/token")
def gcp_token():
    return jsonify(access_token="mock-access-token", expires_in=3600, token_type="Bearer")


@app.post("/compute/v1/projects/<project>/global/networks")
def gcp_create_network(project):
    base = request.host_url.rstrip("/")
    return jsonify(name="op-net", selfLink=f"{base}/compute/v1/op/op-net", status="PENDING")


@app.post("/compute/v1/projects/<project>/regions/<region>/subnetworks")
def gcp_create_subnet(project, region):
    base = request.host_url.rstrip("/")
    return jsonify(name="op-subnet", selfLink=f"{base}/compute/v1/op/op-subnet", status="PENDING")


@app.get("/compute/v1/op/<op>")
def gcp_operation(op):
    base = request.host_url.rstrip("/")
    if op == "op-net":
        return jsonify(name=op, status="DONE", targetId="1111",
                       targetLink=f"{base}/compute/v1/projects/mock-project/global/networks/mock-net")
    return jsonify(name=op, status="DONE", targetId="2222",
                   targetLink=f"{base}/compute/v1/projects/mock-project/regions/us-central1/subnetworks/mock-subnet")


# --- Azure -------------------------------------------------------------------
# client-credentials token; subscriptions (paginated via nextLink) → VNets per sub → subnets per VNet.


@app.post("/<tenant>/oauth2/v2.0/token")
def azure_token(tenant):
    return jsonify(access_token="mock-azure-token", expires_in=3600, token_type="Bearer")


def require_azure_token():
    """Reject a request whose bearer is absent or empty, exactly as real ARM does. Without this the
    mock answers 200 to an unauthenticated call, so a plan that binds an EMPTY {token} — e.g. a token
    exchange that yielded no access_token — passes here and only fails against the real cloud with
    `AuthenticationFailedMissingToken`. Returns an error response, or None when the token is good."""
    header = request.headers.get("Authorization", "")
    token = header[len("Bearer "):].strip() if header.startswith("Bearer ") else ""
    if not token:
        return jsonify(error={
            "code": "AuthenticationFailedMissingToken",
            "message": "Authentication failed. The 'Authorization' header is missing the access token.",
        }), 401
    return None


@app.get("/subscriptions")
def azure_subscriptions():
    if (err := require_azure_token()):
        return err
    base = request.host_url.rstrip("/")
    if request.args.get("_page") == "2":
        return jsonify(value=[{"subscriptionId": "sub-2"}])
    # page 1 carries a nextLink to page 2 (exercises httpx ContFollow).
    return jsonify(value=[{"subscriptionId": "sub-1"}],
                   nextLink=f"{base}/subscriptions?api-version=2020-01-01&_page=2")


@app.get("/subscriptions/<sub>/providers/Microsoft.Network/virtualNetworks")
def azure_vnets(sub):
    vid = f"/subscriptions/{sub}/resourceGroups/rg1/providers/Microsoft.Network/virtualNetworks/vnet-{sub}"
    return jsonify(value=[{"id": vid, "name": f"vnet-{sub}", "location": "eastus"}])


@app.get("/subscriptions/<sub>/resourceGroups/<rg>/providers/Microsoft.Network/virtualNetworks/<vnet>/subnets")
def azure_subnets(sub, rg, vnet):
    vid = f"/subscriptions/{sub}/resourceGroups/{rg}/providers/Microsoft.Network/virtualNetworks/{vnet}"
    return jsonify(value=[
        {"id": f"{vid}/subnets/sn-a", "name": "sn-a", "properties": {"addressPrefix": "10.0.0.0/24"}},
        {"id": f"{vid}/subnets/sn-b", "name": "sn-b", "properties": {"addressPrefix": "10.0.1.0/24"}},
    ])


@app.get("/subscriptions/<sub>/providers/Microsoft.Storage/storageAccounts")
def azure_storage_accounts(sub):
    if (err := require_azure_token()):
        return err
    return jsonify(value=[
        {"name": f"stor{sub}a", "properties": {
            "encryption": {"keySource": "Microsoft.Storage"},
            "allowBlobPublicAccess": False, "supportsHttpsTrafficOnly": True}},
        {"name": f"stor{sub}b", "properties": {
            "encryption": {"keySource": "Microsoft.Keyvault"},
            "allowBlobPublicAccess": True, "supportsHttpsTrafficOnly": True}},
    ])


# --- GCP Cloud Storage -------------------------------------------------------


# Cloud Resource Manager v3: the org → folder → project hierarchy the recursive descent walks.
# Tree: org 123456789 ─ folders/100 ─ folders/200 (2 deep), a project at each node — so a full
# descent finds proj-deep (under a nested folder) that a direct-children-only pass would miss.
_GCP_FOLDERS = {
    "organizations/123456789": ["folders/100"],
    "folders/100": ["folders/200"],
}
_GCP_PROJECTS = {
    "organizations/123456789": ["proj-root"],
    "folders/100": ["proj-100"],
    # folders/200 intentionally has NO project: an empty scope must not leak a ?project= call.
}


@app.get("/v3/folders")
def gcp_v3_folders():
    parent = request.args.get("parent", "")
    if "000000000" in parent:  # org whose hierarchy the caller can't list (grants at project, not org)
        return Response(
            '{"error":{"code":403,"message":"The caller does not have permission","status":"PERMISSION_DENIED"}}',
            status=403, mimetype="application/json")
    return jsonify(folders=[{"name": f} for f in _GCP_FOLDERS.get(parent, [])])


@app.get("/v3/projects")
def gcp_v3_projects():
    parent = request.args.get("parent", "")
    return jsonify(projects=[{"projectId": p} for p in _GCP_PROJECTS.get(parent, [])])


@app.get("/storage/v1/b")
def gcp_storage_buckets():
    return jsonify(items=[
        {"name": "gcs-plain", "iamConfiguration": {"publicAccessPrevention": "inherited"},
         "versioning": {"enabled": False}},
        {"name": "gcs-cmek", "encryption": {"defaultKmsKeyName": "projects/p/locations/l/keyRings/r/cryptoKeys/k"},
         "iamConfiguration": {"publicAccessPrevention": "enforced"}, "versioning": {"enabled": True}},
    ])


if __name__ == "__main__":
    app.run(host="127.0.0.1", port=int(os.environ.get("PORT", "8085")), threaded=True)
