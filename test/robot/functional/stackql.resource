*** Variables ***
${LOCAL_LIB_HOME}        ${CURDIR}/../lib
${EXECUTION_PLATFORM}    native   # to be overridden from command line, eg "docker"
${SQL_BACKEND}           sqlite_embedded   # to be overridden from command line, eg "postgres_tcp"

*** Settings ***
Library           Process
Library           OperatingSystem 
Variables         ${LOCAL_LIB_HOME}/stackql_context.py    ${EXECUTION_PLATFORM}    ${SQL_BACKEND}
Library           Process
Library           OperatingSystem
Library           String
Library           ${LOCAL_LIB_HOME}/StackQLInterfaces.py    ${EXECUTION_PLATFORM}    ${SQL_BACKEND}
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
    Generate Container Credentials for StackQL PG Server mTLS
    Start StackQL PG Server mTLS    ${PG_SRV_PORT_MTLS}    ${PG_SRV_MTLS_CFG_STR}    {}    {}    ${SQL_BACKEND_CFG_STR_CANONICAL}    ${PG_SRV_PORT_DOCKER_MTLS}
    Start StackQL PG Server mTLS    ${PG_SRV_PORT_MTLS_WITH_NAMESPACES}    ${PG_SRV_MTLS_CFG_STR}    ${NAMESPACES_TTL_SPECIALCASE_TRANSPARENT}    {}    ${SQL_BACKEND_CFG_STR_CANONICAL}    ${PG_SRV_PORT_DOCKER_MTLS_WITH_NAMESPACES}
    Start StackQL PG Server mTLS    ${PG_SRV_PORT_MTLS_WITH_EAGER_GC}    ${PG_SRV_MTLS_CFG_STR}    {}    ${GC_CFG_EAGER}    ${SQL_BACKEND_CFG_STR_CANONICAL}    ${PG_SRV_PORT_DOCKER_MTLS_WITH_EAGER_GC}
    Start StackQL PG Server unencrypted    ${PG_SRV_PORT_UNENCRYPTED}    {}
    Sleep    50s

Generate Container Credentials for StackQL PG Server mTLS
    IF    "${EXECUTION_PLATFORM}" == "docker"
        ${res} =         Run Process    docker\-compose    \-\-log\-level    ERROR
                        ...  run    \-\-rm    credentialsgen
        Log     Credentials gen completed
        Should Be Equal As Integers    ${res.rc}    0
    END

Start StackQL PG Server mTLS
    [Arguments]    ${_SRV_PORT_MTLS}    ${_MTLS_CFG_STR}    ${_NAMESPACES_CFG}    ${_GC_CFG}    ${_SQL_BACKEND_CFG}    ${_DOCKER_PORT}
    IF    "${EXECUTION_PLATFORM}" == "native"
        ${process} =    Start Process    ${STACKQL_EXE}
                        ...  srv    \-\-registry\=${REGISTRY_NO_VERIFY_CFG_STR.get_config_str('native')}
                        ...  \-\-auth\=${AUTH_CFG_STR}
                        ...  \-\-tls\.allowInsecure\=true
                        ...  \-\-pgsrv\.address\=0.0.0.0 
                        ...  \-\-pgsrv\.port\=${_SRV_PORT_MTLS} 
                        ...  \-\-pgsrv\.tls    ${_MTLS_CFG_STR}
                        ...  \-\-namespaces\=${_NAMESPACES_CFG}
                        ...  \-\-gc\=${_GC_CFG}
                        ...  \-\-sqlBackend\=${_SQL_BACKEND_CFG}
                        ...  stderr=${CURDIR}/tmp/stdout-stackql-srv-mtls-${_SRV_PORT_MTLS}.txt
                        ...  stdout=${CURDIR}/tmp/stderr-stackql-srv-mtls-${_SRV_PORT_MTLS}.txt
    ELSE IF    "${EXECUTION_PLATFORM}" == "docker"
        ${process} =    Start Process    docker\-compose
                        ...  \-p     stackqlpgsrv\-mtls\-${_DOCKER_PORT}
                        ...  run
                        ...  \-\-rm    \-p${_SRV_PORT_MTLS}:${_DOCKER_PORT}/tcp
                        ...  stackqlsrv
                        ...  bash
                        ...  \-c
                        ...  stackql srv \-\-registry\='${REGISTRY_NO_VERIFY_CFG_STR.get_config_str('docker')}' \-\-auth\='${AUTH_CFG_STR}' \-\-namespaces\='${_NAMESPACES_CFG}' \-\-gc\='${_GC_CFG}' \-\-sqlBackend\='${_SQL_BACKEND_CFG}' \-\-tls\.allowInsecure\=true \-\-pgsrv\.address\='0.0.0.0' \-\-pgsrv\.port\=${_DOCKER_PORT} \-\-pgsrv\.tls\='{\"keyFilePath\": \"/opt/stackql/srv/credentials/pg_server_key.pem\", \"certFilePath\": \"/opt/stackql/srv/credentials/pg_server_cert.pem\", \"clientCAs\": [\"'$(base64 -w 0 /opt/stackql/srv/credentials/pg_client_cert.pem)'\"]}'
                        ...  stderr=${CURDIR}/tmp/stdout-stackql-srv-mtls-${_SRV_PORT_MTLS}.txt
                        ...  stdout=${CURDIR}/tmp/stderr-stackql-srv-mtls-${_SRV_PORT_MTLS}.txt
    END
    [Return]    ${process}

Start StackQL PG Server unencrypted
    [Arguments]    ${_SRV_PORT_UNENCRYPTED}    ${_NAMESPACES_CFG}
    IF    "${EXECUTION_PLATFORM}" == "native"
        ${process} =    Start Process    ${STACKQL_EXE}
                        ...  srv    \-\-registry\=${REGISTRY_NO_VERIFY_CFG_STR.get_config_str('native')}
                        ...  \-\-auth\=${AUTH_CFG_STR}
                        ...  \-\-tls\.allowInsecure\=true
                        ...  \-\-pgsrv\.address\=0.0.0.0 
                        ...  \-\-pgsrv\.port\=${_SRV_PORT_UNENCRYPTED}
                        ...  \-\-namespaces\=${_NAMESPACES_CFG}
                        ...  stderr=${CURDIR}/tmp/stdout-stackql-srv-unencrypted-${_SRV_PORT_UNENCRYPTED}.txt
                        ...  stdout=${CURDIR}/tmp/stderr-stackql-srv-unencrypted-${_SRV_PORT_UNENCRYPTED}.txt
    ELSE IF    "${EXECUTION_PLATFORM}" == "docker"
        ${process} =    Start Process    docker\-compose    \-\-log\-level    ERROR
                        ...  \-p     stackqlpgsrv\-unencrypted
                        ...  run
                        ...  \-\-rm    \-p${_SRV_PORT_UNENCRYPTED}:${PG_SRV_PORT_DOCKER_UNENCRYPTED}/tcp
                        ...  stackqlsrv
                        ...  bash
                        ...  \-c
                        ...  stackql srv \-\-registry\='${REGISTRY_NO_VERIFY_CFG_STR.get_config_str('docker')}' \-\-auth\='${AUTH_CFG_STR}' \-\-namespaces\='${_NAMESPACES_CFG}' \-\-tls\.allowInsecure\=true \-\-pgsrv\.address\=0.0.0.0 \-\-pgsrv\.port\=${PG_SRV_PORT_DOCKER_UNENCRYPTED}
                        ...  stderr=${CURDIR}/tmp/stdout-stackql-srv-unencrypted-${_SRV_PORT_UNENCRYPTED}.txt
                        ...  stdout=${CURDIR}/tmp/stderr-stackql-srv-unencrypted-${_SRV_PORT_UNENCRYPTED}.txt
    END
    [Return]    ${process}

Stackql Per Test Teardown
    IF    "${EXECUTION_PLATFORM}" == "docker" and "${SQL_BACKEND}" == "postgres_tcp"
        ${res} =         Run Process    bash    \-c    docker kill $(docker ps \-\-filter name\=execrun \-q)
        Log     Container killed
        # Should Be Equal As Integers    ${res.rc}    0
        ${restwo} =         Run Process    bash    \-c    docker rm $(docker ps \-\-filter status\=exited \-q)
        Log     Container removed
        # Should Be Equal As Integers    ${restwo.rc}    0
    END
