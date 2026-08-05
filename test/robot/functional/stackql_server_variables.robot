*** Settings ***
Resource          ${CURDIR}/stackql.resource
Test Teardown     Stackql Per Test Teardown
Documentation     x-stackQL-envVar resolution of OpenAPI server variables (issue 706).
...               The {deployment} server variable is a path segment on the native_test
...               mock, which echoes the segment it was hit on, so tests assert the
...               env-resolved vs WHERE-supplied value and the requiredness flow-through
...               into SHOW METHODS.

*** Test Cases ***
Server Variable Resolves From Env Var
    Set Environment Variable    STACKQL_NATIVE_TEST_DEPLOYMENT    env-deployment
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, deployment from stackql_native_test.srvvar.items;
    ...    env-deployment

Server Variable Where Clause Wins Over Env Var
    Set Environment Variable    STACKQL_NATIVE_TEST_DEPLOYMENT    env-deployment
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, deployment from stackql_native_test.srvvar.items where deployment \= 'where-deployment';
    ...    where-deployment

Server Variable Unset Env Var Fails Server Resolution
    [Documentation]    With the env var unset and no WHERE value the server URL cannot
    ...    resolve; the variable itself is named in SHOW METHODS RequiredParams (below).
    Remove Environment Variable    STACKQL_NATIVE_TEST_DEPLOYMENT
    Should Stackql Exec Inline Contain Stderr
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, deployment from stackql_native_test.srvvar.items;
    ...    cannot find any viable servers

Server Variable Where Clause Satisfies Requirement With Env Unset
    Remove Environment Variable    STACKQL_NATIVE_TEST_DEPLOYMENT
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, deployment from stackql_native_test.srvvar.items where deployment \= 'where-deployment';
    ...    where-deployment

Show Methods Omits Env Resolved Server Variable
    Set Environment Variable    STACKQL_NATIVE_TEST_DEPLOYMENT    env-deployment
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |------------|----------------|---------|
    ...    |${SPACE}MethodName${SPACE}|${SPACE}RequiredParams${SPACE}|${SPACE}SQLVerb${SPACE}|
    ...    |------------|----------------|---------|
    ...    |${SPACE}select${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}SELECT${SPACE}${SPACE}|
    ...    |------------|----------------|---------|
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    show methods in stackql_native_test.srvvar.items;
    ...    ${outputStr}

Show Methods Includes Unresolved Server Variable
    Remove Environment Variable    STACKQL_NATIVE_TEST_DEPLOYMENT
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    show methods in stackql_native_test.srvvar.items;
    ...    deployment
