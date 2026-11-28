*** Settings ***
Documentation     MCP extensions (skills, tasks, file uploads) through the real native binary with raw JSON-RPC.
...               Every extension is off by default; with all of them off the advertised surface is unchanged.
...               Application roots, databases and loopback ports are unique per test.
Library           MCPExtensionsLibrary.py
Test Setup        Prepare Native MCP Extensions
Test Teardown     Stop MCP Extensions

*** Variables ***
${STACKQL_EXE}     ${EXECDIR}${/}build${/}stackql

*** Keywords ***
Prepare Native MCP Extensions
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Prepare MCP Extensions    ${STACKQL_EXE}

*** Test Cases ***
MCP Discover Capabilities Unchanged With Extensions Disabled
    [Documentation]    server/discover (2026-07-28) and initialize (2025-11-25) capabilities and the tool,
    ...                prompt, resource and template names match the main baseline with every extension off.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Discover Capabilities Unchanged With Extensions Disabled

MCP Discover Advertises Enabled Extensions
    [Documentation]    Each extension toggled on alone is advertised on both revisions and nothing else changes.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Discover Advertises Enabled Extensions

MCP Skills List Returns Complete Entries With Valid Digests
    [Documentation]    Over stdio and stateless HTTP every manifest file's sha256 and size are recomputed from
    ...                resources/read output; 2026-07-28 results carry resultType, ttlMs and cacheScope.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Skills List Returns Complete Entries With Valid Digests

MCP Skills Get Unknown URI Is Invalid Params
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Skills Get Unknown URI Is Invalid Params

MCP Skills Absent When Disabled
    [Documentation]    No skills/list handler (-32601), no extension advertised, resources/list unchanged.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Skills Absent When Disabled

MCP Skills Allowlist Filters Entries
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Skills Allowlist Filters Entries

MCP IaC Run Returns Task When Client Declares Tasks
    [Documentation]    stdio and stateless HTTP: CreateTaskResult shape, tasks/get to completed, rows equal to a
    ...                synchronous run of the same statement, and decision plus execution audit records with the task id.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    IaC Run Returns Task When Client Declares Tasks

MCP IaC Run Stays Synchronous Without Tasks Capability
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    IaC Run Stays Synchronous Without Tasks Capability

MCP Ordinary Mutation Stays Synchronous With Tasks Capability
    [Documentation]    Recover, lifecycle-tool converge, plan SELECT and multi-statement batches never become tasks.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Ordinary Mutation Stays Synchronous With Tasks Capability

MCP Task Outlives Request Timeout
    [Documentation]    request_timeout 1s with a 3s provider create: the task completes, the synchronous call hits the limit.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Task Outlives Request Timeout

MCP Task Approval Precedes Task Creation
    [Documentation]    safe mode: input_required first, a decline returns no task, an accept returns the task.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Task Approval Precedes Task Creation

MCP Tasks Get Unknown Id Is Invalid Params
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Tasks Get Unknown Id Is Invalid Params

MCP Task Tool Error Completes With The Synchronous Error Result
    [Documentation]    A provider error is a tool error: the task completes with the isError result a synchronous call
    ...                returns (ext-tasks: non-JSON-RPC errors must not use failed). The failed status with an error
    ...                object is covered by the Go test TestTaskFailureCarriesErrorObject.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Task Tool Error Completes With The Synchronous Error Result

MCP Task Cancel Acknowledges And Run Completes
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Task Cancel Acknowledges And Run Completes

MCP Attach Policy Document Inline Round Trip
    [Documentation]    stdio and stateless HTTP: attach a small Rego file inline and read the same bytes back
    ...                through resources/read; x-mcp-file sits on the document property only.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Attach Policy Document Inline Round Trip

MCP Attach Policy Document Rejects Size Mismatch
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Attach Policy Document Rejects    size mismatch

MCP Attach Policy Document Rejects Digest Mismatch
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Attach Policy Document Rejects    digest mismatch

MCP Attach Policy Document Rejects Over Inline Limit
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Attach Policy Document Rejects    over inline limit

MCP Attach Policy Document Rejects Missing Data
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Attach Policy Document Rejects    missing data

MCP Attach Policy Document Rejects Binary Content
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Attach Policy Document Rejects    binary content

MCP Attach Policy Document Accepts Undeclared Inline Call
    [Documentation]    Inline data is never rejected because the request did not declare the extension (2026-07-28
    ...                without the declaration, and a 2025-11-25 session).
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Attach Policy Document Accepts Undeclared Inline Call

MCP Attach Policy Document Allowed In Read Only Mode
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Attach Policy Document Allowed In Read Only Mode

MCP Policy Document Data Never Audited
    [Documentation]    Neither the base64 data nor the decoded text reaches the audit file or stderr; args keep
    ...                only name, mimeType, size and digest.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Policy Document Data Never Audited
