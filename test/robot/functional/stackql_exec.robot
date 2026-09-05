*** Settings ***
Resource          ${CURDIR}/stackql.resource
Test Teardown     Stackql Per Test Teardown
Documentation     EXEC plan building against methods whose 2xx response schema lacks a
...               property matching an exec parameter (stackql #705 / #726, any-sdk #132).
...               Before the any-sdk fix the parameter-only projected column carried a
...               nil schema wrapped into a non-nil interface, so plan build dereferenced
...               nil in GenerateSelectDML (SIGSEGV). A clean despatch is the assertion.

*** Test Cases ***
Exec Method Whose Response Omits The Exec Parameter Dispatches
    [Documentation]    The activate response is {status, requestId} with no id property;
    ...    the @id exec argument is admitted as a parameter-only projection.
    Should Stackql Exec Inline Equal Stderr
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    exec stackql_native_test.casing.activate.activate @id \= 'x';
    ...    The operation was despatched successfully

Exec Method Whose Response Omits The Exec Parameter Dispatches With Json Payload
    [Documentation]    stackql #726 shape: the same method with an @@json body.
    Should Stackql Exec Inline Equal Stderr
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    exec stackql_native_test.casing.activate.activate @id \= 'x' @@json \= '{"reason": "test"}';
    ...    The operation was despatched successfully

Exec Method Whose Response Omits The Exec Parameter Projects Response Row
    [Documentation]    The response body projects; the schema-less id column no longer
    ...    poisons the select tabulation.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select status, request_id from (exec stackql_native_test.casing.activate.activate @id \= 'x');
    ...    | ACTIVE | req-activate-1 |
