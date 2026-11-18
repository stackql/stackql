---
name: run_select_query
title: Run SELECT Query
open_world: true
---
Execute a SELECT. Returns {rows}, plus {messages} when the engine emitted
per-statement messages (eg upstream provider diagnostics such as a 404 for a
lookup of an object that does not exist; see the error handling instructions).
Reads only; INSERT, UPDATE, DELETE and EXEC
are refused in every mode. Consult query_library_search for a vetted template before composing SQL from scratch; when the SQL came from a
library entry, pass its id as source.
