*** Variables ***
${LOCAL_LIB_HOME}                      ${CURDIR}/../lib
${REPOSITORY_ROOT}                     ${CURDIR}${/}..${/}..${/}..
${EXECUTION_PLATFORM}                  native   # to be overridden from command line, eg "docker"
${SQL_BACKEND}                         sqlite_embedded   # to be overridden from command line, eg "postgres_tcp"
${IS_WSL}                              false   # to be overridden from command line, with string "true"
${SHOULD_RUN_DOCKER_EXTERNAL_TESTS}    false   # to be overridden from command line, with string "true"
${CONCURRENCY_LIMIT}                   1       # to be overridden from command line, with integer value, -1 for no limit
${USE_STACKQL_PREINSTALLED}            false   # to be overridden from command line, with string "true"
${SUNDRY_CONFIG}                       {}  # to be overridden from command line, with string value

*** Settings ***
Library           Process
Library           OperatingSystem 
Variables         ${LOCAL_LIB_HOME}/stackql_context.py    ${EXECUTION_PLATFORM}    ${SQL_BACKEND}    ${USE_STACKQL_PREINSTALLED}
...               ${SUNDRY_CONFIG}
Library           Process
Library           OperatingSystem
Library           String
Library           ${LOCAL_LIB_HOME}/StackQLInterfaces.py    ${EXECUTION_PLATFORM}    ${SQL_BACKEND}    ${CONCURRENCY_LIMIT}
Library           ${LOCAL_LIB_HOME}/CloudIntegration.py
Library           ${LOCAL_LIB_HOME}/web_service_keywords.py

*** Keywords ***

Start All Mock Servers
    ${port_dict} =    Create Dictionary    
    ...    oauth_client_credentials_token=${MOCKSERVER_PORT_OAUTH_CLIENT_CREDENTIALS_TOKEN}
    ...    github=${MOCKSERVER_PORT_GITHUB}
    ...    google=${MOCKSERVER_PORT_GOOGLE}
    ...    okta=${MOCKSERVER_PORT_OKTA}
    ...    aws=${MOCKSERVER_PORT_AWS}
    ...    stackql_auth_testing=${MOCKSERVER_PORT_STACKQL_AUTH_TESTING}
    ...    googleadmin=${MOCKSERVER_PORT_GOOGLEADMIN}
    ...    k8s=${MOCKSERVER_PORT_K8S}
    ...    registry=${MOCKSERVER_PORT_REGISTRY}
    ...    azure=${MOCKSERVER_PORT_AZURE}
    ...    sumologic=${MOCKSERVER_PORT_SUMOLOGIC}
    ...    digitalocean=${MOCKSERVER_PORT_DIGITALOCEAN}
    Start All Webservers    port_dict=${port_dict}


Prepare StackQL Environment
    Set Environment Variable    OKTA_SECRET_KEY    ${OKTA_SECRET_STR}
    Set Environment Variable    GITHUB_SECRET_KEY    ${GITHUB_SECRET_STR}
    Set Environment Variable    K8S_SECRET_KEY    ${K8S_SECRET_STR}
    Set Environment Variable    AZ_ACCESS_TOKEN    ${AZURE_SECRET_STR}
    Set Environment Variable    SUMO_CREDS    ${SUMOLOGIC_SECRET_STR}
    Set Environment Variable    DIGITALOCEAN_TOKEN    ${DIGITALOCEAN_SECRET_STR}
    Set Environment Variable    DUMMY_DIGITALOCEAN_USERNAME    ${DUMMY_DIGITALOCEAN_USERNAME_STR}
    Set Environment Variable    DUMMY_DIGITALOCEAN_PASSWORD    ${DUMMY_DIGITALOCEAN_PASSWORD_STR}
    Set Environment Variable    DB_SETUP_SRC    ${DB_SETUP_SRC}
    Set Environment Variable    GOOGLE_APPLICATION_CREDENTIALS    ${GOOGLE_APPLICATION_CREDENTIALS}
    Set Environment Variable    DD_API_KEY    %{DD_API_KEY=myusername}
    Set Environment Variable    DD_APPLICATION_KEY    %{DD_APPLICATION_KEY=mypassword}
    Start All Mock Servers
    Generate Container Credentials for StackQL PG Server mTLS
    Start StackQL PG Server mTLS    ${PG_SRV_PORT_MTLS}    ${PG_SRV_MTLS_CFG_STR}    {}    {}    ${SQL_BACKEND_CFG_STR_CANONICAL}    ${PG_SRV_PORT_DOCKER_MTLS}
    Start StackQL PG Server mTLS    ${PG_SRV_PORT_MTLS_WITH_NAMESPACES}    ${PG_SRV_MTLS_CFG_STR}    ${NAMESPACES_TTL_SPECIALCASE_TRANSPARENT}    {}    ${SQL_BACKEND_CFG_STR_CANONICAL}    ${PG_SRV_PORT_DOCKER_MTLS_WITH_NAMESPACES}
    Start StackQL PG Server mTLS    ${PG_SRV_PORT_MTLS_WITH_EAGER_GC}    ${PG_SRV_MTLS_CFG_STR}    {}    ${GC_CFG_EAGER}    ${SQL_BACKEND_CFG_STR_CANONICAL}    ${PG_SRV_PORT_DOCKER_MTLS_WITH_EAGER_GC}
    Start StackQL PG Server unencrypted    ${PG_SRV_PORT_UNENCRYPTED}    {}    ${SQL_BACKEND_CFG_STR_CANONICAL}
    Start StackQL PG Server mTLS Expose Backing DB    ${PG_SRV_PORT_MTLS_EXPORT}    ${PG_SRV_MTLS_CFG_STR}    {}    {}    ${SQL_BACKEND_CFG_STR_CANONICAL}    ${PG_SRV_PORT_DOCKER_MTLS_EXPORT}
    Start Postgres External Source If Viable
    Sleep    50s

Generate Container Credentials for StackQL PG Server mTLS
    IF    "${EXECUTION_PLATFORM}" == "docker"
        ${res} =         Run Process    docker    compose    \-f    docker\-compose\-credentials.yml
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
                        ...  \-\-pgsrv\.debug\.enable\=true
                        ...  \-\-pgsrv\.tls    ${_MTLS_CFG_STR}
                        ...  \-\-namespaces\=${_NAMESPACES_CFG}
                        ...  \-\-gc\=${_GC_CFG}
                        ...  \-\-execution\.concurrency\.limit\=${CONCURRENCY_LIMIT}
                        ...  \-\-sqlBackend\=${_SQL_BACKEND_CFG}
                        ...  \-\-dbInternal\=${DB_INTERNAL_CFG_LAX}
                        ...  stderr=${CURDIR}/tmp/stdout-stackql-srv-mtls-${_SRV_PORT_MTLS}.txt
                        ...  stdout=${CURDIR}/tmp/stderr-stackql-srv-mtls-${_SRV_PORT_MTLS}.txt
    ELSE IF    "${EXECUTION_PLATFORM}" == "docker"
        ${process} =    Start Process    docker    compose
                        ...  \-p     stackqlpgsrv\-mtls\-${_DOCKER_PORT}
                        ...  run
                        ...  \-\-rm    \-p${_SRV_PORT_MTLS}:${_DOCKER_PORT}/tcp
                        ...  stackqlsrv
                        ...  bash
                        ...  \-c
                        ...  sleep 2 && stackql srv \-\-execution\.concurrency\.limit\=${CONCURRENCY_LIMIT} \-\-registry\='${REGISTRY_NO_VERIFY_CFG_STR.get_config_str('docker')}' \-\-auth\='${AUTH_CFG_STR}' \-\-namespaces\='${_NAMESPACES_CFG}' \-\-gc\='${_GC_CFG}' \-\-sqlBackend\='${_SQL_BACKEND_CFG}' \-\-dbInternal\='${DB_INTERNAL_CFG_LAX}' \-\-tls\.allowInsecure\=true \-\-pgsrv\.address\='0.0.0.0' \-\-pgsrv\.port\=${_DOCKER_PORT} \-\-pgsrv\.debug\.enable\=true \-\-pgsrv\.tls\='{\"keyFilePath\": \"/opt/stackql/srv/credentials/pg_server_key.pem\", \"certFilePath\": \"/opt/stackql/srv/credentials/pg_server_cert.pem\", \"clientCAs\": [\"'$(base64 -w 0 /opt/stackql/srv/credentials/pg_client_cert.pem)'\"]}'
                        ...  stderr=${CURDIR}/tmp/stdout-stackql-srv-mtls-${_SRV_PORT_MTLS}.txt
                        ...  stdout=${CURDIR}/tmp/stderr-stackql-srv-mtls-${_SRV_PORT_MTLS}.txt
    END
    RETURN    ${process}

Start StackQL PG Server mTLS Expose Backing DB
    [Arguments]    ${_SRV_PORT_MTLS}    ${_MTLS_CFG_STR}    ${_NAMESPACES_CFG}    ${_GC_CFG}    ${_SQL_BACKEND_CFG}    ${_DOCKER_PORT}
    IF    "${EXECUTION_PLATFORM}" == "native"
        ${process} =    Start Process    ${STACKQL_EXE}
                        ...  srv    \-\-registry\=${REGISTRY_NO_VERIFY_CFG_STR.get_config_str('native')}
                        ...  \-\-auth\=${AUTH_CFG_STR}
                        ...  \-\-tls\.allowInsecure\=true
                        ...  \-\-pgsrv\.address\=0.0.0.0 
                        ...  \-\-pgsrv\.port\=${_SRV_PORT_MTLS} 
                        ...  \-\-pgsrv\.debug\.enable\=true
                        ...  \-\-pgsrv\.tls    ${_MTLS_CFG_STR}
                        ...  \-\-namespaces\=${_NAMESPACES_CFG}
                        ...  \-\-gc\=${_GC_CFG}
                        ...  \-\-execution\.concurrency\.limit\=${CONCURRENCY_LIMIT}
                        ...  \-\-sqlBackend\=${_SQL_BACKEND_CFG}
                        ...  \-\-dbInternal\=${DB_INTERNAL_CFG_LAX}
                        ...  \-\-export.alias\=stackql_export
                        ...  stderr=${CURDIR}/tmp/stdout-stackql-srv-mtls-backing-${_SRV_PORT_MTLS}.txt
                        ...  stdout=${CURDIR}/tmp/stderr-stackql-srv-mtls-backing-${_SRV_PORT_MTLS}.txt
    ELSE IF    "${EXECUTION_PLATFORM}" == "docker"
        ${process} =    Start Process    docker    compose
                        ...  \-p     stackqlpgsrv\-mtls\-backing\-${_DOCKER_PORT}
                        ...  \-f    ./docker\-compose\-persist\-postgres.yml
                        ...  run
                        ...  \-\-rm    \-p${_SRV_PORT_MTLS}:${_DOCKER_PORT}/tcp
                        ...  stackqlsrv
                        ...  bash
                        ...  \-c
                        ...  sleep 2 && stackql srv \-\-execution\.concurrency\.limit\=${CONCURRENCY_LIMIT} \-\-registry\='${REGISTRY_NO_VERIFY_CFG_STR.get_config_str('docker')}' \-\-auth\='${AUTH_CFG_STR}' \-\-namespaces\='${_NAMESPACES_CFG}' \-\-gc\='${_GC_CFG}' \-\-sqlBackend\='${_SQL_BACKEND_CFG}' \-\-export.alias\='stackql_export' \-\-dbInternal\='${DB_INTERNAL_CFG_LAX}' \-\-tls\.allowInsecure\=true \-\-pgsrv\.address\='0.0.0.0' \-\-pgsrv\.port\=${_DOCKER_PORT} \-\-pgsrv\.debug\.enable\=true \-\-pgsrv\.tls\='{\"keyFilePath\": \"/opt/stackql/srv/credentials/pg_server_key.pem\", \"certFilePath\": \"/opt/stackql/srv/credentials/pg_server_cert.pem\", \"clientCAs\": [\"'$(base64 -w 0 /opt/stackql/srv/credentials/pg_client_cert.pem)'\"]}'
                        ...  stderr=${CURDIR}/tmp/stdout-stackql-srv-mtls-backing-${_SRV_PORT_MTLS}.txt
                        ...  stdout=${CURDIR}/tmp/stderr-stackql-srv-mtls-backing-${_SRV_PORT_MTLS}.txt
    END
    RETURN    ${process}

Start StackQL PG Server unencrypted
    [Arguments]    ${_SRV_PORT_UNENCRYPTED}    ${_NAMESPACES_CFG}   ${_SQL_BACKEND_CFG}
    IF    "${EXECUTION_PLATFORM}" == "native"
        ${process} =    Start Process    ${STACKQL_EXE}
                        ...  srv    \-\-registry\=${REGISTRY_NO_VERIFY_CFG_STR.get_config_str('native')}
                        ...  \-\-auth\=${AUTH_CFG_STR}
                        ...  \-\-tls\.allowInsecure\=true
                        ...  \-\-pgsrv\.address\=0.0.0.0 
                        ...  \-\-pgsrv\.port\=${_SRV_PORT_UNENCRYPTED}
                        ...  \-\-namespaces\=${_NAMESPACES_CFG}
                        ...  \-\-sqlBackend\=${_SQL_BACKEND_CFG}
                        ...  \-\-dbInternal\=${DB_INTERNAL_CFG_LAX}
                        ...  \-\-execution\.concurrency\.limit\=${CONCURRENCY_LIMIT}
                        ...  stderr=${CURDIR}/tmp/stdout-stackql-srv-unencrypted-${_SRV_PORT_UNENCRYPTED}.txt
                        ...  stdout=${CURDIR}/tmp/stderr-stackql-srv-unencrypted-${_SRV_PORT_UNENCRYPTED}.txt
    ELSE IF    "${EXECUTION_PLATFORM}" == "docker"
        ${process} =    Start Process    docker    compose
                        ...  \-p     stackqlpgsrv\-unencrypted
                        ...  run
                        ...  \-\-rm    \-p${_SRV_PORT_UNENCRYPTED}:${PG_SRV_PORT_DOCKER_UNENCRYPTED}/tcp
                        ...  stackqlsrv
                        ...  bash
                        ...  \-c
                        ...  sleep 2 && stackql srv \-\-execution\.concurrency\.limit\=${CONCURRENCY_LIMIT} \-\-registry\='${REGISTRY_NO_VERIFY_CFG_STR.get_config_str('docker')}' \-\-auth\='${AUTH_CFG_STR}' \-\-namespaces\='${_NAMESPACES_CFG}' \-\-sqlBackend\='${_SQL_BACKEND_CFG}' \-\-dbInternal\='${DB_INTERNAL_CFG_LAX}' \-\-tls\.allowInsecure\=true \-\-pgsrv\.address\=0.0.0.0 \-\-pgsrv\.port\=${PG_SRV_PORT_DOCKER_UNENCRYPTED}
                        ...  stderr=${CURDIR}/tmp/stdout-stackql-srv-unencrypted-${_SRV_PORT_UNENCRYPTED}.txt
                        ...  stdout=${CURDIR}/tmp/stderr-stackql-srv-unencrypted-${_SRV_PORT_UNENCRYPTED}.txt
    END
    RETURN    ${process}

Start Postgres External Source If Viable
    IF    "${SHOULD_RUN_DOCKER_EXTERNAL_TESTS}" == "true"
        ${process} =    Start Process    docker    compose
                        ...  \-p     pg\-data\-external
                        ...  \-f     docker\-compose\-externals.yml
                        ...  up
                        ...  postgres_external_data_source
    END

Stackql Per Test Teardown
    IF    "${EXECUTION_PLATFORM}" == "docker" and "${SQL_BACKEND}" == "postgres_tcp"
        ${res} =         Run Process    bash    \-c    docker kill $(docker ps \-\-filter name\=execrun \-q)
        Log     Container killed
        # Should Be Equal As Integers    ${res.rc}    0
        ${restwo} =         Run Process    bash    \-c    docker rm $(docker ps \-\-filter status\=exited \-q)
        Log     Container removed
        # Should Be Equal As Integers    ${restwo.rc}    0
    END
