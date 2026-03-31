
# Data flow analysis in stackql

Data flow analysis is impplmented as multiple passes on:

- An inital abstract syntax tree (AST) from the parser.
   - Annotated derivatives of the AST.
- `any-sdk`  `{ provider, service, resource, method, schema... }` graphs.
- `gonum` DAG adaptations with data flow dependencies representing edges.

Some other aspects of data flow analysis:

- Relational algebra is implemented in a coupled RDBMS (embedded `sqlite` or `postgres` over TCP).  There is a query rewriting process to stringify "containers" for this.
- There are `transaction control counter` objects and corresponding RDBMS columns to bound relational algebra "containers" and future proof for gargage collection.  Some mutex protection is in place.
- Views in `stackql` permit clobbering of where clause arguments from outside the view.  The canonical case is a document-based view in a provider document.  A good example are in [test/registry/src/aws/v0.1.0/services/pseudo_s3.yaml](/test/registry/src/aws/v0.1.0/services/pseudo_s3.yaml)at `...s3_bucket_list_and_detail.config.views.select`; one can overwrite `region` here.
- Views, subqueries, materialized views and user space tables are modelled as "indirections".


## Open Issues

## Indirection Data Flow Analysis and Query Execution

Data flow analysis for indirections is not composable:

- It it impossible to join heterogenous collections of these with each other or conventional resources.  There is no recusrsive and stable data flow analysis.
- While `stackql` does have a `max depth` parameter, I do not believe it is stable enfoced eagerly.  Ie: queries too complex should fail at analysis time.  Cannot remember param name of=r default.

The expected fix for this issue:

- Joins, unions etc on indirections work to arbitrary and configurable depth.  For depth violations, failure is eager in the analysis phase and error message is plain and in the canonical err stream already widely used.
- Data flow analysis includes assurance on reuired poarams and viability of projections, joins, etc.
- Support for CTEs internal to these indirections is in place.
- Mocked robot tests are added to the canonical test suite, covering off this function.


## Glossary of terms

| Term | Expansion |
|---|---|
| AST  | Abstract Syntax Tree  |
| CTE  | Common Table Expression  |
| DAG  | Directed Acyclic Graph  |
| GC  | Garbage Collection   |
| RDBMS  | Relational Database Management System  |
| TCP  | Transmission Control Protocol  |
|   |   |
