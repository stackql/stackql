*** Variables ***
${LOCAL_LIB_HOME}               ${CURDIR}${/}..${/}..${/}python
${REPOSITORY_ROOT}              ${CURDIR}${/}..${/}..${/}..
${EXECUTION_PLATFORM}           native   # to be overridden from command line, eg "docker"
${SQL_BACKEND}                  sqlite_embedded   # to be overridden from command line, eg "postgres_tcp"
${IS_WSL}                       false   # to be overridden from command line, with string "true"
${USE_STACKQL_PREINSTALLED}     false   # to be overridden from command line, with string "true"
${SUNDRY_CONFIG}                {}  # to be overridden from command line, with string value
${STACKQL_INTERFACE_LIBRARY}    stackql_test_tooling.StackQLInterfaces
${CLOUD_INTEGRATION_LIBRARY}    stackql_test_tooling.CloudIntegration

*** Settings ***
Library           Process
Library           OperatingSystem 
Variables         ${LOCAL_LIB_HOME}/stackql_test_tooling/stackql_context.py    ${REPOSITORY_ROOT}    ${EXECUTION_PLATFORM}    ${SQL_BACKEND}    ${USE_STACKQL_PREINSTALLED}
...               ${SUNDRY_CONFIG}
Library           Process
Library           OperatingSystem
Library           String
Library           ${STACKQL_INTERFACE_LIBRARY}    ${EXECUTION_PLATFORM}    ${SQL_BACKEND}
Library           ${CLOUD_INTEGRATION_LIBRARY}