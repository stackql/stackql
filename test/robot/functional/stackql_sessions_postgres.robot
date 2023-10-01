*** Settings ***
Resource          ${CURDIR}/stackql.resource

*** Test Cases *** 

SQLAlchemy Session Positive Control
    Pass Execution If    "${SQL_BACKEND}" != "postgres_tcp"    This is a postgres only test
    Should SQLALchemy Raw Session Inline Equal
    ...    ${POSTGRES_URL_UNENCRYPTED_CONN}
    ...    ${PG_CLIENT_SETUP_QUERIES}
    ...    ${PG_CLIENT_SETUP_QUERIES_TUPLES_EXPECTED}
    ...    stdout=${CURDIR}/tmp/SQLAlchemy-Session-Positive-Control.tmp
    [Teardown]    NONE

SQLAlchemy Session Postgres Catalog Join
    Pass Execution If    "${SQL_BACKEND}" != "postgres_tcp"    This is a postgres only test
    Should SQLALchemy Raw Session Inline Contain
    ...    ${POSTGRES_URL_UNENCRYPTED_CONN}
    ...    ${SELECT_POSTGRES_CATALOG_JOIN_ARR}
    ...    ${SELECT_POSTGRES_CATALOG_JOIN_TUPLE_EXPECTED}
    ...    stdout=${CURDIR}/tmp/SQLAlchemy-Session-Postgres-Catalog-Join.tmp
    [Teardown]    NONE

SQLAlchemy Session Postgres PID Function
    Pass Execution If    "${SQL_BACKEND}" != "postgres_tcp"    This is a postgres only test
    Should SQLALchemy Raw Session Inline Have Length
    ...    ${POSTGRES_URL_UNENCRYPTED_CONN}
    ...    ${SELECT_POSTGRES_BACKEND_PID_ARR}
    ...    1
    ...    stdout=${CURDIR}/tmp/SQLAlchemy-Session-Postgres-PID-Function.tmp
    [Teardown]    NONE

SQLAlchemy Session Postgres Intel Views Exist
    Log    This test expects exactly 4 results per query in the sequence
    Pass Execution If    "${SQL_BACKEND}" != "postgres_tcp"    This is a postgres only test
    Should SQLALchemy Raw Session Inline Have Length Greater Than Or Equal To
    ...    ${POSTGRES_URL_UNENCRYPTED_CONN}
    ...    ${SELECT_ACCELERATOR_TYPES_DESC_SEQUENCE}
    ...    12
    ...    stdout=${CURDIR}/tmp/SQLAlchemy-Session-Postgres-Intel-Views-Exist.tmp
    [Teardown]    NONE

SQLAlchemy Session Materialized View Lifecycle
    Pass Execution If    "${SQL_BACKEND}" != "postgres_tcp"    This is a postgres only test
    ${inputStr} =    Catenate
    ...    [
    ...    "create materialized view vw_aws_usr as select Arn, UserName, UserId, region from aws.iam.users where region = 'us-east-1';",
    ...    "select u1.UserName, u2.UserId, u2.Arn, u1.region from aws.iam.users u1 inner join vw_aws_usr u2 on u1.Arn = u2.Arn where u1.region = 'us-east-1' and u2.region = 'us-east-1' order by u1.UserName desc;",
    ...    "drop materialized view vw_aws_usr;"
    ...    ]
    ${inputList} =    Evaluate    ${inputStr}
    Should SQLALchemy Raw Session Inline Have Length
    ...    ${POSTGRES_URL_UNENCRYPTED_CONN}
    ...    ${inputList}
    ...    2
    ...    stdout=${CURDIR}/tmp/SQLAlchemy-Session-v2-Materialized-View-Lifecycle.tmp
    ...    stderr=${CURDIR}/tmp/SQLAlchemy-Session-v2-Materialized-View-Lifecycle-stderr.tmp
    [Teardown]    NONE
