

## TODO

- [ ]Source registry files from other repository, where possible.
- [x]"reuired" string.
- [ ]clean up this readme.
- [ ]integration tests for different registry configurations.

### Example query executed by the functional tests

```bash
/Users/admin/stackql/stackql-devel/build/stackql exec "--registry={\"url\": \"file://${HOME}/stackql/stackql-devel/test/registry\", \"localDocRoot\": \"${HOME}/stackql/stackql-devel/test/registry\", \"useEmbedded\": false, \"verifyConfig\": {\"nopVerify\": true}}" "--auth={\"google\": {\"credentialsfilepath\": \"${HOME}/stackql/stackql-devel/test/assets/credentials/dummy/google/functional-test-dummy-sa-key.json\", \"type\": \"service_account\"}, \"okta\": {\"credentialsenvvar\": \"OKTA_SECRET_KEY\", \"type\": \"api_key\"}}" --tls.allowInsecure=true "select ipCidrRange, sum(5) cc  from  google.container.\`projects.aggregated.usableSubnetworks\` where projectsId = 'testing-project' group by \"ipCidrRange\" having sum(5) >= 5 order by ipCidrRange desc;"


/Users/admin/stackql/stackql-devel/build/stackql exec "--registry={\"url\": \"file://${HOME}/stackql/stackql-devel/test/empty\", \"localDocRoot\": \"${HOME}/stackql/stackql-devel/test/empty\", \"useEmbedded\": false, \"verifyConfig\": {\"nopVerify\": true}}" "--auth={\"google\": {\"credentialsfilepath\": \"${HOME}/stackql/stackql-devel/test/assets/credentials/dummy/google/functional-test-dummy-sa-key.json\", \"type\": \"service_account\"}, \"okta\": {\"credentialsenvvar\": \"OKTA_SECRET_KEY\", \"type\": \"api_key\"}}" --tls.allowInsecure=true "show providers;"
```