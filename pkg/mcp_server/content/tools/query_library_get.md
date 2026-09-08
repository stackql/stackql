---
name: query_library_get
---
Retrieve a query library entry by id. Without params: returns the raw template, param declarations and notes (adapt from it when no exact match exists). With
params: the server validates values and returns rendered SQL plus which tool to execute it with (run_select_query or run_mutation_query). Read-only, no credentials.
