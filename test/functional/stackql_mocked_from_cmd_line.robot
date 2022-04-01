*** Settings ***
Library    Process
Library    OperatingSystem
Library    String

*** Settings ***
Variables         ${CURDIR}/variables/stackql_context.py
Suite Setup       Prepare StackQL Environment
Suite Teardown    Terminate All Processes

*** Test Cases *** 
Google Container Agg Desc
    Should StackQL Exec Equal
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${SELECT_CONTAINER_SUBNET_AGG_DESC}
    ...    ${SELECT_CONTAINER_SUBNET_AGG_DESC_EXPECTED}

Google Container Agg Asc
    Should StackQL Exec Equal
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${SELECT_CONTAINER_SUBNET_AGG_ASC}
    ...    ${SELECT_CONTAINER_SUBNET_AGG_ASC_EXPECTED}

Google IAM Policy Agg
    Should StackQL Exec Equal
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    \-\-infile\=${GET_IAM_POLICY_AGG_ASC_INPUT_FILE}
    ...    ${GET_IAM_POLICY_AGG_ASC_EXPECTED}
    ...    \-o\=csv


Google AcceleratorTypes SQL verb pre changeover
    Should StackQL Exec Equal
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${SELECT_ACCELERATOR_TYPES_DESC}
    ...    ${SELECT_ACCELERATOR_TYPES_DESC_EXPECTED}

Google AcceleratorTypes SQL verb post changeover
    Should StackQL Exec Equal
    ...    ${REGISTRY_SQL_VERB_CONTRIVED_NO_VERIFY_CFG_STR}
    ...    ${SELECT_ACCELERATOR_TYPES_DESC}
    ...    ${SELECT_ACCELERATOR_TYPES_DESC_EXPECTED}

Basic Query mTLS Returns OK
    Should PG Client Inline Contain
    ...    ${PSQL_MTLS_CONN_STR}
    ...    ${SELECT_CONTAINER_SUBNET_AGG_ASC}
    ...    ipCidrRange

Basic Query unencrypted Returns OK
    Should PG Client Inline Contain
    ...    ${PSQL_UNENCRYPTED_CONN_STR}
    ...    ${SELECT_CONTAINER_SUBNET_AGG_ASC}
    ...    ipCidrRange

Erroneous mTLS Config Plus Basic Query Returns Error
    Should PG Client Error Inline Contain
    ...    ${PSQL_MTLS_INVALID_CONN_STR}
    ...    ${SELECT_CONTAINER_SUBNET_AGG_ASC}
    ...    error

*** Keywords ***
Start Mock Server
    [Arguments]    ${_JSON_INIT_FILE_PATH}    ${_MOCKSERVER_JAR}    ${_MOCKSERVER_PORT}
    ${process} =    Start Process    java    \-Dfile.encoding\=UTF-8
    ...  \-Dmockserver.initializationJsonPath\=${_JSON_INIT_FILE_PATH}
    ...  \-jar    ${_MOCKSERVER_JAR}
    ...  \-serverPort    ${_MOCKSERVER_PORT}    \-logLevel    INFO
    Sleep    5s
    [Return]    ${process}


Prepare StackQL Environment
    Set Environment Variable    OKTA_SECRET_KEY    ${OKTA_SECRET_STR}
    Start Mock Server    ${JSON_INIT_FILE_PATH}    ${MOCKSERVER_JAR}    ${MOCKSERVER_PORT}
    Start StackQL PG Server mTLS    ${PG_SRV_PORT_MTLS}    ${PG_SRV_MTLS_CFG_STR}
    Start StackQL PG Server unencrypted    ${PG_SRV_PORT_UNENCRYPTED}


Run StackQL Exec Command
    [Arguments]    ${_REGISTRY_CFG_STR}    ${_EXEC_CMD_STR}    @{varargs}
    Set Environment Variable    OKTA_SECRET_KEY    ${OKTA_SECRET_STR}
    ${result} =     Run Process    
                    ...  ${STACKQL_EXE}
                    ...  exec    \-\-registry\=${_REGISTRY_CFG_STR}
                    ...  \-\-auth\=${AUTH_CFG_STR}
                    ...  \-\-tls.allowInsecure\=true
                    ...  ${_EXEC_CMD_STR}    @{varargs}
    Log             ${result.stdout}
    Log             ${result.stderr}
    [Return]    ${result}


Should StackQL Exec Equal
    [Arguments]    ${_REGISTRY_CFG_STR}    ${_EXEC_CMD_STR}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}
    ${result} =    Run StackQL Exec Command    ${_REGISTRY_CFG_STR}    ${_EXEC_CMD_STR}    @{varargs}
    Should Be Equal    ${result.stdout}    ${_EXEC_CMD_EXPECTED_OUTPUT}

Start StackQL PG Server mTLS
    [Arguments]    ${_SRV_PORT_MTLS}    ${_MTLS_CFG_STR}
    ${process} =    Start Process    ${STACKQL_EXE}
                    ...  srv    \-\-registry\=${REGISTRY_NO_VERIFY_CFG_STR}
                    ...  \-\-auth\=${AUTH_CFG_STR}
                    ...  \-\-tls\.allowInsecure\=true
                    ...  \-\-pgsrv\.address\=0.0.0.0 
                    ...  \-\-pgsrv\.port\=${_SRV_PORT_MTLS} 
                    ...  \-\-pgsrv\.tls    ${_MTLS_CFG_STR}
    Sleep    15s
    [Return]    ${process}


Start StackQL PG Server unencrypted
    [Arguments]    ${_SRV_PORT_UNENCRYPTED}
    ${process} =    Start Process    ${STACKQL_EXE}
                    ...  srv    \-\-registry\=${REGISTRY_NO_VERIFY_CFG_STR}
                    ...  \-\-auth\=${AUTH_CFG_STR}
                    ...  \-\-tls\.allowInsecure\=true
                    ...  \-\-pgsrv\.address\=0.0.0.0 
                    ...  \-\-pgsrv\.port\=${_SRV_PORT_UNENCRYPTED}
    Sleep    15s
    [Return]    ${process}


Run PG Client Command
    [Arguments]    ${_PSQL_CONN_STR}    ${_QUERY}
    ${_MOD_CONN} =    Replace String    ${_PSQL_CONN_STR}    \\    /
    Log To Console    CURDIR = '${CURDIR}'
    Log To Console    PSQL_EXE = '${PSQL_EXE}'
    ${result} =     Run Process    
                    ...  ${PSQL_EXE}
                    ...  -d    ${_MOD_CONN}
                    ...  -c    ${_QUERY}
    Log             ${result.stdout}
    Log             ${result.stderr}
    [Return]    ${result}


Should PG Client Inline Equal
    [Arguments]    ${_CONN_STR}   ${_QUERY}    ${_EXPECTED_OUTPUT}
    ${result} =    Run PG Client Command    ${_CONN_STR}    ${_QUERY}
    Should Be Equal    ${result.stdout}    ${_EXPECTED_OUTPUT}

Should PG Client Inline Contain
    [Arguments]    ${_CONN_STR}   ${_QUERY}    ${_EXPECTED_OUTPUT}
    ${result} =    Run PG Client Command    ${_CONN_STR}    ${_QUERY}
    Should Contain    ${result.stdout}    ${_EXPECTED_OUTPUT}

Should PG Client Error Inline Contain
    [Arguments]    ${_CONN_STR}   ${_QUERY}    ${_EXPECTED_OUTPUT}
    ${result} =    Run PG Client Command    ${_CONN_STR}    ${_QUERY}
    Should Contain    ${result.stderr}    ${_EXPECTED_OUTPUT}
