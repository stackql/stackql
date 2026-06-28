*** Settings ***
Resource          ${CURDIR}/stackql.resource
Test Teardown     Stackql Per Test Teardown
Documentation     Functional coverage for the GraphQL acquire path against a mocked,
...               no-auth provider (stackql_native_test.graph.things, backed by the
...               native_test flask mock). Covers the any-sdk cursor_after pagination
...               strategy, the stackql GraphQL LIMIT push-down (SQL LIMIT -> the query's
...               {{ .limit }} / first: N), and the alpha08 --http.log.enabled wire-request
...               logging (graphql.ContextWithHTTPLogger).

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

GraphQL Limit Pushed Into Query And Wire Logged
    [Documentation]    SQL LIMIT 3 renders as `first: 3` in the wire query, emitted to stderr under --http.log.enabled.
    Should StackQL Exec Inline Contain Stderr
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name from stackql_native_test.graph.things limit 3;
    ...    first: 3
    ...    stackql_debug_http=${True}

GraphQL Pagination Follows Cursor In Wire Request
    [Documentation]    Subsequent pages carry the Relay-style after: cursor in the wire query.
    Should StackQL Exec Inline Contain Stderr
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name from stackql_native_test.graph.things limit 100;
    ...    after:
    ...    stackql_debug_http=${True}
