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

PG Session Azure Compute Table Nomenclature Mutation Guard
    Should PG Client Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX}
    ...    ${SHELL_COMMANDS_AZURE_COMPUTE_MUTATION_GUARD}
    ...    ${SHELL_COMMANDS_AZURE_COMPUTE_MUTATION_GUARD_JSON_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Azure-Compute-Table-Nomenclature-Mutation-Guard.tmp

PG Session Postgres Client Setup Queries
    Should PG Client Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX}
    ...    ${PG_CLIENT_SETUP_QUERIES}
    ...    ${PG_CLIENT_SETUP_QUERIES_JSON_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Postgres-Client-Setup-Queries.tmp

PG Session Postgres Client V2 Setup Queries
    Should PG Client V2 Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX}
    ...    ${PG_CLIENT_SETUP_QUERIES}
    ...    ${PG_CLIENT_SETUP_QUERIES_JSON_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Postgres-Client-V2-Setup-Queries.tmp

PG Session Postgres Client Typed Queries
    Should PG Client Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX}
    ...    ${SELECT_AWS_CLOUD_CONTROL_EVENTS_MINIMAL}
    ...    ${SELECT_AWS_CLOUD_CONTROL_EVENTS_MINIMAL_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Postgres-Client-Typed-Queries.tmp

PG Session Postgres Client V2 Typed Queries
    Should PG Client V2 Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX}
    ...    ${SELECT_AWS_CLOUD_CONTROL_EVENTS_MINIMAL}
    ...    ${SELECT_AWS_CLOUD_CONTROL_EVENTS_MINIMAL_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Postgres-Client-V2-Typed-Queries.tmp
