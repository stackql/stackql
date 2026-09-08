#!/usr/bin/env python3
"""
check-published.py - read-only assurance that a stackql release has its MCP
server published to every distribution target. Never publishes, never
mutates: every probe is an anonymous HTTPS GET/HEAD against a public API.

Checks (V = the release version, defaulting to the latest GitHub release):

  release       stackql/stackql release vV carries the 5 .mcpb + 5 .sha256 assets
  proxy         releases.stackql.io/stackql/V/<bundle> serves the 4 platform bundles
                (what the npm/pypi/SDK wrappers download at runtime)
  oci           Docker Hub stackql/stackql-mcp:V exists with linux/amd64 + linux/arm64;
                :latest points at the same manifest digest
  npm           @stackql/mcp-server@V exists; dist-tag latest == V
  pypi          stackql-mcp-server V exists; latest == V
  cargo         crates.io stackql-mcp V exists and is not yanked; newest == V
  go            stackql/stackql-mcp-go has tag vV; proxy.golang.org resolves vV and @latest
  dotnet        NuGet StackQL.Mcp + StackQL.Mcp.AgentFramework V exist and are the highest
  mcp-registry  Official MCP Registry io.github.stackql/stackql-mcp V exists, is active
                and isLatest, its packages match registry/server.template.json rendered
                for V, and the mcpb fileSha256 pins equal the release .sha256 assets
  github-mcp    GitHub MCP Registry (api.mcp.github.com) lists the server at V. It syncs
                from the Official MCP Registry on an hourly sweep and its read API caches
                the listing for ~20 min, so this row can trail a registry publish by up
                to 2 hours (measured 2026-09-08: published 02:54Z, visible 04:0xZ)

Usage:
  scripts/check-published.py                  # latest stackql release
  scripts/check-published.py --version X.Y.Z
  scripts/check-published.py --json           # machine-readable report

GH_TOKEN / GITHUB_TOKEN, when set, authenticate the api.github.com calls
(3 per run) to avoid the anonymous rate limit; nothing else needs credentials.

Exit status: 0 all checks pass, 1 at least one check failed, 2 usage or
version-resolution error.
"""
import argparse
import json
import os
import re
import sys
import urllib.error
import urllib.parse
import urllib.request

ROOT_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
SERVER_TEMPLATE = os.path.join(ROOT_DIR, "registry", "server.template.json")

UA = "stackql-mcp-packaging-check"
TIMEOUT = 30

STACKQL_REPO = "stackql/stackql"
GO_MIRROR_REPO = "stackql/stackql-mcp-go"
RELEASE_BASE = "https://github.com/stackql/stackql/releases/download"
PROXY_BASE = "https://releases.stackql.io/stackql"
OCI_IMAGE = "stackql/stackql-mcp"
NPM_PACKAGE = "@stackql/mcp-server"
PYPI_PACKAGE = "stackql-mcp-server"
CRATE = "stackql-mcp"
GO_MODULE = "github.com/stackql/stackql-mcp-go"
NUGET_PACKAGES = ("StackQL.Mcp", "StackQL.Mcp.AgentFramework")
MCP_SERVER_NAME = "io.github.stackql/stackql-mcp"
MCP_REGISTRY = "https://registry.modelcontextprotocol.io/v0"
GITHUB_MCP_REGISTRY = "https://api.mcp.github.com/v0"

PLATFORM_TARGETS = ("linux-x64", "linux-arm64", "windows-x64", "darwin-universal")
BUNDLE_TARGETS = PLATFORM_TARGETS + ("multiplatform",)


class Report:
    def __init__(self):
        self.rows = []

    def add(self, venue, ok, detail):
        self.rows.append({"venue": venue, "ok": bool(ok), "detail": detail})

    @property
    def failed(self):
        return [r for r in self.rows if not r["ok"]]


# --- HTTP helpers -------------------------------------------------------------

def _headers(url):
    h = {"User-Agent": UA, "Accept": "application/json"}
    if url.startswith("https://api.github.com/"):
        token = os.environ.get("GH_TOKEN") or os.environ.get("GITHUB_TOKEN")
        if token:
            h["Authorization"] = "Bearer " + token
    return h


def fetch(url, method="GET"):
    """Return (status, body_bytes). HTTP errors are returned, not raised;
    transport errors surface as status 0 with the error text as body."""
    req = urllib.request.Request(url, headers=_headers(url), method=method)
    try:
        with urllib.request.urlopen(req, timeout=TIMEOUT) as resp:
            return resp.status, (b"" if method == "HEAD" else resp.read())
    except urllib.error.HTTPError as e:
        return e.code, e.read() if method != "HEAD" else b""
    except (urllib.error.URLError, OSError) as e:
        return 0, str(e).encode()


def fetch_json(url):
    status, body = fetch(url)
    if status != 200:
        return status, None
    try:
        return status, json.loads(body.decode("utf-8"))
    except ValueError:
        return status, None


def describe(status, body=b""):
    if status == 0:
        return "unreachable (%s)" % body.decode(errors="replace").strip()
    return "HTTP %d" % status


def version_key(v):
    """Sort key for dotted versions with optional pre-release suffixes;
    numeric parts compare numerically, non-numeric parts lexically after."""
    parts = re.split(r"[.\-+]", v.lstrip("v"))
    return tuple((0, int(p)) if p.isdigit() else (1, p) for p in parts)


# --- version resolution -------------------------------------------------------

def resolve_latest_version():
    status, data = fetch_json("https://api.github.com/repos/%s/releases/latest" % STACKQL_REPO)
    if status != 200 or not data or not data.get("tag_name"):
        sys.exit("error: could not resolve the latest %s release (%s)" % (STACKQL_REPO, describe(status)))
    return data["tag_name"].lstrip("v")


# --- checks -------------------------------------------------------------------

def check_release(version, report):
    """The GitHub release must carry every bundle and its checksum file.
    Returns the sha256 pins read from the release for the registry check."""
    url = "https://api.github.com/repos/%s/releases/tags/v%s" % (STACKQL_REPO, version)
    status, data = fetch_json(url)
    if status != 200 or not data:
        report.add("release", False, "%s release v%s not found (%s)" % (STACKQL_REPO, version, describe(status)))
        return {}
    names = {a["name"] for a in data.get("assets", [])}
    expected = []
    for t in BUNDLE_TARGETS:
        expected += ["stackql-mcp-%s.mcpb" % t, "stackql-mcp-%s.mcpb.sha256" % t]
    missing = [n for n in expected if n not in names]
    if missing:
        report.add("release", False, "v%s is missing %d/%d mcpb assets: %s (mcp-packaging dispatch)"
                   % (version, len(missing), len(expected), ", ".join(missing)))
    else:
        report.add("release", True, "%s v%s carries all %d mcpb assets" % (STACKQL_REPO, version, len(expected)))

    pins = {}
    for t in PLATFORM_TARGETS:
        name = "stackql-mcp-%s.mcpb.sha256" % t
        if name not in names:
            continue
        s, body = fetch("%s/v%s/%s" % (RELEASE_BASE, version, name))
        if s == 200:
            pins[t] = body.decode(errors="replace").split()[0].lower()
    return pins


def check_proxy(version, report):
    """The wrappers download from the releases.stackql.io front door, which
    only serves a version once its tag is pushed to stackql/releases.stackql.io."""
    bad = []
    for t in PLATFORM_TARGETS:
        status, body = fetch("%s/%s/stackql-mcp-%s.mcpb" % (PROXY_BASE, version, t), method="HEAD")
        if status != 200:
            bad.append("%s (%s)" % (t, describe(status, body)))
    if bad:
        report.add("proxy", False, "%s/%s not serving: %s (push tag v%s to stackql/releases.stackql.io)"
                   % (PROXY_BASE, version, "; ".join(bad), version))
    else:
        report.add("proxy", True, "%s/%s serves all %d platform bundles" % (PROXY_BASE, version, len(PLATFORM_TARGETS)))


def check_oci(version, report):
    base = "https://hub.docker.com/v2/repositories/%s/tags" % OCI_IMAGE
    status, tag = fetch_json("%s/%s" % (base, version))
    if status != 200 or not tag:
        report.add("oci", False, "docker.io/%s:%s not on Docker Hub (%s) (mcp-packaging oci-publish job)"
                   % (OCI_IMAGE, version, describe(status)))
        return
    archs = sorted("%s/%s" % (i.get("os"), i.get("architecture")) for i in tag.get("images", []))
    want = ["linux/amd64", "linux/arm64"]
    missing = [a for a in want if a not in archs]
    if missing:
        report.add("oci", False, "docker.io/%s:%s lacks %s (has %s)" % (OCI_IMAGE, version, ", ".join(missing), ", ".join(archs)))
    else:
        report.add("oci", True, "docker.io/%s:%s present (%s)" % (OCI_IMAGE, version, ", ".join(archs)))
    status, latest = fetch_json("%s/latest" % base)
    if status != 200 or not latest:
        report.add("oci", False, "docker.io/%s:latest not resolvable (%s)" % (OCI_IMAGE, describe(status)))
    elif latest.get("digest") != tag.get("digest"):
        report.add("oci", False, "docker.io/%s:latest digest differs from :%s" % (OCI_IMAGE, version))
    else:
        report.add("oci", True, "docker.io/%s:latest matches :%s" % (OCI_IMAGE, version))


def check_npm(version, report):
    status, _ = fetch_json("https://registry.npmjs.org/%s/%s" % (NPM_PACKAGE, version))
    if status != 200:
        report.add("npm", False, "%s@%s not published (%s) (release doc step 7a)" % (NPM_PACKAGE, version, describe(status)))
        return
    report.add("npm", True, "%s@%s published" % (NPM_PACKAGE, version))
    status, meta = fetch_json("https://registry.npmjs.org/%s" % NPM_PACKAGE)
    latest = (meta or {}).get("dist-tags", {}).get("latest")
    report.add("npm", latest == version, "%s dist-tag latest is %s" % (NPM_PACKAGE, latest or "unknown"))


def check_pypi(version, report):
    status, _ = fetch_json("https://pypi.org/pypi/%s/%s/json" % (PYPI_PACKAGE, version))
    if status != 200:
        report.add("pypi", False, "%s %s not published (%s) (release doc step 7b)" % (PYPI_PACKAGE, version, describe(status)))
        return
    report.add("pypi", True, "%s %s published" % (PYPI_PACKAGE, version))
    status, meta = fetch_json("https://pypi.org/pypi/%s/json" % PYPI_PACKAGE)
    latest = ((meta or {}).get("info") or {}).get("version")
    report.add("pypi", latest == version, "%s latest is %s" % (PYPI_PACKAGE, latest or "unknown"))


def check_cargo(version, report):
    status, data = fetch_json("https://crates.io/api/v1/crates/%s/%s" % (CRATE, version))
    if status != 200 or not data:
        report.add("cargo", False, "crates.io %s %s not published (%s) (release doc step 7d)" % (CRATE, version, describe(status)))
        return
    yanked = (data.get("version") or {}).get("yanked", False)
    report.add("cargo", not yanked, "crates.io %s %s %s" % (CRATE, version, "is yanked" if yanked else "published"))
    status, meta = fetch_json("https://crates.io/api/v1/crates/%s" % CRATE)
    newest = ((meta or {}).get("crate") or {}).get("newest_version")
    report.add("cargo", newest == version, "crates.io %s newest is %s" % (CRATE, newest or "unknown"))


def check_go(version, report):
    tag = "v" + version
    status, _ = fetch_json("https://api.github.com/repos/%s/git/ref/tags/%s" % (GO_MIRROR_REPO, tag))
    if status != 200:
        report.add("go", False, "%s has no tag %s (%s) (release doc step 7e)" % (GO_MIRROR_REPO, tag, describe(status)))
    else:
        report.add("go", True, "%s tagged %s" % (GO_MIRROR_REPO, tag))
    status, info = fetch_json("https://proxy.golang.org/%s/@v/%s.info" % (GO_MODULE, tag))
    if status != 200 or not info:
        report.add("go", False, "proxy.golang.org cannot resolve %s@%s (%s)" % (GO_MODULE, tag, describe(status)))
    else:
        report.add("go", True, "proxy.golang.org resolves %s@%s" % (GO_MODULE, tag))
    status, latest = fetch_json("https://proxy.golang.org/%s/@latest" % GO_MODULE)
    got = (latest or {}).get("Version")
    report.add("go", got == tag, "proxy.golang.org %s@latest is %s" % (GO_MODULE, got or "unknown"))


def check_dotnet(version, report):
    for pkg in NUGET_PACKAGES:
        lower = pkg.lower()
        status, index = fetch_json("https://api.nuget.org/v3-flatcontainer/%s/index.json" % lower)
        versions = (index or {}).get("versions") or []
        if status != 200 or version not in versions:
            have = ", ".join(versions) if versions else "none"
            why = describe(status) if status != 200 else "have: " + have
            report.add("dotnet", False, "NuGet %s %s not published (%s) (release doc step 7f)" % (pkg, version, why))
            continue
        highest = max(versions, key=version_key)
        if highest != version:
            report.add("dotnet", False, "NuGet %s %s published but %s is higher" % (pkg, version, highest))
        else:
            report.add("dotnet", True, "NuGet %s %s published and highest" % (pkg, version))


def expected_registry_packages(version):
    """(registryType, identifier, version) triples from server.template.json rendered for V."""
    with open(SERVER_TEMPLATE, encoding="utf-8") as f:
        rendered = json.loads(f.read().replace("__VERSION__", version))
    return {(p.get("registryType"), p.get("identifier"), p.get("version")) for p in rendered.get("packages", [])}


def check_mcp_registry(version, pins, report):
    name = urllib.parse.quote(MCP_SERVER_NAME, safe="")
    status, data = fetch_json("%s/servers/%s/versions/%s" % (MCP_REGISTRY, name, version))
    if status != 200 or not data:
        report.add("mcp-registry", False, "Official MCP Registry has no %s %s (%s) (release doc step 7c)"
                   % (MCP_SERVER_NAME, version, describe(status)))
        return
    server = data.get("server", data)
    meta = (data.get("_meta") or {}).get("io.modelcontextprotocol.registry/official") or {}
    active = meta.get("status") == "active"
    is_latest = meta.get("isLatest") is True
    report.add("mcp-registry", active, "Official MCP Registry %s %s status %s" % (MCP_SERVER_NAME, version, meta.get("status", "unknown")))
    report.add("mcp-registry", is_latest, "Official MCP Registry %s %s isLatest=%s" % (MCP_SERVER_NAME, version, meta.get("isLatest")))

    got = {(p.get("registryType"), p.get("identifier"), p.get("version")) for p in server.get("packages", [])}
    try:
        want = expected_registry_packages(version)
    except (OSError, ValueError) as e:
        report.add("mcp-registry", False, "cannot read %s: %s" % (SERVER_TEMPLATE, e))
        want = None
    if want is not None:
        missing = sorted(want - got, key=str)
        extra = sorted(got - want, key=str)
        if missing or extra:
            parts = []
            if missing:
                parts.append("missing " + ", ".join("%s %s" % (t, i) for t, i, _ in missing))
            if extra:
                parts.append("unexpected " + ", ".join("%s %s" % (t, i) for t, i, _ in extra))
            report.add("mcp-registry", False, "registry packages differ from server.template.json for %s: %s" % (version, "; ".join(parts)))
        else:
            report.add("mcp-registry", True, "registry lists all %d packages from server.template.json at %s" % (len(want), version))

    mismatched = []
    for p in server.get("packages", []):
        if p.get("registryType") != "mcpb":
            continue
        m = re.search(r"stackql-mcp-([a-z0-9-]+)\.mcpb$", p.get("identifier", ""))
        target = m.group(1) if m else None
        expected = pins.get(target)
        if not expected:
            mismatched.append("%s (no release .sha256 to compare)" % (target or p.get("identifier")))
        elif (p.get("fileSha256") or "").lower() != expected:
            mismatched.append(target)
    if mismatched:
        report.add("mcp-registry", False, "registry mcpb sha256 pins do not match the release .sha256 assets: %s" % ", ".join(mismatched))
    else:
        report.add("mcp-registry", True, "registry mcpb sha256 pins match the release .sha256 assets")


def check_github_mcp_registry(version, report):
    """api.mcp.github.com ignores ?search= and exposes only a paged listing,
    so walk it (100 per page) until the server name turns up."""
    entry = None
    cursor = None
    for _ in range(50):
        url = "%s/servers?limit=100" % GITHUB_MCP_REGISTRY
        if cursor:
            url += "&cursor=" + urllib.parse.quote(cursor, safe="")
        status, data = fetch_json(url)
        if status != 200 or not data:
            report.add("github-mcp", False, "GitHub MCP Registry listing failed (%s)" % describe(status))
            return
        for s in data.get("servers", []):
            server = s.get("server", s)
            if server.get("name") == MCP_SERVER_NAME:
                entry = server
                break
        if entry:
            break
        cursor = (data.get("metadata") or {}).get("next_cursor")
        if not cursor:
            break
    if not entry:
        report.add("github-mcp", False, "GitHub MCP Registry does not list %s (ingests from the Official MCP Registry)" % MCP_SERVER_NAME)
        return
    got = (entry.get("version_detail") or {}).get("version")
    if got == version:
        report.add("github-mcp", True, "GitHub MCP Registry lists %s at %s" % (MCP_SERVER_NAME, version))
    else:
        report.add("github-mcp", False, "GitHub MCP Registry lists %s at %s, expected %s (syncs from the Official MCP Registry hourly, plus read-side caching: allow up to 2 hours after mcp-registry-publish)"
                   % (MCP_SERVER_NAME, got or "unknown", version))


# --- main ---------------------------------------------------------------------

def main():
    ap = argparse.ArgumentParser(description="Read-only check that a stackql release's MCP server is published everywhere.")
    ap.add_argument("--version", help="stackql release version without the leading v (default: latest GitHub release)")
    ap.add_argument("--json", action="store_true", help="emit a JSON report instead of the table")
    args = ap.parse_args()

    resolved_latest = args.version is None
    version = args.version.lstrip("v") if args.version else resolve_latest_version()

    report = Report()
    pins = check_release(version, report)
    check_proxy(version, report)
    check_oci(version, report)
    check_npm(version, report)
    check_pypi(version, report)
    check_cargo(version, report)
    check_go(version, report)
    check_dotnet(version, report)
    check_mcp_registry(version, pins, report)
    check_github_mcp_registry(version, report)

    failed = report.failed
    if args.json:
        json.dump({"version": version, "resolved_latest": resolved_latest, "ok": not failed,
                   "checks": report.rows}, sys.stdout, indent=2)
        sys.stdout.write("\n")
    else:
        print("stackql MCP server publication check - v%s%s" % (version, " (latest release)" if resolved_latest else ""))
        print()
        for r in report.rows:
            print("%s  %-13s %s" % ("PASS" if r["ok"] else "FAIL", r["venue"], r["detail"]))
        print()
        print("%d checks: %d pass, %d fail" % (len(report.rows), len(report.rows) - len(failed), len(failed)))
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
