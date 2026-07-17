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

# ===========================================================================
# Issue #684: pluggable cursor strategies (keyset / offset / page_info).
# Each strategy resource is backed by its own mock endpoint that reflects the
# strategy-specific wire argument into the rows, so both the traversal (the
# rank-5 `purple` row is only served on the third page) and the wire shape
# are asserted from STDOUT.
# ===========================================================================

GraphQL Keyset Strategy Traverses All Pages
    [Documentation]    keyset injects a rankGt comparator on the last row's sort key;
    ...                termination is an empty row array.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, rank from stackql_native_test.graph.keyset_things order by rank limit 100;
    ...    purple

GraphQL Keyset Strategy Advances Comparator On The Wire
    [Documentation]    The third page's request carried rankGt: 4 (the last rank of page two).
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name from stackql_native_test.graph.keyset_things where wire_rank_gt \= 4 limit 100;
    ...    purple

GraphQL Offset Strategy Traverses All Pages
    [Documentation]    offset substitutes a client-side running row count; termination is
    ...                an empty row array (or a short page under the configured pageSize).
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, rank from stackql_native_test.graph.offset_things order by rank limit 100;
    ...    purple

GraphQL Offset Strategy Advances Offset On The Wire
    [Documentation]    The third page's request carried offset: 4 (rows already returned).
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name from stackql_native_test.graph.offset_things where wire_offset \= 4 limit 100;
    ...    purple

GraphQL PageInfo Strategy Terminates On HasNextPage Flag
    [Documentation]    Relay-strict: the mock's endCursor stays NON-EMPTY on the final
    ...                page, so completing the traversal (reaching rank-5 purple) proves
    ...                termination came from pageInfo.hasNextPage, not cursor emptiness -
    ...                a cursor-emptiness reader would loop forever here.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, rank from stackql_native_test.graph.pageinfo_things order by rank limit 100;
    ...    purple

GraphQL PageInfo Strategy Follows EndCursor On The Wire
    [Documentation]    The third page's request carried after: "c3" (page two's endCursor).
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name from stackql_native_test.graph.pageinfo_things where wire_after \= 'c3' limit 100;
    ...    purple
