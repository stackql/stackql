<!-- language: lang-none -->

![Platforms](https://img.shields.io/badge/platform-windows%20macos%20linux-brightgreen)
![Go](https://github.com/stackql/stackql/workflows/Go/badge.svg)
![License](https://img.shields.io/github/license/stackql/stackql)
![Lines](https://img.shields.io/tokei/lines/github/stackql/stackql)  
[![StackQL](https://stackql.io/img/stackql-banner.png)](https://stackql.io/)  


# Deploy, Manage and Query Cloud Infrastructure using SQL

[[Documentation](https://docs.stackql.io/)]  [[Developer Guide](/docs/developer_guide.md)] [[BYO Providers](/docs/registry_contribution.md)]

## Cloud infrastructure coding using SQL

> StackQL allows you to create, modify and query the state of services and resources across all three major public cloud providers (Google, AWS and Azure) using a common, widely known DSL...SQL.

----
## Its as easy as...
    SELECT * FROM google.compute.instances WHERE zone = 'australia-southeast1-b' AND project = 'my-project' ;

----

```
select d1.name, d1.id, d2.name as d2_name, d2.status, d2.label, d2.id as d2_id from google.compute.disks d1 inner join okta.application.apps d2 on d1.name = d2.label where d1.project = 'lab-kr-network-01' and d1.zone = 'australia-southeast1-a' and d2.subdomain = 'dev-79923018-admin';
```

## Provider development

Keen to expose some new functionality though `stackql`?  We are very keen on this!  

Please see [registry_contribution.md](/docs/registry_contribution.md).

---

## Design

[HLDD](/docs/high-level-design.md)


---

## Providers

Please see [the stackql-provider-registry repository](https://github.com/stackql/stackql-provider-registry)

Providers include:

- Google.
- Okta.
- ...

---

## Build

### Native Build

#### In shell

```bash
env CGO_ENABLED=1 go build \
  --tags "json1 sqleanall" \
  -ldflags "-X github.com/stackql/stackql/internal/stackql/cmd.BuildMajorVersion=${BUILDMAJORVERSION:-1} \
  -X github.com/stackql/stackql/internal/stackql/cmd.BuildMinorVersion=${BUILDMINORVERSION:-1} \
  -X github.com/stackql/stackql/internal/stackql/cmd.BuildPatchVersion=${BUILDPATCHVERSION:-1} \
  -X github.com/stackql/stackql/internal/stackql/cmd.BuildCommitSHA=$BUILDCOMMITSHA \
  -X github.com/stackql/stackql/internal/stackql/cmd.BuildShortCommitSHA=$BUILDSHORTCOMMITSHA \
  -X \"github.com/stackql/stackql/internal/stackql/cmd.BuildDate=$BUILDDATE\" \
  -X \"stackql/internal/stackql/planbuilder.PlanCacheEnabled=$PLANCACHEENABLED\" \
  -X github.com/stackql/stackql/internal/stackql/cmd.BuildPlatform=$BUILDPLATFORM" -o ./build ./...


```

#### System requirements

These are the system requirements for local development, build and test

- golang>=1.18
- openssl>=1.1.1
- python>=3.10
    - python packages as per [the requirements file](/requirements.txt)
- docker

### Docker Build

```bash
docker build -t stackql:${STACKQL_TAG} -t stackql:latest .
```

## Run

### Native Run

#### Help message

```bash
./build/stackql --help

```

#### Shell

```bash

# Amend STACKQL_AUTH as required, angle bracketed strings must be replaced.
export STACKQL_AUTH='{ "google": { "credentialsfilepath": "</path/to/google/sa-key.json>", "type": "service_account" }, "okta": { "credentialsenvvar": "<OKTA_SECRET_KEY>", "type": "api_key" }, "github": { "type": "basic", "credentialsenvvar": "<GITHUB_CREDS>" }, "aws": { "type": "aws_signing_v4", "credentialsfilepath": "</path/to/aws/secret-key.txt>", "keyID": "<YOUR_AWS_KEY_NOT_A_SECRET>" }, "k8s": { "credentialsenvvar": "<K8S_TOKEN>", "type": "api_key", "valuePrefix": "Bearer " } }'

./build/stackql --auth="${STACKQL_AUTH}" shell

```

### Docker Run

**NOTE**: on some docker versions, the argument `--security-opt seccomp=unconfined` is required as a hack for a [known issue in docker](https://github.com/containers/skopeo/issues/1501). 

#### Docker single query

```bash
docker compose run --rm stackqlsrv "bash" "-c" "stackql exec 'show providers;'"
```

#### Docker interactive shell

```bash

export AWS_KEY_ID='<YOUR_AWS_KEY_ID_NOT_A_SECRET>'

export DOCKER_AUTH_STR='{ "google": { "credentialsfilepath": "/opt/stackql/keys/sa-key.json", "type": "service_account" }, "okta": { "credentialsenvvar": "OKTA_SECRET_KEY", "type": "api_key" }, "github": { "type": "basic", "credentialsenvvar": "GITHUB_CREDS" }, "aws": { "type": "aws_signing_v4", "credentialsfilepath": "/opt/stackql/keys/integration/aws-secret-key.txt", "keyID": "'${AWS_KEY_ID}'" }, "k8s": { "credentialsenvvar": "K8S_TOKEN", "type": "api_key", "valuePrefix": "Bearer " } }'

export DOCKER_REG_CFG='{ "url": "https://registry.stackql.app/providers" }'

docker compose -p shellrun run --rm -e OKTA_SECRET_KEY=some-dummy-api-key -e GITHUB_SECRET_KEY=some-dummy-github-key -e K8S_SECRET_KEY=some-k8s-token -e REGISTRY_SRC=test/registry-mocked stackqlsrv bash -c "stackql shell --registry='${DOCKER_REG_CFG}' --auth='${DOCKER_AUTH_STR}'"
```

#### Docker PG Server

#### mTLS Server Stock as a rock

From the root directory of this repository...

```bash
docker compose -f docker-compose-credentials.yml run --rm credentialsgen 

docker compose up stackqlsrv
```

Then...

```bash
psql -d "host=127.0.0.1 port=5576 user=myuser sslmode=verify-full sslcert=./vol/srv/credentials/pg_client_cert.pem sslkey=./vol/srv/credentials/pg_client_key.pem sslrootcert=./vol/srv/credentials/pg_server_cert.pem dbname=mydatabase"
```

When finished, clean up with:

```bash
docker compose down
```



## Examples

```
./stackql exec "show extended services from google where title = 'Service Directory API';"
```

More examples in [examples/examples.md](/examples/examples.md).

---

## Developers

- [docs/developer_guide.md](/docs/developer_guide.md).
- [contributing](/CONTRIBUTING.md).

## Testing

- [test/README.md](/test/README.md).
- [docs/integration_testing.md](/docs/integration_testing.md).

## Server mode

Please see [the server mode section of the developer docs](/docs/developer_guide.md#server-mode).

## Alpha Features

- [GC, cacheing and concurrent users](/docs/GC_cache_concurrency.md)

## Acknowledgements

Forks of the following support our work:

  - [vitess](https://vitess.io/)
  - [readline](https://github.com/chzyer/readline)
  - [color](https://github.com/fatih/color)

We gratefully acknowledge these pieces of work.

## Licensing

Please see the [stackql LICENSE](/LICENSE).

Licenses for third party software we are using are included in the [/licenses](/licenses) directory.
