
# Garbage Collection, Caching, Concurrency and Views

**NOTE**: The features here described are in ***alpha*** and **owner onus applies**.


## Background


### Observations on relationship GC plus cache plus concurrency 

If we want to implement a large result set / analytics cache, then:

- Cache tables need some nuanced GC, probably with their own, configurable polic(y/ies), as distinct from GC on the bog standard data store. *Only difference to normal queries is probably object lifetime.*
- Probably makes sense to namespace cache tables independently.  *Might be ok to just use cache tables for everything in cache namespace and live with the consequences.*
- When a `cacheable` query is analyzed, the cache GC control parameters must be consumed and used.  *Don't think this need be any different to normal queries.*
- GC needs to be aware of dialect.
- **concurrent** access to data can be brokered via control columns.
    - GC needs to be aware which records are involved in *live transactions*.
    - Regarding limiting selection of result sets in concurrency and GC:
        - **If** existing data is to be consumed, then the set of control predicates must be enforced.
        - **Else If** new data is added, control predicates used identically.
        - For either case, the relationship between active txn and control params must be updated atomically and the analyzer must accumulate the parameters for use during selection.
        - Naive assumption is that predicate sets apply only to individual insertion sets.
    - This means that complex queries effectively possess a list of `{ Table, ControlPredicate }` pairs.  The `Table` is early-bound, the `ControlPredicate` is late-bound.
    - In a `query` (that is a simple SQL query, subquery or CTE), **all** of the tables analyzed must have control parameters persisted for the selection to later use.  This feature is not yet implemented.  The RW on control parameters must be thread safe, either using locking primitives or ordering invariants.


### Cache ideation

- ~~Async query priming annotation (directive in MySQL parlance).~~ If cache is not primed, then initial queries run online.
- Scheduling via config (or extensible to same).
- Query accesses cache if allowed, TTL alive, and/or some annotation in place.
- TTL, schedule, access policy all configurable.
- Boils down to a priming operation followed by OLAP.

### Initial cache read POC

- Stuff data into empty cache table in setup script.
- Analysis phase to include awareness of cache prefix.
- Bingo!


### Usage


Here is an example usage for 1 hour duration caching of github responses:

```
export NAMESPACES='{ "analytics": { "ttl": 86400, "regex": "^(?P<objectName>github.*)$", "template": "stackql_analytics_{{ .objectName }}" } }'


stackql ... --namespaces="${NAMESPACES}" ... shell
```

### MVCC and Postgres?

- [Postgres MVCC high level rundown](https://devcenter.heroku.com/articles/postgresql-concurrency#:~:text=a%20hard%20problem.-,How%20MVCC%20works,statements%20together%20via%20BEGIN%20%2D%20COMMIT%20).  Cool side notes from this article:
    - The `t_xmin` and `t_xmax` counters are visible on explicit calls; `SELECT *, xmin, xmax FROM table_name;`.
    - The Txn ID is also visible; `SELECT txid_current();`.
- [Postgres official MVCC](https://www.postgresql.org/docs/current/mvcc.html) 
- [Postgres official VACUUM + ID wraparound etc](https://www.postgresql.org/docs/current/routine-vacuuming.html)

Postgres implements [RW locking on indexes](https://www.postgresql.org/docs/current/locking-indexes.html) which varies by index type and comes with differing performance and deadlock headaches.  `B-tree`, the default type, is the go to for `scalar` data.

At record level, MVCC takes over, leveraging [the tuple header](https://www.postgresql.org/docs/current/storage-page-layout.html#STORAGE-TUPLE-LAYOUT).  The key fields here are `t_xmin` for the Txn that created the record and `t_xmax`, which ***usually*** records the Txn that deleted the record.  Both `t_xmin` and `t_xmax` are 32 bit counters and overflow anomalies are only prevented by the action of [`VACUUM` garbage collection](https://www.postgresql.org/docs/current/routine-vacuuming.html).  Canonically, updates result in a new row being created (deletes are the same, but no new tuple created) and the tuple header `t_max` set to whichever Txn made the update.  One interesting variation is a scenario where multiple Txns have a lock on the tuple; in such a case the tuple header field `t_infomask` [will indicate that `t_xmax` should be interpreted as a `MultiXactId`](https://github.com/postgres/postgres/blob/ce20f8b9f4354b46b40fd6ebf7ce5c37d08747e0/src/include/access/htup_details.h#L208).  `MultiXactId` is itself a 32 bit counter that acts as an indirection to a list of txn IDs stored elsewhere.  There are other variations and the algorithms to maintain all of the requisite data coherently are non-trivial.


## Short term approach for StackQL

### v1

In the first instance, MVCC is likely overkill for stackql.  This is because stackql does not actually **need** to update non-control portions of database records and txns **never** directly delete records.  Therefore the following information is sufficient to infer if a record is a deletion candidate:

  - `txn_max_id` is the maximum Txn ID that has locked the record.  
  - `txn_running_min_id` is the minimum Txn ID still running in the system.

If `txn_running_min_id` > `txn_max_id` then the record can safely be removed.  This is definitely not the only or optimal approach but it is simple and requires only one (maximum) Txn ID to be stored against each record. The simplest technique is to store this Txn ID as a control column in each record tuple.

A **hard requirement** of this in-row pattern is that row updates in the database backend can be done atomically.  This is definitely the case for SQLite, Postgres, MySQL, etc but may not hold up for all future backends.  It will be adequate for `v1`.

Long lived or dormant `phantom` Txns must be killable through some aspect of the GC process.  This is because otherwise `txn_running_min_id` will not advance and eventually the required monotonic Txn counter invariant will fail.  This we can initially address by killing all transactions older than some threshold timestamp or the timestamp corresponding to Txn ID of some threshold count.  It can be improved upon later.

A sensible `v1` approach to concurrency and GC in `stackql` is therefore:

- A Txn ID, such as may be required in various aspects of the system, is a monotonically increasing counter.  The chosen implementation is fixed size integer type used as a ring and periodically re-zeroed.  **Update**: this will be implemented in the DB backend, through SQL.  **TBD**: The re-zero logic is yet to be implemented at this early *alpha* stage.  
- A global list of live (Txn ID, timestamp begun) tuples must be maintained. `txn_running_min_id` can be inferred from this list.  This is a la Postgres.  **TBD**: the Txn ID store is completed, however timestamps are not yet persisted, so ***old*** Txns are effectively invisible.
- The `acquire` phase (write to DB after REST call or read from cache) must update `txn_max_id` using conditional SQL logic (if greater than existing).
- **TBD**: GC cycles to be triggered by:
  - Threshold of live Txns reached.
  - Schedule.
  - ...
- In GC cycles:
  - **TBD**: No new Txns may begin.
  - **Hypothesis**: existing Txns can sit dormant.
  - If there are too many Txns alive and/or some are too old, then destroy the abberant Txns.
  - If `txn_running_min_id` > `txn_max_id` then destroy record.
  - Admin activities to be supported:
    - Purge interventions for Txns and records.  This is available through the `PURGE` grammar.
- **TBD**: Pre-emptible handler threads to be implemented.


This naive approach avoids deadlock and provides break glass.

Please **watch this space** on all items which are **TBD**-inclusive.

### v2

- Admin activities to be supported supported (`v2` perhaps):
  - Purge interventions.
  - List active Txns.
  - Halt / Allow new Txns.
  - Cancel Txns (filtered / unfiltered).
