

# Examples

This directory contains examples of:

- [direct use of stackql in the stackql subdirectory](/docs/examples/stackql)
- Registry examples:
    - [empty registry](/docs/examples/empty-registry).
    - [a reference implementation of a simple registry](/docs/examples/registry).
- Examples of [stackql invocation using various tooling in the scripts directory](/docs/examples/scripts).

## Assumptions for direct stackql invocation examples

- `stackql` is in your `${PATH}`.
- Authentication particulars are supplied as a json string in the arg `--auth`.  Per provider, you supply a key/val pair.  The val iteslf is a json string, optionally specifying `type` (defaulted to `service_account`, which represents a google service account key). The val minimally contains either:
    - An appropriate key file at the file location `{ "credentialsfilepath": "/PATH/TO/KEY/FILE" }`.  For example, with the google provider, one might use a service account json key.
    - An appropriate key plaintext stored in an (exported) environment variable.  Eg: `{ "credentialsenvvar": "OKTA_SECRET_KEY" }`.  For example, with the google provider, one might use a service account json key.

If using `service account` auth against the `google` provider, then no ancillary information is required.  If however, you are using another key type / provider, then more runtime information is required, eg:

## Running stackql

Simplest example is using the interactive shell.

Google:

```sh

export OKTA_SECRET_KEY="$(cat ${HOME}/stackql/stackql-devel/cicd/keys/okta-token.txt)"

export AUTH_STR='{ "google": { "credentialsfilepath": "'${HOME}'/stackql/stackql-devel/cicd/keys/sa-key.json", "type": "service_account" }, "okta": { "credentialsenvvar": "OKTA_SECRET_KEY", "type": "api_key" } }'

./stackql shell --auth="${AUTH_STR}"


```

## Queries

### SELECT

```
stackql \
  --auth="${AUTH_STR}" exec  \
  "select * from google.compute.instances WHERE zone = '${YOUR_GOOGLE_ZONE}' AND project = '${YOUR_GOOGLE_PROJECT}' ;" ; echo

```

Or...

```
stackql \
  --auth="${AUTH_STR}" exec  \
  "select selfLink, projectNumber from google.storage.buckets WHERE location = '${YOUR_GOOGLE_ZONE}' AND project = '${YOUR_GOOGLE_PROJECT}' ;" ; echo

```

For example:
```sql
select d1.name, d1.id from google.compute.disks d1 where d1.project = 'lab-kr-network-01' and d1.zone = 'australia-southeast1-a' ;
```

### Joins

- Please [follow this link for a self join](/docs/examples/stackql/self-join.sql).
- Please [follow this link for a three way join](/docs/examples/stackql/three-way-join.sql).
- Please [follow this link for a cross-provider join](/docs/examples/stackql/cross-provider-join.sql).

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

## okta query examples

### app insert

```
insert into okta.application.apps(subdomain, data__name, data__label, data__signOnMode, data__settings) SELECT 'dev-79923018-admin', 'template_basic_auth', 'some other4 new app', 'BASIC_AUTH', '{ "app": { "authURL": "https://example.com/auth.html", "url": "https://example.com/bookmark.html" } }';
```

### aliased table select

```
select * from okta.application.apps;
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

### Access from python

- Python access requires a running server, simplest is `stackql srv` which will serve on default port with zero credentials.
- For integration testing, we use `psycopg` which is currently `v3`.
    - Run / adapt [this script](/docs/examples/scripts/python/psycopg3_scratchpad.py) for troubleshooting `psycopg` `v3`. 
- `superset` uses `sqlalchemy` which consumes `psycopg2` **note different version**.
    - Run / adapt [this script](/docs/examples/scripts/python/psycopg2_scratchpad.py) for troubleshooting `psycopg2`. 
    - Run / adapt [this script](/docs/examples/scripts/python/sqlalchemy_scratchpad.py) for troubleshooting `sqlalchemy`. 

### With docker

From the repository root directory.

```bash
docker compose up stackqlsrv
```

```bash
psql -d "host=127.0.0.1 port=5576 user=silly dbname=silly sslmode=verify-full sslcert=./cicd/vol/srv/credentials/pg_client_cert.pem sslkey=./cicd/vol/srv/credentials/pg_client_key.pem sslrootcert=./cicd/vol/srv/credentials/pg_server_cert.pem"
```
