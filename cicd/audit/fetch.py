#!/usr/bin/env python3
"""Download the pinned audit engine (run.sh + queries + scripts) from
stackql/stackql-audit-action into /tmp/engine. Stdlib only — no curl/git needed.
Pinned by AUDIT_ENGINE_REF; subdir/repo overridable via env."""
import io
import os
import tarfile
import urllib.request

repo = os.environ.get("AUDIT_ENGINE_REPO", "stackql/stackql-audit-action")
ref = os.environ.get("AUDIT_ENGINE_REF", "main")
sub = os.environ.get("AUDIT_ENGINE_SUBDIR", "cicd/audit/quickstart").strip("/") + "/"

url = f"https://codeload.github.com/{repo}/tar.gz/{ref}"
req = urllib.request.Request(url, headers={"User-Agent": "curl/8"})  # github 403s default urllib UA
data = urllib.request.urlopen(req).read()

prefix = f"{repo.split('/')[-1]}-{ref}/{sub}"
with tarfile.open(fileobj=io.BytesIO(data), mode="r:gz") as t:
    for m in t.getmembers():
        if m.name.startswith(prefix):
            m.name = m.name[len(prefix):]
            if m.name:
                t.extract(m, "/tmp/engine", filter="data")
print(f"fetched {repo}@{ref} ({sub}) -> /tmp/engine")
