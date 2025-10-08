# Repository Guidelines

These guidelines help contributors work effectively on this repository.  We gratefully acknowledge [mcp-postgres](https://github.com/gldc/mcp-postgres) as the chief inspiration for the MCP server function and this document.

We also encourage reading [`docs/developer_guide.md`](/docs/developer_guide.md) for further useful information.


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

- Create env: `python -m venv .venv && source .venv/bin/activate`
- Install deps: `pip install -r requirements.txt`
- Run server (no DB): `python postgres_server.py`
- Run with DB: `POSTGRES_CONNECTION_STRING="postgresql://user:pass@host:5432/db" python postgres_server.py`
- Docker build/run: `docker build -t mcp-postgres .` then `docker run -e POSTGRES_CONNECTION_STRING=... -p 8000:8000 mcp-postgres`

## Coding Style & Naming Conventions

- Publish and program to abstractions.

## Testing Guidelines

- Black box regression tests are effectively mandatory.  The canaonical ones reside in [`test/robot/functional`](/test/robot/functional).

## Tools & Resources

- Please inspect using the API.


## Commit & Pull Request Guidelines

- Fork and pull model for general public; we **strongly** welcome public contributions, comment and issues.

## Security & Configuration Tips

- WIP.

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
