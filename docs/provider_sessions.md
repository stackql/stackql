# Provider sessions, variables and secrets

Client sessions own their variables, runtime configuration and default in-memory secret store.
Independent SQL connections do not share mutations. MCP request contexts are isolated; configure
and query in one SQL batch when using MCP.

## Assignment and reset

The dollar prefix selects intrinsic runtime configuration. A plain name selects a scalar variable;
dots alone do not activate configuration. Both forms affect this session only.

```sql
SET region = 'ap-southeast-2';
SET minimum_count = 3;
SET enabled = true;
SET optional_value = null;
SET "providers.saved" = '{"provider":"aws","tenancy":"production"}';

SET "$.auth.example.password" = 'session-password';
UNSET "$.auth.example.password";
UNSET region;
```

`SET SESSION` and `UNSET SESSION` make the default scope explicit. Global/principal/group shared
storage is not implemented. Unsupported scopes fail, and CLI configuration errors exit nonzero.
`SET name = DEFAULT` is equivalent to `UNSET name`. Explicit null is a stored value, not a reset.
Assignments currently accept scalar literals or direct value references. Apply conversions and
other SQL expressions in expression positions such as SELECT and WHERE. The earlier
`variables.name` spelling remains an alias for a plain variable named `name`. SET lists apply in
order; they are not an RDBMS transaction across assignments or stores.

## Configure two AWS accounts

These aliases use separate environment-variable names without changing the server environment:

```sql
SET "$.providers.production" = '{
  "provider":"aws",
  "tenancy":"production-account",
  "auth":{
    "type":"aws_signing_v4",
    "access_key_id_env_var":"PRODUCTION_AWS_ACCESS_KEY_ID",
    "secret_access_key_env_var":"PRODUCTION_AWS_SECRET_ACCESS_KEY"
  },
  "params":{"region":"ap-southeast-2"},
  "tuning":{"max_concurrent":4,"max_per_host":4}
}';
SET "$.providers.development" = '{
  "provider":"aws",
  "tenancy":"development-account",
  "auth":{
    "type":"aws_signing_v4",
    "access_key_id_env_var":"DEVELOPMENT_AWS_ACCESS_KEY_ID",
    "secret_access_key_env_var":"DEVELOPMENT_AWS_SECRET_ACCESS_KEY"
  },
  "params":{"region":"ap-southeast-2"},
  "tuning":{"max_concurrent":2}
}';
SELECT * FROM production.s3.buckets;
SELECT * FROM development.s3.buckets;
```

Provider and tenancy are required. Tenancy is a grouping label, not an account identity verified
by StackQL. Auth must be explicit and does not borrow a canonical provider's credentials. Temporary
AWS credentials additionally need the appropriate session-token field/environment reference.
Environment references use the server process environment; there is no session-local environment overlay.

Independent server sessions can query concurrently with different bindings. `UNION ALL` branches
currently execute sequentially. Aliases sharing a provider/tenancy share HTTP concurrency controls
within a session and must declare identical tuning. SQL predicates override parameter defaults;
defaults are supplied only for declared provider parameters, including declared server parameters.

Metadata lookup resolves the canonical provider; execution retains the alias and detached auth
snapshot. Updating credentials affects subsequent queries, not queries already running. Structural
binding/tuning changes start a new governor generation. Aliases cover document-driven metadata,
reads, mutations and legacy EXEC; they do not extend native built-in or IaC resource addresses.

## Typed dereferencing and expressions

Use `$name` without single quotes, including for string variables. Resolution replaces AST nodes
with typed values before Omni planning, rather than interpolating SQL text. Missing values fail;
arrays and objects are not scalar variables. A JSON configuration document may be held as a string.

```sql
SET some_id = 42;
SET suffix = 'East';
SELECT CAST($some_id AS TEXT) AS id_text,
       CONVERT($some_id, TEXT) || '-' || lower($suffix) AS label,
       md5(lower(CONVERT($some_id, TEXT) || '-' || $suffix)) AS digest,
       '$some_id' AS literal_text;

-- Predicate form: WHERE id = CAST($some_id AS TEXT)
```

Use the predicate form with a real resource and its declared identifier column. `'$some_id'` is literally the text `$some_id`. Strings are escaped as SQL literals; numbers,
booleans and null retain their types. CAST and CONVERT use backend/catalogue type semantics. `md5`
returns a lowercase hexadecimal digest, propagates null, and is not a password/security primitive.
`||` concatenates; `OR` is boolean OR. Prepared statements resolve values at execution, not preparation.

A variable named `providers.saved` is separate from active provider configuration:

```sql
SET "providers.saved" = '{"provider":"aws","tenancy":"production-account","auth":{"type":"aws_signing_v4","access_key_id_env_var":"PRODUCTION_AWS_ACCESS_KEY_ID","secret_access_key_env_var":"PRODUCTION_AWS_SECRET_ACCESS_KEY"}}';
SET "$.providers.production" = "$providers.saved";
UNSET "providers.saved";
```

Deleting the saved variable does not delete the active binding. `UNSET "$.providers.production"`
removes the active binding. Variable names shadow equally named runtime paths for SQL dereferencing;
credential resolution still uses active runtime configuration.

## Object names

In relation positions, `$name` resolves to an identifier. In expression positions, use
`IDENTIFIER($name)` to request a column name rather than a scalar string. Identifier values are
converted to AST names and never spliced into SQL. Dotted relation values have up to four components.
Missing/non-string/empty identifiers and statement-injection values fail before provider access.

```sql
SET relation_name = 'production.s3.buckets';
SET column_name = 'name';
SET output_name = 'bucket';
SET row_count = 10;
SELECT IDENTIFIER($column_name) AS "$output_name"
FROM $relation_name
ORDER BY IDENTIFIER($column_name)
LIMIT $row_count;
```

Use column names actually exposed by your provider version. Qualified relation components may also
be variables, for example `FROM $provider_name.$service_name.$resource_name`. Double-quoted aliases
such as `AS "$output_name"` resolve to identifier values. DDL column names, INSERT column lists
and UPDATE targets also accept double-quoted dollar references; use `IDENTIFIER($name)` in
ordinary expression positions. Joins, CTEs and subqueries resolve their relation variables too. Variables are not arbitrary SQL fragments:
you cannot substitute keywords, a comma-separated column list or a whole WHERE clause.

## Default session-only secrets

No store configuration or persistence flag is needed for memory-only secrets:

```sql
SET "$.secrets.production" = '{"type":"aws_signing_v4","access_key_id_env_var":"PRODUCTION_AWS_ACCESS_KEY_ID","secret_access_key_env_var":"PRODUCTION_AWS_SECRET_ACCESS_KEY"}';
SET "$.providers.production" = '{"provider":"aws","tenancy":"production-account","auth_secret":"production","params":{"region":"ap-southeast-2"}}';
SELECT * FROM production.s3.buckets;
UNSET "$.secrets.production";
```

Values are literal strings, including JSON Auth documents. `auth_secret` resolves the named secret
into a detached query snapshot. Rotation affects later queries; deletion makes later uses fail.
A new client session receives a fresh memory store, not another session's store. Ending the session
releases its store with the session's state; Go memory reclamation is not a cryptographic zeroization
guarantee. `UNSET "$.secret_store"` discards the current store binding and starts empty session memory.

Normal credential use and variable/configuration SET statements do not persist secrets in the RDBMS.
The explicit disclosure opt-in below is an exception. Putting sensitive data into an ordinary variable
or table explicitly is the caller's responsibility.

## Optional persistent OS storage

Persistent storage is disabled by default. The operator must enable `--allow-persistent-secrets`;
SQL cannot enable that permission. Only an explicit OS backend is accepted. Flat-file stores,
file paths, and the implicit file default are rejected even with the flag enabled.

```sql
SET "$.secret_store" = '{"backend":"os","namespace":"stackql/provider-auth"}';
```

This uses macOS Keychain, Linux Secret Service (`secret-tool` and a working session service), or
Windows Credential Manager. Unavailable facilities fail without file fallback. The process's OS
identity owns access; namespaces are not per-StackQL-principal ACLs. OS-store security depends on
its deployment and permissions. Do not treat this opt-in as a multi-user vault.

Persistent values outlive the SQL session and are deleted explicitly with `UNSET "$.secrets.key"`.
Resetting the store binding does not delete previously persisted OS credentials. Robot tests mock
Unix OS command boundaries; native Windows exercises Credential Manager. Unix mock results do not
prove the security or availability of an actual desktop keyring.

## Explicit SQL disclosure and caller authorization

Secret SQL dereferencing is disabled by default. The trusted operator flag
`--allow-secret-dereference` permits references such as `"$secrets.production.credentials"`,
`"$auth.example.password"` and `"$providers.production.auth.credentials"`. The operator accepts that
explicitly disclosed values may enter SQL staging, results, logs and persisted query artifacts.
Normal provider credential use does not require this flag.

With caller access management enabled, variable reads require `session/variables:read`; writes and
resets require `session/config:write`. Per-variable/per-secret owner and group grants are not yet
implemented. Resolved SQL objects are checked before execution; variable substitution grants no
additional provider/backend permission. Existing credential reload remains a separate mechanism.

## Session document merge

```sql
SET "$.document_merge" = '{"patches":[{"service":"stackql_unstable_google.compute","merge":{"components":{"x-stackQL-resources":{"networks":{"methods":{"list":{"response":{"objectKey":"$.items"}}}}}}}}]}';
-- Alternatively, read the same configuration from a server-local file:
SET "$.document_merge" = '@/absolute/path/merge.json';
UNSET "$.document_merge";
```

Requires the document-driven provider/preview integration. Subsequent document descriptions, queries and IaC use the effective view; the base registry is unchanged. Each SET replaces the patch set against the base (include all desired fragments in order). Failed SET preserves the previous view; UNSET restores the base. New client sessions inherit the configured view but later changes are isolated.

The default merge cache is `<approot>/cache/document-views` (normally under `.stackql/`); `doc_cache.dir` overrides it. IaC ledger and run journals already default to `<approot>/iac`.

Keep the base immutable and the cache available while readers use it. `fresh` is rejected; use a new cache directory instead. File paths refer to the StackQL process's filesystem, not a remote SQL client's filesystem. JSON objects may also be supplied through session variables. Inline JSON follows SQL string escaping; use `@file` to avoid escaping embedded programs.
