


*** Settings ***
Resource          ${CURDIR}/stackql.resource

*** Test Cases *** 
Nop From Lib
    ${result} =     Nop Cloud Integration Keyword
    Should Be Equal    ${result}    PASS


AWS S3 Buckets Location Constraint
    ${locactionConstraintQuery} =    Catenate
    ...    select LocationConstraint from aws.s3.bucket_locations where region = 'ap-southeast-1' and Bucket = 'stackql-trial-bucket-01';
    ${result} =    Run Process
    ...    ${STACKQL_EXE}
    ...    \-\-registry
    ...    { "url": "file://${REPOSITORY_ROOT}/test/registry", "localDocRoot": "${REPOSITORY_ROOT}/test/registry", "verifyConfig": { "nopVerify": true } }
    ...    exec
    ...    ${locactionConstraintQuery}
    ...    cwd=${REPOSITORY_ROOT}
    ...    stdout=${CURDIR}/tmp/AWS-S3-Buckets-Location-Constraint.tmp
    ...    stderr=${CURDIR}/tmp/AWS-S3-Buckets-Location-Constraint-stderr.tmp
    Should Be Equal As Integers    ${result.rc}           0
    Should Contain                 ${result.stdout}       ap\-southeast\-1

AWS S3 Buckets List
    ${bucketsListQuery} =    Catenate
    ...    select * from aws.s3.buckets where region = 'us-east-1' order by BucketArn desc;
   ${result} =    Run Process
    ...    ${STACKQL_EXE}
    ...    \-\-registry
    ...    { "url": "file://${REPOSITORY_ROOT}/test/registry", "localDocRoot": "${REPOSITORY_ROOT}/test/registry", "verifyConfig": { "nopVerify": true } }
    ...    exec
    ...    ${bucketsListQuery}
    ...    cwd=${REPOSITORY_ROOT}
    ...    stdout=${CURDIR}/tmp/AWS-S3-Buckets-List.tmp
    ...    stderr=${CURDIR}/tmp/AWS-S3-Buckets-List-stderr.tmp
    Should Be Equal As Integers    ${result.rc}           0
    Should Contain                 ${result.stdout}       stackql\-trial\-bucket\-02

AWS S3 Bucket Objects List
    ${bucketObjectsListQuery} =    Catenate
    ...    select * from aws.s3.objects where Bucket = 'stackql-trial-bucket-02' and region = 'ap-southeast-2';
    ${result} =    Run Process
    ...    ${STACKQL_EXE}
    ...    \-\-registry
    ...    { "url": "file://${REPOSITORY_ROOT}/test/registry", "localDocRoot": "${REPOSITORY_ROOT}/test/registry", "verifyConfig": { "nopVerify": true } }
    ...    exec
    ...    ${bucketObjectsListQuery}
    ...    cwd=${REPOSITORY_ROOT}
    ...    stdout=${CURDIR}/tmp/AWS-S3-Bucket-Objects-List.tmp
    ...    stderr=${CURDIR}/tmp/AWS-S3-Bucket-Objects-List-stderr.tmp
    Should Be Equal As Integers    ${result.rc}           0
    Should Contain                 ${result.stdout}       docs/advanced
