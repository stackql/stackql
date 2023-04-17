*** Settings ***
Resource          ${CURDIR}/stackql.resource
Test Teardown     Stackql Per Test Teardown

*** Test Cases *** 

Select Class Oid
    Pass Execution If    "${SQL_BACKEND}" != "postgres_tcp"    This is a postgres only test
    ${inputStr} =    Catenate
    ...    SELECT c.oid
    ...    FROM pg_catalog.pg_class c
    ...    LEFT JOIN pg_catalog.pg_namespace n 
    ...    ON n.oid \= c.relnamespace
    ...    WHERE (n.nspname \= 'information_schema')
    ...    AND c.relname \= 'attributes' AND c.relkind in
    ...    ('r', 'v', 'm', 'f', 'p')
    ...    ;
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |-------|
    ...    |${SPACE}${SPACE}oid${SPACE}${SPACE}|
    ...    |-------|
    ...    |${SPACE}13429${SPACE}|
    ...    |-------|
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    {"digitalocean": { "username_var": "DUMMY_DIGITALOCEAN_USERNAME", "password_var": "DUMMY_DIGITALOCEAN_PASSWORD", "type": "basic", "valuePrefix": "TOTALLY_CONTRIVED "}}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Select-Class-Oid.tmp

Select Attributes
    Pass Execution If    "${SQL_BACKEND}" != "postgres_tcp"    This is a postgres only test
    ${inputStr} =    Catenate
    ...    SELECT 
    ...    a.attname,
    ...    pg_catalog.format_type(a.atttypid, a.atttypmod),
    ...    (
    ...    SELECT pg_catalog.pg_get_expr(d.adbin, d.adrelid)
    ...    FROM pg_catalog.pg_attrdef d
    ...    WHERE d.adrelid \= a.attrelid AND d.adnum = a.attnum
    ...    AND a.atthasdef
    ...    ) AS DEFAULT,
    ...    a.attnotnull,
    ...    a.attrelid as table_oid,
    ...    pgd.description as comment,
    ...    a.attgenerated as generated,
    ...    (
    ...    SELECT json_build_object(
    ...    'always', a.attidentity \= 'a',
    ...    'start', s.seqstart,
    ...    'increment', s.seqincrement,
    ...    'minvalue', s.seqmin,
    ...    'maxvalue', s.seqmax,
    ...    'cache', s.seqcache,
    ...    'cycle', s.seqcycle)
    ...    FROM pg_catalog.pg_sequence s
    ...    JOIN pg_catalog.pg_class c on s.seqrelid \= c."oid"
    ...    WHERE c.relkind \= 'S'
    ...    AND a.attidentity !\= ''
    ...    AND s.seqrelid = pg_catalog.pg_get_serial_sequence(
    ...    a.attrelid\:\:regclass\:\:text, a.attname
    ...    )\:\:regclass\:\:oid
    ...    ) as identity_options
    ...    FROM pg_catalog.pg_attribute a
    ...    LEFT JOIN pg_catalog.pg_description pgd ON (
    ...    pgd.objoid \= a.attrelid AND pgd.objsubid \= a.attnum)
    ...    WHERE a.attrelid \= '13429'
    ...    AND a.attnum > 0 AND NOT a.attisdropped
    ...    ORDER BY a.attnum
    ...    ;
    Should Stackql Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    {"digitalocean": { "username_var": "DUMMY_DIGITALOCEAN_USERNAME", "password_var": "DUMMY_DIGITALOCEAN_PASSWORD", "type": "basic", "valuePrefix": "TOTALLY_CONTRIVED "}}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    information_schema.sql_identifier
    ...    stdout=${CURDIR}/tmp/Select-Attributes.tmp

Select Attribute Metadata
    Pass Execution If    "${SQL_BACKEND}" != "postgres_tcp"    This is a postgres only test
    ${inputStr} =    Catenate
    ...    SELECT 
    ...    t.typname as "name",
    ...    pg_catalog.format_type(t.typbasetype, t.typtypmod) as "attype",
    ...    not t.typnotnull as "nullable",
    ...    t.typdefault as "default",
    ...    pg_catalog.pg_type_is_visible(t.oid) as "visible",
    ...    n.nspname as "schema"
    ...    FROM pg_catalog.pg_type t
    ...    LEFT JOIN pg_catalog.pg_namespace n ON n.oid \= t.typnamespace
    ...    WHERE t.typtype \= 'd'
    ...    ;
    Should Stackql Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    {"digitalocean": { "username_var": "DUMMY_DIGITALOCEAN_USERNAME", "password_var": "DUMMY_DIGITALOCEAN_PASSWORD", "type": "basic", "valuePrefix": "TOTALLY_CONTRIVED "}}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    sql_identifier
    ...    stdout=${CURDIR}/tmp/Select-Attribute-Metadata.tmp

Select Enums Returns Header Only
    Pass Execution If    "${SQL_BACKEND}" != "postgres_tcp"    This is a postgres only test
    ${inputStr} =    Catenate    SEPARATOR=\n  # linebreaks required due to inline comments
    ...    SELECT 
    ...    t.typname as "name",
    ...    -- no enum defaults in 8.4 at least
    ...    -- t.typdefault as "default",
    ...    pg_catalog.pg_type_is_visible(t.oid) as "visible",
    ...    n.nspname as "schema",
    ...    e.enumlabel as "label"
    ...    FROM pg_catalog.pg_type t
    ...    LEFT JOIN pg_catalog.pg_namespace n ON n.oid \= t.typnamespace
    ...    LEFT JOIN pg_catalog.pg_enum e ON t.oid \= e.enumtypid
    ...    WHERE t.typtype \= 'e'
    ...    ORDER BY "schema", "name", e.oid
    ...    ;
    Should Stackql Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    {"digitalocean": { "username_var": "DUMMY_DIGITALOCEAN_USERNAME", "password_var": "DUMMY_DIGITALOCEAN_PASSWORD", "type": "basic", "valuePrefix": "TOTALLY_CONTRIVED "}}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    visible
    ...    stdout=${CURDIR}/tmp/Select-Enums-Returns-Header-Only.tmp

Select pg_attribute Returns Header Only
    Pass Execution If    "${SQL_BACKEND}" != "postgres_tcp"    This is a postgres only test
    ${inputStr} =    Catenate
    ...    SELECT a.attname 
    ...    FROM pg_attribute a 
    ...    JOIN ( 
    ...    SELECT unnest(ix.indkey) attnum, generate_subscripts(ix.indkey, 1) ord FROM pg_index ix WHERE ix.indrelid \= '13420' AND ix.indisprimary 
    ...    ) k 
    ...    ON a.attnum \= k.attnum 
    ...    WHERE a.attrelid \= '13420'  
    ...    ORDER BY k.ord
    ...    ;
    Should Stackql Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    {"digitalocean": { "username_var": "DUMMY_DIGITALOCEAN_USERNAME", "password_var": "DUMMY_DIGITALOCEAN_PASSWORD", "type": "basic", "valuePrefix": "TOTALLY_CONTRIVED "}}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    attname
    ...    stdout=${CURDIR}/tmp/Select-pg_attribute-Returns-Header-Only.tmp

Information Schema Schemata Returns Data
    Pass Execution If    "${SQL_BACKEND}" != "postgres_tcp"    This is a postgres only test
    ${inputStr} =    Catenate
    ...    SELECT * 
    ...    FROM information_schema.schemata
    ...    ;
    Should Stackql Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    {"digitalocean": { "username_var": "DUMMY_DIGITALOCEAN_USERNAME", "password_var": "DUMMY_DIGITALOCEAN_PASSWORD", "type": "basic", "valuePrefix": "TOTALLY_CONTRIVED "}}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    stackql
    ...    stdout=${CURDIR}/tmp/Information-Schema-Schemata-Returns-Data.tmp
