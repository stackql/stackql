

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
      This is advantagesous on RDBMS systems where view creation will fail if physical tables do not exist.
    - We may need a layer of indirection for views to execute, wrt table names containing generation ID.
      Simplest option is input table name.
- SQL view definitions (translated to physical tables) are stored in the RDBMS.
    - This implies that even quite early in analysis, it must be known that a view is being referenced.
    - Some part of the namespace must be reserved for these views; configurable using existing regex / template namespacing?
    - Quite possibly some specialised object(s) or extension of the `table` interface stages are used for view analysis and parameter routing.
- Once analysis is complete:
    - Acquistion occurs as normal through primitive DAG.
    - Selection phase uses physical views.


## Subqueries

Some aspects of subquery analysis and execution will be similar to views, but not all.  What are the considerations for view implementation in the short term such that subsequent subquery implmentation is expedited and natural.

To be continued...


