

## BYO Registry


```bash

PROVIDER_REGISTRY_ROOT_DIR="$(pwd)/../examples/registry"

REG_STR='{ "url": "file://'${PROVIDER_REGISTRY_ROOT_DIR}'", "localDocRoot": "'${PROVIDER_REGISTRY_ROOT_DIR}'",  "useEmbedded": false, "verifyConfig": {"nopVerify": true } }'


## All auth required ahead of time at this stage, even for no auth providers
AUTH_STR='{ "publicapis": { "type": "null_auth" }  }'

./stackql --auth="${AUTH_STR}" --registry="${REG_STR}" exec "select API from publicapis.api.apis where API like 'Dog%' limit 10;"

./stackql --auth="${AUTH_STR}" --registry="${REG_STR}" exec "select API from publicapis.api.random where title =  'Dog';"

## The single, anonmyous column returned from selecing an array of strings is a core issue to fix separate to this
./stackql --auth="${AUTH_STR}" --registry="${REG_STR}" exec "select * from publicapis.api.categories limit 5;"

```