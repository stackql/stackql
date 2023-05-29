*** Settings ***
Resource          ${CURDIR}/stackql.resource

*** Test Cases *** 
Positive Control
    Should contain    ''    ''

Get Providers
    Should StackQL Exec Contain    ${SHOW_PROVIDERS_STR}   okta
    Should StackQL Novel Exec Contain    ${SHOW_PROVIDERS_STR}   v0.3.1
    Should StackQL Exec Contain JSON output    ${SHOW_PROVIDERS_STR}   okta

Get Providers No Config
    Should StackQL No Cfg Exec Contain    ${SHOW_PROVIDERS_STR}   name

Get Okta Services
    Should StackQL Exec Contain    ${SHOW_OKTA_SERVICES_FILTERED_STR}    Application${SPACE}API

Get Okta Application Resources
    Should StackQL Exec Contain    ${SHOW_OKTA_APPLICATION_RESOURCES_FILTERED_STR}    grants    groups

Describe GitHub Repos Pages
    Should StackQL Novel Exec Contain    ${DESCRIBE_GITHUB_REPOS_PAGES}    https_certificate    url

Describe AWS EC2 Instances Exemplifies Deep XPath for schema
    Should StackQL NoVerify Only Exec Contain    ${DESCRIBE_AWS_EC2_INSTANCES}
                                                 ...  architecture    bootMode    subnetId
                                                 ...  stdout=${CURDIR}/tmp/describe-aws-ec2-instances.tmp

Describe AWS EC2 Default KMS Key ID Exemplifies Top Level XPath for schema
    Should StackQL NoVerify Only Exec Contain    ${DESCRIBE_AWS_EC2_DEFAULT_KMS_KEY_ID}
                                                 ...  kmsKeyId
                                                 ...  stdout=${CURDIR}/tmp/describe-aws-ec2-default-kms-key-id.tmp

Show Methods GitHub
    Should StackQL Novel Exec Equal    ${SHOW_METHODS_GITHUB_REPOS_REPOS}   ${SHOW_METHODS_GITHUB_REPOS_REPOS_EXPECTED}

Show Methods including server params AWS 
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |---------------|----------------|---------|
    ...    |${SPACE}${SPACE}MethodName${SPACE}${SPACE}${SPACE}|${SPACE}RequiredParams${SPACE}|${SPACE}SQLVerb${SPACE}|
    ...    |---------------|----------------|---------|
    ...    |${SPACE}vpcs_Describe${SPACE}|${SPACE}region${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}SELECT${SPACE}${SPACE}|
    ...    |---------------|----------------|---------|
    ...    |${SPACE}vpc_Create${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}region${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}INSERT${SPACE}${SPACE}|
    ...    |---------------|----------------|---------|
    ...    |${SPACE}vpc_Delete${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}VpcId,${SPACE}region${SPACE}${SPACE}|${SPACE}DELETE${SPACE}${SPACE}|
    ...    |---------------|----------------|---------|
    Should StackQL Current Exec Equal    
    ...    show methods in aws.ec2.vpcs;
    ...    ${outputStr}

Show Insert Google Container Clusters
    Should StackQL Exec Contain    
    ...    SHOW INSERT INTO google.container."projects.zones.clusters";
    ...    ${SHOW_INSERT_GOOGLE_CONTAINER_CLUSTERS} 
    ...    stackql_H=True
    ...    stdout=${CURDIR}/tmp/Show-Insert-Google-Container-Clusters.tmp
    
JSONNET Plus Env Vars
    ${varList}=    Create List    project=stackql-demo    region=australia-southeast1
    Should StackQL Exec Equal    
    ...    ""
    ...    ${JSONNET_PLUS_ENV_VARS_EXPECTED} 
    ...    stackql_H=True 
    ...    stackql_dryrun=True
    ...    stackql_i=${JSONNET_PLUS_ENV_VARS_QUERY_FILE}
    ...    stackql_vars=${varList}
    ...    stackql_iqldata=${JSONNET_PLUS_ENV_VARS_VAR_FILE}
    ...    stdout=${CURDIR}/tmp/JSONNET-Plus-Env-Vars.tmp

Show Extended Insert Google BQ Datasets
    Should StackQL Exec Contain    
    ...    SHOW EXTENDED INSERT INTO google.bigquery.datasets;
    ...    ${SHOW_INSERT_GOOGLE_BIGQUERY_DATASET} 
    ...    stackql_H=True
    ...    stdout=${CURDIR}/tmp/Show-Extended-Insert-Google-BQ-Datasets.tmp   

Show Insert Google BQ Datasets
    Should StackQL Exec Contain    
    ...    SHOW INSERT INTO google.bigquery.datasets;
    ...    ${SHOW_INSERT_GOOGLE_BIGQUERY_DATASET} 
    ...    stackql_H=True
    ...    stdout=${CURDIR}/tmp/Show-Insert-Google-BQ-Datasets.tmp

*** Keywords ***
Should StackQL Exec Equal
    [Arguments]    ${_EXEC_CMD_STR}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}    &{kwargs}
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    &{kwargs}
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_CANONICAL_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    &{kwargs}
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_DEPRECATED_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    &{kwargs}

Should StackQL Current Exec Equal
    [Arguments]    ${_EXEC_CMD_STR}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}    &{kwargs}
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    &{kwargs}

Should StackQL Novel Exec Equal
    [Arguments]    ${_EXEC_CMD_STR}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}    &{kwargs}
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    &{kwargs}
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_CANONICAL_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    &{kwargs}

Should StackQL Exec Contain
    [Arguments]    ${_EXEC_CMD_STR}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}    &{kwargs}
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    &{kwargs}
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_CANONICAL_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    &{kwargs}
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_DEPRECATED_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    &{kwargs}

Should StackQL Exec Contain JSON output
    [Arguments]    ${_EXEC_CMD_STR}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}    &{kwargs}
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    \-o\=json
    ...    &{kwargs}
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_CANONICAL_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    \-o\=json
    ...    &{kwargs}
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_DEPRECATED_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    \-o\=json
    ...    &{kwargs}

Should StackQL Novel Exec Contain
    [Arguments]    ${_EXEC_CMD_STR}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}    &{kwargs}
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    &{kwargs}
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_CANONICAL_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    &{kwargs}

Should StackQL NoVerify Only Exec Contain
    [Arguments]    ${_EXEC_CMD_STR}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}    &{kwargs}
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    &{kwargs}

Should StackQL No Cfg Exec Contain
    [Arguments]    ${_EXEC_CMD_STR}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}    &{kwargs}
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NULL}
    ...    ${EMPTY}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    &{kwargs}


