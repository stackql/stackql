
delete from "__iql__.control.gc.txn_table_x_ref"
where
iql_generation_id = ? and iql_session_id = ? and iql_transaction_id = ?
and
collected_dttm IS null
;