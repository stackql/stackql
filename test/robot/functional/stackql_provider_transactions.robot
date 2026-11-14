*** Settings ***
Resource    ${CURDIR}/stackql.resource
Library     stackql_test_tooling.access_management_keywords.AccessManagementKeywords

*** Test Cases ***
Provider Lazy Rollback Discards Unexecuted Mutations
    Verify Caller Access    ${STACKQL_EXE}    ${SQL_BACKEND_CFG_STR_CANONICAL}    ${EXECUTION_PLATFORM}    provider_transaction_lazy_rollback

Provider Read Drains Preceding Mutation And Rollback Uses IaC Inverse
    Verify Caller Access    ${STACKQL_EXE}    ${SQL_BACKEND_CFG_STR_CANONICAL}    ${EXECUTION_PLATFORM}    provider_transaction_read_barrier

Provider Commit Persists Journal And Transaction Status
    Verify Caller Access    ${STACKQL_EXE}    ${SQL_BACKEND_CFG_STR_CANONICAL}    ${EXECUTION_PLATFORM}    provider_transaction_commit

Provider Irreversible Mutation Is Rejected Before Dispatch
    Verify Caller Access    ${STACKQL_EXE}    ${SQL_BACKEND_CFG_STR_CANONICAL}    ${EXECUTION_PLATFORM}    provider_transaction_irreversible

Provider Transaction Begin Requires Authorization
    Verify Caller Access    ${STACKQL_EXE}    ${SQL_BACKEND_CFG_STR_CANONICAL}    ${EXECUTION_PLATFORM}    provider_transaction_denied

Provider Eager Rollback Uses The Shared Inverse
    Verify Caller Access    ${STACKQL_EXE}    ${SQL_BACKEND_CFG_STR_CANONICAL}    ${EXECUTION_PLATFORM}    provider_transaction_eager_rollback

Provider Returning Preserves Inverse Identity Outside Projection
    Verify Caller Access    ${STACKQL_EXE}    ${SQL_BACKEND_CFG_STR_CANONICAL}    ${EXECUTION_PLATFORM}    provider_transaction_returning

Provider Create Without Declared Inverse Never Dispatches
    Verify Caller Access    ${STACKQL_EXE}    ${SQL_BACKEND_CFG_STR_CANONICAL}    ${EXECUTION_PLATFORM}    provider_transaction_missing_inverse

Provider Transaction Status Is An Authorized Session Bound Read
    Verify Caller Access    ${STACKQL_EXE}    ${SQL_BACKEND_CFG_STR_CANONICAL}    ${EXECUTION_PLATFORM}    provider_transaction_status

Provider Committed Effects Resume After Process Exit And Lease Expiry
    Verify Caller Access    ${STACKQL_EXE}    ${SQL_BACKEND_CFG_STR_CANONICAL}    ${EXECUTION_PLATFORM}    provider_transaction_resume

Provider Deferred Queue Survives Process Exit Without Duplicate Creation
    Verify Caller Access    ${STACKQL_EXE}    ${SQL_BACKEND_CFG_STR_CANONICAL}    ${EXECUTION_PLATFORM}    provider_transaction_resume_queue

Provider Missing Handle Requires Explicit Recovery Before Rollback
    Verify Caller Access    ${STACKQL_EXE}    ${SQL_BACKEND_CFG_STR_CANONICAL}    ${EXECUTION_PLATFORM}    provider_transaction_recovery

Provider Transactions Work Without Opting Into Caller Auth
    Verify Caller Access    ${STACKQL_EXE}    ${SQL_BACKEND_CFG_STR_CANONICAL}    ${EXECUTION_PLATFORM}    provider_transaction_auth_disabled

Unconfigured SQL Transactions Use Omni And Keep Creates Distinct
    Verify Caller Access    ${STACKQL_EXE}    ${SQL_BACKEND_CFG_STR_CANONICAL}    ${EXECUTION_PLATFORM}    provider_transaction_default
