---
name: reload_credentials
---
Live-reload provider credentials: re-sources the server's configured env file into the process environment, invalidates cached auth contexts, and reports resolution status for every installed provider (optional provider arg filters the report; changed field flags values that differ from before). Never returns secret values. Credentials resolve automatically at query time - never call this at session start or before queries. Call only after a query fails with a credential resolution error (fix the env file first, then reload, then retry) or when the user says credentials have been rotated or changed. If the report shows changed: false after a failure, the file was not updated - ask the user rather than retrying.
