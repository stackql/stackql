
## Distributed testing

It is convenient for development- and release-critical robot tests to reside in this repository, because it accelerates the 
delivery cycle.  There are other repositories in the org that can re-use much of this functionality; ironically the best examples are dependencies:

- [`any-sdk`](https://github.com/stackql/any-sdk).
- [`stackql-provider-registry`](https://github.com/stackql/stackql-provider-registry).

These may consume entire testing modules, or more nuanced [tag-based](https://robotframework.org/robotframework/latest/RobotFrameworkUserGuide.html#tagging-test-cases) approaches, with [include](https://robotframework.org/robotframework/latest/RobotFrameworkUserGuide.html#by-tag-names) and [skip](https://robotframework.org/robotframework/latest/RobotFrameworkUserGuide.html#skip) capabilities.


## Running yourself

### Running mocked provider tests

From the repository root:

```sh
env PYTHONPATH="$PYTHONPATH:$(pwd)/test/python" robot -d test/robot/reports test/robot/functional
```

### Running actual integration tests

From the repository root:

```sh
env PYTHONPATH="$PYTHONPATH:$(pwd)/test/python"  robot -d test/robot/integration \ 
  -v OKTA_CREDENTIALS:"$(cat /path/to/okta/credentials)" \
  -v GCP_CREDENTIALS:"$(cat /path/to/gcp/credentials)" \
  -v AWS_CREDENTIALS:"$(cat /path/to/aws/credentials)" \
  -v AZURE_CREDENTIALS:"$(cat /path/to/azure/credentials)" \
  test/robot/integration
```

For example:

```sh
env PYTHONPATH="$PYTHONPATH:$(pwd)/test/python" robot -d test/robot/integration \ 
  -v OKTA_CREDENTIALS:"$(cat /path/to/okta/credentials)" \
  -v GCP_CREDENTIALS:"$(cat ${HOME}/stack/stackql-devel/cicd/keys/integration/stackql-dev-01-07d91f4abacf.json)" \
  -v AWS_CREDENTIALS:"$(cat ${HOME}/stack/stackql-devel/cicd/keys/integration/aws-auth-val.txt)" \
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

## Windows issues

This somehow works:

```ps1
$v1="SELECT i.zone, i.name, i.machineType, i.deletionProtection, '[{""""""subnetwork"""""":""""""' || JSON_EXTRACT(i.networkInterfaces, '$[0].subnetwork') || '""""""}]', '[{""""""boot"""""": true, """"""initializeParams"""""": { """"""diskSizeGb"""""": """"""' || JSON_EXTRACT(i.disks, '$[0].diskSizeGb') || '"""""", """"""sourceImage"""""": """"""' || d.sourceImage || '""""""}}]', i.labels FROM google.compute.instances i INNER JOIN google.compute.disks d ON i.name = d.name WHERE i.project = 'testing-project' AND i.zone = 'australia-southeast1-a' AND d.project = 'testing-project' AND d.zone = 'australia-southeast1-a' AND i.name LIKE '%' order by i.name DESC;"

.\build\stackql.exe --auth="${AUTH_STR}" --registry="${REG_CFG_MOCKED}" --tls.allowInsecure=true exec "$v1"
```

## Session testing for Server and Shell

**NOTE**: This is deprecated; robot is the way formward.  Do **not** add new tests with this pattern.

Basic idea is have python start a session, run commands with result verification and then terminate.  Probably custom library(ies).

Library would do something like the below adaptation of [this example from stack overflow](https://stackoverflow.com/questions/19880190/interactive-input-output-using-python):

```py
import os
import subprocess
import sys


def start(cmd_arg_list):
  command = [item.encode(sys.getdefaultencoding()) for item in cmd_arg_list]
  return subprocess.Popen(
    command,
    stdin=subprocess.PIPE,
    stdout=subprocess.PIPE,
    stderr=subprocess.PIPE
  )


def read(process):
    return process.stdout.readline().decode("utf-8").strip()


def write(process, message):
    process.stdin.write(f"{message.strip()}\n".encode("utf-8"))
    process.stdin.flush()


def terminate(process):
    process.stdin.close()
    process.terminate()
    process.wait(timeout=0.2)


process = start(
  [ "./stackql",
    f"--registry={os.environ.get('REG_TEST')}",
    f"--auth={os.environ.get('AUTH_STR_INT')}",
    "shell"
  ]
)



write(process, "show providers;")

response_01 = read(process)

print(response_01)

terminate(process)

```

