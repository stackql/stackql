
## Golang SQL drivers

How to use drivers:

- https://go.dev/doc/database/open-handle

List of drivers:

- https://github.com/golang/go/wiki/SQLDrivers

### Data Source Name (DSN) strings

- [SQLite as per golang](https://github.com/mattn/go-sqlite3#dsn-examples).
- [Postgres URI](https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING).

## Backends

### SQLite

The default implementation is **embedded** SQLite.  SQLite does **not** have a wire protocol or TCP-native version.

### Postgres

#### Postgres over TCP

- [Using golang SQL driver interfaces](https://github.com/jackc/pgx/wiki/Getting-started-with-pgx-through-database-sql#hello-world-from-postgresql).
- [PGX native (improved performance)](https://github.com/jackc/pgx/wiki/Getting-started-with-pgx).

#### Embedded Postgres

https://github.com/fergusstrange/embedded-postgres


#### Setup postgres in docker

```sh
docker run -v "${PWD}/vol/postgres/setup:/docker-entrypoint-initdb.d:ro" -it --entrypoint bash postgres:14.5-bullseye
```

```sh
docker run --rm -v "$(pwd)/vol/postgres/setup:/docker-entrypoint-initdb.d:ro" -p 127.0.0.1:6532:5432/tcp -e POSTGRES_PASSWORD=password postgres:14.5-bullseye
```

Docker compose troubleshoot

```
docker-compose -p execrun run --rm -e OKTA_SECRET_KEY=some-dummy-api-key -e GITHUB_SECRET_KEY=some-dummy-github-key -e K8S_SECRET_KEY=some-k8s-token -e AZ_ACCESS_TOKEN=dummy_azure_token stackqlsrv bash -c "sleep 5 && stackql exec --registry='{\"url\": \"file:///opt/stackql/registry\", \"localDocRoot\": \"/opt/stackql/registry\", \"verifyConfig\": {\"nopVerify\": true}}' --auth='{\"google\": {\"credentialsfilepath\": \"/opt/stackql/credentials/dummy/google/docker-functional-test-dummy-sa-key.json\", \"type\": \"service_account\"}, \"okta\": {\"credentialsenvvar\": \"OKTA_SECRET_KEY\", \"type\": \"api_key\"}, \"aws\": {\"type\": \"aws_signing_v4\", \"credentialsfilepath\": \"/opt/stackql/credentials/dummy/aws/functional-test-dummy-aws-key.txt\", \"keyID\": \"NON_SECRET\"}, \"github\": {\"type\": \"basic\", \"credentialsenvvar\": \"GITHUB_SECRET_KEY\"}, \"k8s\": {\"credentialsenvvar\": \"K8S_SECRET_KEY\", \"type\": \"api_key\", \"valuePrefix\": \"Bearer \"}, \"azure\": {\"type\": \"api_key\", \"valuePrefix\": \"Bearer \", \"credentialsenvvar\": \"AZ_ACCESS_TOKEN\"}}' --sqlBackend='{\"dbEngine\":\"postgres_tcp\",\"dsn\":\"postgres://stackql:stackql@postgres_stackql:6532/stackql\",\"sqlDialect\":\"postgres\"}' --tls.allowInsecure=true 'select name from google.compute.machineTypes where project = '\"'\"'testing-project'\"'\"' and zone = '\"'\"'australia-southeast1-a'\"'\"' order by name desc;'"
```

#### Setup postgres DB locally

```sql

CREATE database "stackql";

CREATE user stackql with password 'stackql';

GRANT ALL PRIVILEGES on DATABASE stackql to stackql;

```

#### Postgres integration bug checklist

- [ ] ERROR: function group_concat(text) does not exist
- [ ] syntax error at or near "like"
- [ ] ERROR: column "projectsId" of relation "google.cloudresourcemanager.Binding.generation_1" does not exist
- [ ] ERROR: syntax error at or near "".profile, '$.login')""
- [ ] ERROR: column "VolumeId" does not exist
- [ ] `AWS IAM Users Select Simple` sort order different to SQLite.
- [ ] sql insert error: 'failed to encode args[11]: unable to encode 13 into text format for text
- [ ] ERROR: function json_extract(text, unknown) does not exist
- [ ] ERROR: column "JSON_EXTRACT(samlIdentity, '$.nameId')" does not exist
- [ ] ERROR: column "count(*)" does not exist
- [ ] ERROR: syntax error at or near "PRAGMA"
- [ ] ERROR: column "split_part(teams_url, '/', 4)" does not exist
- [ ] ERROR: syntax error at or near "".""
- [ ] "err = sql: no rows in result set for tableNamePattern = 'k8s.core_v1.io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta.generation_%' and tableNameLHSRemove = 'k8s.core_v1.io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta.generation_'"
- [ ] ERROR: column "eTag" does not exist
- [ ] `PG Session Anayltics Cache Behaviour Canonical` cacheing does not work, same other cache test.

Failing tests checklist:

- [ ] Google IAM Policy Agg                                                 
- [ ] Google Select Project IAM Policy                                      
- [ ] Google Select Project IAM Policy Filtered And Verify Like Filtering   
- [ ] Google Select Project IAM Policy Filtered And Verify Where Filtering  
- [ ] Google Join Plus String Concatenated Select Expressions               
- [ ] Google AcceleratorTypes SQL verb pre changeover                       
- [ ] Google Machine Types Select Paginated                                 
- [ ] Google AcceleratorTypes SQL verb post changeover                      
- [ ] Okta Users Select Simple Paginated                                    
- [ ] AWS EC2 Volumes Select Simple                                         
- [ ] AWS IAM Users Select Simple                                           
- [ ] AWS S3 Objects Select Simple                                          
- [ ] AWS Cloud Control Operations Select Simple                            
- [ ] GitHub Scim Users Select                                              
- [ ] GitHub SAML Identities Select GraphQL                                 
- [ ] GitHub Tags Paginated Count                                           
- [ ] GitHub Analytics Simple Select Repositories Collaborators             
- [ ] GitHub Analytics Transparent Select Repositories Collaborators        
- [ ] GitHub Repository With Functions Select                               
- [ ] Join GCP Okta Cross Provider                                          
- [ ] Join GCP Okta Cross Provider JSON Dependent Keyword in Table Name     
- [ ] Join GCP Three Way                                                    
- [ ] Join GCP Self                                                         
- [ ] K8S Nodes Select Leveraging JSON Path                                 
- [ ] Google Compute Instance IAM Policy Select                             
- [ ] Paginated and Data Flow Sequential Join Github Okta SAML              
- [ ] Data Flow Sequential Join Select With Functions Github                
- [ ] Functional.Stackql Mocked From Cmd Line                               
- [ ] PG Session Anayltics Cache Behaviour Canonical                        
- [ ] PG Session Postgres Client Typed Queries                              
- [ ] PG Session Postgres Client V2 Typed Queries  


Some debugging:

- Google Select Project IAM Policy


