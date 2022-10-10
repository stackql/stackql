with tables_to_delete as
(
select 
"name" 
from sqlite_schema
where 
"type" = 'table'
and
"name" not like 'sqlite%'
and
"name" not like '__iql__%'
EXCEPT
select ss.name 
from sqlite_schema ss,
(
select 
discovery_name || '%' || cast(iql_discovery_generation_id as text) as name_like 
from "__iql__.control.discovery_generation"
) foo 
where 
ss.name like foo.name_like
and
ss.type = 'table'
)
select 
'drop table if exists "' || "name" || '" cascade;'
from
tables_to_delete
;