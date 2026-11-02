*** Settings ***
Documentation     Query, SHOW METHODS, EXEC and mutations share opt-in endpoint selection and binding precedence.
Resource          ${CURDIR}/stackql.resource
Library           EndpointSelectorLibrary.py
Suite Setup       Prepare Endpoint Selectors    ${STACKQL_EXE}    ${EXECUTION_PLATFORM}    ${SQL_BACKEND_CFG_STR_CANONICAL}
Suite Teardown    Stop Endpoint Selectors

*** Test Cases ***
Default Selector Matches Query Show And Exec
    Default Selector Matches Query Show And Exec

Selector And Direct Override Apply To Exec
    Selector And Direct Override Apply To Exec

SQL And Alias Defaults Override Environment
    SQL And Alias Defaults Override Environment

Mutation Uses The Same Selector
    Mutation Uses The Same Selector

Unknown Selector Fails Before Requests
    Unknown Selector Fails Before Requests

Missing Declared Source Stays Required
    Missing Declared Source Stays Required
