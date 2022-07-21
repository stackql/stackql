

# HTTPS server for simulated integration / regression testing

## Mock Server

We are using the java [mockserver](https://www.mock-server.com/) tool.

Some doco on creating expectations [here](https://www.mock-server.com/mock_server/creating_expectations.html#button_match_request_by_query_parameter_name_regex).

### To install

```bash
mvn org.apache.maven.plugins:maven-dependency-plugin:3.2.0:copy -Dartifact=org.mock-server:mockserver-netty:5.12.0:jar:shaded -DoutputDirectory=${HOME}/stackql/stackql-devel/test/downloads -DdestFileName=mockserver-netty.jar -DoverWrite=true
```

### To Run

GCP mocks:

```bash
java  -Dfile.encoding=UTF-8 -Dmockserver.initializationJsonPath=${HOME}/stackql/stackql-devel/test/mockserver/expectations/static-gcp-expectations.json -jar /usr/local/lib/mockserver/mockserver-netty-jar-with-dependencies.jar  -serverPort 1080 -logLevel INFO
```

Okta mocks:

```bash
java  -Dfile.encoding=UTF-8 -Dmockserver.initializationJsonPath=${HOME}/stackql/stackql-devel/test/mockserver/expectations/static-okta-expectations.json -jar /usr/local/lib/mockserver/mockserver-netty-jar-with-dependencies.jar  -serverPort 1090 -logLevel INFO
```

AWS mocks:

```bash
java  -Dfile.encoding=UTF-8 -Dmockserver.initializationJsonPath=${HOME}/stackql/stackql-devel/test/mockserver/expectations/static-aws-expectations.json -jar /usr/local/lib/mockserver/mockserver-netty-jar-with-dependencies.jar  -serverPort 1090 -logLevel INFO
```

Github mocks:

```bash
java  -Dfile.encoding=UTF-8 -Dmockserver.initializationJsonPath=${HOME}/stackql/stackql-devel/test/mockserver/expectations/static-github-expectations.json -jar /usr/local/lib/mockserver/mockserver-netty-jar-with-dependencies.jar  -serverPort 1093 -logLevel INFO
```


### Expectations from local file

As per [expectations/static-gcp-expectations.json](/test/server/expectations/static-gcp-expectations.json)


Basic idea is to rewrite openapi docs and also dummy credentials file such that 
all requests go to localhost.  We will pass in the dummy server CA to StackQL at init time.
This will obviously only occur in testing.

```
"select ipCidrRange, sum(5) cc  from  google.container.`projects.aggregated.usableSubnetworks` where projectsId = 'testing-project' group by \"ipCidrRange\" having sum(5) >= 5 order by ipCidrRange desc;"
```