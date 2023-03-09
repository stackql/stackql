


*** Settings ***
Resource          ${CURDIR}/stackql.resource

*** Test Cases *** 
Nop From Lib
    ${result} =     Nop Cloud Integration Keyword
    Should Be Equal    ${result}    PASS

Azure Authenticated VM Sizes
    Should Stackql Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_CANONICAL_NO_VERIFY_CFG_STR}
    ...    {}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${AZURE_VM_SIZES_ENUMERATION}
    ...    Standard_
    ...    stdout=${CURDIR}/tmp/Azure-Authenticated-VM-Sizes.tmp

Faulty Auth Azure Authenticated VM Sizes
    Should Stackql Exec Inline Contain stderr
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_CANONICAL_NO_VERIFY_CFG_STR}
    ...    ${AUTH_AZURE_FAULTY}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${AZURE_VM_SIZES_ENUMERATION}
    ...    credentials error
    ...    stdout=${CURDIR}/tmp/Faulty-Azure-Authenticated-VM-Sizes.tmp
    ...    stderr=${CURDIR}/tmp/Faulty-Azure-Authenticated-VM-Sizes-stderr.tmp

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
    Set Environment Variable    GITHUB_SECRET_KEY    ${GITHUB_SECRET_STR}
    Set Environment Variable    K8S_SECRET_KEY    ${K8S_SECRET_STR}
    Set Environment Variable    DB_SETUP_SRC    ${DB_SETUP_SRC}
    Start Mock Server    ${JSON_INIT_FILE_PATH_GOOGLE}    ${MOCKSERVER_JAR}    ${MOCKSERVER_PORT_GOOGLE}
    Start Mock Server    ${JSON_INIT_FILE_PATH_OKTA}    ${MOCKSERVER_JAR}    ${MOCKSERVER_PORT_OKTA}
    Start StackQL PG Server mTLS    ${PG_SRV_PORT_MTLS}    ${PG_SRV_MTLS_CFG_STR}
    Start StackQL PG Server unencrypted    ${PG_SRV_PORT_UNENCRYPTED}


Run StackQL Exec Command
    [Arguments]    ${_REGISTRY_CFG_STR}    ${_EXEC_CMD_STR}    @{varargs}
    Set Environment Variable    OKTA_SECRET_KEY    ${OKTA_SECRET_STR}
    Set Environment Variable    GITHUB_SECRET_KEY    ${GITHUB_SECRET_STR}
    Set Environment Variable    K8S_SECRET_KEY    ${K8S_SECRET_STR}
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

