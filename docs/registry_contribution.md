
# Provider Registry Contribution

Community development of provider functionality is key to `stackql` and our mission.
Here, we describe the process to develop, test and integrate a new provider from zero to General Availability.

We include an example integration for a simple Provider, `publicapis`.

## Workflow

1. Create a Provider definition document.
2. Create at least one Service definition document.
3. Run `stackql` with config to support local Provider development.
4. Iterate and verify that `stackql` works as expected against your new provider.
5. Submit a Pull Request against the Provider Registry repository.


### 1. Create a Provider definition document

This is a yaml or json document which contains Provider metadata and reference(s) to any Service document(s) through which it will expose functionality.

Example as per [examples/registry/src/publicapis/v1/provider.yaml](/examples/registry/src/publicapis/v1/provider.yaml).

### 2. Create at least one Service definition document.

This is an [openapi document spec](https://swagger.io/specification/) in either yaml or json format, **plus** a legal annotation `components.x-stackQL-resources`, which defines the Resource portion of the `stackql` heirarchy.

Example as per [examples/registry/src/publicapis/v1/services/api-v1.yaml](/examples/registry/src/publicapis/v1/services/api-v1.yaml).


### 3. Run `stackql` with config to support local Provider development.

Configure `stackql` to use your docs, via the `--registry` command line argument.  The [example integration section](#Configuring-StackQL-to-consume-a-local-development-registry) expands upon this.

### 4. Iterate and verify that `stackql` works as expected against your new provider.

Iterate upon steps 2 and 3 until API coverage and functionality fulfil your requirements.

### 5. Submit a Pull Request against the Provider Registry repository.

As per [the Provider Registry Contributor Guide](https://github.com/stackql/stackql-provider-registry/blob/main/.github/CONTRIBUTING.md).  The team will review as rapidly as possible and work with you to complete the integration.  Once this is done, your functionality with be available via `registry pull...`

## Example integration

In this walkthrough, we demonstrate the bare bones of integrating a simple Provider,
`publicapis`, which is included here within source control.

The assumptions for this walkthrough are:

1. You have either built (eg: via `cmake` as per [the readme](/README.md#build)) or copied the `stackql` executable for your platform into the [build](/build) directory.
2. The current working directory is same as this document, otherwise relative paths to `pwd` will need to be adjusted.

### Configuring StackQL to consume a local development registry

Registry access config:

```bash
PROVIDER_REGISTRY_ROOT_DIR="$(pwd)/../examples/registry"

REG_STR='{ "url": "file://'${PROVIDER_REGISTRY_ROOT_DIR}'", "localDocRoot": "'${PROVIDER_REGISTRY_ROOT_DIR}'",  "verifyConfig": {"nopVerify": true } }'
```

Provider auth config as per [developer guide](/docs/developer_guide.md#provider-authentication):

```bash
AUTH_STR='{ "publicapis": { "type": "null_auth" }  }'
```

### Interacting with the local development registry

```bash
$(pwd)/../build/stackql --auth="${AUTH_STR}" --registry="${REG_STR}" exec "select API from publicapis.api.apis where API like 'Dog%' limit 10;"

$(pwd)/../build/stackql --auth="${AUTH_STR}" --registry="${REG_STR}" exec "select API from publicapis.api.random where title =  'Dog';"

## The single, anonmyous column returned from selecing an array of strings is a core issue to fix separate to this
$(pwd)/../build/stackql --auth="${AUTH_STR}" --registry="${REG_STR}" exec "select * from publicapis.api.categories limit 5;"
```
