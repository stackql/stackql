

# Views

## *a priori*

At definition time, it is apparent:

- The possible permutations (note plural) of required parameters to support execution.
- Optional parameters.
- View schema:
    - `openapi` schema.
    - Relational schema.

## Runtime

The runtime representation of views must support:

- Views can be aliased as per tables.
- View columns can be aliased in the same way as table columns (even and **especially** those that are aliased inside the view itself).

## Ideation

- StackQL views DDL stored in some special stackql table designated for this purpose.
    - Physical table name such as `__iql__.views`.
    - Views need not exist until the `SELECT ... FROM <view>` portion of the query is executed.
      This is advantageous on RDBMS systems where view creation will fail if physical tables do not exist.
    - We may need a layer of indirection for views to execute, wrt table names containing generation ID.
      Simplest option is input table name.
- SQL view definitions (translated to physical tables) are stored in the RDBMS.
    - This implies that even quite early in analysis, it must be known that a view is being referenced.
    - Some part of the namespace must be reserved for these views; configurable using existing regex / template namespacing?
    - Quite possibly some specialised object(s) or extension of the `table` interface stages are used for view analysis and parameter routing.
- Once analysis is complete:
    - Acquisition occurs as normal through primitive DAG.
    - Selection phase uses physical views.

## Materialized views

Materialized views are similar in nature to views, although eager executed and lacking in mutation of internal `WHERE` clauses from outside.

## User space tables

These map to RDBMS tables.  The DDL is somewhat impaired; we imagine these are useful for staging in general and applications across: ELT, IAC.


## Subqueries

Some aspects of subquery analysis and execution will be similar to views, but not all.  What are the considerations for view implementation in the short term such that subsequent subquery implementation is expedited and natural.

To be continued...


## Joins and aliasing on Views etc

### Views (lazy evaluated)

Views are rendered as inline subqueries `( SELECT ... ) AS "alias"` in the final SQL.  When a user alias is provided (e.g. `FROM my_view v1`), the alias `v1` replaces the view name in the `AS` clause.

**Supported:**
- View aliased and selected from: `SELECT * FROM my_view v1`.
- View JOIN view: `SELECT ... FROM v1 INNER JOIN v2 ON ...`.
- View JOIN provider table: `SELECT ... FROM my_view v1 INNER JOIN provider.svc.resource r ON ...`.
- View JOIN subquery: `SELECT ... FROM my_view v1 INNER JOIN (SELECT ...) sq ON ...`.
- View JOIN materialized view: `SELECT ... FROM my_view v1 INNER JOIN mv ON ...`.
- Nested views (view wrapping a view): supported up to configurable depth (`--indirect-depth-max`, default 5).
- WHERE clause parameter clobbering from outside the view, using **unqualified** parameters (e.g. `WHERE region = 'us-east-1'`).

**Not supported:**
- Table-qualified parameter clobbering into views (e.g. `WHERE v1.region = 'us-east-1'` will not override the view's internal `region` parameter).

### Materialized views (eager evaluated)

Materialized views are persisted as physical tables in the RDBMS.  They are referenced by their table name directly (not as inline subqueries).

**Supported:**
- Materialized view aliased and selected from.
- Materialized view joined with provider tables, user space tables, views and subqueries.
- `CREATE`, `DROP`, `REFRESH`, `CREATE OR REPLACE` lifecycle.

**Not supported:**
- WHERE clause parameter clobbering from outside (materialized views are snapshot-based).

### Subqueries

Subqueries appear as inline `( SELECT ... )` expressions. CTEs (`WITH ... AS`) are converted to subqueries at AST level and handled identically.

### User space tables

User space tables are RDBMS-resident tables created via `CREATE TABLE`. They can participate in joins with any other indirection type.
