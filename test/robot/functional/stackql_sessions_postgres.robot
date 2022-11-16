*** Settings ***
Resource          ${CURDIR}/stackql.resource

*** Test Cases *** 

SQLAlchemy Session Positive Control
    Pass Execution If    "${SQL_BACKEND}" != "postgres_tcp"    This is a postgres only test
    Should SQLALchemy Raw Session Inline Equal
    ...    ${POSTGRES_URL_UNENCRYPTED_CONN}
    ...    ${PG_CLIENT_SETUP_QUERIES}
    ...    ${PG_CLIENT_SETUP_QUERIES_TUPLE_EXPECTED}
    ...    stdout=${CURDIR}/tmp/SQLAlchemy-Session-Positive-Control.tmp
    [Teardown]    NONE

