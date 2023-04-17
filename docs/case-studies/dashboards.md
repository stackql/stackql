

```sql

ROLLBACK;

BEGIN;

select pg_catalog.version();

select current_schema();

show transaction isolation level;

show standard_conforming_strings;

ROLLBACK;

SET DATESTYLE TO 'ISO';

SELECT c.oid
FROM pg_catalog.pg_class c
LEFT JOIN pg_catalog.pg_namespace n 
ON n.oid = c.relnamespace
WHERE (n.nspname = 'information_schema')
AND c.relname = 'attributes' AND c.relkind in
('r', 'v', 'm', 'f', 'p')
;

SELECT 
  a.attname,
  pg_catalog.format_type(a.atttypid, a.atttypmod),
  (
    SELECT pg_catalog.pg_get_expr(d.adbin, d.adrelid)
    FROM pg_catalog.pg_attrdef d
    WHERE d.adrelid = a.attrelid AND d.adnum = a.attnum
    AND a.atthasdef
  ) AS DEFAULT,
  a.attnotnull,
  a.attrelid as table_oid,
  pgd.description as comment,
  a.attgenerated as generated,
  (
    SELECT json_build_object(
        'always', a.attidentity = 'a',
        'start', s.seqstart,
        'increment', s.seqincrement,
        'minvalue', s.seqmin,
        'maxvalue', s.seqmax,
        'cache', s.seqcache,
        'cycle', s.seqcycle)
    FROM pg_catalog.pg_sequence s
    JOIN pg_catalog.pg_class c on s.seqrelid = c."oid"
    WHERE c.relkind = 'S'
    AND a.attidentity != ''
    AND s.seqrelid = pg_catalog.pg_get_serial_sequence(
        a.attrelid::regclass::text, a.attname
    )::regclass::oid
        ) as identity_options
    FROM pg_catalog.pg_attribute a
    LEFT JOIN pg_catalog.pg_description pgd ON (
        pgd.objoid = a.attrelid AND pgd.objsubid = a.attnum)
    WHERE a.attrelid = '13429'
    AND a.attnum > 0 AND NOT a.attisdropped
    ORDER BY a.attnum
;


SELECT 
  t.typname as "name",
  pg_catalog.format_type(t.typbasetype, t.typtypmod) as "attype",
  not t.typnotnull as "nullable",
  t.typdefault as "default",
  pg_catalog.pg_type_is_visible(t.oid) as "visible",
  n.nspname as "schema"
FROM pg_catalog.pg_type t
LEFT JOIN pg_catalog.pg_namespace n ON n.oid = t.typnamespace
WHERE t.typtype = 'd'
;


SELECT 
t.typname as "name",
-- no enum defaults in 8.4 at least
-- t.typdefault as "default",
pg_catalog.pg_type_is_visible(t.oid) as "visible",
n.nspname as "schema",
e.enumlabel as "label"
FROM pg_catalog.pg_type t
LEFT JOIN pg_catalog.pg_namespace n ON n.oid = t.typnamespace
LEFT JOIN pg_catalog.pg_enum e ON t.oid = e.enumtypid
WHERE t.typtype = 'e'
ORDER BY "schema", "name", e.oid
;


SELECT t.oid, NULL
FROM pg_type t JOIN pg_namespace ns
ON typnamespace = ns.oid
WHERE typname = 'hstore'
;


SELECT t.typname as "name",
-- no enum defaults in 8.4 at least
-- t.typdefault as \"default\",
pg_catalog.pg_type_is_visible(t.oid) as "visible",
n.nspname as "schema",
e.enumlabel as "label"
FROM pg_catalog.pg_type t
LEFT JOIN pg_catalog.pg_namespace n ON n.oid = t.typnamespace
LEFT JOIN pg_catalog.pg_enum e ON t.oid = e.enumtypid
WHERE t.typtype = 'e'
ORDER BY "schema", "name", e.oid
;


SELECT a.attname
FROM pg_attribute a JOIN (
SELECT unnest(ix.indkey) attnum,
generate_subscripts(ix.indkey, 1) ord
FROM pg_index ix
WHERE ix.indrelid = '13420' AND ix.indisprimary
) k ON a.attnum=k.attnum
WHERE a.attrelid = '13420' 
ORDER BY k.ord
;


SELECT a.attname 
FROM pg_attribute a 
JOIN ( 
SELECT unnest(ix.indkey) attnum, generate_subscripts(ix.indkey, 1) ord FROM pg_index ix WHERE ix.indrelid = '13420' AND ix.indisprimary 
) k 
ON a.attnum=k.attnum 
WHERE a.attrelid = '13420'  
ORDER BY k.ord
;


```