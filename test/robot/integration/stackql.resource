*** Variables ***
${LOCAL_LIB_HOME}        ${CURDIR}/../lib
${EXECUTION_PLATFORM}    native   # to be overridden from command line, eg "docker"
${SQL_BACKEND}           sqlite_embedded   # to be overridden from command line, eg "postgres_tcp"
${IS_WSL}                false   # to be overridden from command line, with string "true"

*** Settings ***
Library           Process
Library           OperatingSystem 
Variables         ${LOCAL_LIB_HOME}/stackql_context.py    ${EXECUTION_PLATFORM}    ${SQL_BACKEND}
Library           Process
Library           OperatingSystem
Library           String
Library           ${LOCAL_LIB_HOME}/StackQLInterfaces.py    ${EXECUTION_PLATFORM}    ${SQL_BACKEND}
Library           ${LOCAL_LIB_HOME}/CloudIntegration.py