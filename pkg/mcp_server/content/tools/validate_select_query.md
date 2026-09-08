---
name: validate_select_query
---
Parse and plan a SELECT without executing; the provider's credentials must still resolve, as for execution. Returns {valid, errors}. Use when iterating on syntax or plan errors, or pre-flighting complex/expensive queries - don't validate routinely; run_select_query surfaces the same errors.
