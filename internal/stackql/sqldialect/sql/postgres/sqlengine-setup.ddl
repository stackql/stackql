
CREATE TABLE IF NOT EXISTS "__iql__.control.generation" (
   iql_generation_id BIGSERIAL PRIMARY KEY
  ,generation_description TEXT
  ,created_dttm TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
  ,collected_dttm TIMESTAMP WITH TIME ZONE DEFAULT null
)
;

CREATE INDEX IF NOT EXISTS "idx.__iql__.control.generation.created_dttm" 
ON "__iql__.control.generation" (created_dttm)
;

CREATE TABLE IF NOT EXISTS "__iql__.control.discovery_generation" (
   iql_discovery_generation_id BIGSERIAL PRIMARY KEY
  ,discovery_name TEXT NOT NULL
  ,discovery_generation_description TEXT
  ,created_dttm TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
  ,collected_dttm TIMESTAMP WITH TIME ZONE DEFAULT null
)
;

CREATE INDEX IF NOT EXISTS "idx.__iql__.control.discovery_generation.created_dttm" 
ON "__iql__.control.discovery_generation" (created_dttm)
;

CREATE TABLE IF NOT EXISTS "__iql__.control.session" (
   iql_session_id BIGSERIAL PRIMARY KEY
  ,iql_generation_id INTEGER NOT NULL
  ,session_description TEXT
  ,created_dttm TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
  ,collected_dttm TIMESTAMP WITH TIME ZONE DEFAULT null
  ,FOREIGN KEY(iql_generation_id) REFERENCES "__iql__.control.generation"(iql_generation_id)
)
;

CREATE INDEX IF NOT EXISTS "idx.__iql__.control.session.created_dttm" 
ON "__iql__.control.session" (created_dttm)
;

CREATE TABLE IF NOT EXISTS "__iql__.cache.key_val" (
   k TEXT NOT NULL UNIQUE
  ,v BYTEA
  ,tablespace TEXT
  ,tablespace_id INTEGER 
);

CREATE TABLE IF NOT EXISTS "__iql__.control.gc.txn_table_x_ref" (
   iql_generation_id INTEGER not null
  ,iql_session_id INTEGER not null
  ,iql_transaction_id INTEGER not null
  ,table_name TEXT not null
  ,created_dttm TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
  ,collected_dttm TIMESTAMP WITH TIME ZONE DEFAULT null
  ,PRIMARY KEY (iql_generation_id, iql_session_id, iql_transaction_id, table_name)
)
;

CREATE TABLE IF NOT EXISTS "__iql__.control.gc.rings" (
   ring_id BIGSERIAL PRIMARY KEY
  ,ring_name TEXT not null UNIQUE
  ,current_value INTEGER not null DEFAULT 0
  ,current_offset INTEGER not null DEFAULT 0
  ,width_bits INTEGER not null DEFAULT 32
  ,created_dttm TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
  ,collected_dttm TIMESTAMP WITH TIME ZONE DEFAULT null
)
;

CREATE INDEX IF NOT EXISTS "idx.__iql__.control.gc.rings.ring_name" 
ON "__iql__.control.gc.rings" (ring_name)
;

INSERT INTO "__iql__.control.gc.rings" (ring_name) 
VALUES ('transaction_id')
ON CONFLICT (ring_name) DO NOTHING
;

INSERT INTO "__iql__.control.gc.rings" (ring_name) 
VALUES ('session_id')
ON CONFLICT (ring_name) DO NOTHING
;


