*** Settings ***
Library    Process    

*** Settings ***
Variables    ${CURDIR}/variables/stackql_context.py

*** Test Cases *** 
Positive Control
    Should contain    ''    ''

Get Providers
    ${result} =     Run Process    ${STACKQL_EXE}    exec    \-\-registry\=${REGISTRY_CFG_STR}    ${SHOW_PROVIDERS_STR} 
    Log    ${result.stdout}
    Should contain    ${result.stdout}   okta

Get Okta Services
    ${result} =     Run Process    ${STACKQL_EXE}    exec    \-\-registry\=${REGISTRY_CFG_STR}    ${SHOW_OKTA_SERVICES_FILTERED_STR} 
    Log    ${result.stdout}
    Should contain    ${result.stdout}   Application${SPACE}API

Get Okta Application Resources
    ${result} =     Run Process    ${STACKQL_EXE}    exec    \-\-registry\=${REGISTRY_CFG_STR}    ${SHOW_OKTA_APPLICATION_RESOURCES_FILTERED_STR} 
    Log    ${result.stdout}
    Should contain    ${result.stdout}   grants    groups
