---
name: write_safe_select
description: Guidance for writing safe SELECT queries against stackql resources.
---
In order to ascertain the best safe select query, the correct query form is:
	>   SHOW methods IN <provider>.<service>.<resource>;
	From the output, one can infer the best access method for the SQL "select" verb and the **required** WHERE clause attributes.
