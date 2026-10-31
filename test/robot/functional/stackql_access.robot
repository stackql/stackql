*** Settings ***
Resource    ${CURDIR}/stackql.resource
Library     stackql_test_tooling.access_management_keywords.AccessManagementKeywords

*** Test Cases ***

JWT CLI Authenticates And Uses Directory Groups
    [Documentation]    Authenticate a signed JWT; inspect its principal/groups; read a provider; refuse missing, invalid and unauthorized credentials.
    Verify Access Scenario    jwt_cli

JWT Supports Every Configured Signing Algorithm
    [Documentation]    Read with EdDSA, RSA, RSA-PSS and ECDSA JWTs; reject a valid token when the configured key algorithm differs.
    Verify Access Scenario    jwt_algorithms

JWT HTTPS Isolates Callers And Refreshes Membership
    [Documentation]    Use request bearer credentials over verified HTTPS; deny bad tokens, concurrent caller leakage and revoked membership.
    Verify Access Scenario    jwt_https

JWT Shell Uses The Local Credential
    [Documentation]    Feed a real shell session a provider query; repeat without a token and require authentication denial.
    Verify Access Scenario    jwt_shell

JWT MCP Stdio Uses The Local Credential
    [Documentation]    Initialize a real MCP stdio session and call run_select_query; repeat without a token and require denial.
    Verify Access Scenario    jwt_stdio

Static Bearer CLI Verifies A Provisioned Digest
    [Documentation]    Provision SHA-256 token digests; accept the reader token and deny missing, wrong and unauthorized tokens.
    Verify Access Scenario    bearer_cli

Static Bearer HTTPS Isolates Callers And Refreshes Membership
    [Documentation]    Use actual HTTPS bearer headers; deny invalid tokens and concurrent identity leakage; revoke membership between requests.
    Verify Access Scenario    bearer_https

Static Bearer Shell Uses The Local Credential
    [Documentation]    Read from the shell with a provisioned token; refuse the same query without it.
    Verify Access Scenario    bearer_shell

Static Bearer MCP Stdio Uses The Local Credential
    [Documentation]    Read through MCP stdio with a provisioned token; refuse the same query without it.
    Verify Access Scenario    bearer_stdio

MTLS HTTPS Verifies Client Certificates And SAN Identity
    [Documentation]    Verify the client CA and unique URI SAN; reject missing, untrusted, expired, ambiguous and unmapped certificates; refresh directory groups.
    Verify Access Scenario    mtls_https

MTLS HTTPS Supports A Unique DNS SAN Identity
    [Documentation]    Map a verified unique DNS SAN through the directory; reject invalid chains, expiry, ambiguous SANs and unknown identities.
    Verify Access Scenario    mtls_https_dns

Local OS CLI Uses The Effective User And Directory Groups
    [Documentation]    Map the effective UID/SID to the reader; read a provider; deny mutation and an unprovisioned OS identity.
    Verify Access Scenario    local_os_cli

Local OS Shell Uses The Effective User
    [Documentation]    Read a provider through the real shell using the process identity, without a token.
    Verify Access Scenario    local_os_shell

Local OS MCP Stdio Uses The Effective User
    [Documentation]    Read a provider through MCP stdio using the process identity, without a token.
    Verify Access Scenario    local_os_stdio

Local OS Cannot Authenticate Remote HTTP Callers
    [Documentation]    Refuse HTTP startup rather than authenticate remote callers as the server OS account.
    Verify Access Scenario    local_os_http_refused

Access Configuration Accepts A Strict Path Envelope
    [Documentation]    Use JSON path and legacy filename forms; deny unknown fields, missing paths, trailing data and invalid authentication modes.
    Verify Access Scenario    config

Reader Policy Denies Effects And Protects Directory Tables
    [Documentation]    Deny DML, EXEC, IaC, directory-table access and effectful functions; isolate concurrent principals.
    Verify Access Scenario    policy

Directory Revocation And Malformed Attributes Fail Closed
    [Documentation]    Revoke a reader membership and corrupt operator attributes; deny execution before provider effects.
    Verify Access Scenario    revocation

Protected IAC Applies Updates Prunes And Confines State
    [Documentation]    Apply and update two resources; refuse another principal and path escape; prune one and destroy the rest; require exit failure for unsuccessful IaC.
    Verify Access Scenario    iac

Disabled Caller Access Preserves Legacy Queries
    [Documentation]    Read the mock provider with no access configuration or caller credential.
    Verify Access Scenario    compatibility

JWT Rejects Missing Token
    [Documentation]    Require CLI failure and zero provider requests for this invalid credential.
    Verify Access Scenario    jwt_invalid:missing

JWT Rejects Invalid Signature
    [Documentation]    Require CLI failure and zero provider requests for this invalid credential.
    Verify Access Scenario    jwt_invalid:signature

JWT Rejects Wrong Issuer
    [Documentation]    Require CLI failure and zero provider requests for this invalid credential.
    Verify Access Scenario    jwt_invalid:issuer

JWT Rejects Wrong Audience
    [Documentation]    Require CLI failure and zero provider requests for this invalid credential.
    Verify Access Scenario    jwt_invalid:audience

JWT Rejects Missing Expiry
    [Documentation]    Require CLI failure and zero provider requests for this invalid credential.
    Verify Access Scenario    jwt_invalid:expiry_required

JWT Rejects Expired Token
    [Documentation]    Require CLI failure and zero provider requests for this invalid credential.
    Verify Access Scenario    jwt_invalid:expired

JWT Rejects Future Not Before
    [Documentation]    Require CLI failure and zero provider requests for this invalid credential.
    Verify Access Scenario    jwt_invalid:not_before

JWT Rejects Future Issued At
    [Documentation]    Require CLI failure and zero provider requests for this invalid credential.
    Verify Access Scenario    jwt_invalid:issued_at

JWT Rejects Unknown Key ID
    [Documentation]    Require CLI failure and zero provider requests for this invalid credential.
    Verify Access Scenario    jwt_invalid:key_id

JWT Rejects Unknown Identity
    [Documentation]    Require CLI failure and zero provider requests for this invalid credential.
    Verify Access Scenario    jwt_invalid:identity

Documented JWT Walkthrough Works
    [Documentation]    Execute the published quickstart script; provision the directory and authenticate its reader using this mode.
    Verify Access Walkthrough    ${STACKQL_EXE}    jwt    ${REPOSITORY_ROOT}    ${EXECUTION_PLATFORM}

Documented Static Bearer Walkthrough Works
    [Documentation]    Execute the published quickstart script; provision the directory and authenticate its reader using this mode.
    Verify Access Walkthrough    ${STACKQL_EXE}    bearer    ${REPOSITORY_ROOT}    ${EXECUTION_PLATFORM}

Documented MTLS Walkthrough Works
    [Documentation]    Execute the published quickstart script; provision the directory and authenticate its reader using this mode.
    Verify Access Walkthrough    ${STACKQL_EXE}    mtls    ${REPOSITORY_ROOT}    ${EXECUTION_PLATFORM}

Documented Local OS Walkthrough Works
    [Documentation]    Execute the published quickstart script; provision the directory and authenticate its reader using this mode.
    Verify Access Walkthrough    ${STACKQL_EXE}    local_os    ${REPOSITORY_ROOT}    ${EXECUTION_PLATFORM}

Queries Have No Authentication Deadline By Default
    [Documentation]    Complete a slow provider query beyond max_session_seconds when deadline enforcement is disabled.
    Verify Access Scenario    query_deadline_default

Authentication Query Deadline Is Opt In
    [Documentation]    Enable deadline enforcement and require cancellation of a slow provider query at max_session_seconds.
    Verify Access Scenario    query_deadline_opt_in

*** Keywords ***
Verify Access Scenario
    [Arguments]    ${scenario}
    Verify Caller Access    ${STACKQL_EXE}    ${SQL_BACKEND_CFG_STR_CANONICAL}    ${EXECUTION_PLATFORM}    ${scenario}
