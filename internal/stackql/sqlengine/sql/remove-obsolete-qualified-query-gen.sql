
select distinct
  'with table_exists as (SELECT count(*) FROM sqlite_master WHERE type=''table'' AND name=''' || table_name || ''') delete from "' || table_name || 
  '" where 1 in table_exists and iql_generation_id = ? and iql_session_id = ? and iql_txn_id = ? ;'
from "__iql__.control.gc.txn_table_x_ref"
where
collected_dttm IS null
;