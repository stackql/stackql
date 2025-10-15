# Repository Guidelines

These guidelines help contributors, human and otherwise, understand and work effectively on this repository.

We also encourage reading [`docs/developer_guide.md`](/docs/developer_guide.md) for further useful information.  For deeper understanding of the "brains" of `stackql`, it is worthwhile to consult [the `AGENTS.md` file of `any-sdk`](https://github.com/stackql/any-sdk/blob/main/AGENTS.md) and linked documents from there.


## Project Structure & Module Organization

- Entrypoint: [`stackql/main.go`](/stackql/main.go).
- Ideally, foregin system semantics are dealt with in the `any-sdk` repository.
- Loose adherence to popular idioms:
    - App internals in [`internal`](/internal).
    - Re-usable modules in [`pkg`](/pkg).
— The MCP server function is built upon the golang MCP SDK.
- CICD: please see [the github actions workflows](/.github/workflows).
- Docs: `README.md`, this `AGENTS.md`.

## Build, Test, and Development Commands

Please refer to [the developer guide](/docs/developer_guide.md), [the testing summary document](/test/README.md), [the robot testing document](/test/robot/README.md), and possibly most useful of all, [the doco explaining testing with mocks](/test/python/stackql_test_tooling/flask/README.md).  For CI in the wild, please see [`.github/workflows/build.yml`](/.github/workflows/build.yml).

## Coding Style & Naming Conventions

- Publish and program to abstractions.

## Testing Guidelines

- Black box regression tests are effectively mandatory.  The canaonical ones reside in [`test/robot`](/test/robot/README.md).

## Tools & Resources

- Please inspect using the API.


## Commit & Pull Request Guidelines

- Fork and pull model for general public; we **strongly** welcome public contributions, comment and issues.

## Security & Configuration Tips

StackQL supports flexible configuration management, including authentication secrets and connection parameters, through environment variables and command-line arguments. This design enables seamless integration with standard deployment mechanisms such as container orchestration platforms, CI/CD pipelines, and configuration management tools, allowing sensitive credentials to be managed securely through established DevOps practices rather than hard-coded in scripts or configuration files.

---

## StackQL Resource Key Encoding Quirk

### Hierarchical Keys and URL Encoding

StackQL resource methods (especially for GitHub refs/tags) require exact matches for hierarchical keys, such as `refs/tags/x1.3`. However, due to dependency library behavior (e.g., gorilla/mux in Go), forward slashes in parameters may be interpreted as path delimiters, causing query failures or incorrect resource matching.

**Workaround:** Always URL encode forward slashes in resource keys when constructing queries. For example:

- **Incorrect:**
  ```sql
  SELECT ref FROM github.git.refs WHERE owner = 'stackql' AND repo = 'mcp-stackql' AND ref = 'refs/tags/x2.0';
  ```
- **Correct:**
  ```sql
  SELECT ref FROM github.git.refs WHERE owner = 'stackql' AND repo = 'mcp-stackql' AND ref = 'tags%2Fx2.0';
  ```

This ensures the backend treats the parameter as a literal string, not a path.

### Guidance for LLMs and Contributors

- Always encode slashes in hierarchical resource keys for StackQL queries.
- Do not use wildcards or inequalities for `WHERE` clauses that map to parameters (eg: HTTP path parameters); in such cases, only exact matches are supported.
- Document this quirk in scripts, tests, and code comments to avoid confusion.
- Example for LLM prompt:
  > "When querying StackQL resources with keys containing slashes, always URL encode the slash."

### Why This Is Necessary

Many RESTful routing libraries (like gorilla/mux) treat slashes as path separators. Encoding slashes prevents misinterpretation and ensures correct resource access.

Refer to this section whenever you encounter issues with resource keys containing slashes or hierarchical identifiers.
