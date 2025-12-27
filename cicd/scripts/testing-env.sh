#!/usr/bin/env bash

CUR_DIR="$(cd "$(dirname "${BASH_SOURCE[0]:-${(%):-%x}}")" && pwd)"

source "${CUR_DIR}/context.sh"

export REPOSITORY_ROOT="${REPOSITORY_ROOT}"
export workspaceFolder="${REPOSITORY_ROOT}"

export OKTA_SECRET_KEY='some-dummy-api-key'
export GITHUB_SECRET_KEY='some-dummy-github-key'
export K8S_SECRET_KEY='some-k8s-token'
export AZ_ACCESS_TOKEN='dummy_azure_token'
export SUMO_CREDS='somesumologictoken'
export DIGITALOCEAN_TOKEN='somedigitaloceantoken'
export DUMMY_DIGITALOCEAN_USERNAME='myusername'
export DUMMY_DIGITALOCEAN_PASSWORD='mypassword'


googleCredentialsFilePath="${workspaceFolder}/test/assets/credentials/dummy/google/functional-test-dummy-sa-key.json"

export stackqlMockedRegistryStr="{ \"url\": \"file://${workspaceFolder}/test/registry-mocked\", \"localDocRoot\": \"${workspaceFolder}/test/registry-mocked\", \"verifyConfig\": { \"nopVerify\": true } }"

export stackqlTestRegistryStr="{ \"url\": \"file://${workspaceFolder}/test/registry\", \"localDocRoot\": \"${workspaceFolder}/test/registry\", \"verifyConfig\": { \"nopVerify\": true } }"

export stackqlAuthStr="{\"google\": {\"credentialsfilepath\": \"${googleCredentialsFilePath}\", \"type\": \"service_account\"}, \"okta\": {\"credentialsenvvar\": \"OKTA_SECRET_KEY\", \"type\": \"api_key\"}, \"aws\": {\"type\": \"aws_signing_v4\", \"credentialsfilepath\": \"${REPOSITORY_ROOT}/test/assets/credentials/dummy/aws/functional-test-dummy-aws-key.txt\", \"keyID\": \"NON_SECRET\"}, \"github\": {\"type\": \"basic\", \"credentialsenvvar\": \"GITHUB_SECRET_KEY\"}, \"k8s\": {\"credentialsenvvar\": \"K8S_SECRET_KEY\", \"type\": \"api_key\", \"valuePrefix\": \"Bearer \"}, \"azure\": {\"type\": \"api_key\", \"valuePrefix\": \"Bearer \", \"credentialsenvvar\": \"AZ_ACCESS_TOKEN\"}, \"sumologic\": {\"type\": \"basic\", \"credentialsenvvar\": \"SUMO_CREDS\"}, \"digitalocean\": {\"type\": \"bearer\", \"username\": \"myusername\", \"password\": \"mypassword\"}}"




