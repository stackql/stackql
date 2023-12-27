

# HTTPS server for simulated integration / regression testing

## Mock Server

We are using the java [mockserver](https://www.mock-server.com/) tool.  As of now, this requires `java 11` and **not** some newer version; otherwise consequences are errors in the `BouncyCastle` TLS library.

Some doco on creating expectations [here](https://www.mock-server.com/mock_server/creating_expectations.html#button_match_request_by_query_parameter_name_regex).

### To install

```bash
mvn org.apache.maven.plugins:maven-dependency-plugin:3.2.0:copy -Dartifact=org.mock-server:mockserver-netty:5.12.0:jar:shaded -DoutputDirectory=${HOME}/stackql/stackql-devel/test/downloads -DdestFileName=mockserver-netty.jar -DoverWrite=true
```

### To Run

GCP mocks:

```bash
java  -Dfile.encoding=UTF-8 -Dmockserver.initializationJsonPath=${HOME}/stackql/stackql-devel/test/mockserver/expectations/static-gcp-expectations.json -jar ${HOME}/stackql/stackql-devel/test/downloads/mockserver-netty-5.12.0-shaded.jar  -serverPort 1080 -logLevel INFO
```

Azure mocks:

```bash
java  -Dfile.encoding=UTF-8 -Dmockserver.initializationJsonPath=${HOME}/stackql/stackql-devel/test/mockserver/expectations/static-azure-expectations.json -jar ${HOME}/stackql/stackql-devel/test/downloads/mockserver-netty-5.12.0-shaded.jar  -serverPort 1095 -logLevel INFO
```

Okta mocks:

```bash
java  -Dfile.encoding=UTF-8 -Dmockserver.initializationJsonPath=${HOME}/stackql/stackql-devel/test/mockserver/expectations/static-okta-expectations.json -jar ${HOME}/stackql/stackql-devel/test/downloads/mockserver-netty-5.12.0-shaded.jar  -serverPort 1090 -logLevel INFO
```

AWS mocks:

```bash
java  -Dfile.encoding=UTF-8 -Dmockserver.initializationJsonPath=${HOME}/stackql/stackql-devel/test/mockserver/expectations/static-aws-expectations.json -jar ${HOME}/stackql/stackql-devel/test/downloads/mockserver-netty-5.12.0-shaded.jar  -serverPort 1091 -logLevel INFO
```

Github mocks:

```bash
java  -Dfile.encoding=UTF-8 -Dmockserver.initializationJsonPath=${HOME}/stackql/stackql-devel/test/mockserver/expectations/static-github-expectations.json -jar ${HOME}/stackql/stackql-devel/test/downloads/mockserver-netty-5.12.0-shaded.jar  -serverPort 1093 -logLevel INFO
```

Sumologic mocks:

```bash
java  -Dfile.encoding=UTF-8 -Dmockserver.initializationJsonPath=${HOME}/stackql/stackql-devel/test/mockserver/expectations/static-sumologic-expectations.json -jar ${HOME}/stackql/stackql-devel/test/downloads/mockserver-netty-5.12.0-shaded.jar  -serverPort 1096 -logLevel INFO
```

Digitalocean mocks:

```bash
java  -Dfile.encoding=UTF-8 -Dmockserver.initializationJsonPath=${HOME}/stackql/stackql-devel/test/mockserver/expectations/static-digitalocean-expectations.json -jar ${HOME}/stackql/stackql-devel/test/downloads/mockserver-netty-5.12.0-shaded.jar  -serverPort 1097 -logLevel INFO
```

`googleadmin` mocks:

```bash
java  -Dfile.encoding=UTF-8 -Dmockserver.initializationJsonPath=${HOME}/stackql/stackql-devel/test/mockserver/expectations/static-google-admin-expectations.json -jar ${HOME}/stackql/stackql-devel/test/downloads/mockserver-netty-5.12.0-shaded.jar  -serverPort 1098 -logLevel INFO
```

stackql auth testing mocks:

```bash
java  -Dfile.encoding=UTF-8 -Dmockserver.initializationJsonPath=${HOME}/stackql/stackql-devel/test/mockserver/expectations/static-auth-testing-expectations.json -jar ${HOME}/stackql/stackql-devel/test/downloads/mockserver-netty-5.12.0-shaded.jar  -serverPort 1170 -logLevel INFO
```

### Expectations from local file

As per [expectations/static-gcp-expectations.json](/test/server/expectations/static-gcp-expectations.json)


Basic idea is to rewrite openapi docs and also dummy credentials file such that 
all requests go to localhost.  We will pass in the dummy server CA to StackQL at init time.
This will obviously only occur in testing.

```
"select ipCidrRange, sum(5) cc  from  google.container.`projects.aggregated.usableSubnetworks` where projectsId = 'testing-project' group by \"ipCidrRange\" having sum(5) >= 5 order by ipCidrRange desc;"
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
