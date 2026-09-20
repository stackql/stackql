# Work order: migrate embedded SQLite backend to modernc.org/sqlite

## Objective

Replace the embedded relational backend driver `github.com/stackql/stackql-go-sqlite3`
(a cgo fork of mattn/go-sqlite3) with `modernc.org/sqlite` (pure Go, CGo-free),
including a pure Go port of the custom extension functions currently provided by
`github.com/stackql/sqlite-ext-functions`. This is a single big-bang PR: no dual
backend, no feature flag, no cgo fallback. The full robot test suite passing is
the merge gate.

## Context

- The embedded backend sits behind an existing SQL backend abstraction (the same
  seam that supports the postgres backend). Do not touch the postgres backend.
- The extension surface is exactly six deterministic scalar functions:
  `split_part`, `regexp_like`, `regexp_substr`, `regexp_replace`, `json_equal`,
  `aws_policy_equal`. No aggregates, no window functions, no virtual tables.
- Conventions for the ported functions live in `internal/sqlfuncs/CLAUDE.md`
  (already in the repo). Treat it as authoritative for that package.
- The C reference implementations and their test suites are at
  https://github.com/stackql/sqlite-ext-functions (see `src/` and `test/`).

## Hard constraints

- Pin `modernc.org/sqlite` at >= v1.57.0 (caller-constructed Driver function
  registration). Do not fork or vendor-patch it.
- No package-global function registration. Construct a `*sqlite.Driver`,
  register functions on it via `internal/sqlfuncs.Register(drv)`, and register
  it under the driver name `stackql-sqlite`.
- Do not silently change function semantics. Divergences from the C
  implementations (RE2 regexp limits in particular) are documented in
  `internal/sqlfuncs/DIVERGENCES.md`, never absorbed.
- Deliver as one PR with three clean commit groups (below) so the revert path
  is `git revert`, not archaeology.

## Stop conditions - halt and report, do not work around

1. Run `PRAGMA compile_options;` on the current cgo build and on a minimal
   modernc.org/sqlite program. Diff them. If the fork enables any
   `SQLITE_ENABLE_*` option that stackql code paths depend on and modernc's
   build lacks, STOP. The fix is an upstream request to the modernc maintainer,
   not a workaround.
2. If any golden vector from the C test suite cannot be satisfied by a Go
   implementation without ambiguity about intended behavior, STOP and list the
   cases for human decision.
3. If the robot suite reveals a driver behavior difference not covered by this
   work order, STOP and report before patching around it.

## Commit group 1: internal/sqlfuncs

1. Clone stackql/sqlite-ext-functions to a temp location. Copy every test
   vector from its `test/` directory into `internal/sqlfuncs/testdata/`,
   converted to a table-driven format. These are the compatibility contract.
2. Implement the six functions in pure Go per `internal/sqlfuncs/CLAUDE.md`:
   - `split_part`: strings-based; match the C behavior exactly for 1-based
     indexing, negative indexes counting from the end, and out-of-range parts.
   - `regexp_like` / `regexp_substr` / `regexp_replace`: Go `regexp` (RE2).
     Determine from the C source what replacement token syntax it used
     ($1 vs \1) and translate. Record every pattern-class divergence
     (backreferences, lookaround) in DIVERGENCES.md.
   - `json_equal`: canonicalized deep comparison via encoding/json. Match the
     C implementation for number forms (1 vs 1.0, big integers) and for
     invalid-JSON inputs.
   - `aws_policy_equal`: port the normalization logic - scalar vs array forms
     of Action/Resource/Principal/NotAction etc., statement order
     insensitivity, IAM case rules.
   - NULL propagation per function, verified against the vectors, not assumed.
3. `register.go`: `func Register(drv *sqlite.Driver) error` registering all six
   as deterministic scalars on the passed driver.
4. Native Go fuzz targets for each function. Seed corpora from the golden
   vectors. Run each fuzz target briefly (e.g. 30s) and commit any findings as
   regression cases.

## Commit group 2: driver swap

1. Remove the `stackql-go-sqlite3` / `mattn` import entirely, first. Fix every
   resulting compile error - this is the mechanical inventory of touchpoints
   (error type assertions, ConnectHook usage, driver name literals, DSN
   construction).
2. The compiler will not catch everything. Grep and fix:
   - string matching on driver error messages
   - DSN strings built by concatenation anywhere outside the central builder
   - the literal driver name `sqlite3` in Open calls, config, and docs
3. Centralize DSN construction in one `BuildDSN(opts)` function. Translate
   mattn-style params to modernc `_pragma=` syntax, e.g.
   `_busy_timeout=5000` -> `_pragma=busy_timeout(5000)`,
   `_journal_mode=WAL` -> `_pragma=journal_mode(WAL)`,
   `_foreign_keys=on` -> `_pragma=foreign_keys(1)`.
   modernc IGNORES unknown mattn-style params without error - a missed
   translation fails silently, hence step 5.
4. Wrap all driver error inspection behind internal predicates
   (`isBusy(err)`, `isConstraintViolation(err)`, etc.) mapping
   `*sqlite.Error` codes. No `*sqlite.Error` type assertions outside that file.
5. Startup pragma assertion: immediately after opening the embedded backend,
   query `PRAGMA busy_timeout;`, `PRAGMA journal_mode;`, `PRAGMA foreign_keys;`
   (and any other pragma the DSN sets) and fail fast if they do not match
   intent. This assertion is permanent, not scaffolding.
6. Verify in-memory database semantics under the existing connection pool
   settings. With database/sql pooling, each new connection to a plain
   `:memory:` DSN is a separate database. Confirm the existing
   SetMaxOpenConns / shared-cache policy still yields one coherent database;
   single writer connection is the safe default for modernc.
7. If any control-plane table stores timestamps, add a round-trip test
   (insert time.Time -> select -> compare) and set `_time_format` in the DSN
   if needed to preserve prior behavior.
8. Add a concurrency stress test: N goroutines issuing reads plus a writer
   against the embedded backend, asserting no unhandled SQLITE_BUSY.

## Commit group 3: CI and release

1. Set `CGO_ENABLED=0` everywhere. Remove C toolchain setup from CI and
   release workflows (mingw, osxcross, musl, gcc steps, related caches).
2. Cross-compile every existing release GOOS/GOARCH target natively. Add a
   per-artifact smoke test to the release pipeline:
   `stackql exec "SELECT split_part('a,b', ',', 2)"` plus one invocation per
   extension function - this proves the binary and function registration on
   every platform.
3. Add a note to dependabot config or CONTRIBUTING: bumps of modernc.org/sqlite
   merge only after the full robot suite passes (transpile regenerations
   occasionally regress).
4. Append to the repo root CLAUDE.md, under a "SQLite backend" heading:
   - embedded backend is modernc.org/sqlite (pure Go), driver name
     `stackql-sqlite`, pinned >= v1.57.0
   - DSN construction only via BuildDSN; driver errors only via the
     predicates file; extension functions only via internal/sqlfuncs
   - never reintroduce mattn/go-sqlite3, cgo, or package-global function
     registration
5. Release notes entry: pure Go backend, CGO_ENABLED=0 static binaries,
   library embedders inherit the modernc dependency tree, external loadable
   SQLite extensions (.load) are no longer supported.

## Acceptance criteria

- [ ] Full robot test suite green
- [ ] internal/sqlfuncs unit tests green, golden vectors all passing
- [ ] Fuzz targets run clean for the smoke duration
- [ ] Startup pragma assertions in place and passing
- [ ] Concurrency stress test passing
- [ ] Timestamp round-trip verified (or shown not applicable)
- [ ] `grep -r "mattn\|stackql-go-sqlite3"` returns nothing outside docs/changelog
- [ ] All release targets build with CGO_ENABLED=0 and pass the smoke test
- [ ] DIVERGENCES.md complete, root CLAUDE.md updated, release notes drafted

## Out of scope

- The postgres backend
- Any change to SQL semantics visible to stackql queries beyond documented
  regexp divergences
- Archiving the old repos (handled post-merge, outside this PR)
