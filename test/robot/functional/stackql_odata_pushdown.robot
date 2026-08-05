*** Settings ***
Resource          ${CURDIR}/stackql.resource
Test Teardown     Stackql Per Test Teardown
Documentation     OData query-option push-down (issue 659) via any-sdk
...               WithPushdownIntent: stackql computes a neutral PushdownIntent,
...               any-sdk renders the OData dialect. The native_test mock echoes the
...               decoded request query so each test asserts the wire shape.
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

OData Select Union Includes Where And Order By Columns
    [Documentation]    Issue #682: a pushed $select must include WHERE / ORDER BY-only
    ...                columns; the echoed wire query proves the union reached the server.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, echoed from stackql_native_test.odata.people where city \= 'NYC' order by age asc;
    ...    $select\=name,echoed,city,age

OData Where Only Column Still Filters Rows
    [Documentation]    Issue #682 end-to-end: the WHERE column is absent from the SELECT
    ...                list, the mock strips unselected fields, the row must still return.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name from stackql_native_test.odata.people where city \= 'NYC';
    ...    Alice

OData Restricted Select Superset Includes Supported Filter Column
    [Documentation]    any-sdk #116 with a supportedColumns allowlist: the pushed filter
    ...                column (city, select-supported) joins the emitted $select.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, echoed from stackql_native_test.odata.people_restricted where city \= 'NYC';
    ...    $select\=name,echoed,city

OData Restricted Select Superset Includes Supported Order By Column
    [Documentation]    any-sdk #116: the pushed ORDER BY column (city, select-supported)
    ...                joins the emitted $select and $orderby is unchanged.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, echoed from stackql_native_test.odata.people_restricted order by city asc;
    ...    $select\=name,echoed,city

OData Restricted Filter Outside Select Allowlist Never Reaches Select
    [Documentation]    any-sdk #116: age is filter-supported but outside the select
    ...                allowlist. The filter still pushes; because stackql unions the
    ...                WHERE column into the pushdown projection, the all-or-nothing
    ...                allowlist gate suppresses $select entirely rather than emit an
    ...                incoherent one (the trailing cell boundary proves its absence).
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, echoed from stackql_native_test.odata.people_restricted where age \= 30;
    ...    $filter\=age eq 30${SPACE}|

OData Restricted Select Star Emits No Select
    [Documentation]    any-sdk #116: an empty projection (SELECT *) with a pushed filter
    ...                must not conjure a $select; the echoed query terminates at the filter.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select * from stackql_native_test.odata.people_restricted where city \= 'NYC';
    ...    eq 'NYC'${SPACE}|
