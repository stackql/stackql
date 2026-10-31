# Opt-in caller access and IaC

Caller access is disabled by default. Enable it with
`--access.config '{"path":"/path/to/access.json"}'`. The source envelope requires a nonempty `path`
and rejects unknown fields and trailing JSON. The previous bare filename form remains accepted.

Start with the [four executable walkthroughs](examples/access-management/README.md).
They are run directly by Robot tests and need no cloud account.
Caller credentials are independent of provider credentials and SQL `AUTH` statements.

## Configuration

Select one authentication `type`: `jwt`, `bearer`, `mtls` or `local_os`. Omitted `type` means `jwt`
for compatibility. Configuration is strict JSON; unknown fields and fields belonging to another mode
are rejected. All modes use the same principal directory and policy rules.

JWT uses an explicit issuer, audience and pinned public keys:

```json
{
  "type": "jwt",
  "issuer": "https://identity.example",
  "audience": "stackql",
  "enforce_session_deadline": false,
  "max_session_seconds": 300,
  "keys": [
    {"kid": "primary", "algorithm": "EdDSA", "public_key_pem": "-----BEGIN PUBLIC KEY-----\n...\n-----END PUBLIC KEY-----\n"}
  ],
  "rules": [
    {"subject": "group:readers", "resource": "operation/demo/list/listItems", "action": "execute"},
    {"subject": "group:readers", "resource": "credential/demo", "action": "use"}
  ]
}
```

For the other modes, keep the same `rules` and optional deadline settings, but replace JWT's
`audience` and `keys` with the following mode-specific fields:

```json
{
  "type": "bearer",
  "issuer": "local-tokens",
  "tokens": [{"subject": "alice", "sha256": "<64 hexadecimal characters: SHA-256 of the token>"}],
  "rules": [{"subject": "group:readers", "resource": "host/sql/read", "action": "execute"}]
}
```

```json
{
  "type": "mtls",
  "issuer": "client-certificates",
  "client_ca_pem": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----\n",
  "certificate_identity": "uri_san",
  "rules": [{"subject": "group:readers", "resource": "host/sql/read", "action": "execute"}]
}
```

```json
{
  "type": "local_os",
  "issuer": "this-host",
  "rules": [{"subject": "group:readers", "resource": "host/sql/read", "action": "execute"}]
}
```

The identity subject is the configured bearer subject, a verified client URI/DNS SAN, or the effective
Unix UID / Windows SID. mTLS requires exactly one SAN of the configured kind (`uri_san` or `dns_san`);
no Common Name fallback is used. Local OS mode ignores `STACKQL_CALLER_TOKEN` and is local-only.
It obtains the process identity from the OS, including numeric container UIDs without passwd entries.
Neither OS groups nor token group claims replace directory membership. Static bearer digests must be
unique SHA-256 values; comparisons happen in the SDK. Basic username/password and OAuth/OIDC discovery
are not provided. An external issuer can issue JWTs, but its keys/issuer/audience must be pinned here.

JWT accepts EdDSA, RS256/384/512, PS256/384/512 and ES256/384/512. RSA keys require at least 2048 bits.

Use the real compiled method and operation identifiers from trusted provider documents. An operation
wildcard can grant mutation methods as well as reads. Keys are bound to their configured algorithms;
unknown key IDs, invalid signatures, missing/expired expiry and invalid temporal/issuer/audience
claims fail closed. Token-supplied URLs and groups are not used for trust or directory membership.
Configuration is loaded at startup; rotate policy or keys through a controlled restart.
Queries have no authentication-derived execution deadline by default. Credentials are verified and
directory membership is refreshed for each new statement; expired credentials cannot dispatch new
work, but expiry does not interrupt an already authorized query. Explicit caller timeouts and
cancellation still apply.

Set `"enforce_session_deadline": true` to opt into cancellation at `max_session_seconds` (default
300; maximum 86400), shortened by JWT or mTLS certificate expiry. `max_session_seconds` has no effect
on query execution while enforcement is disabled. This option does not enable token renewal.

## Principal directory

The configured SQLite or PostgreSQL backend stores:

- `stackql_access_principals`: ID and JSON object of string attributes.
- `stackql_access_identities`: verified `(issuer, subject)` to principal ID.
- `stackql_access_groups`: group IDs.
- `stackql_access_memberships`: principal/group bindings.

Schema creation is explicit: `accessmanagement.InitializeSchema(ctx, engine)` is an administration
API, not an authentication side effect. Provision tables, identities and memberships through a
trusted database administration path before serving protected queries. Unknown identities, absent
schema, malformed data and directory outages deny access. Ordinary protected SQL cannot address
directory tables. SQLite runtime connections enable foreign keys. Protect the database file and
use separate administrative/runtime database privileges where supported.

For PostgreSQL, create the four tables through your database administration connection, using the
schema shown in [the walkthrough script](examples/access-management/quickstart.py). Use a dedicated
schema/search path and provision identities before starting the runtime. For example:

```sql
INSERT INTO stackql_access_principals (id, attributes) VALUES ('alice', '{}');
INSERT INTO stackql_access_groups (id) VALUES ('readers');
INSERT INTO stackql_access_identities (issuer, subject, principal_id)
VALUES ('https://identity.example', 'alice', 'alice');
INSERT INTO stackql_access_memberships (principal_id, group_id) VALUES ('alice', 'readers');
```

Use the authentication mode's actual issuer/subject: for local OS this might be `('this-host', '1000')`,
and for mTLS `('client-certificates', 'spiffe://example/alice')`. Select the backend explicitly:

```sh
stackql --access.config '{"path":"/etc/stackql/access.json"}' \
  --sqlBackend '{"dbEngine":"postgres_tcp","dsn":"postgres://runtime@localhost/stackql?sslmode=require","sqlDialect":"postgres"}' \
  exec 'SELECT whoami(), groups();'
```

Runtime credentials need directory SELECT access; keep membership writes in the trusted administration
connection. Revocation is `DELETE FROM stackql_access_memberships WHERE principal_id = 'alice'` followed
by commit. The next statement resolves a new snapshot. SQLite and PostgreSQL Robot scenarios test this.

The directory produces an immutable per-statement snapshot. Omni receives that snapshot and an
authorizer; database connections, drivers and SQL remain in StackQL.

## Entry points and permissions

| Authentication | `exec`, shell, MCP stdio | HTTPS MCP |
| --- | --- | --- |
| JWT | `STACKQL_CALLER_TOKEN` | Request bearer header |
| Static bearer | `STACKQL_CALLER_TOKEN` | Request bearer header |
| Local OS | Effective process UID/SID; no credential | Explicitly refused |
| mTLS | Refused: no verified TLS connection | Verified client chain and unique configured SAN |

Supply local token credentials through the process environment, never a SQL literal or command-line
argument. Protected `exec` failures return exit status 1 after presenting the error.

HTTPS MCP obtains credentials from the actual request or TLS connection. Server TLS certificate/key
configuration is mandatory. mTLS additionally requires and verifies client certificates using the configured
client CA. The caller verifier replaces the legacy shared HTTP bearer gate when access is enabled.
Missing request credentials never fall back to the server's environment token.

Protected PostgreSQL-wire startup and MCP reverse proxy currently refuse explicitly. The current
wire dependency does not expose verified per-connection authentication evidence to the backend.
They must gain that integration before protection can be enabled on those transports. Unprotected
legacy deployments continue to operate as before. Provider SQL `AUTH`/`AUTH REVOKE` administration also refuses in protected mode; provision provider
credentials through trusted host configuration instead. Other standalone helper/registry commands refuse
when caller access is enabled, rather than bypassing the protected SQL path.

The `readers` directory group grants the host's safe-read SQL capability. SDK operations and provider
credential use still require explicit policy. Other statements require `execute` on
`host/sql/<statement-kind>`; reads outside the implicit reader group require `host/sql/read`.
Additional host checks include:

| Resource | Action | Scope |
| --- | --- | --- |
| `host/iac/run`, `host/iac/destroy` | `execute` | Effectful IaC relations, despite SELECT syntax |
| `host/sql/exec` | `execute` | Native EXEC, including nested EXEC |
| `host/function/<name>` | `execute` | Functions outside the small known-pure allowlist |
| `host/backend` | `read` | Opaque stored/external backend relations |
| `host/credentials` | `reload` | MCP credential reload administration |

SDK collection, operation, credential, relation and transport checks are additional requirements.
A collection grant does not grant its underlying methods. Stored query expansion is checked before
execution. Non-integrated legacy execution is refused while protection is enabled. Legacy plan and
row-cache reuse is disabled on protected paths.

Inspect the established snapshot using:

```sql
SELECT whoami() AS principal, groups() AS memberships;
```

## IaC surface

For a create, rerun, update and destroy sequence, see the [AWS VPC/subnet walkthrough](iac_walkthrough.md).

Use `stackql_preview.iac.run` and `.destroy` with the existing preview opt-in. Resource JSON uses
Omni's shared parser: `write_once` carries initial request inputs, `desired` carries maintained
state, and typed `input`, `env`, `self` and sibling bindings remain intact. Retired `params` and
`correlationParam` resource fields are rejected. Resource keys must be unique.

`inputs` and `environment` predicates accept JSON objects. JSON numbers retain their precision;
strings are literal values, not SQL or templates. Remaining predicates are provider execution scope.
Always supply explicit collection and cloud scope. The host-configured state root confines relative
`state` names, including checks against existing symlink escapes. Protected state is additionally
namespaced by the directory's `tenant` attribute, or by principal ID when no tenant is configured.
Only trusted directory administrators should assign tenant membership.

```sql
-- Explicit partial deletion. An empty selection is an error.
SELECT * FROM stackql_preview.iac.destroy
WHERE collection = 'deployment' AND state = 'production'
  AND keys = '["resource-key"]' AND run_id = 'prune-001';

-- Full teardown: omit keys.
SELECT * FROM stackql_preview.iac.destroy
WHERE collection = 'deployment' AND state = 'production' AND run_id = 'destroy-001';
```

Omit `run_id` to generate a unique ID automatically. Explicit IDs must be unique within the state
directory; reusing one is rejected before provider execution. Reapply the latest desired-state file:
returning to an older file can request removal of fields it no longer declares. After pruning, remove the deleted declarations before applying again; omission
alone does not delete, and reapplying the original declaration requests recreation. Omni blocks
selected deletion when retained resources depend on it. Failed/error/outstanding result rows become
SQL failures and CLI failure status, while diagnostic rows remain available. `applied` can mean no-op.
Cancellation and authenticated context propagate to SDK execution, completion polls and compensation;
compensation does not retry anonymously after access expires or fails.

`test/robot/functional/stackql_access.robot` contains separately named scenarios for each authentication
mode, transport, invalid JWT constraint, policy boundary and IaC lifecycle. It exercises all accepted JWT
signing algorithms and runs the published walkthrough script for all four modes. The fixture uses a local
API and the configured SQLite/PostgreSQL directory. Test tokens remain outside Robot arguments and artifacts.
