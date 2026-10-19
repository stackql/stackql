*** Settings ***
Resource          ${CURDIR}/stackql.resource
Suite Setup       Legacy Resolve Platform Paths
Test Teardown     Stackql Per Test Teardown
Documentation     Port of the retired ``test/deprecated/python/main.py`` suite. Covers the
...               output-format surface (json / csv / table / text, custom delimiter, ``--outfile``)
...               over offline metadata statements, plus the templated ``--infile`` cases
...               (json, jsonnet, external ``--iqldata`` context, headless dry-run). Every case
...               runs on all platforms via the robot harness; nothing here hits the network.

*** Variables ***
${OUTPUT_ROOT}        ${CURDIR}${/}tmp
${OUTFILE_HOST_ROOT}    ${REPOSITORY_ROOT}${/}test${/}tmp

*** Keywords ***
Legacy Resolve Platform Paths
    # docker-compose mounts test/assets/input at /opt/stackql/input and test/tmp at /opt/test/tmp
    ${is_docker} =    Set Variable If    "${EXECUTION_PLATFORM}" == "docker"    ${True}    ${False}
    ${input_root} =    Set Variable If    ${is_docker}    /opt/stackql/input    ${REPOSITORY_ROOT}${/}test${/}assets${/}input
    ${outfile_root} =    Set Variable If    ${is_docker}    /opt/test/tmp    ${OUTFILE_HOST_ROOT}
    Set Suite Variable    ${INPUT_ROOT}    ${input_root}
    Set Suite Variable    ${OUTFILE_ROOT}    ${outfile_root}

Legacy Exec Should Contain
    # stdout is file-backed: Robot's in-memory capture deadlocks once a process writes >64KB.
    [Arguments]    ${query}    ${expected}    @{args}    &{cfg}
    Should Stackql Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${query}
    ...    ${expected}
    ...    @{args}
    ...    stdout=${OUTPUT_ROOT}${/}${TEST NAME}.tmp
    ...    stderr=${OUTPUT_ROOT}${/}${TEST NAME}-stderr.tmp
    ...    &{cfg}

Legacy Exec Outfile Should Contain
    [Arguments]    ${query}    ${outfile_name}    ${expected}    @{args}    &{cfg}
    Should Stackql Exec Inline Outfile Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${query}
    ...    ${OUTFILE_ROOT}/${outfile_name}
    ...    ${expected}
    ...    @{args}
    ...    host_outfile=${OUTFILE_HOST_ROOT}${/}${outfile_name}
    ...    stdout=${OUTPUT_ROOT}${/}${TEST NAME}.tmp
    ...    stderr=${OUTPUT_ROOT}${/}${TEST NAME}-stderr.tmp
    ...    &{cfg}

*** Test Cases ***
Legacy USE Then Show Services JSON Is Array
    Legacy Exec Should Contain    USE google; SHOW SERVICES FROM google;    [{"id":    stackql_output=json

Legacy Show Providers JSON Is Array
    Legacy Exec Should Contain    SHOW PROVIDERS;    [{"name":    stackql_output=json

Legacy Show Services JSON Is Array
    Legacy Exec Should Contain    SHOW SERVICES FROM google;    [{"id":    stackql_output=json

Legacy Show Resources JSON Is Array
    Legacy Exec Should Contain    SHOW RESOURCES FROM google.compute;    [{"id":"google.compute.    stackql_output=json

Legacy Show Extended Providers JSON Is Array
    Legacy Exec Should Contain    SHOW EXTENDED PROVIDERS;    [{"name":    stackql_output=json

Legacy Show Extended Services JSON Is Array
    Legacy Exec Should Contain    SHOW EXTENDED SERVICES FROM google;    [{"description":    stackql_output=json

Legacy Show Extended Resources JSON Is Array
    Legacy Exec Should Contain    SHOW EXTENDED RESOURCES FROM google.compute;    [{"description":    stackql_output=json

Legacy Describe Resource JSON Is Array
    Legacy Exec Should Contain    DESCRIBE google.compute.instances;    [{"name":    stackql_output=json

Legacy Describe Extended Resource JSON Is Array
    Legacy Exec Should Contain    DESCRIBE EXTENDED google.compute.instances;    [{"description":    stackql_output=json

Legacy Show Providers CSV Outfile Has Header
    Legacy Exec Outfile Should Contain    SHOW PROVIDERS;    legacy-show-providers.csv    name    stackql_output=csv

Legacy Show Providers Semicolon CSV Outfile
    Legacy Exec Outfile Should Contain    SHOW PROVIDERS;    legacy-show-providers-alt.csv    google    stackql_output=csv    stackql_delimiter=;

Legacy Show Extended Services CSV Outfile Has Header
    Legacy Exec Outfile Should Contain    SHOW EXTENDED SERVICES FROM google;    legacy-show-services.csv    id,name,title,description    stackql_output=csv

Legacy Show Extended Services Semicolon CSV Outfile Has Header
    Legacy Exec Outfile Should Contain    SHOW EXTENDED SERVICES FROM google;    legacy-show-services-alt.csv    id;name;title;description    stackql_output=csv    stackql_delimiter=;

Legacy Filtered Show Extended Services Semicolon CSV
    Legacy Exec Should Contain
    ...    show extended services from google where description like 'Provides natural language%' and version \= 'v1';
    ...    language:v1;
    ...    stackql_output=csv
    ...    stackql_delimiter=;

Legacy Filtered Show Extended Resources Semicolon CSV
    Legacy Exec Should Contain
    ...    show extended resources from google.compute where name \= 'resourcePolicies' and id like '%.resourcePol%';
    ...    resourcePolicies;
    ...    stackql_output=csv
    ...    stackql_delimiter=;

Legacy Tabular Show Extended Services
    Legacy Exec Should Contain
    ...    show extended services from google where description like 'Provides natural language%' and version \= 'v1';
    ...    | language:v1
    ...    stackql_output=table

Legacy Tabular Show Insert
    Legacy Exec Should Contain    show insert into google.compute.instances;    INSERT INTO    stackql_output=table

Legacy Tabular Show Methods
    Legacy Exec Should Contain    show methods in google.compute.instances;    MethodName    stackql_output=table

Legacy JSON Templated Infile Filtered Services
    Legacy Exec Should Contain
    ...    ${EMPTY}
    ...    language:v1;
    ...    stackql_output=csv
    ...    stackql_delimiter=;
    ...    stackql_i=${INPUT_ROOT}${/}json-templated-show-services-extended-filtered.iql

Legacy JSONNET Templated Infile Filtered Services
    Legacy Exec Should Contain
    ...    ${EMPTY}
    ...    language:v1;
    ...    stackql_output=csv
    ...    stackql_delimiter=;
    ...    stackql_i=${INPUT_ROOT}${/}jsonnet-templated-show-services-extended-filtered.iql

Legacy External JSON Context Infile Filtered Services
    Legacy Exec Should Contain
    ...    ${EMPTY}
    ...    language:v1;
    ...    stackql_output=csv
    ...    stackql_delimiter=;
    ...    stackql_i=${INPUT_ROOT}${/}empty-templated-show-services-extended-filtered.iql
    ...    stackql_iqldata=${INPUT_ROOT}${/}show-services-ctx.json

Legacy External JSONNET Context Infile Filtered Services
    Legacy Exec Should Contain
    ...    ${EMPTY}
    ...    language:v1;
    ...    stackql_output=csv
    ...    stackql_delimiter=;
    ...    stackql_i=${INPUT_ROOT}${/}empty-templated-show-services-extended-filtered.iql
    ...    stackql_iqldata=${INPUT_ROOT}${/}show-services-ctx.jsonnet

Legacy Headless Dry Run Templated Insert Compute Network
    Legacy Exec Should Contain
    ...    ${EMPTY}
    ...    '{"routingMode":"REGIONAL"}'
    ...    stackql_H=True
    ...    stackql_dryrun=True
    ...    stackql_i=${INPUT_ROOT}${/}templated-insert-compute-network.iql

Legacy Headless Dry Run K8s The Hard Way
    Legacy Exec Should Contain
    ...    ${EMPTY}
    ...    'kubernetes-the-hard-way-vpc',
    ...    stackql_H=True
    ...    stackql_dryrun=True
    ...    stackql_i=${INPUT_ROOT}${/}k8s-the-hard-way${/}k8s-the-hard-way.iql
    ...    stackql_iqldata=${INPUT_ROOT}${/}k8s-the-hard-way${/}vars.jsonnet

Legacy Headless Show Insert With Column Selection
    Legacy Exec Should Contain
    ...    show insert into google.compute.disks(zone, data__resourcePolicies);
    ...    data__resourcePolicies
    ...    stackql_H=True
