


*** Settings ***
Resource          ${CURDIR}/stackql.resource

*** Test Cases ***

ID Fed AWS S3 Buckets List
    Sleep    2s
    ${awsAuthCfg} =    Catenate
    ...    { "aws": { "type":"aws_signing_v4", "keyIDenvvar": "AWS_ACCESS_KEY_ID", "credentialsenvvar": "AWS_SECRET_ACCESS_KEY" } }
    ${bucketsListQuery} =    Catenate
    ...    select * from aws.pseudo_s3.buckets_list_only where region = 'ap-southeast-2';
    ${result} =    Run Process
    ...    ${STACKQL_EXE}
    ...    \-\-auth
    ...    ${awsAuthCfg}
    ...    \-\-registry
    ...    { "url": "file://${REPOSITORY_ROOT}/test/registry", "localDocRoot": "${REPOSITORY_ROOT}/test/registry", "verifyConfig": { "nopVerify": true } }
    ...    exec
    ...    ${bucketsListQuery}
    ...    cwd=${REPOSITORY_ROOT}
    ...    stdout=${CURDIR}/tmp/ID-Fed-AWS-S3-Buckets-List.tmp
    ...    stderr=${CURDIR}/tmp/ID-Fed-AWS-S3-Buckets-List-stderr.tmp
    Should Be Equal As Integers    ${result.rc}           0
    Should Be Empty                ${result.stderr}
    Should Contain                 ${result.stdout}       stackql\-trial\-bucket\-02


ID Fed Azure VNETs List
    Sleep    2s
    ${azureTargetSubscription} =    OperatingSystem.Get Environment Variable    AZURE_TARGET_SUBSCRIPTION_ID
    Should Not Be Empty    ${azureTargetSubscription}
    ${azureAuthCfg} =    Catenate
    ...    { "azure": { "type":"azure_default" } }
    ${bucketsListQuery} =    Catenate
    ...    select location, name from azure.network.virtual_networks where subscriptionId = '${azureTargetSubscription}';
    ${result} =    Run Process
    ...    ${STACKQL_EXE}
    ...    \-\-auth
    ...    ${azureAuthCfg}
    ...    \-\-registry
    ...    { "url": "file://${REPOSITORY_ROOT}/test/registry", "localDocRoot": "${REPOSITORY_ROOT}/test/registry", "verifyConfig": { "nopVerify": true } }
    ...    exec
    ...    ${bucketsListQuery}
    ...    cwd=${REPOSITORY_ROOT}
    ...    stdout=${CURDIR}/tmp/ID-Fed-Azure-VNETs-List.tmp
    ...    stderr=${CURDIR}/tmp/ID-Fed-Azure-VNETs-List-stderr.tmp
    Should Be Equal As Integers    ${result.rc}           0
    Should Be Empty                ${result.stderr}
    Should Contain                 ${result.stdout}       inspector\-network


ID Fed Google Buckets List
    Sleep    2s
    ${gcpAccessToken} =    OperatingSystem.Get Environment Variable    GCP_ACCESS_TOKEN
    Should Not Be Empty    ${gcpAccessToken}
    ${gcpAuthCfg} =    Catenate
    ...    { "google": { "type":"bearer", "credentialsenvvar": "GCP_ACCESS_TOKEN" } }
    ${bucketsListQuery} =    Catenate
    ...    select location, name from google.storage.buckets where project = 'stackql-demo';
    ${result} =    Run Process
    ...    ${STACKQL_EXE}
    ...    \-\-auth
    ...    ${gcpAuthCfg}
    ...    \-\-registry
    ...    { "url": "file://${REPOSITORY_ROOT}/test/registry", "localDocRoot": "${REPOSITORY_ROOT}/test/registry", "verifyConfig": { "nopVerify": true } }
    ...    exec
    ...    ${bucketsListQuery}
    ...    cwd=${REPOSITORY_ROOT}
    ...    stdout=${CURDIR}/tmp/ID-Fed-Google-Buckets-List.tmp
    ...    stderr=${CURDIR}/tmp/ID-Fed-Google-Buckets-List-stderr.tmp
    Should Be Equal As Integers    ${result.rc}           0
    Should Be Empty                ${result.stderr}
    Should Contain                 ${result.stdout}       stackql\-demo\-bucket
