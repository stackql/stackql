---
name: attach_policy_document
title: Attach Policy Document
open_world: false
---
Attach one policy or compliance document as text so it can be read back as an MCP resource: an OPA/Rego policy, a compliance control set or a GRC document (.rego, .yaml, .yml, .json, .md, .txt or .csv).
Send the file inline in `document` as an MCP FileInput: `id`, `mimeType` and `size` are required, `data` carries the bytes base64-encoded, and `name` and a sha-256 `digest` are optional. Add an optional `label` to name the document.

Limits: one UTF-8 text file per call, at most the server's inline limit (the `maxSize` on `document`, 1 MiB by default); binary and empty files are rejected. Returns a resource_link to stackql://policies/<id>. Documents are held in server memory only and do not survive a restart. Writes no cloud or local state.
