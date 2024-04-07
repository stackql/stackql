*** Settings ***
Resource          ${CURDIR}/stackql.resource
Test Setup        Remove File    ${EXPORT_SQLITE_FILE_PATH}
Test Teardown     Stackql Per Test Teardown

*** Test Cases *** 
Export Materialized View and then Access From RDBMSs
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Engineer a way to do postgres and sqlite export testing in same test case
    ${ddlInputStr} =    Catenate
    ...    create materialized view nv as select BackupId, BackupState 
    ...    from aws.cloudhsm.backups where region = 'ap-southeast-2' order by BackupId;
    ${ddlOutputStr} =    Catenate    SEPARATOR=\n
    ...    DDL Execution Completed
    ${queryStringSQLite} =    Catenate
    ...    select BackupId, BackupState from "stackql_export.nv" order by BackupId;
    ${queryStringPostgres} =    Catenate
    ...    select "BackupId", "BackupState" from stackql_export.nv order by "BackupId";
    ${dbName} =    Set Variable If    "${SQL_BACKEND}" == "postgres_tcp"     postgres    sqlite
    ${queryString} =    Set Variable If    "${SQL_BACKEND}" == "postgres_tcp"     ${queryStringPostgres}    ${queryStringSQLite}
    ${queryOutputStr} =    Catenate    SEPARATOR=\n
    ...    bkp-000001,READY
    ...    bkp-000002,READY
    Should Stackql Exec Inline Equal Both Streams
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_CLIENT_EXPORT_BACKEND}
    ...    ${ddlInputStr}
    ...    ${EMPTY}
    ...    ${ddlOutputStr}
    ...    \-\-export.alias\=stackql_export
    ...    stdout=${CURDIR}/tmp/Export-Materialized-View-and-then-Access-From-RDBMSs.tmp
    ...    stderr=${CURDIR}/tmp/Export-Materialized-View-and-then-Access-From-RDBMSs-stderr.tmp
    Should RDBMS Query Return CSV Result
    ...    ${dbName}
    ...    ${SQL_CLIENT_EXPORT_CONNECTION_ARG}
    ...    ${queryString}
    ...    ${queryOutputStr}
    ...    ${EMPTY}
    ...    stdout=${CURDIR}/tmp/Export-Materialized-View-and-then-Access-From-RDBMSs-stage-2.tmp
    ...    stderr=${CURDIR}/tmp/Export-Materialized-View-and-then-Access-From-RDBMSs-stage-2-stderr.tmp

Export Materialized View and then Access From RDBMSs Over Stackql Postgres Server
    Pass Execution If    "${SQL_BACKEND}" != "postgres_tcp"    TODO: FIX THIS... Engineer a way to do postgres and sqlite export testing in same test case
    ${ddlInputStr} =    Catenate
    ...    create materialized view nv as select BackupId, BackupState 
    ...    from aws.cloudhsm.backups where region = 'ap-southeast-2' order by BackupId;
    ${posixInput} =     Catenate
    ...    "${PSQL_EXE}"    -d     "${PSQL_MTLS_CONN_STR_EXPORT_UNIX}"   -c   "${ddlInputStr}"
    ${windowsInput} =     Catenate
    ...    &    ${posixInput}
    ${input} =    Set Variable If    "${IS_WINDOWS}" == "1"    ${windowsInput}    ${posixInput}
    ${shellExe} =    Set Variable If    "${IS_WINDOWS}" == "1"    powershell    sh
    ${result} =    Run Process
    ...    ${shellExe}     \-c    ${input}
    ...    stdout=${CURDIR}/tmp/Export-Materialized-View-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server.tmp
    ...    stderr=${CURDIR}/tmp/Export-Materialized-View-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server-stderr.tmp
    Should Be Equal    ${result.stdout}    OK
    ${queryStringSQLite} =    Catenate
    ...    select BackupId, BackupState from "stackql_export.nv" order by BackupId;
    ${queryStringPostgres} =    Catenate
    ...    select "BackupId", "BackupState" from stackql_export.nv order by "BackupId";
    ${dbName} =    Set Variable If    "${SQL_BACKEND}" == "postgres_tcp"     postgres    sqlite
    ${queryString} =    Set Variable If    "${SQL_BACKEND}" == "postgres_tcp"     ${queryStringPostgres}    ${queryStringSQLite}
    ${queryOutputStr} =    Catenate    SEPARATOR=\n
    ...    BackupId,BackupState
    ...    bkp-000001,READY
    ...    bkp-000002,READY
    Should RDBMS Query Return CSV Result
    ...    ${dbName}
    ...    ${SQL_CLIENT_EXPORT_CONNECTION_ARG}
    ...    ${queryString}
    ...    ${queryOutputStr}
    ...    ${EMPTY}
    ...    stdout=${CURDIR}/tmp/Export-Materialized-View-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server-stage-2.tmp
    ...    stderr=${CURDIR}/tmp/Export-Materialized-View-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server-stage-2-stderr.tmp 