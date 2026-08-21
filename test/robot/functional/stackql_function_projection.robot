*** Settings ***
Resource          ${CURDIR}/stackql.resource
Test Teardown     Stackql Per Test Teardown
Documentation     Scalar-function projections over provider table columns (issue 687):
...               a function projection must not inherit its argument column's schema
...               type (typeof/date/datetime yielded 0/null and corrupted same-named
...               siblings). Also covers function projections over a subquery
...               (issue 352), where an expression that does not resolve to a source
...               column of the subquery must not abort query rewriting.
...               Uses the no-auth stackql_native_test provider.

*** Test Cases ***
Typeof Over Bare Integer Column Returns Underlying Type
    [Documentation]    Issue #687: typeof(size) previously returned 0 because the text
    ...                result was scanned through size's declared integer type.
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    typeof is a sqlite-native function; asserted on the sqlite backend.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select volume_id, typeof(size) as t from stackql_native_test.xml_ec2.volumes order by volume_id;
    ...    integer

Datetime Over Bare Integer Column Projects Timestamp
    [Documentation]    Issue #687: datetime(size, 'unixepoch') previously returned 0; the
    ...                bare-column argument form must match the expression-wrapped form.
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    datetime(..., 'unixepoch') is sqlite-native syntax; asserted on the sqlite backend.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select volume_id, datetime(size, 'unixepoch') as dt from stackql_native_test.xml_ec2.volumes order by volume_id;
    ...    1970-01-01 00:00:08

Function Projection Does Not Corrupt Sibling Column
    [Documentation]    Issue #687 contagion guard: co-projecting typeof(size) with the
    ...                bare size column must not null the sibling.
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    typeof is a sqlite-native function; asserted on the sqlite backend.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select volume_id, typeof(size) as t, size from stackql_native_test.xml_ec2.volumes order by volume_id;
    ...    16

Aggregate Over Bare Column Unaffected
    [Documentation]    Control: aggregate typing (sum over the integer column) is unchanged
    ...                by the issue #687 fix.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select sum(size) as total from stackql_native_test.xml_ec2.volumes;
    ...    24

Function Expression Supplies Required Parameter
    [Documentation]    Issue #686: a pure-literal function expression constraining a
    ...                required parameter is constant-folded and supplied to the request.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select volumeId, size from aws.ec2.volumes where region \= lower('AP-SOUTHEAST-2') order by volumeId asc;
    ...    vol-00100000000000000

Function Expression Containing Column Reference Stays Client Side
    [Documentation]    Issue #686 scope guard: an expression referencing a column is not
    ...                folded; it remains an authoritative client-side filter.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select volumeId from aws.ec2.volumes where region \= lower('AP-SOUTHEAST-2') and size \= abs(size) order by volumeId asc;
    ...    vol-00200000000000000

Json Extract Over Subquery Function Argument Projects Value
    [Documentation]    Issue #352: a JSON_EXTRACT projection over a subquery whose argument
    ...                is itself an expression previously failed with
    ...                "query rewriting for indirection: cannot find col".
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    json_extract and json_object are sqlite-native functions; asserted on the sqlite backend.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select json_extract(json_object('vid', volume_id), '$.vid') as vid from (select volume_id from stackql_native_test.xml_ec2.volumes) foo;
    ...    vol-1

Literal Leading Function Projection Over Subquery Projects Value
    [Documentation]    Issue #352 generalisation: a function projection over a subquery whose
    ...                leading argument is a literal has no inferable source column and must
    ...                still be projected, on either backend.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select coalesce('vol-fallback', volume_id) as vid from (select volume_id from stackql_native_test.xml_ec2.volumes) foo;
    ...    vol-fallback
