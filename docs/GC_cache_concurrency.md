
# Garbage Collection, Caching, Concurrency and Views

**NOTE**: The features here described are in ***alpha*** and **owner onus applies**.


## Observations on relationship GC plus cache plus concurrency 

If we want to implement a large result set / analytics cache, then:

- Cache tables need to be outside existing GC, probably with their own, configurable GC. *Only difference to normal queries is probably object lifetime.*
- Probably makes sense to namespace cache tables independently.  *Might be ok to just use cache tables for everything in cache namespace and live with the consequences.*
- When a `cacheable` query is analyzed, the cache GC control parameters must be consumed and used.  *Don't think this need be any different to normal queries.*
- GC needs to be aware of dialect.
- **concurrent** access to data can be brokered via control columns.
    - GC needs to be aware which records are involved in *live transactions*.
    - Regarding limiting selection of result sets in concurrency and GC:
        - **If** existing data is to be consumed, then the set of control predicates must be enforced.
        - **Else If** new data is added, control predicates used identically.
        - For either case, the relationship between active txn and control params must be updated atomically and the analyzer must accumulate the parameters for use during selection.
        - Naive assumption is that perdicate sets apply only to individual insertion sets.
    - This means that complex queries effectively possess a list of `{ Table, ControlPredicate }` pairs.  The `Table` is early-bound, the `ControlPredicate` is late-bound.
    - In a `query` (that is a simple SQL query, subquery or CTE), **all** of the tables analyzed must have control parameters persisted for the selection to later use.  This feature is not yet implemented.  The RW on control parameters must be thread safe, either using locking primitives or ordering invariants.


## Cache ideation

- Async query priming annotation (directive in MySQL parlance).
- Scheduling via config (or extensible to same).
- Query accesses cache if allowed, TTL alive, and/or some annotation in place.
- TTL, schedule, access policy all configurable.
- Boils down to a priming operation followed by OLAP.

### Initial cache read POC

- Stuff data into empty cache table in setup script.
- Analysis phase to include awareness of cache prefix.
- Bingo!


## Usage


Here is an example usage for 1 hour duration caching of github responses:

```
export NAMESPACES='{ "analytics": { "ttl": 86400, "regex": "^(?P<objectName>github.*)$", "template": "stackql_analytics_{{ .objectName }}" } }'


stackql ... --namespaces="${NAMESPACES}" ... shell
```

