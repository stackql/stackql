*** Settings ***
Resource          ${CURDIR}/stackql.resource
Test Teardown     Stackql Per Test Teardown
Documentation     Functional coverage for the GraphQL acquire path against a mocked,
...               no-auth provider (stackql_native_test.graph.things, backed by the
...               native_test flask mock). Covers the any-sdk cursor_after pagination
...               strategy and the stackql GraphQL LIMIT push-down (SQL LIMIT -> the
...               query's {{ .limit }} / first: N). The mock reflects the wire page
...               args back into each node (wire_first / wire_after) so the push-down
...               and cursor-follow are asserted from STDOUT (the --http.log.enabled
...               wire log is not portably captured under the docker execution platform).

*** Test Cases ***
GraphQL Cursor Pagination Returns All Pages
    [Documentation]    The mock serves two things per page; the reader follows endCursor until exhausted.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, rank from stackql_native_test.graph.things order by rank limit 100;
    ...    purple

GraphQL Limit Pushed Into Query First Arg
    [Documentation]    SQL LIMIT 42 renders as `first: 42` in the wire query; the mock reflects it as wire_first.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, wire_first from stackql_native_test.graph.things limit 42;
    ...    42

GraphQL Pagination Follows Cursor In Wire Request
    [Documentation]    Subsequent pages carry the Relay-style after: cursor; the mock reflects it as wire_after.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, wire_after from stackql_native_test.graph.things limit 42;
    ...    c1
