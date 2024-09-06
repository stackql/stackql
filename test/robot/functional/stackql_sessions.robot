*** Settings ***
Resource          ${CURDIR}/stackql.resource

*** Test Cases *** 
Shell Session Simple
    Pass Execution If    "${IS_WINDOWS}" == "1"    Skipping session test in windows
    Should StackQL Shell Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SHELL_SESSION_SIMPLE_COMMANDS}
    ...    ${SHELL_SESSION_SIMPLE_EXPECTED}
    ...    stdout=${CURDIR}/tmp/Shell-Session-Simple.tmp
    [Teardown]    Stackql Per Test Teardown

Shell Session Azure Compute Table Nomenclature Mutation Guard
    Pass Execution If    "${IS_WINDOWS}" == "1"    Skipping session test in windows
    Should StackQL Shell Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SHELL_COMMANDS_AZURE_COMPUTE_MUTATION_GUARD}
    ...    ${SHELL_COMMANDS_AZURE_COMPUTE_MUTATION_GUARD_EXPECTED}
    ...    stdout=${CURDIR}/tmp/Shell-Session-Azure-Compute-Table-Nomenclature-Mutation-Guard.tmp
    [Teardown]    Stackql Per Test Teardown

PG Session GC Manual Behaviour Canonical
    Should PG Client Session Inline Equal Strict
    ...    ${PSQL_MTLS_CONN_STR_UNIX}
    ...    ${SHELL_COMMANDS_GC_SEQUENCE_CANONICAL}
    ...    ${SHELL_COMMANDS_GC_SEQUENCE_CANONICAL_JSON_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-GC-Manual-Behaviour-Canonical.tmp
    [Teardown]    NONE

PG Session GC Eager Behaviour Canonical
    Should PG Client Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX_WITH_EAGER_GC}
    ...    ${SHELL_COMMANDS_GC_SEQUENCE_EAGER}
    ...    ${SHELL_COMMANDS_GC_SEQUENCE_EAGER_JSON_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-GC-Eager-Behaviour-Canonical.tmp
    [Teardown]    NONE

PG Session View Handling Behaviour Canonical
    Should PG Client Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX_WITH_EAGER_GC}
    ...    ${SHELL_COMMANDS_VIEW_HANDLING_SEQUENCE}
    ...    ${SHELL_COMMANDS_VIEW_HANDLING_SEQUENCE_JSON_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-View-Handling-Behaviour-Canonical.tmp
    [Teardown]    NONE

PG Session View Handling With Replacement Behaviour Canonical
    Should PG Client Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX_WITH_EAGER_GC}
    ...    ${SHELL_COMMANDS_VIEW_HANDLING_WITH_REPLACEMENT_SEQUENCE}
    ...    ${SHELL_COMMANDS_VIEW_HANDLING_WITH_REPLACEMENT_SEQUENCE_JSON_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-View-Handling-With-Replacement-Behaviour-Canonical.tmp
    [Teardown]    NONE

PG Session Aliased Cross Cloud Disks View Handling Behaviour Canonical
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Skipping postgres backend test due to split_part function
    Should PG Client Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX_WITH_EAGER_GC}
    ...    ${SHELL_COMMANDS_DISKS_VIEW_ALIASED_SEQUENCE}
    ...    ${SHELL_COMMANDS_DISKS_VIEW_ALIASED_SEQUENCE_JSON_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Aliased-Cross-Cloud-Disks-View-Handling-Behaviour-Canonical.tmp
    [Teardown]    NONE

PG Session NOT Aliased Cross Cloud Disks View Handling Behaviour Canonical
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Skipping postgres backend test due to split_part function
    Should PG Client Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX_WITH_EAGER_GC}
    ...    ${SHELL_COMMANDS_DISKS_VIEW_NOT_ALIASED_SEQUENCE}
    ...    ${SHELL_COMMANDS_DISKS_VIEW_NOT_ALIASED_SEQUENCE_JSON_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-NOT-Aliased-Cross-Cloud-Disks-View-Handling-Behaviour-Canonical.tmp
    [Teardown]    NONE

PG Session Azure Compute Table Nomenclature Mutation Guard
    Should PG Client Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX}
    ...    ${SHELL_COMMANDS_AZURE_COMPUTE_MUTATION_GUARD}
    ...    ${SHELL_COMMANDS_AZURE_COMPUTE_MUTATION_GUARD_JSON_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Azure-Compute-Table-Nomenclature-Mutation-Guard.tmp
    [Teardown]    NONE

PG Session Materialized View Lifecycle
    ${inputStr} =    Catenate
    ...    [
    ...    "create materialized view vw_aws_usr as select Arn, UserName, UserId, region from aws.iam.users where region = 'us-east-1';",
    ...    "select u1.UserName, u2.UserId, u2.Arn, u1.region from aws.iam.users u1 inner join vw_aws_usr u2 on u1.Arn = u2.Arn where u1.region = 'us-east-1' and u2.region = 'us-east-1' order by u1.UserName desc;",
    ...    "drop materialized view vw_aws_usr;"
    ...    ]
    ${inputList} =    Evaluate    ${inputStr}
    ${outputStr} =    Catenate
    ...    [
    ...    { "UserName":  "Jackie", "UserId": "AIDIODR4TAW7CSEXAMPLE", "Arn": "arn:aws:iam::123456789012:user/division_abc/subdivision_xyz/engineering/Jackie", "region": "us-east-1" },
    ...    { "UserName":  "Andrew", "UserId": "AID2MAB8DPLSRHEXAMPLE", "Arn": "arn:aws:iam::123456789012:user/division_abc/subdivision_xyz/engineering/Andrew", "region": "us-east-1" }
    ...    ]
    ${outputList} =    Evaluate    ${outputStr}
    Should PG Client Session Inline Equal
    ...    ${POSTGRES_URL_UNENCRYPTED_CONN}
    ...    ${inputList}
    ...    ${outputList}
    ...    stdout=${CURDIR}/tmp/PG-Session-Materialized-View-Lifecycle.tmp
    ...    stderr=${CURDIR}/tmp/PG-Session-Materialized-View-Lifecycle-stderr.tmp
    [Teardown]    NONE

PG Session V2 Materialized View Lifecycle
    ${inputStr} =    Catenate
    ...    [
    ...    "create materialized view vw_aws_usr as select Arn, UserName, UserId, region from aws.iam.users where region = 'us-east-1';",
    ...    "select u1.UserName, u2.UserId, u2.Arn, u1.region from aws.iam.users u1 inner join vw_aws_usr u2 on u1.Arn = u2.Arn where u1.region = 'us-east-1' and u2.region = 'us-east-1' order by u1.UserName desc;",
    ...    "drop materialized view vw_aws_usr;"
    ...    ]
    ${inputList} =    Evaluate    ${inputStr}
    ${outputStr} =    Catenate
    ...    [
    ...    { "UserName":  "Jackie", "UserId": "AIDIODR4TAW7CSEXAMPLE", "Arn": "arn:aws:iam::123456789012:user/division_abc/subdivision_xyz/engineering/Jackie", "region": "us-east-1" },
    ...    { "UserName":  "Andrew", "UserId": "AID2MAB8DPLSRHEXAMPLE", "Arn": "arn:aws:iam::123456789012:user/division_abc/subdivision_xyz/engineering/Andrew", "region": "us-east-1" }
    ...    ]
    ${outputList} =    Evaluate    ${outputStr}
    Should PG Client V2 Session Inline Equal
    ...    ${POSTGRES_URL_UNENCRYPTED_CONN}
    ...    ${inputList}
    ...    ${outputList}
    ...    stdout=${CURDIR}/tmp/PG-Session-v2-Materialized-View-Lifecycle.tmp
    ...    stderr=${CURDIR}/tmp/PG-Session-v2-Materialized-View-Lifecycle-stderr.tmp
    [Teardown]    NONE

PG Session Wrongly Named Column error recovery Azure Compute Table Nomenclature
    Should PG Client Session Inline Contain
    ...    ${PSQL_MTLS_CONN_STR_UNIX}
    ...    ${SHELL_SESSION_SIMPLE_COMMANDS_AFTER_ERROR}
    ...    ${SHELL_SESSION_SIMPLE_COMMANDS_AFTER_ERROR_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Wrongly-Named-Column-error-recovery-Azure-Compute-Table-Nomenclature.tmp
    [Teardown]    NONE

Shell Session Azure Billing Path Interrogation Regression Guard
    Should PG Client Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX}
    ...    ${SHELL_COMMANDS_AZURE_BILLING_PATH_SPLIT_GUARD}
    ...    ${SHELL_COMMANDS_AZURE_BILLING_PATH_SPLIT_GUARD_JSON_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Azure-Compute-Table-Nomenclature-Mutation-Guard.tmp
    [Teardown]    NONE

PG Session Anayltics Cache Behaviour Canonical
    Should PG Client Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX_WITH_NAMESPACES}
    ...    ${SHELL_COMMANDS_SPECIALCASE_REPEATED_CACHED}
    ...    ${SHELL_COMMANDS_SPECIALCASE_REPEATED_CACHED_JSON_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Anayltics-Cache-Behaviour-Canonical.tmp
    [Teardown]    NONE

PG Session Postgres Client Setup Queries
    Should PG Client Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX}
    ...    ${PG_CLIENT_SETUP_QUERIES}
    ...    ${PG_CLIENT_SETUP_QUERIES_JSON_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Postgres-Client-Setup-Queries.tmp
    [Teardown]    NONE

PG Session Postgres Client V2 Setup Queries
    Should PG Client V2 Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX}
    ...    ${PG_CLIENT_SETUP_QUERIES}
    ...    ${PG_CLIENT_SETUP_QUERIES_JSON_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Postgres-Client-V2-Setup-Queries.tmp
    [Teardown]    NONE

PG Session Postgres Client AWS Method Signature Polymorphism
    Should PG Client Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX}
    ...    ${AWS_CLOUD_CONTROL_METHOD_SIGNATURE_CMD_ARR}
    ...    ${AWS_CLOUD_CONTROL_METHOD_SIGNATURE_CMD_ARR_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Postgres-Client-AWS-Method-Signature-Polymorphism.tmp
    [Teardown]    NONE

PG Session Postgres Client Typed Queries
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Skipping postgres backend test likely due to typing issues
    Should PG Client Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX}
    ...    ${SELECT_AWS_CLOUD_CONTROL_EVENTS_MINIMAL}
    ...    ${SELECT_AWS_CLOUD_CONTROL_EVENTS_MINIMAL_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Postgres-Client-Typed-Queries.tmp
    [Teardown]    NONE

PG Session Server Survives Defective Query
    Should PG Client Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX}
    ...    ${AWS_CLOUD_CONTROL_BUCKET_DETAIL_PROJECTION_DEFECTIVE_CMD_ARR}
    ...    ${AWS_CLOUD_CONTROL_BUCKET_DETAIL_PROJECTION_DEFECTIVE_CMD_ARR_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Server-Survives-Defective-Query-and-Subsequently-Serves-Valid-Query.tmp
    [Teardown]    NONE

PG Session Postgres Client V2 Typed Queries
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"     TODO: FIX THIS... Skipping postgres backend test likely due to typing issues
    Should PG Client V2 Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX}
    ...    ${SELECT_AWS_CLOUD_CONTROL_EVENTS_MINIMAL}
    ...    ${SELECT_AWS_CLOUD_CONTROL_EVENTS_MINIMAL_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Postgres-Client-V2-Typed-Queries.tmp
    [Teardown]    NONE



High Volume IN Query Is Correct and Performs OK
    Pass Execution If    "${CONCURRENCY_LIMIT}" == "1"    We only expect performance when concurrency settings are aggressive.
    ${inputStr} =    Catenate
    ...              select 
    ...              instanceId, 
    ...              ipAddress 
    ...              from aws.ec2.instances 
    ...              where 
    ...              instanceId not in ('some-silly-id')  
    ...              and region in (
    ...              'us-east-1',
    ...              'us-east-2',
    ...              'us-west-1',
    ...              'us-west-2',
    ...              'ap-south-1',
    ...              'ap-northeast-3',
    ...              'ap-northeast-2',
    ...              'ap-southeast-1',
    ...              'ap-southeast-2',
    ...              'ap-northeast-1',
    ...              'ca-central-1',
    ...              'eu-central-1',
    ...              'eu-west-1',
    ...              'eu-west-2',
    ...              'eu-west-3',
    ...              'eu-north-1',
    ...              'sa-east-1'
    ...              )
    ...              ;
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...               ${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}instanceId${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}ipAddress${SPACE}${SPACE}${SPACE}${SPACE}
    ...               ---------------------+----------------
    ...               ${SPACE}i-1234567890abcdef0${SPACE}|${SPACE}54.194.252.215
    ...               ${SPACE}i-1234567890abcdef0${SPACE}|${SPACE}54.194.252.215
    ...               ${SPACE}i-1234567890abcdef0${SPACE}|${SPACE}54.194.252.215
    ...               ${SPACE}i-1234567890abcdef0${SPACE}|${SPACE}54.194.252.215
    ...               ${SPACE}i-1234567890abcdef0${SPACE}|${SPACE}54.194.252.215
    ...               ${SPACE}i-1234567890abcdef0${SPACE}|${SPACE}54.194.252.215
    ...               ${SPACE}i-1234567890abcdef0${SPACE}|${SPACE}54.194.252.215
    ...               ${SPACE}i-1234567890abcdef0${SPACE}|${SPACE}54.194.252.215
    ...               ${SPACE}i-1234567890abcdef0${SPACE}|${SPACE}54.194.252.215
    ...               ${SPACE}i-1234567890abcdef0${SPACE}|${SPACE}54.194.252.215
    ...               ${SPACE}i-1234567890abcdef0${SPACE}|${SPACE}54.194.252.215
    ...               ${SPACE}i-1234567890abcdef0${SPACE}|${SPACE}54.194.252.215
    ...               ${SPACE}i-1234567890abcdef0${SPACE}|${SPACE}54.194.252.215
    ...               ${SPACE}i-1234567890abcdef0${SPACE}|${SPACE}54.194.252.215
    ...               ${SPACE}i-1234567890abcdef0${SPACE}|${SPACE}54.194.252.215
    ...               ${SPACE}i-1234567890abcdef0${SPACE}|${SPACE}54.194.252.215
    ...               ${SPACE}i-1234567890abcdef0${SPACE}|${SPACE}54.194.252.215
    ...               (17${SPACE}rows)
    ...               ${EMPTY}
    Should PG Client Inline Equal Bench
    ...    ${CURDIR}
    ...    ${PSQL_EXE}
    ...    ${PSQL_MTLS_CONN_STR}
    ...    ${inputStr}
    ...    ${outputStr}
    ...    max_mean_time=1.7

Acceptable Secure Connection to mTLS Server Returns Success Message
    ${input} =     Catenate
    ...    echo     ""     |
    ...    openssl     s_client     -starttls    postgres 
    ...    -connect     ${PSQL_CLIENT_HOST}:${PG_SRV_PORT_MTLS}
    ...    -cert       ${STACKQL_PG_CLIENT_CERT_PATH}
    ...    -key        ${STACKQL_PG_CLIENT_KEY_PATH}
    ...    -CAfile     ${STACKQL_PG_SERVER_CERT_PATH}
    ${shellExe} =    Set Variable If    "${IS_WINDOWS}" == "1"    powershell    sh
    ${result} =    Run Process
    ...    ${shellExe}     \-c    ${input}
    ...    stdout=${CURDIR}/tmp/Acceptable-Secure-Connection-to-mTLS-Server-Returns-Success-Message.tmp
    ...    stderr=${CURDIR}/tmp/Acceptable-Secure-Connection-to-mTLS-Server-Returns-Success-Message-stderr.tmp
    Should Contain    ${result.stdout}    Verify return code: 0

Unacceptable Insecure Connection to mTLS Server Returns Error Message
    ${input} =     Catenate
    ...    echo     ""     |
    ...    openssl     s_client     -starttls    postgres 
    ...    -connect     ${PSQL_CLIENT_HOST}:${PG_SRV_PORT_MTLS}
    ...    -cert       ${STACKQL_PG_CLIENT_CERT_PATH}
    ...    -key        ${STACKQL_PG_CLIENT_KEY_PATH}
    ...    -CAfile     ${STACKQL_PG_RUBBISH_CERT_PATH}
    ${shellExe} =    Set Variable If    "${IS_WINDOWS}" == "1"    powershell    sh
    ${result} =    Run Process
    ...    ${shellExe}     \-c    ${input}
    ...    stdout=${CURDIR}/tmp/Unacceptable-Insecure-Connection-to-mTLS-Server-Returns-Error-Message.tmp
    ...    stderr=${CURDIR}/tmp/Unacceptable-Insecure-Connection-to-mTLS-Server-Returns-Error-Message-stderr.tmp
    Should Contain    ${result.stdout}    Verify return code: 18

Acceptable Secure PSQL Connection to mTLS Server With Diagnostic Query Returns Connection Info
    ${posixInput} =     Catenate
    ...    "${PSQL_EXE}"    -d     "${PSQL_MTLS_CONN_STR_UNIX}" -c "\\conninfo"
    ${windowsInput} =     Catenate
    ...    &    ${posixInput}
    ${input} =    Set Variable If    "${IS_WINDOWS}" == "1"    ${windowsInput}    ${posixInput}
    ${shellExe} =    Set Variable If    "${IS_WINDOWS}" == "1"    powershell    sh
    ${result} =    Run Process
    ...    ${shellExe}     \-c    ${input}
    ...    stdout=${CURDIR}/tmp/Acceptable-Secure-PSQL-Connection-to-mTLS-Server-With-Diagnostic-Query-Returns-Connection-Info.tmp
    ...    stderr=${CURDIR}/tmp/Acceptable-Secure-PSQL-Connection-to-mTLS-Server-With-Diagnostic-Query-Returns-Connection-Info-stderr.tmp
    Should Contain    ${result.stdout}    SSL connection (protocol: TLSv1.3

Acceptable Password Only PSQL Connection Defined by Env Vars to Server With Diagnostic Query Returns Connection Info
    Set Environment Variable    PGHOST           ${PSQL_CLIENT_HOST}
    Set Environment Variable    PGPORT           ${PG_SRV_PORT_UNENCRYPTED}
    Set Environment Variable    PGUSER           stackql
    Set Environment Variable    PGPASSWORD       ${PSQL_PASSWORD} 
    ${posixInput} =     Catenate
    ...    "${PSQL_EXE}" -c "\\conninfo"
    ${windowsInput} =     Catenate
    ...    &    ${posixInput}
    ${input} =    Set Variable If    "${IS_WINDOWS}" == "1"    ${windowsInput}    ${posixInput}
    ${shellExe} =    Set Variable If    "${IS_WINDOWS}" == "1"    powershell    sh
    ${result} =    Run Process
    ...    ${shellExe}     \-c    ${input}
    ...    stdout=${CURDIR}/tmp/Acceptable-Password-Only-PSQL-Connection-Defined-by-Env-Vars-to-Server-With-Diagnostic-Query-Returns-Connection-Info.tmp
    ...    stderr=${CURDIR}/tmp/Acceptable-Password-Only-PSQL-Connection-Defined-by-Env-Vars-to-Server-With-Diagnostic-Query-Returns-Connection-Info-stderr.tmp
    Should Contain    ${result.stdout}    You are connected to database
    [Teardown]  Run Keywords    Remove Environment Variable     PGHOST
    ...         AND             Remove Environment Variable     PGPORT
    ...         AND             Remove Environment Variable     PGUSER 
    ...         AND             Remove Environment Variable     PGPASSWORD

Acceptable Secure PSQL Connection Defined by Env Vars to mTLS Server With Diagnostic Query Returns Connection Info
    Set Environment Variable    PGHOST           ${PSQL_CLIENT_HOST}
    Set Environment Variable    PGPORT           ${PG_SRV_PORT_MTLS}
    Set Environment Variable    PGSSLMODE        verify\-full 
    Set Environment Variable    PGSSLCERT        ${STACKQL_PG_CLIENT_CERT_PATH} 
    Set Environment Variable    PGSSLKEY         ${STACKQL_PG_CLIENT_KEY_PATH} 
    Set Environment Variable    PGSSLROOTCERT    ${STACKQL_PG_SERVER_CERT_PATH} 
    ${posixInput} =     Catenate
    ...    "${PSQL_EXE}" -c "\\conninfo"
    ${windowsInput} =     Catenate
    ...    &    ${posixInput}
    ${input} =    Set Variable If    "${IS_WINDOWS}" == "1"    ${windowsInput}    ${posixInput}
    ${shellExe} =    Set Variable If    "${IS_WINDOWS}" == "1"    powershell    sh
    ${result} =    Run Process
    ...    ${shellExe}     \-c    ${input}
    ...    stdout=${CURDIR}/tmp/Acceptable-Secure-PSQL-Connection-Defined-by-Env-Vars-to-mTLS-Server-With-Diagnostic-Query-Returns-Connection-Info.tmp
    ...    stderr=${CURDIR}/tmp/Acceptable-Secure-PSQL-Connection-Defined-by-Env-Vars-to-mTLS-Server-With-Diagnostic-Query-Returns-Connection-Info-stderr.tmp
    Should Contain    ${result.stdout}    SSL connection (protocol: TLSv1.3
    [Teardown]  Run Keywords    Remove Environment Variable     PGHOST
    ...         AND             Remove Environment Variable     PGPORT
    ...         AND             Remove Environment Variable     PGSSLMODE 
    ...         AND             Remove Environment Variable     PGSSLCERT 
    ...         AND             Remove Environment Variable     PGSSLKEY
    ...         AND             Remove Environment Variable     PGSSLROOTCERT

Unacceptable Insecure PSQL Connection to mTLS Server Returns Error Message
    ${posixInput} =     Catenate
    ...    "${PSQL_EXE}"    -d    "${PSQL_MTLS_DISABLE_CONN_STR_UNIX}" -c "\\conninfo"
    ${windowsInput} =     Catenate
    ...    &    ${posixInput}
    ${input} =    Set Variable If    "${IS_WINDOWS}" == "1"    ${windowsInput}    ${posixInput}
    ${shellExe} =    Set Variable If    "${IS_WINDOWS}" == "1"    powershell    sh
    ${result} =    Run Process
    ...    ${shellExe}     \-c    ${input}
    ...    stdout=${CURDIR}/tmp/Unacceptable-Insecure-PSQL-Connection-to-mTLS-Server-Returns-Error-Message.tmp
    ...    stderr=${CURDIR}/tmp/Unacceptable-Insecure-PSQL-Connection-to-mTLS-Server-Returns-Error-Message-stderr.tmp
    Should Contain    ${result.stderr}    server closed the connection unexpectedly
