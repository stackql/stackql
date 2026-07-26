# Server modes

The `mode` from `server_info` sets the write contract for the session. Every mode allows SELECT and metadata reads; they differ only on INSERT, UPDATE, REPLACE, DELETE and EXEC:

- `read_only` - all mutations and lifecycle ops are refused immediately. Do not attempt them; tell the user the server is read-only and what they would need to change.
- `safe` (the default) - each mutation or lifecycle op requires per-statement user approval: submitting the tool call triggers an approval prompt in the client (MCP elicitation), and a decline comes back as "user declined approval". Attempting the call is the correct way to request permission - state the intended change in conversation first so the user approves from an informed position. If the client does not support elicitation, the call is refused with guidance for the operator.
- `delete_safe` - INSERT, UPDATE and REPLACE proceed without prompting; DELETE and EXEC still require approval.
- `full_access` - everything proceeds without prompting. The protocol will not ask for you, so confirm destructive intent conversationally before issuing DELETE or EXEC.

The mode is global per server and operator-set; there is no per-tool override. A refusal is enforcement, not an error to engineer around (see error handling).

`pull_provider` and `reload_credentials` write only to the local provider cache and process environment respectively; they touch no cloud resource and are allowed in every mode.
