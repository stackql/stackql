*** Settings ***
Resource          ${CURDIR}/stackql.resource
Test Teardown     Stackql Per Test Teardown
Documentation     Functional coverage for the any-sdk casing engine (PR 107), exercised
...               through the no-auth stackql_native_test provider against the local
...               native_test flask echo mock. These cover the parts that are wired
...               end-to-end today: provider config.snake_case_aliases renames response
...               columns to snake_case, and PascalCase wire request parameters are
...               transmitted as declared.
...
...               KNOWN any-sdk GAPS (tracked as separate issues, deliberately NOT asserted
...               here): (1) a snake WHERE/INSERT key is not reverse-resolved to its
...               PascalCase wire parameter via request.nativeCasing; (2) a multi-word
...               column whose snake alias differs from the wire name projects a null
...               VALUE. When those land upstream this suite should be extended.

*** Test Cases ***
Snake Case Aliases Rename Multi Word Response Columns
    [Documentation]    Wire fields VpcId/SubnetId are exposed as snake_case columns vpc_id/subnet_id.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select vpc_id, subnet_id, echoed_query from stackql_native_test.casing.echo where VpcId \= 'abc123';
    ...    vpc_id

Snake Case Aliases Single Word Column Projects Value
    [Documentation]    echoed_query (snake alias == wire name) projects its value normally.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select echoed_query from stackql_native_test.casing.echo where VpcId \= 'abc123';
    ...    abc123

Pascal Case Wire Parameter Transmitted As Declared
    [Documentation]    The VpcId query parameter reaches the wire with its declared PascalCase name.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select echoed_query from stackql_native_test.casing.echo where VpcId \= 'abc123';
    ...    VpcId\=abc123

Multiple Pascal Case Wire Parameters Transmitted
    [Documentation]    Two PascalCase wire parameters in one WHERE both reach the wire.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select echoed_query from stackql_native_test.casing.echo where VpcId \= 'abc123' and SubnetId \= 'sub-9';
    ...    SubnetId\=sub-9
