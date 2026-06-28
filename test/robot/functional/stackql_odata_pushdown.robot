*** Settings ***
Resource          ${CURDIR}/stackql.resource
Test Teardown     Stackql Per Test Teardown
Documentation     Functional coverage for OData query-option push-down, wiring the any-sdk
...               formulation.ApplyPushdown helper into the stackql REST acquire path.
...               The no-auth stackql_native_test.odata.people resource carries a
...               queryParamPushdown config; the native_test flask mock echoes the decoded
...               request query into an `echoed` column so each test can assert which
...               OData option ($filter/$select/$orderby/$top/$skip/$count) was pushed.
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
    ...    stackql_debug_http=${True}

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
    ...    stackql_debug_http=${True}

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
    ...    stackql_debug_http=${True}

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
    ...    stackql_debug_http=${True}

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
    ...    stackql_debug_http=${True}

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
    ...    stackql_debug_http=${True}

OData Count Pushed From Count Star
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
    ...    stackql_debug_http=${True}

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
    ...    stackql_debug_http=${True}

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
    ...    stackql_debug_http=${True}
