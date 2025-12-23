

# HTTP(S) servers for simulated integration / regression testing

## Flask

We have now migrated entirely to [flask](https://flask.palletsprojects.com/en/stable/), from the prior java [mockserver](https://www.mock-server.com/).  There is no disparaging of mockserver whatsoever; rather this was motivated in large part by different behaviour against versions of `java` / dependency libraries, also by the community support and knowledge base for `flask` and `jinja`.  That said, the mock defninitions to some degree are a holdover from `mockserver`; this should diminish over time.

One pertinent fact in life with `flask` is that processes die hard; so it generally pays this before testing mocks:

```bash
pgrep -f flask | xargs kill -9
```

### Running in docker

This is far and away the easiest option, given you have docker on your system.  You must let the sequence run before starting any `stackql` client, or the TLS credentials will be overwritten.  Then:

```bash

docker compose -f docker-compose-testing.yml up -d --build --force-recreate

```

To kill all:

```bash

docker compose -f docker-compose-testing.yml down --volumes

```

### To Run Natively

In order to get the environmental variables required, you can go to the repository root and then `source cicd/scripts/context.sh`, or set manually; hopefully self-explanatory.

GCP mocks:

```bash
flask --app=${REPOSITORY_ROOT}/test/python/stackql_test_tooling/flask/gcp/app run --cert=${REPOSITORY_ROOT}/test/server/mtls/credentials/pg_server_cert.pem --key=${REPOSITORY_ROOT}/test/server/mtls/credentials/pg_server_key.pem --host 0.0.0.0 --port  1080
```

Azure mocks:

```bash
flask --app=${REPOSITORY_ROOT}/test/python/stackql_test_tooling/flask/azure/app run --cert=${REPOSITORY_ROOT}/test/server/mtls/credentials/pg_server_cert.pem --key=${REPOSITORY_ROOT}/test/server/mtls/credentials/pg_server_key.pem --host 0.0.0.0 --port 1095
```

Okta mocks:

```bash
flask --app=${REPOSITORY_ROOT}/test/python/stackql_test_tooling/flask/okta/app run --cert=${REPOSITORY_ROOT}/test/server/mtls/credentials/pg_server_cert.pem --key=${REPOSITORY_ROOT}/test/server/mtls/credentials/pg_server_key.pem --host 0.0.0.0  --port 1090
```

AWS mocks:

```bash
flask --app=${REPOSITORY_ROOT}/test/python/stackql_test_tooling/flask/aws/app run --cert=${REPOSITORY_ROOT}/test/server/mtls/credentials/pg_server_cert.pem --key=${REPOSITORY_ROOT}/test/server/mtls/credentials/pg_server_key.pem --host 0.0.0.0  --port 1091
```

Github mocks:

```bash
flask --app=${REPOSITORY_ROOT}/test/python/stackql_test_tooling/flask/github/app run --cert=${REPOSITORY_ROOT}/test/server/mtls/credentials/pg_server_cert.pem --key=${REPOSITORY_ROOT}/test/server/mtls/credentials/pg_server_key.pem --host 0.0.0.0 --port 1093
```

Sumologic mocks:

```bash
flask --app=${REPOSITORY_ROOT}/test/python/stackql_test_tooling/flask/sumologic/app run --cert=${REPOSITORY_ROOT}/test/server/mtls/credentials/pg_server_cert.pem --key=${REPOSITORY_ROOT}/test/server/mtls/credentials/pg_server_key.pem --host 0.0.0.0 --port 1096
```

Digitalocean mocks:

```bash
flask --app=${REPOSITORY_ROOT}/test/python/stackql_test_tooling/flask/digitalocean/app run --cert=${REPOSITORY_ROOT}/test/server/mtls/credentials/pg_server_cert.pem --key=${REPOSITORY_ROOT}/test/server/mtls/credentials/pg_server_key.pem --host 0.0.0.0 --port 1097
```

`googleadmin` mocks:

```bash
flask --app=${REPOSITORY_ROOT}/test/python/stackql_test_tooling/flask/googleadmin/app run --cert=${REPOSITORY_ROOT}/test/server/mtls/credentials/pg_server_cert.pem --key=${REPOSITORY_ROOT}/test/server/mtls/credentials/pg_server_key.pem --host 0.0.0.0 --port 1098
```

`k8s` mocks:

```bash
flask --app=${REPOSITORY_ROOT}/test/python/stackql_test_tooling/flask/k8s/app run --cert=${REPOSITORY_ROOT}/test/server/mtls/credentials/pg_server_cert.pem --key=${REPOSITORY_ROOT}/test/server/mtls/credentials/pg_server_key.pem --host 0.0.0.0 --port 1092
```

stackql auth testing mocks:

```bash
flask --app=${REPOSITORY_ROOT}/test/python/stackql_test_tooling/flask/static_auth/app run --cert=${REPOSITORY_ROOT}/test/server/mtls/credentials/pg_server_cert.pem --key=${REPOSITORY_ROOT}/test/server/mtls/credentials/pg_server_key.pem --host 0.0.0.0 --port  1170
```

Token server mocks:

```bash
flask --app=${REPOSITORY_ROOT}/test/python/stackql_test_tooling/flask/oauth2/token_srv run --cert=${REPOSITORY_ROOT}/test/server/mtls/credentials/pg_server_cert.pem --key=${REPOSITORY_ROOT}/test/server/mtls/credentials/pg_server_key.pem --host 0.0.0.0 --port  2091
```

Registry mocks:

```bash
flask --app=${REPOSITORY_ROOT}/test/python/stackql_test_tooling/flask/registry/app run --host 0.0.0.0 --port 1094
```


### Manually testing mocks

With embedded `sqlite` (default), from the root of this repository:

```bash
source cicd/scripts/testing-env.sh

stackql --registry="${stackqlMockedRegistryStr}" --auth="${stackqlAuthStr}" --tls.allowInsecure shell
```

With `postgres`, from the root of this repository:

```bash
docker compose -f docker-compose-externals.yml up postgres_stackql -d

source cicd/scripts/testing-env.sh

stackql --registry="${stackqlMockedRegistryStr}" --tls.allowInsecure --sqlBackend="{ \"dbEngine\": \"postgres_tcp\", \"sqlDialect\": \"postgres\", \"dsn\": \"postgres://stackql:stackql@127.0.0.1:7432/stackql\" }" shell
```

## Sources of Mock Data

There are some decent examples in vendor documentation, eg:

- [Azure vendor documenation](https://learn.microsoft.com/en-us/rest/api/azure/).


