*** Variables ***
${LOCAL_LIB_HOME}        ${CURDIR}/../lib
${EXECUTION_PLATFORM}    native   # to be overridden from command line, eg "docker"

*** Settings ***
Library           Process
Library           OperatingSystem 
Variables         ${LOCAL_LIB_HOME}/stackql_context.py    ${EXECUTION_PLATFORM}
Library           Process
Library           OperatingSystem
Library           String
Library           ${LOCAL_LIB_HOME}/StackQLInterfaces.py    ${EXECUTION_PLATFORM}
Library           ${LOCAL_LIB_HOME}/CloudIntegration.py

*** Keywords ***
Start Mock Server
    [Arguments]    ${_JSON_INIT_FILE_PATH}    ${_MOCKSERVER_JAR}    ${_MOCKSERVER_PORT}
    ${process} =    Start Process    java    \-Dfile.encoding\=UTF-8
    ...  \-Dmockserver.initializationJsonPath\=${_JSON_INIT_FILE_PATH}
    ...  \-Dmockserver.logLevel\=OFF
    ...  \-Droot.logLevel\=OFF
    ...  \-jar    ${_MOCKSERVER_JAR}
    ...  \-serverPort    ${_MOCKSERVER_PORT}
    ...  stderr=${CURDIR}/tmp/stdout-mockserver-${_MOCKSERVER_PORT}.txt
    ...  stdout=${CURDIR}/tmp/stderr-mockserver-${_MOCKSERVER_PORT}.txt
    [Return]    ${process}

Start All Mock Servers
    Start Mock Server    ${JSON_INIT_FILE_PATH_GOOGLE}    ${MOCKSERVER_JAR}    ${MOCKSERVER_PORT_GOOGLE}
    Start Mock Server    ${JSON_INIT_FILE_PATH_OKTA}    ${MOCKSERVER_JAR}    ${MOCKSERVER_PORT_OKTA}
    Start Mock Server    ${JSON_INIT_FILE_PATH_GITHUB}    ${MOCKSERVER_JAR}    ${MOCKSERVER_PORT_GITHUB}
    Start Mock Server    ${JSON_INIT_FILE_PATH_AWS}    ${MOCKSERVER_JAR}    ${MOCKSERVER_PORT_AWS}
    Start Mock Server    ${JSON_INIT_FILE_PATH_K8S}    ${MOCKSERVER_JAR}    ${MOCKSERVER_PORT_K8S}
    Start Mock Server    ${JSON_INIT_FILE_PATH_REGISTRY}    ${MOCKSERVER_JAR}    ${MOCKSERVER_PORT_REGISTRY}
    Start Mock Server    ${JSON_INIT_FILE_PATH_AZURE}    ${MOCKSERVER_JAR}    ${MOCKSERVER_PORT_AZURE}


Prepare StackQL Environment
    Set Environment Variable    OKTA_SECRET_KEY    ${OKTA_SECRET_STR}
    Set Environment Variable    GITHUB_SECRET_KEY    ${GITHUB_SECRET_STR}
    Set Environment Variable    K8S_SECRET_KEY    ${K8S_SECRET_STR}
    Set Environment Variable    AZ_ACCESS_TOKEN    ${AZURE_SECRET_STR}
    Start All Mock Servers
    Start StackQL PG Server mTLS    ${PG_SRV_PORT_MTLS}    ${PG_SRV_MTLS_CFG_STR}
    Start StackQL PG Server unencrypted    ${PG_SRV_PORT_UNENCRYPTED}
    Sleep    50s

Start StackQL PG Server mTLS
    [Arguments]    ${_SRV_PORT_MTLS}    ${_MTLS_CFG_STR}
    IF    "${EXECUTION_PLATFORM}" == "native"
        ${process} =    Start Process    ${STACKQL_EXE}
                        ...  srv    \-\-registry\=${REGISTRY_NO_VERIFY_CFG_STR.get_config_str('native')}
                        ...  \-\-auth\=${AUTH_CFG_STR}
                        ...  \-\-tls\.allowInsecure\=true
                        ...  \-\-pgsrv\.address\=0.0.0.0 
                        ...  \-\-pgsrv\.port\=${_SRV_PORT_MTLS} 
                        ...  \-\-pgsrv\.tls    ${_MTLS_CFG_STR}
                        ...  stderr=${CURDIR}/tmp/stdout-stackql-srv-mtls-${_SRV_PORT_MTLS}.txt
                        ...  stdout=${CURDIR}/tmp/stderr-stackql-srv-mtls-${_SRV_PORT_MTLS}.txt
    ELSE IF    "${EXECUTION_PLATFORM}" == "docker"
        ${res} =         Run Process    docker\-compose    \-\-log\-level    ERROR
                        ...  run    \-\-rm    credentialsgen
        Log     Credentials gen completed
        Should Be Equal As Integers    ${res.rc}    0
        ${process} =    Start Process    docker\-compose
                        ...  \-p     mtls
                        ...  run
                        ...  \-p${_SRV_PORT_MTLS}:${PG_SRV_PORT_DOCKER_MTLS}/tcp
                        ...  stackqlsrv
                        ...  bash
                        ...  \-c
                        ...  stackql srv \-\-registry\='${REGISTRY_NO_VERIFY_CFG_STR.get_config_str('docker')}' \-\-auth\='${AUTH_CFG_STR}' \-\-tls\.allowInsecure\=true \-\-pgsrv\.address\=0.0.0.0 \-\-pgsrv\.port\=${PG_SRV_PORT_DOCKER_MTLS} \-\-pgsrv\.tls\='{\"keyFilePath\": \"/opt/stackql/srv/credentials/pg_server_key.pem\", \"certFilePath\": \"/opt/stackql/srv/credentials/pg_server_cert.pem\", \"clientCAs\": [\"'$(base64 -w 0 /opt/stackql/srv/credentials/pg_client_cert.pem)'\"]}'
                        ...  stderr=${CURDIR}/tmp/stdout-stackql-srv-mtls-${_SRV_PORT_MTLS}.txt
                        ...  stdout=${CURDIR}/tmp/stderr-stackql-srv-mtls-${_SRV_PORT_MTLS}.txt
    END
    [Return]    ${process}

Start StackQL PG Server unencrypted
    [Arguments]    ${_SRV_PORT_UNENCRYPTED}
    IF    "${EXECUTION_PLATFORM}" == "native"
        ${process} =    Start Process    ${STACKQL_EXE}
                        ...  srv    \-\-registry\=${REGISTRY_NO_VERIFY_CFG_STR.get_config_str('native')}
                        ...  \-\-auth\=${AUTH_CFG_STR}
                        ...  \-\-tls\.allowInsecure\=true
                        ...  \-\-pgsrv\.address\=0.0.0.0 
                        ...  \-\-pgsrv\.port\=${_SRV_PORT_UNENCRYPTED}
                        ...  stderr=${CURDIR}/tmp/stdout-stackql-srv-unencrypted-${_SRV_PORT_UNENCRYPTED}.txt
                        ...  stdout=${CURDIR}/tmp/stderr-stackql-srv-unencrypted-${_SRV_PORT_UNENCRYPTED}.txt
    ELSE IF    "${EXECUTION_PLATFORM}" == "docker"
        ${process} =    Start Process    docker\-compose    \-\-log\-level    ERROR
                        ...  \-p     unencrypted
                        ...  run
                        ...  \-\-rm    \-p${_SRV_PORT_UNENCRYPTED}:${PG_SRV_PORT_DOCKER_UNENCRYPTED}/tcp
                        ...  stackqlsrv
                        ...  bash
                        ...  \-c
                        ...  stackql srv \-\-registry\='${REGISTRY_NO_VERIFY_CFG_STR.get_config_str('docker')}' \-\-auth\='${AUTH_CFG_STR}' \-\-tls\.allowInsecure\=true \-\-pgsrv\.address\=0.0.0.0 \-\-pgsrv\.port\=${PG_SRV_PORT_DOCKER_UNENCRYPTED}
                        ...  stderr=${CURDIR}/tmp/stdout-stackql-srv-unencrypted-${_SRV_PORT_UNENCRYPTED}.txt
                        ...  stdout=${CURDIR}/tmp/stderr-stackql-srv-unencrypted-${_SRV_PORT_UNENCRYPTED}.txt
    END
    [Return]    ${process}
