---
name: run_select_query
title: Run SELECT Query
open_world: true
---
Execute a SELECT. Returns {rows}. Reads only; INSERT, UPDATE, DELETE and EXEC
are refused in every mode. Consult query_library_search for a vetted template before composing SQL from scratch; when the SQL came from a
library entry, pass its id as source.
