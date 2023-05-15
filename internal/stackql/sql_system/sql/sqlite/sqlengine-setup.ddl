
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
  view_ddl
) 
VALUES (
  'stackql_repositories',
  'select id, name, url from github.repos.repos where org = ''stackql'';'
)
;


INSERT OR IGNORE INTO "__iql__.views" (
  view_name,
  view_ddl
) 
VALUES (
  'aws_ec2_all_volumes',
  'select 
    ''ap-southeast-2'' AS aws_region, 
    volumeId, 
    encrypted, 
    size
  from aws.ec2.volumes 
  where region = ''ap-southeast-2'' 
  UNION 
  SELECT 
    ''ap-southeast-1'' AS aws_region, 
    volumeId, 
    encrypted, 
    size 
  from aws.ec2.volumes 
  where region = ''ap-southeast-1''
  UNION 
  SELECT 
    ''ap-northeast-1'' AS aws_region, 
    volumeId, 
    encrypted, 
    size 
  from aws.ec2.volumes 
  where region = ''ap-northeast-1''
  UNION 
  SELECT 
    ''ap-northeast-2'' AS aws_region, 
    volumeId, 
    encrypted, 
    size 
  from aws.ec2.volumes 
  where region = ''ap-northeast-2''
  UNION 
  SELECT 
    ''ap-northeast-3'' AS aws_region, 
    volumeId, 
    encrypted, 
    size 
  from aws.ec2.volumes 
  where region = ''ap-northeast-3''
  UNION 
  SELECT 
    ''ap-south-1'' AS aws_region, 
    volumeId, 
    encrypted, 
    size 
  from aws.ec2.volumes 
  where region = ''ap-south-1''
  UNION 
  SELECT 
    ''us-east-1'' AS aws_region, 
    volumeId, 
    encrypted, 
    size 
  from aws.ec2.volumes 
  where region = ''us-east-1''
  UNION 
  SELECT 
    ''us-east-2'' AS aws_region, 
    volumeId, 
    encrypted, 
    size 
  from aws.ec2.volumes 
  where region = ''us-east-2''
  UNION
  SELECT 
    ''us-west-1'' AS aws_region, 
    volumeId, 
    encrypted, 
    size 
  from aws.ec2.volumes 
  where region = ''us-west-1''
  UNION 
  SELECT 
    ''us-west-2'' AS aws_region, 
    volumeId, 
    encrypted, 
    size 
  from aws.ec2.volumes 
  where region = ''us-west-2''
  UNION 
  SELECT 
    ''ca-central-1'' AS aws_region, 
    volumeId, 
    encrypted, 
    size 
  from aws.ec2.volumes 
  where region = ''ca-central-1''
  UNION 
  SELECT 
    ''sa-east-1'' AS aws_region, 
    volumeId, 
    encrypted, 
    size 
  from aws.ec2.volumes 
  where region = ''sa-east-1''
  UNION 
  SELECT 
    ''eu-central-1'' AS aws_region, 
    volumeId, 
    encrypted, 
    size 
  from aws.ec2.volumes 
  where region = ''eu-central-1''
  UNION 
  SELECT 
    ''eu-north-1'' AS aws_region, 
    volumeId, 
    encrypted, 
    size 
  from aws.ec2.volumes 
  where region = ''eu-north-1''
  UNION 
  SELECT 
    ''eu-west-1'' AS aws_region, 
    volumeId, 
    encrypted, 
    size 
  from aws.ec2.volumes 
  where region = ''eu-west-1''
  UNION 
  SELECT 
    ''eu-west-2'' AS aws_region, 
    volumeId, 
    encrypted, 
    size 
  from aws.ec2.volumes 
  where region = ''eu-west-2''
  UNION 
  SELECT 
    ''eu-west-3'' AS aws_region, 
    volumeId, 
    encrypted, 
    size 
  from aws.ec2.volumes 
  where region = ''eu-west-3''
  ORDER BY size DESC
  ;'
)
;

INSERT OR IGNORE INTO "__iql__.views" (
  view_name,
  view_ddl
) 
VALUES (
  'aws_cc_bucket_detail',
  '
  SELECT 
  JSON_EXTRACT(Properties, ''$.Arn'') as Arn,
  JSON_EXTRACT(Properties, ''$.BucketName'') as BucketName,
  JSON_EXTRACT(Properties, ''$.DomainName'') as DomainName,
  JSON_EXTRACT(Properties, ''$.RegionalDomainName'') as RegionalDomainName,
  JSON_EXTRACT(Properties, ''$.DualStackDomainName'') as DualStackDomainName,
  JSON_EXTRACT(Properties, ''$.WebsiteURL'') as WebsiteURL,
  JSON_EXTRACT(Properties, ''$.OwnershipControls.Rules[0].ObjectOwnership'') as ObjectOwnership,
  IIF(JSON_EXTRACT(Properties, ''$.PublicAccessBlockConfiguration.RestrictPublicBuckets'') = 0, ''false'', ''true'') as RestrictPublicBuckets,
  IIF(JSON_EXTRACT(Properties, ''$.PublicAccessBlockConfiguration.BlockPublicPolicy'') = 0, ''false'', ''true'') as BlockPublicPolicy,
  IIF(JSON_EXTRACT(Properties, ''$.PublicAccessBlockConfiguration.BlockPublicAcls'') = 0, ''false'', ''true'') as BlockPublicAcls,
  IIF(JSON_EXTRACT(Properties, ''$.PublicAccessBlockConfiguration.IgnorePublicAcls'') = 0, ''false'', ''true'') as IgnorePublicAcls,
  JSON_EXTRACT(Properties, ''$.Tags'') as Tags
  FROM aws.cloud_control.resources WHERE region = ''ap-southeast-2'' and data__TypeName = ''AWS::S3::Bucket'' and data__Identifier = ''stackql-trial-bucket-01''
  ;'
);

INSERT OR IGNORE INTO "__iql__.views" (
  view_name,
  view_ddl
) 
VALUES (
  'aws_cc_bucket_unfiltered',
  '
  SELECT 
  JSON_EXTRACT(Properties, ''$.Arn'') as Arn,
  JSON_EXTRACT(Properties, ''$.BucketName'') as BucketName,
  JSON_EXTRACT(Properties, ''$.DomainName'') as DomainName,
  JSON_EXTRACT(Properties, ''$.RegionalDomainName'') as RegionalDomainName,
  JSON_EXTRACT(Properties, ''$.DualStackDomainName'') as DualStackDomainName,
  JSON_EXTRACT(Properties, ''$.WebsiteURL'') as WebsiteURL,
  JSON_EXTRACT(Properties, ''$.OwnershipControls.Rules[0].ObjectOwnership'') as ObjectOwnership,
  IIF(JSON_EXTRACT(Properties, ''$.PublicAccessBlockConfiguration.RestrictPublicBuckets'') = 0, ''false'', ''true'') as RestrictPublicBuckets,
  IIF(JSON_EXTRACT(Properties, ''$.PublicAccessBlockConfiguration.BlockPublicPolicy'') = 0, ''false'', ''true'') as BlockPublicPolicy,
  IIF(JSON_EXTRACT(Properties, ''$.PublicAccessBlockConfiguration.BlockPublicAcls'') = 0, ''false'', ''true'') as BlockPublicAcls,
  IIF(JSON_EXTRACT(Properties, ''$.PublicAccessBlockConfiguration.IgnorePublicAcls'') = 0, ''false'', ''true'') as IgnorePublicAcls,
  JSON_EXTRACT(Properties, ''$.Tags'') as Tags
  FROM aws.cloud_control.resources WHERE region = ''ap-southeast-2'' and data__TypeName = ''AWS::S3::Bucket''
  ;'
);

CREATE TABLE IF NOT EXISTS "__iql__.external.columns" (
   iql_column_id BIGSERIAL PRIMARY KEY
  ,connection_name TEXT
  ,catalog_name TEXT
  ,schema_name TEXT
  ,table_name TEXT
  ,column_name TEXT
  ,column_type TEXT
  ,ordinal_position INT
  ,"oid" INT
  ,column_width INT
  ,column_precision TEXT
  ,UNIQUE(connection_name, catalog_name, schema_name, table_name, column_name)
)
;
