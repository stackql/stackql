
CREATE TABLE IF NOT EXISTS "__iql__.control.generation" (
   iql_generation_id INTEGER PRIMARY KEY AUTOINCREMENT
  ,generation_description TEXT
  ,created_dttm INTEGER not null
  ,collected_dttm INTEGER default null
)
;

CREATE INDEX IF NOT EXISTS "idx.__iql__.control.generation.created_dttm" 
ON "__iql__.control.generation" (created_dttm)
;

CREATE TABLE IF NOT EXISTS "__iql__.control.discovery_generation" (
   iql_discovery_generation_id INTEGER PRIMARY KEY AUTOINCREMENT
  ,discovery_name TEXT NOT NULL
  ,discovery_generation_description TEXT
  ,created_dttm INTEGER not null
  ,collected_dttm INTEGER default null
)
;

CREATE INDEX IF NOT EXISTS "idx.__iql__.control.discovery_generation.created_dttm" 
ON "__iql__.control.discovery_generation" (created_dttm)
;

CREATE TABLE IF NOT EXISTS "__iql__.control.session" (
   iql_session_id INTEGER PRIMARY KEY AUTOINCREMENT
  ,iql_generation_id INTEGER NOT NULL
  ,session_description TEXT
  ,created_dttm INTEGER not null
  ,collected_dttm INTEGER default null
  ,FOREIGN KEY(iql_generation_id) REFERENCES "__iql__.control.generation"(iql_generation_id)
)
;

CREATE INDEX IF NOT EXISTS "idx.__iql__.control.session.created_dttm" 
ON "__iql__.control.session" (created_dttm)
;

CREATE TABLE IF NOT EXISTS "__iql__.cache.key_val" (
   k TEXT NOT NULL UNIQUE
  ,v BLOB
  ,tablespace TEXT
  ,tablespace_id INTEGER 
);

CREATE TABLE IF NOT EXISTS "__iql__.control.gc.txn_table_x_ref" (
   iql_generation_id INTEGER not null
  ,iql_session_id INTEGER not null
  ,iql_transaction_id INTEGER not null
  ,table_name TEXT not null
  ,created_dttm not null default CURRENT_TIMESTAMP
  ,collected_dttm INTEGER default null
  ,PRIMARY KEY (iql_generation_id, iql_session_id, iql_transaction_id, table_name)
)
;
