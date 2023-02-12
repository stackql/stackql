SELECT a.attname,
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
                (SELECT json_build_object(
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
WHERE a.attrelid = '112'
AND a.attnum > 0 AND NOT a.attisdropped
ORDER BY a.attnum
;