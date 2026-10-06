# SHOW DEPENDENCIES

```sql
SHOW DEPENDENCIES;
SHOW EXTENDED DEPENDENCIES;
SHOW DEPENDENCIES LIKE '%stackql%';
SHOW EXTENDED DEPENDENCIES LIKE '%vitess%';
```

These commands return a build-specific, offline inventory of the Go modules
contributing packages to the StackQL executable. The inventory excludes CI tools,
test-only modules, installed provider definitions, separate wrappers, container
images and operating-system packages. This is binary supply-chain attribution,
not a complete deployment SBOM or a legal-compliance determination.

## Columns

The default columns, in order, are `name`, `version`, `license`, `description`.
`EXTENDED` appends `source_url`, `license_url`, `purl`, `checksum`,
`original_name`, `original_version`.

Rows are sorted by effective module name, then exact version. Go pseudo-versions
and prerelease versions are preserved. Replacements use the replacement's name,
version, checksum and attribution; the originally requested identity appears in
`original_name` and `original_version`. Those two columns are SQL `NULL` when no
replacement applies.

Unavailable metadata is SQL `NULL`, except unidentified licences are `UNKNOWN`.
Local replacements have no fabricated version, URL, Package URL or checksum.
Checksums retain Go's `h1:` prefix: these are module checksums, not executable or
archive SHA-256 digests. Licence links are pinned to the module tag or, for Go
pseudo-versions, its source revision.

## LIKE

Matching is case-insensitive against `name`, `original_name` and `description`.
`%` matches zero or more characters; `_` matches exactly one character. `NULL`
does not match. `EXTENDED` changes columns, not selected rows. No matches returns
the normal columns with zero rows.
The existing JSON renderer represents an empty result as `null`; CSV preserves
the column header without adding a data row.

SQL string literals use the existing parser's escaping rules, including doubled
single quotes. After SQL literal decoding, backslashes are literal, not a separate
LIKE escape character. Unlike the existing provider SHOW matcher, this command
also supports `_` and is explicitly case-insensitive.

## Build inventory

The normal build script generates and embeds a static dataset before compilation:

```sh
go generate ./internal/stackql/dependencies
```

The generated dataset is a build artifact, not a source-controlled dependency
list. The checked-in seed allows bare development builds without generation.

The generator reads the target binary's package graph (`go list -deps`, not all
of `go.mod` or `go.sum`) and classifies licence files from those exact module
versions in the build-time module cache. It never scans Python, CI, or container
dependencies. Descriptions are maintained in the generator. Ambiguous multiple
licences are reported as `UNKNOWN` rather than guessing an `AND`/`OR` expression.

Cross-builds should generate with the same target and `GOFLAGS` as compilation:

```sh
go run ./cicd/go/dependency-metadata -targets linux/amd64,linux/arm64
```

Run this generator on the host before setting cross-compilation environment
variables. The normal Python build script handles host/target separation
automatically, including Darwin ARM64 builds.

At runtime, the dataset is reconciled with the executable's immutable Go linker
build information. Bare development builds with absent or stale datasets still
include every linked module; unmatched attribution is explicitly unknown or
missing. Missing Go build information produces an error. No query performs a
network request or requires Go, a source checkout, or a module cache.

CLI output formats, PostgreSQL wire clients and permitted MCP SQL execution use
the same result sets, on either the SQLite or PostgreSQL backend.
