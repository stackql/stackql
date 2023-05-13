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
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SHELL_SESSION_SIMPLE_COMMANDS}
    ...    ${SHELL_SESSION_SIMPLE_EXPECTED}
    ...    stdout=${CURDIR}/tmp/Shell-Session-Simple.tmp
    [Teardown]    Stackql Per Test Teardown

Shell Session Azure Compute Table Nomenclature Mutation Guard
    Pass Execution If    "${IS_WINDOWS}" == "1"    Skipping session test in windows
    Should StackQL Shell Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SHELL_COMMANDS_AZURE_COMPUTE_MUTATION_GUARD}
    ...    ${SHELL_COMMANDS_AZURE_COMPUTE_MUTATION_GUARD_EXPECTED}
    ...    stdout=${CURDIR}/tmp/Shell-Session-Azure-Compute-Table-Nomenclature-Mutation-Guard.tmp
    [Teardown]    Stackql Per Test Teardown

PG Session GC Manual Behaviour Canonical
    Should PG Client Session Inline Equal Strict
    ...    ${PSQL_MTLS_CONN_STR_UNIX}
    ...    ${SHELL_COMMANDS_GC_SEQUENCE_CANONICAL}
    ...    ${SHELL_COMMANDS_GC_SEQUENCE_CANONICAL_JSON_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-GC-Manual-Behaviour-Canonical.tmp
    [Teardown]    NONE

PG Session GC Eager Behaviour Canonical
    Should PG Client Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX_WITH_EAGER_GC}
    ...    ${SHELL_COMMANDS_GC_SEQUENCE_EAGER}
    ...    ${SHELL_COMMANDS_GC_SEQUENCE_EAGER_JSON_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-GC-Eager-Behaviour-Canonical.tmp
    [Teardown]    NONE

PG Session View Handling Behaviour Canonical
    Should PG Client Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX_WITH_EAGER_GC}
    ...    ${SHELL_COMMANDS_VIEW_HANDLING_SEQUENCE}
    ...    ${SHELL_COMMANDS_VIEW_HANDLING_SEQUENCE_JSON_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-View-Handling-Behaviour-Canonical.tmp
    [Teardown]    NONE

PG Session Aliased Cross Cloud Disks View Handling Behaviour Canonical
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Skipping postgres backend test due to split_part function
    Should PG Client Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX_WITH_EAGER_GC}
    ...    ${SHELL_COMMANDS_DISKS_VIEW_ALIASED_SEQUENCE}
    ...    ${SHELL_COMMANDS_DISKS_VIEW_ALIASED_SEQUENCE_JSON_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Aliased-Cross-Cloud-Disks-View-Handling-Behaviour-Canonical.tmp
    [Teardown]    NONE

PG Session NOT Aliased Cross Cloud Disks View Handling Behaviour Canonical
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Skipping postgres backend test due to split_part function
    Should PG Client Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX_WITH_EAGER_GC}
    ...    ${SHELL_COMMANDS_DISKS_VIEW_NOT_ALIASED_SEQUENCE}
    ...    ${SHELL_COMMANDS_DISKS_VIEW_NOT_ALIASED_SEQUENCE_JSON_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-NOT-Aliased-Cross-Cloud-Disks-View-Handling-Behaviour-Canonical.tmp
    [Teardown]    NONE

PG Session Azure Compute Table Nomenclature Mutation Guard
    Should PG Client Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX}
    ...    ${SHELL_COMMANDS_AZURE_COMPUTE_MUTATION_GUARD}
    ...    ${SHELL_COMMANDS_AZURE_COMPUTE_MUTATION_GUARD_JSON_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Azure-Compute-Table-Nomenclature-Mutation-Guard.tmp
    [Teardown]    NONE

PG Session Wrongly Named Column error recovery Azure Compute Table Nomenclature
    Should PG Client Session Inline Contain
    ...    ${PSQL_MTLS_CONN_STR_UNIX}
    ...    ${SHELL_SESSION_SIMPLE_COMMANDS_AFTER_ERROR}
    ...    ${SHELL_SESSION_SIMPLE_COMMANDS_AFTER_ERROR_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Wrongly-Named-Column-error-recovery-Azure-Compute-Table-Nomenclature.tmp
    [Teardown]    NONE

Shell Session Azure Billing Path Interrogation Regression Guard
    Should PG Client Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX}
    ...    ${SHELL_COMMANDS_AZURE_BILLING_PATH_SPLIT_GUARD}
    ...    ${SHELL_COMMANDS_AZURE_BILLING_PATH_SPLIT_GUARD_JSON_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Azure-Compute-Table-Nomenclature-Mutation-Guard.tmp
    [Teardown]    NONE

PG Session Anayltics Cache Behaviour Canonical
    Should PG Client Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX_WITH_NAMESPACES}
    ...    ${SHELL_COMMANDS_SPECIALCASE_REPEATED_CACHED}
    ...    ${SHELL_COMMANDS_SPECIALCASE_REPEATED_CACHED_JSON_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Anayltics-Cache-Behaviour-Canonical.tmp
    [Teardown]    NONE

PG Session Postgres Client Setup Queries
    Should PG Client Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX}
    ...    ${PG_CLIENT_SETUP_QUERIES}
    ...    ${PG_CLIENT_SETUP_QUERIES_JSON_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Postgres-Client-Setup-Queries.tmp
    [Teardown]    NONE

PG Session Postgres Client V2 Setup Queries
    Should PG Client V2 Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX}
    ...    ${PG_CLIENT_SETUP_QUERIES}
    ...    ${PG_CLIENT_SETUP_QUERIES_JSON_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Postgres-Client-V2-Setup-Queries.tmp
    [Teardown]    NONE

PG Session Postgres Client AWS Method Signature Polymorphism
    Should PG Client Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX}
    ...    ${AWS_CLOUD_CONTROL_METHOD_SIGNATURE_CMD_ARR}
    ...    ${AWS_CLOUD_CONTROL_METHOD_SIGNATURE_CMD_ARR_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Postgres-Client-AWS-Method-Signature-Polymorphism.tmp
    [Teardown]    NONE

PG Session Postgres Client Typed Queries
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Skipping postgres backend test likely due to typing issues
    Should PG Client Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX}
    ...    ${SELECT_AWS_CLOUD_CONTROL_EVENTS_MINIMAL}
    ...    ${SELECT_AWS_CLOUD_CONTROL_EVENTS_MINIMAL_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Postgres-Client-Typed-Queries.tmp
    [Teardown]    NONE

PG Session Server Survives Defective Query
    Should PG Client Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX}
    ...    ${AWS_CLOUD_CONTROL_BUCKET_DETAIL_PROJECTION_DEFECTIVE_CMD_ARR}
    ...    ${AWS_CLOUD_CONTROL_BUCKET_DETAIL_PROJECTION_DEFECTIVE_CMD_ARR_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Server-Survives-Defective-Query-and-Subsequently-Serves-Valid-Query.tmp
    [Teardown]    NONE

PG Session Postgres Client V2 Typed Queries
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"     TODO: FIX THIS... Skipping postgres backend test likely due to typing issues
    Should PG Client V2 Session Inline Equal
    ...    ${PSQL_MTLS_CONN_STR_UNIX}
    ...    ${SELECT_AWS_CLOUD_CONTROL_EVENTS_MINIMAL}
    ...    ${SELECT_AWS_CLOUD_CONTROL_EVENTS_MINIMAL_EXPECTED}
    ...    stdout=${CURDIR}/tmp/PG-Session-Postgres-Client-V2-Typed-Queries.tmp
    [Teardown]    NONE
