


*** Settings ***
Resource          ${CURDIR}/stackql.resource

*** Test Cases ***

Foreign AWS S3 Buckets List
    Sleep    2s
    ${awsRoleArn} =    OperatingSystem.Get Environment Variable    STACKQL_AUDIT_ROLE_ARN
    Should Not Be Empty    ${awsRoleArn}
    ${awsAuthCfg} =    Catenate
    ...    { "aws": { "type":"aws_assume_role", "keyIDenvvar": "AWS_ACCESS_KEY_ID", "credentialsenvvar": "AWS_SECRET_ACCESS_KEY", "aws_role_arn": "${awsRoleArn}" } }
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
    ...    stdout=${CURDIR}/tmp/AWS-S3-Buckets-List.tmp
    ...    stderr=${CURDIR}/tmp/AWS-S3-Buckets-List-stderr.tmp
    Should Be Equal As Integers    ${result.rc}           0
    Should Contain                 ${result.stdout}       stackql\-trial\-bucket\-02
