*** Settings ***
Resource          ${CURDIR}/stackql.resource
Test Teardown     Stackql Per Test Teardown
Documentation     Functional coverage for scalar-function projections over provider table
...               columns (issue 687). A function projection must NOT inherit its argument
...               column's provider schema type: binding it made type-changing functions
...               (typeof / date / datetime) scan their text results through the argument's
...               declared type, yielding 0/null, and corrupted same-named sibling
...               projections in the same SELECT (the "contagion"). Exercised through the
...               no-auth stackql_native_test provider whose xml_ec2.volumes fixture
...               declares `size` as integer. The sqlite-native date/time/typeof functions
...               are asserted on the sqlite backend only.

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
    [Documentation]    Issue #687 contagion guard: co-projecting typeof(size) with the bare
    ...                size column previously nulled the sibling; the size value 16 must
    ...                still appear in the result set.
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
