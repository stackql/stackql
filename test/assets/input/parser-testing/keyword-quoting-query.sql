SELECT
    cons.conname as name,
    cons.conkey as key,
    a.attnum as col_num,
    a.attname as col_name
FROM
    pg_catalog.pg_constraint cons
    join pg_attribute a
      on cons.conrelid = a.attrelid AND
        a.attnum = ANY(cons.conkey)
WHERE
    cons.conrelid = '16709' AND
    cons.contype = 'u'
;