*** Settings ***
Library    Process
Library    OperatingSystem   

*** Settings ***
Variables    ${CURDIR}/variables/stackql_context.py

*** Test Cases *** 
Positive Control
    Should contain    ''    ''

Get Providers
    Should StackQL Exec Contain    ${SHOW_PROVIDERS_STR}   okta

Get Okta Services
    Should StackQL Exec Contain    ${SHOW_OKTA_SERVICES_FILTERED_STR}    Application${SPACE}API

Get Okta Application Resources
    Should StackQL Exec Contain    ${SHOW_OKTA_APPLICATION_RESOURCES_FILTERED_STR}    grants    groups

*** Keywords ***
Run StackQL Exec Command
    [Arguments]    ${_EXEC_CMD_STR}    @{varargs}
    Set Environment Variable    OKTA_SECRET_KEY    ${OKTA_SECRET_STR}
    ${result} =     Run Process    
                    ...  ${STACKQL_EXE}
                    ...  exec    \-\-registry\=${REGISTRY_NO_VERIFY_CFG_STR}
                    ...  \-\-auth\=${AUTH_CFG_STR}
                    ...  \-\-tls.allowInsecure\=true
                    ...  ${_EXEC_CMD_STR}    @{varargs}
    Log             ${result.stdout}
    Log             ${result.stderr}
    [Return]    ${result}

Run StackQL Embedded Exec Command
    [Arguments]    ${_EXEC_CMD_STR}    @{varargs}
    Set Environment Variable    OKTA_SECRET_KEY    ${OKTA_SECRET_STR}
    ${result} =     Run Process    
                    ...  ${STACKQL_EXE}
                    ...  exec    \-\-registry\=${REGISTRY_EMBEDDED_CFG_STR}
                    ...  \-\-auth\=${AUTH_CFG_STR}
                    ...  \-\-tls.allowInsecure\=true
                    ...  ${_EXEC_CMD_STR}    @{varargs}
    Log             ${result.stdout}
    Log             ${result.stderr}
    [Return]    ${result}


Should StackQL Exec Equal
    [Arguments]    ${_EXEC_CMD_STR}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}
    ${result} =    Run StackQL Exec Command    ${_EXEC_CMD_STR}    @{varargs}
    Should Be Equal    ${result.stdout}    ${_EXEC_CMD_EXPECTED_OUTPUT}

Should StackQL Exec Contain
    [Arguments]    ${_EXEC_CMD_STR}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}
    ${result} =    Run StackQL Exec Command    ${_EXEC_CMD_STR}
    Should contain    ${result.stdout}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}


