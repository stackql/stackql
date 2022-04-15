
# Google provider examples

## Assumptions

- `stackql` is in your `${PATH}`.
- Authentication particulars are supplied as a json string in the arg `--auth`.  Per provider, you supply a key/val pair.  The val iteslf is a json string, optionally specifying `type` (defaulted to `service_account`, which represents a google service account key). The val minimally contains either:
    - An appropriate key file at the file location `{ "credentialsfilepath": "/PATH/TO/KEY/FILE" }`.  For example, with the google provider, one might use a service account json key.
    - An appropriate key plaintext stored in an (exported) environment variable.  Eg: `{ "credentialsenvvar": "OKTA_SECRET_KEY" }`.  For example, with the google provider, one might use a service account json key.

If using `service account` auth against the `google` provider, then no ancillary information is required.  If howevere, you are using another key type / provider, then more runtime information is required, eg:

## Running stackql

Simplest example is using the interactive shell.

Google:

```sh

export OKTA_SECRET_KEY="$(cat ${HOME}/stackql/stackql-devel/keys/okta-token.txt)"

export AUTH_STR='{ "google": { "credentialsfilepath": "'${HOME}'/stackql/stackql-devel/keys/sa-key.json", "type": "service_account" }, "okta": { "credentialsenvvar": "OKTA_SECRET_KEY", "type": "api_key" } }'

./stackql shell --auth="${AUTH_STR}"


```

## Running in server mode

**Note that this feature is in alpha**, as discussed in [the developer guide](/docs/developer_guide.md#server-mode).


To run a `stackql` server over the `postgres` wire protocol (without client authentication), from the `build` directory.

```bash
./stackql --auth="${AUTH_STR}" --registry="${REG_STR}" srv
```

And then, using the `psql` client:

```bash
psql -d "host=127.0.0.1 port=5466 user=silly dbname=silly"
```

To run using mTLS auth, first prepare collateral as per [the mTLS setup README](/test/server/mtls/README.md).  Important to define the env var `CLIENT_CERT` in the shell session you will use to run the server.

Then:

```bash
STACKQL_SRV_TLS_CFG='{ "keyFilePath": "../test/server/mtls/credentials/pg_server_key.pem", "certFilePath": "../test/server/mtls/credentials/pg_server_cert.pem", "clientCAs": [ "'${CLIENT_CERT}'" ] }'

./stackql --auth="${AUTH_STR}" --registry="${REG_STR}"  --pg.tls="${STACKQL_SRV_TLS_CFG}" 
```

And then, using the `psql` client (from same directory; `build`):

```bash
psql -d "host=127.0.0.1 port=5466 user=silly dbname=silly sslmode=verify-full sslcert=../test/server/mtls/credentials/pg_client_cert.pem sslkey=../test/server/mtls/credentials/pg_client_key.pem sslrootcert=../test/server/mtls/credentials/pg_server_cert.pem"
```

## Queries

### SELECT

```
stackql \
  --auth="${AUTH_STR}" exec  \
  "select * from compute.instances WHERE zone = '${YOUR_GOOGLE_ZONE}' AND project = '${YOUR_GOOGLE_PROJECT}' ;" ; echo

```

Or...

```
stackql \
  --auth="${AUTH_STR}" exec  \
  "select selfLink, projectNumber from storage.buckets WHERE location = '${YOUR_GOOGLE_ZONE}' AND project = '${YOUR_GOOGLE_PROJECT}' ;" ; echo

```

### Joins

Only inner joins suported for now.


Self join:
```sql
select d1.name as n, d1.id, d2.id as d2_id from google.compute.disks d1 inner join google.compute.disks d2 on d1.id = d2.id where d1.project = 'testing-project' and d1.zone = 'australia-southeast1-b' and d2.project = 'testing-project' and d2.zone = 'australia-southeast1-b';
```

Three way join:
```sql
select 
  d1.name as n, 
  d1.id, 
  n1.description, 
  s1.description as s1_description 
from 
  google.compute.disks d1 
  inner join google.compute.networks n1 
  on 
    d1.name = n1.name 
  inner join 
  google.compute.subnetworks s1 
  on 
    d1.name = s1.name  
where 
  d1.project = 'testing-project' and 
  d1.zone = 'australia-southeast1-b' and 
  n1.project = 'testing-project' 
  and s1.project = 'testing-project' 
  and s1.region = 'australia-southeast1'
;
```

Cross-provider join:
```sql
select d1.name, d1.id, d2.name as d2_name, d2.status, d2.label, d2.id as d2_id from google.compute.disks d1 inner join okta.application.apps d2 on d1.name = d2.label where d1.project = 'testing-project' and d1.zone = 'australia-southeast1-b' and d2.subdomain = 'my-subdomain' order by d1.name;
```

### SHOW SERVICES

```
stackql --approot=../test/.stackql \
  --configfile=../test/.stackqlrc exec \
  "SHOW SERVICES from google ;" ; echo

```

### COMPLEX INSERT

```
insert into google.compute.disks(project, zone, data__name) SELECT 'lab-kr-network-01', 'australia-southeast1-a', name || '-new-disk01' as name from google.compute.disks where project = 'lab-kr-network-01' and zone =  'australia-southeast1-a' limit 2;
```

## okta

### app insert

```
insert into okta.application.apps(subdomain, data__name, data__label, data__signOnMode, data__settings) SELECT 'dev-79923018-admin', 'template_basic_auth', 'some other4 new app', 'BASIC_AUTH', '{ "app": { "authURL": "https://example.com/auth.html", "url": "https://example.com/bookmark.html" } }';
```

### aliased table select

```
select * from okta.application.apps;
```