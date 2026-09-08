---
name: validate_select_query
---
Parse and plan a SELECT without executing; no credentials required [confirm this holds]. Returns {valid, errors}. Use when iterating on syntax or plan errors, or pre-flighting complex/expensive queries - don't validate routinely; run_select_query surfaces the same errors.