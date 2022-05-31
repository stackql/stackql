*** Variables ***
${LOCAL_LIB_HOME}    ${CURDIR}/../lib

*** Settings ***
Library    Process
Library    OperatingSystem 
Library    ${LOCAL_LIB_HOME}/StackQLInterfaces.py  

*** Settings ***
Variables    ${CURDIR}/../variables/stackql_context.py

*** Test Cases *** 
Positive Control
    Should contain    ''    ''

Get Providers
    Should StackQL Exec Contain    ${SHOW_PROVIDERS_STR}   okta
    Should StackQL Novel Exec Contain    ${SHOW_PROVIDERS_STR}   v0.3.1
    Should StackQL Exec Contain JSON output    ${SHOW_PROVIDERS_STR}   okta

Get Providers No Config
    Should StackQL No Cfg Exec Contain    ${SHOW_PROVIDERS_STR}   name

Get Okta Services
    Should StackQL Exec Contain    ${SHOW_OKTA_SERVICES_FILTERED_STR}    Application${SPACE}API

Get Okta Application Resources
    Should StackQL Exec Contain    ${SHOW_OKTA_APPLICATION_RESOURCES_FILTERED_STR}    grants    groups

Describe GitHub Repos Pages
    Should StackQL Novel Exec Contain    ${DESCRIBE_GITHUB_REPOS_PAGES}    https_certificate    url

Show Methods GitHub
    Should StackQL Novel Exec Equal    ${SHOW_METHODS_GITHUB_REPOS_REPOS}   ${SHOW_METHODS_GITHUB_REPOS_REPOS_EXPECTED}

*** Keywords ***
Run StackQL Exec Command
    [Arguments]    ${_EXEC_CMD_STR}    @{varargs}
    Set Environment Variable    OKTA_SECRET_KEY    ${OKTA_SECRET_STR}
    Set Environment Variable    GITHUB_SECRET_KEY    ${GITHUB_SECRET_STR}
    Set Environment Variable    K8S_SECRET_KEY    ${K8S_SECRET_STR}
    ${result} =     Run Process    
                    ...  ${STACKQL_EXE}
                    ...  exec    \-\-registry\=${REGISTRY_NO_VERIFY_CFG_STR}
                    ...  \-\-auth\=${AUTH_CFG_STR}
                    ...  \-\-tls.allowInsecure\=true
                    ...  ${_EXEC_CMD_STR}    @{varargs}
    Log             ${result.stdout}
    Log             ${result.stderr}
    [Return]    ${result}

Run StackQL Canonical Exec Command
    [Arguments]    ${_EXEC_CMD_STR}    @{varargs}
    Set Environment Variable    OKTA_SECRET_KEY    ${OKTA_SECRET_STR}
    Set Environment Variable    GITHUB_SECRET_KEY    ${GITHUB_SECRET_STR}
    Set Environment Variable    K8S_SECRET_KEY    ${K8S_SECRET_STR}
    ${result} =     Run Process    
                    ...  ${STACKQL_EXE}
                    ...  exec    \-\-registry\=${REGISTRY_CANONICAL_CFG_STR}
                    ...  \-\-auth\=${AUTH_CFG_STR}
                    ...  \-\-tls.allowInsecure\=true
                    ...  ${_EXEC_CMD_STR}    @{varargs}
    Log             ${result.stdout}
    Log             ${result.stderr}
    [Return]    ${result}

Run StackQL Canonical No Cfg Exec Command
    [Arguments]    ${_EXEC_CMD_STR}    @{varargs}
    Set Environment Variable    OKTA_SECRET_KEY    ${OKTA_SECRET_STR}
    Set Environment Variable    GITHUB_SECRET_KEY    ${GITHUB_SECRET_STR}
    Set Environment Variable    K8S_SECRET_KEY    ${K8S_SECRET_STR}
    ${result} =     Run Process    
                    ...  ${STACKQL_EXE}
                    ...  exec    ${_EXEC_CMD_STR}    @{varargs}
    Log             ${result.stdout}
    Log             ${result.stderr}
    [Return]    ${result}

Run StackQL Deprecated Exec Command
    [Arguments]    ${_EXEC_CMD_STR}    @{varargs}
    Set Environment Variable    OKTA_SECRET_KEY    ${OKTA_SECRET_STR}
    Set Environment Variable    GITHUB_SECRET_KEY    ${GITHUB_SECRET_STR}
    Set Environment Variable    K8S_SECRET_KEY    ${K8S_SECRET_STR}
    ${result} =     Run Process    
                    ...  ${STACKQL_EXE}
                    ...  exec    \-\-registry\=${REGISTRY_DEPRECATED_CFG_STR}
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
    ${result} =    Run StackQL Canonical Exec Command    ${_EXEC_CMD_STR}    @{varargs}
    Should Be Equal    ${result.stdout}    ${_EXEC_CMD_EXPECTED_OUTPUT} 
    ${result} =    Run StackQL Deprecated Exec Command    ${_EXEC_CMD_STR}    @{varargs}
    Should Be Equal    ${result.stdout}    ${_EXEC_CMD_EXPECTED_OUTPUT} 

Should StackQL Novel Exec Equal
    [Arguments]    ${_EXEC_CMD_STR}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}
    ${result} =    Run StackQL Exec Command    ${_EXEC_CMD_STR}    @{varargs}
    Should Be Equal    ${result.stdout}    ${_EXEC_CMD_EXPECTED_OUTPUT} 
    ${result} =    Run StackQL Canonical Exec Command    ${_EXEC_CMD_STR}    @{varargs}
    Should Be Equal    ${result.stdout}    ${_EXEC_CMD_EXPECTED_OUTPUT}
Should StackQL Exec Contain
    [Arguments]    ${_EXEC_CMD_STR}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}
    ${result} =    Run StackQL Exec Command    ${_EXEC_CMD_STR}
    Should contain    ${result.stdout}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}
    ${result} =    Run StackQL Canonical Exec Command    ${_EXEC_CMD_STR}
    Should contain    ${result.stdout}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}
    ${result} =    Run StackQL Deprecated Exec Command    ${_EXEC_CMD_STR}
    Should contain    ${result.stdout}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}

Should StackQL Exec Contain JSON output
    [Arguments]    ${_EXEC_CMD_STR}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}
    ${result} =    Run StackQL Exec Command    ${_EXEC_CMD_STR}    \-o\=json
    Should contain    ${result.stdout}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}
    ${result} =    Run StackQL Canonical Exec Command    ${_EXEC_CMD_STR}    \-o\=json
    Should contain    ${result.stdout}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}
    ${result} =    Run StackQL Deprecated Exec Command    ${_EXEC_CMD_STR}    \-o\=json
    Should contain    ${result.stdout}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}

Should StackQL Novel Exec Contain
    [Arguments]    ${_EXEC_CMD_STR}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}
    ${result} =    Run StackQL Exec Command    ${_EXEC_CMD_STR}
    Should contain    ${result.stdout}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}
    ${result} =    Run StackQL Canonical Exec Command    ${_EXEC_CMD_STR}
    Should contain    ${result.stdout}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}

Should StackQL No Cfg Exec Contain
    [Arguments]    ${_EXEC_CMD_STR}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}
    ${result} =    Run StackQL Canonical No Cfg Exec Command    ${_EXEC_CMD_STR}
    Should contain    ${result.stdout}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}


