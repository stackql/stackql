---
name: stackql-setup
description: Set up StackQL provider credentials and run a first query. Use when the StackQL plugin has just been installed, when the user asks how to configure cloud or SaaS credentials for StackQL, or when a StackQL query fails with a credential resolution error.
---

# StackQL setup

StackQL runs on this computer as a local MCP server. Queries go straight from
this machine to the cloud or SaaS provider APIs with the user's own
credentials. Nothing is sent to StackQL Studios.

The server's tools are only available in sessions that run locally. If the
`stackql` MCP server is not listed in this session, tell the user to open a
session that runs on their computer (a Codex thread or a local ChatGPT desktop
thread; Claude Code or a Cowork session on this machine) and stop here.

## 1. Confirm the server is up

Call `server_info`. It reports the StackQL version, the SQL backend and the
server mode. The default mode is `safe`: SELECT queries run freely, and
mutations ask for approval before they run.

## 2. Run a first query without credentials

GitHub public data needs no credentials, so use it to prove the install:

1. `pull_provider` with provider `github`. This downloads the provider
   definition from the StackQL registry the first time only.
2. `list_services` for `github`, then `list_resources` for the `repos` service.
3. `list_methods` for `github.repos.repos`, then `describe_method` for the
   method that lists an organization's repositories, to get its required
   inputs and output columns.
4. `run_select_query` with a SELECT against `github.repos.repos` that sets the
   required `org` predicate, for example to `stackql`, and selects columns
   named in the method description.

Report the rows returned. If `pull_provider` fails with a network error, the
machine cannot reach the StackQL provider registry; ask the user to check
their connection or proxy and stop.

## 3. Add provider credentials

Credentials live in a dotenv file on this computer. The server creates it
empty on first start and reads no other credential source by default:

- Linux and macOS: `~/.stackql/.env`
- Windows: `%USERPROFILE%\.stackql\.env`

Each provider uses its standard environment variable names, so most users only
add values to that file. For example, AWS and AWS Cloud Control read:

```dotenv
AWS_ACCESS_KEY_ID=...
AWS_SECRET_ACCESS_KEY=...
```

To find the variable names another provider expects, call
`reload_credentials` after pulling the provider: the report lists every
installed provider with the variables it looks for. The provider pages at
https://stackql.io/docs document every option.

After the user edits the file, call `reload_credentials` (optionally with the
provider name). It re-sources the file and reports, per provider, a status of
`ok`, `unresolved` or `not_checked` with the variable names involved. It never
returns credential values, so do not ask the user to paste credentials into
the conversation.

Use `.stackqlrc` in the same directory only when a provider needs an auth type
or credential source that differs from its default. The server reads that file
at startup, so after changing it the user restarts the session.

## 4. Run a query with credentials

Repeat the discovery steps from section 2 for the provider, then run a SELECT.
Mutations (INSERT, UPDATE, DELETE and lifecycle EXEC) have real side effects:
in `safe` mode the server asks for approval on each one, and a client that
cannot show the approval prompt gets an error instead of a change. Say so
before suggesting one.

## When a query fails

- Credential resolution error: the message names the provider and the
  variables. Ask the user to update the env file, call `reload_credentials`
  for that provider, and retry once. If the report shows `changed: false`,
  the file was not updated; ask the user instead of retrying again.
- Unknown column or resource: go back to `describe_method`. Column names come
  from the method the query routes to, so take them from there rather than
  guessing.
