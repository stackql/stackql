

## TODO

- [ ] Source registry files from other repository, where possible.
- [x] "reuired" string.
- [x] clean up this readme.
- [ ] integration tests for different registry configurations.


## Running yourself

### Running mocked provider tests

From the repository root:

```sh
robot -d test/robot/functional test/robot/functional
```

### Running actual integration tests

From the repository root:

```sh
robot -d test/robot/integration \ 
  -v OKTA_CREDENTIALS:"$(cat /path/to/okta/credentials)" \
  -v GCP_CREDENTIALS:"$(cat /path/to/gcp/credentials)" \
  -v AWS_CREDENTIALS:"$(cat /path/to/aws/credentials)" \
  -v AZURE_CREDENTIALS:"$(cat /path/to/azure/credentials)" \
  test/robot/integration
```

For example:

```sh
robot -d test/robot/integration \ 
  -v OKTA_CREDENTIALS:"$(cat /path/to/okta/credentials)" \
  -v GCP_CREDENTIALS:"$(cat ${HOME}/stack/stackql-devel/keys/integration/stackql-dev-01-07d91f4abacf.json)" \
  -v AWS_CREDENTIALS:"$(cat ${HOME}/stack/stackql-devel/keys/integration/aws-auth-val.txt)" \
  -v AZURE_CREDENTIALS:"$(cat /path/to/azure/credentials)" \
  test/robot/integration
```


### Known Queries to add to functional tests

```sql

EXEC github.apps.apps.create_from_manifest ... -- tests allOf in response

SELECT name, ssh_url from github.repos.repos where org = 'stackql' ; -- tests straight to array response

exec /*+ SHOWRESULTS */ github.users.users.get_by_username @username='general-kroll-4-life'; -- was previously busted


```

### Unknown Queries to add to functional tests

- oneOf in response body.
- anyOf in response body.
- allOf in request body.
- oneOf in request body.
- anyOf in request body.


### Other aspects to add to functional tests

- Complete migration of python test script.
- Verify all tables created as expected on document read.
- Verification of GC columns post query.
- Verification of GC.

## Detail

### Example query executed by the functional tests

```bash
/Users/admin/stackql/stackql-devel/build/stackql exec "--registry={\"url\": \"file://${HOME}/stackql/stackql-devel/test/registry\", \"localDocRoot\": \"${HOME}/stackql/stackql-devel/test/registry\", \"useEmbedded\": false, \"verifyConfig\": {\"nopVerify\": true}}" "--auth={\"google\": {\"credentialsfilepath\": \"${HOME}/stackql/stackql-devel/test/assets/credentials/dummy/google/functional-test-dummy-sa-key.json\", \"type\": \"service_account\"}, \"okta\": {\"credentialsenvvar\": \"OKTA_SECRET_KEY\", \"type\": \"api_key\"}}" --tls.allowInsecure=true "select ipCidrRange, sum(5) cc  from  google.container.\`projects.aggregated.usableSubnetworks\` where projectsId = 'testing-project' group by \"ipCidrRange\" having sum(5) >= 5 order by ipCidrRange desc;"


/Users/admin/stackql/stackql-devel/build/stackql exec "--registry={\"url\": \"file://${HOME}/stackql/stackql-devel/test/empty\", \"localDocRoot\": \"${HOME}/stackql/stackql-devel/test/empty\", \"useEmbedded\": false, \"verifyConfig\": {\"nopVerify\": true}}" "--auth={\"google\": {\"credentialsfilepath\": \"${HOME}/stackql/stackql-devel/test/assets/credentials/dummy/google/functional-test-dummy-sa-key.json\", \"type\": \"service_account\"}, \"okta\": {\"credentialsenvvar\": \"OKTA_SECRET_KEY\", \"type\": \"api_key\"}}" --tls.allowInsecure=true "show providers;"
```

