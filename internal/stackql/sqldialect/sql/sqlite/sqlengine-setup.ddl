
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

CREATE TABLE IF NOT EXISTS "__iql__.control.gc.rings" (
   ring_id INTEGER PRIMARY KEY AUTOINCREMENT
  ,ring_name TEXT not null UNIQUE
  ,current_value INTEGER not null DEFAULT 0
  ,current_offset INTEGER not null DEFAULT 0
  ,width_bits INTEGER not null DEFAULT 32
  ,created_dttm DateTime not null default CURRENT_TIMESTAMP
  ,collected_dttm DateTime default null
)
;

CREATE INDEX IF NOT EXISTS "idx.__iql__.control.gc.rings.ring_name" 
ON "__iql__.control.gc.rings" (ring_name)
;

INSERT OR IGNORE INTO "__iql__.control.gc.rings" (ring_name) VALUES ('transaction_id');

INSERT OR IGNORE INTO "__iql__.control.gc.rings" (ring_name) VALUES ('session_id');

CREATE TABLE IF NOT EXISTS "__iql__.views" (
   iql_view_id INTEGER PRIMARY KEY AUTOINCREMENT
  ,view_name TEXT NOT NULL UNIQUE
  ,view_ddl TEXT
  ,view_stackql_ddl TEXT
  ,created_dttm DateTime not null default CURRENT_TIMESTAMP
  ,deleted_dttm DateTime DEFAULT null
)
;

CREATE INDEX IF NOT EXISTS "idx.__iql__.views" 
ON "__iql__.views" (view_name)
;

INSERT OR IGNORE INTO "__iql__.views" (
  view_name,
  view_ddl,
  view_stackql_ddl
) 
VALUES (
  'stackql_providers',
  'SHOW PROVIDERS',
  'SHOW PROVIDERS'
)
;

