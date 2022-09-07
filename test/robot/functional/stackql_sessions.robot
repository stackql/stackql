*** Settings ***
Resource          ${CURDIR}/stackql.resource

*** Test Cases *** 
Shell Session Simple
    Pass Execution If    "${IS_WINDOWS}" == "1"    Skipping session test in windows
    Should StackQL Shell Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SHELL_SESSION_SIMPLE_COMMANDS}
    ...    ${SHELL_SESSION_SIMPLE_EXPECTED}
    ...    stdout=${CURDIR}/tmp/Shell-Session-Simple.tmp

Shell Session Azure Compute Table Nomenclature Mutation Guard
    Pass Execution If    "${IS_WINDOWS}" == "1"    Skipping session test in windows
    Should StackQL Shell Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SHELL_COMMANDS_AZURE_COMPUTE_MUTATION_GUARD}
    ...    ${SHELL_COMMANDS_AZURE_COMPUTE_MUTATION_GUARD_EXPECTED}
    ...    stdout=${CURDIR}/tmp/Shell-Session-Azure-Compute-Table-Nomenclature-Mutation-Guard.tmp
