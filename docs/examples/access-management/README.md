# Try caller access locally

Run these commands **from the StackQL repository root**, in your terminal. Use the built binary
`./build/stackql`. Python 3 provisions the demonstration directory; JWT and mTLS also require OpenSSL.
No cloud account is needed and these examples make no cloud requests.

**`whoami()` and `groups()` require an authenticated StackQL session. They do not work in an ordinary
shell started without `--access.config`.** Start with the local OS example below.

## 1. Local OS identity: set up, then open an authenticated shell

Choose an empty directory. This command creates the principal database and access configuration,
maps your effective Unix UID or Windows SID to principal `reader`, and verifies a query:

```sh
python3 docs/examples/access-management/quickstart.py \
  --stackql ./build/stackql --mode local_os --directory /tmp/access-os
```

Expect the confirmation:

```text
Verified local_os: principal reader; directory group readers.
```

The setup creates `/tmp/access-os/access.json` and `/tmp/access-os/principals.db`.
**Use these actual paths in the commands below.** Paths such as `/etc/stackql/access.json` and
`/var/lib/stackql/principals.db` are deployment placeholders, not files created by this walkthrough.
They only work after you provision those locations yourself; the database's parent directory must
already exist and be accessible.

Now open a shell using that configuration and database:

```sh
./build/stackql \
  --access.config '{"path":"/tmp/access-os/access.json"}' \
  --sqlBackend '{"dsn":"/tmp/access-os/principals.db"}' \
  shell
```

**In this authenticated StackQL shell**, run:

```sql
SELECT whoami() AS principal, groups() AS memberships;
```

Expect one row: `principal` is `reader`, and `memberships` is `["readers"]`.
You can repeat the query or reopen the same configured shell. No token is required. Directory
memberships determine permissions; OS group membership does not grant access. HTTP callers cannot
use the server's OS identity.

The setup command refuses to overwrite a nonempty directory. If you already used `/tmp/access-os`,
reuse its configuration with the shell command above, or choose a new empty directory for setup
and use that path in both shell flags.

## 2. JWT: provision and verify a signed token

```sh
python3 docs/examples/access-management/quickstart.py \
  --stackql ./build/stackql --mode jwt --directory /tmp/access-jwt
```

Expect `Verified jwt: principal reader; directory group readers.`

The script creates a 2048-bit RSA signing key, pins its public key and `kid`, issues a five-minute RS256 JWT,
and supplies it through `STACKQL_CALLER_TOKEN` to one CLI query. Token group claims grant nothing;
the directory supplies `readers` membership. The token is not exported into your terminal, so opening
a separate shell with this configuration alone will not authenticate you. For deployment, obtain
JWTs from your issuer and configure its trusted public keys.

## 3. Static bearer token: provision and verify a token digest

```sh
python3 docs/examples/access-management/quickstart.py \
  --stackql ./build/stackql --mode bearer --directory /tmp/access-bearer
```

Expect `Verified bearer: principal reader; directory group readers.`

The script generates a random token, stores only its SHA-256 digest and subject in `access.json`,
and passes the token to one child CLI query through `STACKQL_CALLER_TOKEN`. The token is not exported
into your terminal or printed. For deployment, provision a token digest and supply its plaintext
credential through your secret-management environment. Changing configured digests requires a restart.

## 4. Mutual TLS: query over HTTPS with a client certificate

```sh
python3 docs/examples/access-management/quickstart.py \
  --stackql ./build/stackql --mode mtls --directory /tmp/access-mtls
```

Expect `Verified mtls: principal reader; directory group readers.`

The script creates a demonstration CA and server/client certificates, starts an HTTPS MCP server,
queries it with the client certificate while verifying the server CA, then stops the server.
The client URI SAN `spiffe://access-demo/reader` maps to principal `reader`. No bearer token is sent.
The server requires a verified client chain and exactly one configured identity SAN; Common Name
is not a fallback. For deployment, use your managed CA and certificates.

## What these examples leave behind

Each directory contains `access.json`, `backend.json` and `principals.db`. JWT/mTLS also create
private keys; mTLS writes `mcp.json`. These credentials are local demonstration credentials.
Use a separate empty directory for each setup command.

See the [configuration reference](../../access_management.md) for policy, principal-directory
provisioning, PostgreSQL and supported transports. For actual infrastructure deployment, see the
[AWS VPC/subnet walkthrough](../../iac_walkthrough.md).

Robot tests execute all four setup scripts. The local OS walkthrough test also starts the configured
shell and runs the identity query inside it. Docker tests use the tested container image; Python
runs on the test host and is not added to the standard image.
