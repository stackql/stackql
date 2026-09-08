---
name: run_mutation_query
---
Execute INSERT/UPDATE/REPLACE/DELETE against the provider. Real side effects. Returns {messages, timestamp}. Gated by server mode. Consult query_library_search for a vetted template before composing SQL from scratch; when the SQL came from a library entry, pass its id as source.
