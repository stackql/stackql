*** Settings ***
Resource          ${CURDIR}/stackql.resource
Test Teardown     Stackql Per Test Teardown
Documentation     Functional coverage for OData query-option push-down (issue 659) via the
...               any-sdk HTTPPreparator.WithPushdownIntent apply-path: stackql computes a
...               neutral PushdownIntent from the SELECT during analysis and hands it to the
...               preparator, which (inside any-sdk) translates it to the OData dialect and
...               sets the request query - stackql never mutates the HTTP request itself.
...               The no-auth stackql_native_test.odata.people resource carries a
...               queryParamPushdown config; the native_test flask mock echoes the decoded
...               request query into an `echoed` column, so each test asserts that the wire
...               shape matches the SQL intent for every OData option
...               ($filter/$select/$orderby/$top/$skip/$count).
...               Push-down is an optimisation only: stackql's client-side WHERE/projection
...               remain authoritative (asserted by the last case).

*** Test Cases ***
OData Filter Eq Pushed From Where
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, echoed from stackql_native_test.odata.people where city \= 'NYC';
    ...    $filter\=city eq 'NYC'

OData Filter Like Becomes Startswith
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, echoed from stackql_native_test.odata.people where name like 'A%';
    ...    startswith(name,'A')

OData Top Pushed From Limit
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, echoed from stackql_native_test.odata.people limit 5;
    ...    $top\=5

OData Skip Pushed From Offset
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, echoed from stackql_native_test.odata.people limit 5 offset 1;
    ...    $skip\=1

OData Orderby Pushed
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, echoed from stackql_native_test.odata.people order by age asc;
    ...    $orderby\=age asc

OData Select Projection Pushed
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, echoed from stackql_native_test.odata.people;
    ...    $select\=name,echoed

OData Count Pushed From Count Star
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    SELECT count(*), <non-grouped col> is sqlite-only syntax (postgres requires GROUP BY); $count push-down is asserted on the sqlite backend.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select count(*) as cnt, echoed from stackql_native_test.odata.people;
    ...    $count\=true

OData Client Side Filter Remains Authoritative
    [Documentation]    Push-down is additive: the client-side WHERE still removes the non-matching row.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name from stackql_native_test.odata.people where name like 'A%';
    ...    Alice

OData Pushdown Suppressed For Grain Changing Query
    [Documentation]    GROUP BY changes grain, so LIMIT must NOT push $top (which the mock honours).
    ...                With the guard the full set is fetched and the client-side aggregate counts all
    ...                3 rows; a wrongly-pushed $top=1 would under-count to 1.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select count(*) as c from stackql_native_test.odata.people group by echoed limit 1;
    ...    3
