*** Settings ***
Resource          ${CURDIR}/stackql.resource
Test Teardown     Stackql Per Test Teardown
Documentation     Functional coverage for the any-sdk casing engine, exercised through the
...               no-auth stackql_native_test provider against the local native_test flask
...               echo mock. Three behaviours are wired end-to-end: (1) config.snake_case_aliases
...               renames response columns to snake_case and the multi-word column VALUE
...               projects via any-sdk GetWireName (issue 108); (2) a snake_case WHERE key is
...               reverse-resolved to its PascalCase wire parameter via the native-casing param
...               set (issue 109); (3) PascalCase wire request parameters are transmitted as
...               declared. Both snake and wire WHERE forms are accepted.

*** Test Cases ***
Snake Case Aliases Multi Word Response Column Projects Value
    [Documentation]    Wire field VpcId is exposed as snake_case column vpc_id and its VALUE
    ...    (echoed back by the mock) now projects via GetWireName (issue 108) instead of NULL.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select vpc_id from stackql_native_test.casing.echo where VpcId \= 'vpc-77';
    ...    vpc-77

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

Snake Case Where Key Resolves To Wire Parameter
    [Documentation]    A snake_case WHERE key (vpc_id) reaches the wire as its PascalCase
    ...    parameter VpcId (issue 109), reverse-resolved via the native-casing param set.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select echoed_query from stackql_native_test.casing.echo where vpc_id \= 'v1';
    ...    VpcId\=v1

Snake Case Multiple Where Keys Resolve To Wire
    [Documentation]    Two snake_case WHERE keys both reverse-resolve to their wire parameters.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select echoed_query from stackql_native_test.casing.echo where vpc_id \= 'v1' and subnet_id \= 's9';
    ...    SubnetId\=s9

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

Describe Extended Shows Snake Case Aliases
    [Documentation]    DESCRIBE surfaces the same snake aliases as SELECT: wire VpcId
    ...    renders as vpc_id (any-sdk ToDescriptionMap parity fix).
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    describe extended stackql_native_test.casing.echo;
    ...    vpc_id

Select Star Projects Snake Aliased Values
    [Documentation]    SELECT * expands to snake-aliased columns (any-sdk GetAllColumns
    ...    parity fix). Before the fix the wire-cased identifiers resolved as string
    ...    literals on a case-sensitive backend and every value projected as its own
    ...    column name; the assertion checks the echoed VALUE, not the header.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select * from stackql_native_test.casing.echo where VpcId \= 'star-val-1';
    ...    star-val-1

Snake Case Where Key Satisfies Required Wire Parameter
    [Documentation]    Method routing accepts a snake key for a REQUIRED wire param
    ...    (any-sdk parameterMatch reverse-casing fix): echo_strict requires VpcId and
    ...    the SQL supplies vpc_id; the echoed wire query proves both routing and
    ...    request construction re-keyed it.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select echoed_query from stackql_native_test.casing.echo_strict where vpc_id \= 'req-9';
    ...    VpcId\=req-9

Base Fallback Body Sent When No Body Params
    [Documentation]    A method with request.base '{}' and no SQL-supplied body fields
    ...    sends the base bytes verbatim (the aws-json no-input pattern); the mock
    ...    echoes the received body.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select echoed_body, ok from stackql_native_test.casing.echo_post;
    ...    {}
