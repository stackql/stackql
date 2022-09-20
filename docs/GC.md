

## GC at present

- `GCCollectObsolete()` in use by pass through primitive.
- For IMDB, pretty much all GC is NOP.
- Some other **unused** algorithms are present in `drm` package.

## Observations on relationship GC vs cache

If we want to implement a large result set / analytics cache, then:

- Cache tables need to be outside existing GC, probably with their own, configurable GC.
- Probably makes sense to namespace cache tables independently.
- When a `cacheable` query is analyzed, the cache GC control parameters must be consumed and used.
- GC needs to be aware of dialect.

## Cache ideation

- Async query priming annotation (directive in MySQL parlance).
- Scheduling via config (or extensible to same).
- Query accesses cache if allowed, TTL alive, and/or some annotation in place.
- TTL, schedule, access policy all configurable.
- Boils down to a priming operation followed by OLAP.
