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

Export User Space Table and then Access From RDBMSs
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Engineer a way to do postgres and sqlite export testing in same test case
    ${ddlInputStr} =    Catenate
    ...    create table my_silly_export_table(id int, name text, magnitude numeric);
    ...    insert into my_silly_export_table(id, name, magnitude) values (1, 'one', 1.0);
    ...    insert into my_silly_export_table(id, name, magnitude) values (2, 'two', 2.0);
    ${ddlOutputStr} =    Catenate    SEPARATOR=\n
    ...    DDL Execution Completed
    ...    insert into table completed
    ...    insert into table completed
    ${queryStringSQLite} =    Catenate
    ...    select id, name, magnitude from "stackql_export.my_silly_export_table" order by id;
    ${queryStringPostgres} =    Catenate
    ...    select id, name, magnitude from stackql_export.my_silly_export_table order by id;
    ${dbName} =    Set Variable If    "${SQL_BACKEND}" == "postgres_tcp"     postgres    sqlite
    ${queryString} =    Set Variable If    "${SQL_BACKEND}" == "postgres_tcp"     ${queryStringPostgres}    ${queryStringSQLite}
    ${queryOutputStr} =    Catenate    SEPARATOR=\n
    ...    1,one,1
    ...    2,two,2
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
    ...    stdout=${CURDIR}/tmp/Export-User-Space-Table-and-then-Access-From-RDBMSs.tmp
    ...    stderr=${CURDIR}/tmp/Export-User-Space-Table-and-then-Access-From-RDBMSs-stderr.tmp
    Should RDBMS Query Return CSV Result
    ...    ${dbName}
    ...    ${SQL_CLIENT_EXPORT_CONNECTION_ARG}
    ...    ${queryString}
    ...    ${queryOutputStr}
    ...    ${EMPTY}
    ...    stdout=${CURDIR}/tmp/Export-User-Space-Table-and-then-Access-From-RDBMSs-stage-2.tmp
    ...    stderr=${CURDIR}/tmp/Export-User-Space-Table-and-then-Access-From-RDBMSs-stage-2-stderr.tmp

Export User Space Table and then Access From RDBMSs Over Stackql Postgres Server
    Pass Execution If    "${SQL_BACKEND}" != "postgres_tcp"    TODO: FIX THIS... Engineer a way to do postgres and sqlite export testing in same test case
    ${ddlInputStr} =    Catenate
    ...    create table my_silly_export_table_two(id int, name text, magnitude numeric);
    ...    insert into my_silly_export_table_two(id, name, magnitude) values (1, 'one', 1.0);
    ...    insert into my_silly_export_table_two(id, name, magnitude) values (2, 'two', 2.0);
    ${posixInput} =     Catenate
    ...    "${PSQL_EXE}"    -d     "${PSQL_MTLS_CONN_STR_EXPORT_UNIX}"   -c   "${ddlInputStr}"
    ${windowsInput} =     Catenate
    ...    &    ${posixInput}
    ${input} =    Set Variable If    "${IS_WINDOWS}" == "1"    ${windowsInput}    ${posixInput}
    ${shellExe} =    Set Variable If    "${IS_WINDOWS}" == "1"    powershell    sh
    ${result} =    Run Process
    ...    ${shellExe}     \-c    ${input}
    ...    stdout=${CURDIR}/tmp/Export-User-Space-Table-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server.tmp
    ...    stderr=${CURDIR}/tmp/Export-User-Space-Table-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server-stderr.tmp
    Should Be Equal    ${result.stdout}    OK
    ${ddlInputStr} =    Catenate
    ...    insert into my_silly_export_table_two(id, name, magnitude) values (1, 'one', 1.0);
    ${posixInput} =     Catenate
    ...    "${PSQL_EXE}"    -d     "${PSQL_MTLS_CONN_STR_EXPORT_UNIX}"   -c   "${ddlInputStr}"
    ${windowsInput} =     Catenate
    ...    &    ${posixInput}
    ${input} =    Set Variable If    "${IS_WINDOWS}" == "1"    ${windowsInput}    ${posixInput}
    ${shellExe} =    Set Variable If    "${IS_WINDOWS}" == "1"    powershell    sh
    ${result} =    Run Process
    ...    ${shellExe}     \-c    ${input}
    ...    stdout=${CURDIR}/tmp/Export-User-Space-Table-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server-insert-one.tmp
    ...    stderr=${CURDIR}/tmp/Export-User-Space-Table-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server-insert-one-stderr.tmp
    Should Be Equal    ${result.stdout}    OK
    ${ddlInputStr} =    Catenate
    ...    insert into my_silly_export_table_two(id, name, magnitude) values (2, 'two', 2.0);
    ${posixInput} =     Catenate
    ...    "${PSQL_EXE}"    -d     "${PSQL_MTLS_CONN_STR_EXPORT_UNIX}"   -c   "${ddlInputStr}"
    ${windowsInput} =     Catenate
    ...    &    ${posixInput}
    ${input} =    Set Variable If    "${IS_WINDOWS}" == "1"    ${windowsInput}    ${posixInput}
    ${shellExe} =    Set Variable If    "${IS_WINDOWS}" == "1"    powershell    sh
    ${result} =    Run Process
    ...    ${shellExe}     \-c    ${input}
    ...    stdout=${CURDIR}/tmp/Export-User-Space-Table-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server-insert-two.tmp
    ...    stderr=${CURDIR}/tmp/Export-User-Space-Table-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server-insert-two-stderr.tmp
    Should Be Equal    ${result.stdout}    OK
    ${queryStringSQLite} =    Catenate
    ...    select id, name, magnitude from "stackql_export.my_silly_export_table_two" order by id;
    ${queryStringPostgres} =    Catenate
    ...    select id, name, magnitude from stackql_export.my_silly_export_table_two order by id;
    ${dbName} =    Set Variable If    "${SQL_BACKEND}" == "postgres_tcp"     postgres    sqlite
    ${queryString} =    Set Variable If    "${SQL_BACKEND}" == "postgres_tcp"     ${queryStringPostgres}    ${queryStringSQLite}
    ${queryOutputStr} =    Catenate    SEPARATOR=\n
    ...    id,name,magnitude
    ...    1,one,1.0
    ...    2,two,2.0
    Should RDBMS Query Return CSV Result
    ...    ${dbName}
    ...    ${SQL_CLIENT_EXPORT_CONNECTION_ARG}
    ...    ${queryString}
    ...    ${queryOutputStr}
    ...    ${EMPTY}
    ...    stdout=${CURDIR}/tmp/Export-User-Space-Table-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server-stage-2.tmp
    ...    stderr=${CURDIR}/tmp/Export-User-Space-Table-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server-stage-2-stderr.tmp 

Lifecycle Export Materialized View and then Access From RDBMSs
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Engineer a way to do postgres and sqlite export testing in same test case
    ${ddlInputStr} =    Catenate
    ...    create materialized view nv as select 'junk' as id, BackupId, BackupState 
    ...    from aws.cloudhsm.backups where region = 'ap-southeast-2' order by BackupId;
    ...    create or replace materialized view nv as select BackupId, BackupState 
    ...    from aws.cloudhsm.backups where region = 'ap-southeast-2' order by BackupId;
    ${ddlOutputStr} =    Catenate    SEPARATOR=\n
    ...    DDL Execution Completed
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
    ...    stdout=${CURDIR}/tmp/Lifecycle-Export-Materialized-View-and-then-Access-From-RDBMSs.tmp
    ...    stderr=${CURDIR}/tmp/Lifecycle-Export-Materialized-View-and-then-Access-From-RDBMSs-stderr.tmp
    Should RDBMS Query Return CSV Result
    ...    ${dbName}
    ...    ${SQL_CLIENT_EXPORT_CONNECTION_ARG}
    ...    ${queryString}
    ...    ${queryOutputStr}
    ...    ${EMPTY}
    ...    stdout=${CURDIR}/tmp/Lifecycle-Export-Materialized-View-and-then-Access-From-RDBMSs-stage-2.tmp
    ...    stderr=${CURDIR}/tmp/Lifecycle-Export-Materialized-View-and-then-Access-From-RDBMSs-stage-2-stderr.tmp

Lifecycle Export Materialized View and then Access From RDBMSs Over Stackql Postgres Server
    Pass Execution If    "${SQL_BACKEND}" != "postgres_tcp"    TODO: FIX THIS... Engineer a way to do postgres and sqlite export testing in same test case
    ${ddlInputStr} =    Catenate
    ...    create materialized view nv as select 'junk' as c1, BackupId, BackupState 
    ...    from aws.cloudhsm.backups where region = 'ap-southeast-2' order by BackupId;
    ${posixInput} =     Catenate
    ...    "${PSQL_EXE}"    -d     "${PSQL_MTLS_CONN_STR_EXPORT_UNIX}"   -c   "${ddlInputStr}"
    ${windowsInput} =     Catenate
    ...    &    ${posixInput}
    ${input} =    Set Variable If    "${IS_WINDOWS}" == "1"    ${windowsInput}    ${posixInput}
    ${shellExe} =    Set Variable If    "${IS_WINDOWS}" == "1"    powershell    sh
    ${result} =    Run Process
    ...    ${shellExe}     \-c    ${input}
    ...    stdout=${CURDIR}/tmp/Lifecycle-Export-Materialized-View-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server.tmp
    ...    stderr=${CURDIR}/tmp/Lifecycle-Export-Materialized-View-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server-stderr.tmp
    Should Be Equal    ${result.stdout}    OK
    ${ddlInputStr} =    Catenate
    ...    create or replace materialized view nv as select BackupId, BackupState 
    ...    from aws.cloudhsm.backups where region = 'ap-southeast-2' order by BackupId;
    ${posixInput} =     Catenate
    ...    "${PSQL_EXE}"    -d     "${PSQL_MTLS_CONN_STR_EXPORT_UNIX}"   -c   "${ddlInputStr}"
    ${windowsInput} =     Catenate
    ...    &    ${posixInput}
    ${input} =    Set Variable If    "${IS_WINDOWS}" == "1"    ${windowsInput}    ${posixInput}
    ${shellExe} =    Set Variable If    "${IS_WINDOWS}" == "1"    powershell    sh
    ${result} =    Run Process
    ...    ${shellExe}     \-c    ${input}
    ...    stdout=${CURDIR}/tmp/Lifecycle-Export-Materialized-View-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server.tmp
    ...    stderr=${CURDIR}/tmp/Lifecycle-Export-Materialized-View-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server-stderr.tmp
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
    ...    stdout=${CURDIR}/tmp/Lifecycle-Export-Materialized-View-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server-stage-2.tmp
    ...    stderr=${CURDIR}/tmp/Lifecycle-Export-Materialized-View-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server-stage-2-stderr.tmp 

Lifecycle Export User Space Table and then Access From RDBMSs
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Engineer a way to do postgres and sqlite export testing in same test case
    ${ddlInputStr} =    Catenate
    ...    create table my_silly_export_table(id int, name text, magnitude numeric);
    ...    drop table my_silly_export_table;
    ...    create table my_silly_export_table(id int, name text, magnitude numeric);
    ...    insert into my_silly_export_table(id, name, magnitude) values (1, 'one', 1.0);
    ...    insert into my_silly_export_table(id, name, magnitude) values (2, 'two', 2.0);
    ${ddlOutputStr} =    Catenate    SEPARATOR=\n
    ...    DDL Execution Completed
    ...    DDL Execution Completed
    ...    DDL Execution Completed
    ...    insert into table completed
    ...    insert into table completed
    ${queryStringSQLite} =    Catenate
    ...    select id, name, magnitude from "stackql_export.my_silly_export_table" order by id;
    ${queryStringPostgres} =    Catenate
    ...    select id, name, magnitude from stackql_export.my_silly_export_table order by id;
    ${dbName} =    Set Variable If    "${SQL_BACKEND}" == "postgres_tcp"     postgres    sqlite
    ${queryString} =    Set Variable If    "${SQL_BACKEND}" == "postgres_tcp"     ${queryStringPostgres}    ${queryStringSQLite}
    ${queryOutputStr} =    Catenate    SEPARATOR=\n
    ...    1,one,1
    ...    2,two,2
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
    ...    stdout=${CURDIR}/tmp/Lifecycle-Export-User-Space-Table-and-then-Access-From-RDBMSs.tmp
    ...    stderr=${CURDIR}/tmp/Lifecycle-Export-User-Space-Table-and-then-Access-From-RDBMSs-stderr.tmp
    Should RDBMS Query Return CSV Result
    ...    ${dbName}
    ...    ${SQL_CLIENT_EXPORT_CONNECTION_ARG}
    ...    ${queryString}
    ...    ${queryOutputStr}
    ...    ${EMPTY}
    ...    stdout=${CURDIR}/tmp/Lifecycle-Export-User-Space-Table-and-then-Access-From-RDBMSs-stage-2.tmp
    ...    stderr=${CURDIR}/tmp/Lifecycle-Export-User-Space-Table-and-then-Access-From-RDBMSs-stage-2-stderr.tmp

Lifecycle Export User Space Table and then Access From RDBMSs Over Stackql Postgres Server
    Pass Execution If    "${SQL_BACKEND}" != "postgres_tcp"    TODO: FIX THIS... Engineer a way to do postgres and sqlite export testing in same test case
    ${ddlInputStr} =    Catenate
    ...    create table my_silly_export_table_two(id int, name text, magnitude numeric);
    ${posixInput} =     Catenate
    ...    "${PSQL_EXE}"    -d     "${PSQL_MTLS_CONN_STR_EXPORT_UNIX}"   -c   "${ddlInputStr}"
    ${windowsInput} =     Catenate
    ...    &    ${posixInput}
    ${input} =    Set Variable If    "${IS_WINDOWS}" == "1"    ${windowsInput}    ${posixInput}
    ${shellExe} =    Set Variable If    "${IS_WINDOWS}" == "1"    powershell    sh
    ${result} =    Run Process
    ...    ${shellExe}     \-c    ${input}
    ...    stdout=${CURDIR}/tmp/Lifecycle-Export-User-Space-Table-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server.tmp
    ...    stderr=${CURDIR}/tmp/Lifecycle-Export-User-Space-Table-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server-stderr.tmp
    Should Be Equal    ${result.stdout}    OK
    ${ddlInputStr} =    Catenate
    ...    drop table my_silly_export_table_two;
    ${posixInput} =     Catenate
    ...    "${PSQL_EXE}"    -d     "${PSQL_MTLS_CONN_STR_EXPORT_UNIX}"   -c   "${ddlInputStr}"
    ${windowsInput} =     Catenate
    ...    &    ${posixInput}
    ${input} =    Set Variable If    "${IS_WINDOWS}" == "1"    ${windowsInput}    ${posixInput}
    ${shellExe} =    Set Variable If    "${IS_WINDOWS}" == "1"    powershell    sh
    ${result} =    Run Process
    ...    ${shellExe}     \-c    ${input}
    ...    stdout=${CURDIR}/tmp/Lifecycle-Export-User-Space-Table-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server-Drop.tmp
    ...    stderr=${CURDIR}/tmp/Lifecycle-Export-User-Space-Table-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server-Drop-stderr.tmp
    Should Be Equal    ${result.stdout}    OK
    ${ddlInputStr} =    Catenate
    ...    create table my_silly_export_table_two(id int, name text, magnitude numeric);
    ${posixInput} =     Catenate
    ...    "${PSQL_EXE}"    -d     "${PSQL_MTLS_CONN_STR_EXPORT_UNIX}"   -c   "${ddlInputStr}"
    ${windowsInput} =     Catenate
    ...    &    ${posixInput}
    ${input} =    Set Variable If    "${IS_WINDOWS}" == "1"    ${windowsInput}    ${posixInput}
    ${shellExe} =    Set Variable If    "${IS_WINDOWS}" == "1"    powershell    sh
    ${result} =    Run Process
    ...    ${shellExe}     \-c    ${input}
    ...    stdout=${CURDIR}/tmp/Lifecycle-Export-User-Space-Table-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server-Recreate.tmp
    ...    stderr=${CURDIR}/tmp/Lifecycle-Export-User-Space-Table-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server-Recreate-stderr.tmp
    Should Be Equal    ${result.stdout}    OK
    ${ddlInputStr} =    Catenate
    ...    insert into my_silly_export_table_two(id, name, magnitude) values (1, 'one', 1.0);
    ${posixInput} =     Catenate
    ...    "${PSQL_EXE}"    -d     "${PSQL_MTLS_CONN_STR_EXPORT_UNIX}"   -c   "${ddlInputStr}"
    ${windowsInput} =     Catenate
    ...    &    ${posixInput}
    ${input} =    Set Variable If    "${IS_WINDOWS}" == "1"    ${windowsInput}    ${posixInput}
    ${shellExe} =    Set Variable If    "${IS_WINDOWS}" == "1"    powershell    sh
    ${result} =    Run Process
    ...    ${shellExe}     \-c    ${input}
    ...    stdout=${CURDIR}/tmp/Lifecycle-Export-User-Space-Table-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server-insert-one.tmp
    ...    stderr=${CURDIR}/tmp/Lifecycle-Export-User-Space-Table-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server-insert-one-stderr.tmp
    Should Be Equal    ${result.stdout}    OK
    ${ddlInputStr} =    Catenate
    ...    insert into my_silly_export_table_two(id, name, magnitude) values (2, 'two', 2.0);
    ${posixInput} =     Catenate
    ...    "${PSQL_EXE}"    -d     "${PSQL_MTLS_CONN_STR_EXPORT_UNIX}"   -c   "${ddlInputStr}"
    ${windowsInput} =     Catenate
    ...    &    ${posixInput}
    ${input} =    Set Variable If    "${IS_WINDOWS}" == "1"    ${windowsInput}    ${posixInput}
    ${shellExe} =    Set Variable If    "${IS_WINDOWS}" == "1"    powershell    sh
    ${result} =    Run Process
    ...    ${shellExe}     \-c    ${input}
    ...    stdout=${CURDIR}/tmp/Lifecycle-Export-User-Space-Table-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server-insert-two.tmp
    ...    stderr=${CURDIR}/tmp/Lifecycle-Export-User-Space-Table-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server-insert-two-stderr.tmp
    Should Be Equal    ${result.stdout}    OK
    ${queryStringSQLite} =    Catenate
    ...    select id, name, magnitude from "stackql_export.my_silly_export_table_two" order by id;
    ${queryStringPostgres} =    Catenate
    ...    select id, name, magnitude from stackql_export.my_silly_export_table_two order by id;
    ${dbName} =    Set Variable If    "${SQL_BACKEND}" == "postgres_tcp"     postgres    sqlite
    ${queryString} =    Set Variable If    "${SQL_BACKEND}" == "postgres_tcp"     ${queryStringPostgres}    ${queryStringSQLite}
    ${queryOutputStr} =    Catenate    SEPARATOR=\n
    ...    id,name,magnitude
    ...    1,one,1.0
    ...    2,two,2.0
    Should RDBMS Query Return CSV Result
    ...    ${dbName}
    ...    ${SQL_CLIENT_EXPORT_CONNECTION_ARG}
    ...    ${queryString}
    ...    ${queryOutputStr}
    ...    ${EMPTY}
    ...    stdout=${CURDIR}/tmp/Lifecycle-Export-User-Space-Table-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server-stage-2.tmp
    ...    stderr=${CURDIR}/tmp/Lifecycle-Export-User-Space-Table-and-then-Access-From-RDBMSs-Over-Stackql-Postgres-Server-stage-2-stderr.tmp 
