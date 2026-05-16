# Repository Guidelines

These guidelines help contributors, human and otherwise, understand and work effectively on this repository.

We also encourage reading [`docs/developer_guide.md`](/docs/developer_guide.md) for further useful information.  For deeper understanding of the "brains" of `stackql`, it is worthwhile to consult [the `AGENTS.md` file of `any-sdk`](https://github.com/stackql/any-sdk/blob/main/AGENTS.md) and linked documents from there.

We have upgraded to golang `>= 1.25.3` in order to take advantage of [time simulation and other capabilities in `testing/synctest`](https://go.dev/blog/testing-time).


## Project Structure & Module Organization

- Entrypoint: [`stackql/main.go`](/stackql/main.go).
- Ideally, foreign system semantics are dealt with in the `any-sdk` repository.
- Loose adherence to popular idioms:
    - App internals in [`internal`](/internal).
    - Re-usable modules in [`pkg`](/pkg).
- The MCP server function is built upon the golang MCP SDK.
- CICD: please see [the github actions workflows](/.github/workflows).
- Docs: `README.md`, this `AGENTS.md`.

## Build, Test, and Development Commands

Authoritative references: [developer guide](/docs/developer_guide.md), [test summary](/test/README.md), [robot tests](/test/robot/README.md), [mock testing](/test/python/stackql_test_tooling/flask/README.md), and [CI workflow](/.github/workflows/build.yml).

Common commands (run from repo root):

- Build: `python cicd/python/build.py --build` (output at `./build/stackql`).
- Unit tests: `python cicd/python/build.py --test` (CI uses `go test -timeout 1200s --tags "sqlite_stackql" -v ./...`).
- Robot tests: `python cicd/python/build.py --robot-test` (requires the binary from the build step).
- Lint: `golangci-lint run` (CI pins the version in [`.github/workflows/lint.yml`](/.github/workflows/lint.yml); config in [`.golangci.yml`](/.golangci.yml)).

## Coding Style & Naming Conventions

- Program to abstractions; concrete types and foreign-system semantics belong behind interfaces, ideally in `any-sdk`.
- `gofmt`/`goimports` formatting; `golangci-lint` must pass (see [`.golangci.yml`](/.golangci.yml)).
- Go identifier conventions: exported `CamelCase`, unexported `camelCase`, acronyms uppercase (`HTTPClient`, not `HttpClient`). Package names short and lowercase.

## Testing Guidelines

- Black-box regression tests are effectively mandatory for behaviour changes. The canonical ones reside in [`test/robot`](/test/robot/README.md) and run against mocks defined in [`test/python/stackql_test_tooling/flask`](/test/python/stackql_test_tooling/flask/README.md).
- Add a new robot scenario when you introduce or fix user-visible query behaviour; unit tests alone are not sufficient evidence.

## Tools & Resources

- Inspect provider surface via the running server: `SHOW PROVIDERS;`, `SHOW SERVICES IN <provider>;`, `DESCRIBE <resource>;`.
- Provider definitions live in the [`stackql-provider-registry`](https://github.com/stackql/stackql-provider-registry); the request-execution brain lives in [`any-sdk`](https://github.com/stackql/any-sdk).

## Commit & Pull Request Guidelines

- Fork-and-pull for external contributors; public contributions, issues, and comments are welcome.
- PR title doubles as the squash-merge commit subject (see recent `main` history for the prevailing style: short hyphenated slug, optional `(#NNN)` appended on merge).
- Keep PRs focused; include a brief summary and a test plan (robot scenario added/modified, or a justification when not applicable).
- All required checks (build, lint, robot) must be green before merge.

## Security & Configuration Tips

StackQL supports flexible configuration management, including authentication secrets and connection parameters, through environment variables and command-line arguments. This design enables seamless integration with standard deployment mechanisms such as container orchestration platforms, CI/CD pipelines, and configuration management tools, allowing sensitive credentials to be managed securely through established DevOps practices rather than hard-coded in scripts or configuration files.

---

## StackQL Resource Key Encoding Quirk

### Hierarchical Keys and URL Encoding

StackQL resource methods require exact matches for hierarchical keys, such as `refs/tags/x1.3`. Historically, due to dependency library behavior (e.g., gorilla/mux in Go), forward slashes in parameters were interpreted as path delimiters, causing query failures or incorrect resource matching.

**Current status (since PR #648, `slash-path-parameters`):** A literal `/` is supported inside a path parameter **only** when the surrounding URL template has no consecutive path parameters. In that case the slash can be passed verbatim. The robot test `Google Cloud Asset IAM Policies Select With Slash In Path Param` covers this.

**When the workaround is still required:** If the slash-bearing parameter is adjacent to another path parameter in the URL template (e.g. consecutive `{p1}/{p2}` slots, as seen in many GitHub routes), the slash must still be URL-encoded as `%2F`. For example, against an endpoint shaped like `/repos/{owner}/{repo}/git/refs/{ref}` where `{ref}` follows other path parameters:

- **May fail:**
  ```sql
  SELECT ref FROM github.git.refs WHERE owner = 'stackql' AND repo = 'mcp-stackql' AND ref = 'refs/tags/x2.0';
  ```
- **Safe fallback:**
  ```sql
  SELECT ref FROM github.git.refs WHERE owner = 'stackql' AND repo = 'mcp-stackql' AND ref = 'tags%2Fx2.0';
  ```

If you are unsure whether a given route has consecutive path parameters, encoding the slash is always safe.

### Guidance for LLMs and Contributors

- Prefer passing the literal value first; if the query fails or returns the wrong resource, fall back to URL-encoding any embedded slashes.
- Do not use wildcards or inequalities for `WHERE` clauses that map to parameters (eg: HTTP path parameters); in such cases, only exact matches are supported.
- Document this quirk in scripts, tests, and code comments to avoid confusion.

### Why This Is Necessary

Many RESTful routing libraries (like gorilla/mux) treat slashes as path separators. PR #648 relaxes this for the single-parameter case, but the consecutive-parameter case remains ambiguous to the router, so encoding is still needed there.

Refer to this section whenever you encounter issues with resource keys containing slashes or hierarchical identifiers.
