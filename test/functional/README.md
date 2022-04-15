

## TODO

- [ ] Source registry files from other repository, where possible.
- [x] "reuired" string.
- [x] clean up this readme.
- [ ] integration tests for different registry configurations.


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

