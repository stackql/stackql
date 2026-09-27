---
name: describe_method
---
Full I/O contract for one method: every input with its param_type (input_required params become exact-match WHERE predicates; input_optional ones may be added) and every output field, which are the columns a SELECT or RETURNING can reference. Call it after list_methods and before the first query against a method; there is no resource-level equivalent because each method returns its own shape. Requires provider, service, resource, method.
