*** Variables ***
${LOCAL_LIB_HOME}        ${CURDIR}/../lib
${EXECUTION_PLATFORM}    native   # to be overridden from command line, eg "docker"

*** Settings ***
Library           Process
Library           OperatingSystem 
Variables         ${LOCAL_LIB_HOME}/stackql_context.py    ${EXECUTION_PLATFORM}
Library           String
Library           ${LOCAL_LIB_HOME}/StackQLInterfaces.py    ${EXECUTION_PLATFORM}
Library           ${LOCAL_LIB_HOME}/CloudIntegration.py

