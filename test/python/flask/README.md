

# HTTP(S) servers for simulated integration / regression testing

## Flask

We have now migrated entirely to [flask](https://flask.palletsprojects.com/en/stable/), from the prior java [mockserver](https://www.mock-server.com/).  There is no disparaging of mockserver whatsoever; rather this was motivated in large part by different behaviour against versions of `java` / dependency libraries, also by the community support and knowledge base for `flask` and `jinja`.  That said, the mock defninitions to some degree are a holdover from `mockserver`; this should diminish over time.

One pertinent fact in life with `flask` is that processes die hard; so it generally pays this before testing mocks:

```bash
pgrep -f flask | xargs kill -9
```


### To Run

GCP mocks:

```bash
flask --app=${HOME}/stackql/stackql-devel/test/python/flask/gcp/app run --cert=${HOME}/stackql/stackql-devel/test/server/mtls/credentials/pg_server_cert.pem --key=${HOME}/stackql/stackql-devel/test/server/mtls/credentials/pg_server_key.pem  --port  1080
```

Azure mocks:

```bash
flask --app=${HOME}/stackql/stackql-devel/test/python/flask/azure/app run --cert=${HOME}/stackql/stackql-devel/test/server/mtls/credentials/pg_server_cert.pem --key=${HOME}/stackql/stackql-devel/test/server/mtls/credentials/pg_server_key.pem --port 1095
```

Okta mocks:

```bash
flask --app=${HOME}/stackql/stackql-devel/test/python/flask/okta/app run --cert=${HOME}/stackql/stackql-devel/test/server/mtls/credentials/pg_server_cert.pem --key=${HOME}/stackql/stackql-devel/test/server/mtls/credentials/pg_server_key.pem  --port 1090
```

AWS mocks:

```bash
flask --app=${HOME}/stackql/stackql-devel/test/python/flask/aws/app run --cert=${HOME}/stackql/stackql-devel/test/server/mtls/credentials/pg_server_cert.pem --key=${HOME}/stackql/stackql-devel/test/server/mtls/credentials/pg_server_key.pem  --port 1091
```

Github mocks:

```bash
flask --app=${HOME}/stackql/stackql-devel/test/python/flask/github/app run --cert=${HOME}/stackql/stackql-devel/test/server/mtls/credentials/pg_server_cert.pem --key=${HOME}/stackql/stackql-devel/test/server/mtls/credentials/pg_server_key.pem  --port 1093
```

Sumologic mocks:

```bash
flask --app=${HOME}/stackql/stackql-devel/test/python/flask/okta/app run --cert=${HOME}/stackql/stackql-devel/test/server/mtls/credentials/pg_server_cert.pem --key=${HOME}/stackql/stackql-devel/test/server/mtls/credentials/pg_server_key.pem  --port 1096
```

Digitalocean mocks:

```bash
flask --app=${HOME}/stackql/stackql-devel/test/python/flask/digitalocean/app run --cert=${HOME}/stackql/stackql-devel/test/server/mtls/credentials/pg_server_cert.pem --key=${HOME}/stackql/stackql-devel/test/server/mtls/credentials/pg_server_key.pem  --port 1097
```

`googleadmin` mocks:

```bash
flask --app=${HOME}/stackql/stackql-devel/test/python/flask/googleadmin/app run --cert=${HOME}/stackql/stackql-devel/test/server/mtls/credentials/pg_server_cert.pem --key=${HOME}/stackql/stackql-devel/test/server/mtls/credentials/pg_server_key.pem  --port 1098
```

stackql auth testing mocks:

```bash
flask --app=${HOME}/stackql/stackql-devel/test/python/flask/static_auth/app run --cert=${HOME}/stackql/stackql-devel/test/server/mtls/credentials/pg_server_cert.pem --key=${HOME}/stackql/stackql-devel/test/server/mtls/credentials/pg_server_key.pem  --port  1170
```


### Manually testing mocks

With embedded `sqlite` (default):

```bash
export workspaceFolder='/path/to/repository/root'  # change this

stackql --registry="{ \"url\": \"file://${workspaceFolder}/test/registry-mocked\", \"localDocRoot\": \"${workspaceFolder}/test/registry-mocked\", \"verifyConfig\": { \"nopVerify\": true } }" --tls.allowInsecure shell
```

With `postgres`:

```bash
docker compose -f docker-compose-externals.yml up postgres_stackql -d

export workspaceFolder='/path/to/repository/root'  # change this

stackql --registry="{ \"url\": \"file://${workspaceFolder}/test/registry-mocked\", \"localDocRoot\": \"${workspaceFolder}/test/registry-mocked\", \"verifyConfig\": { \"nopVerify\": true } }" --tls.allowInsecure --sqlBackend="{ \"dbEngine\": \"postgres_tcp\", \"sqlDialect\": \"postgres\", \"dsn\": \"postgres://stackql:stackql@127.0.0.1:7432/stackql\" }" shell
```