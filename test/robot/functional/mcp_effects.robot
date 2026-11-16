*** Settings ***
Documentation     Issue 818: nested EXEC through the real native MCP binary with a counted POST operation.
...               Fixture registry, application root and database are unique per test; loopback ports are ephemeral.
...               SELECT tool remains read-only in all modes; effectful SQL uses mutation/lifecycle tools.
...               Issue 919: current-revision approval state tampering and replay cause no additional operations.
...               Sessionless HTTP reconnects carry request-local W3C trace context into approval outcome audits.
...               HTTP requests reconnect to one instance, with two interleaved outstanding approval prompts.
...               Concurrent accepted retries must overlap at a provider-side barrier and retain scoped outcome audits.
...               Audit assertions cover completed decisions and caller traces, not pending prompts or authenticated subjects.
...               Backend-owned semantic view and indirection classification is tracked separately in issue 928.
Library           MCPEffectsLibrary.py
Test Setup        Prepare Native MCP Effects
Test Teardown     Stop MCP Effects

*** Variables ***
${STACKQL_EXE}     ${EXECDIR}${/}build${/}stackql

*** Keywords ***
Prepare Native MCP Effects
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Prepare MCP Effects    ${STACKQL_EXE}

*** Test Cases ***
Nested EXEC Read Only Performs Zero Provider Operations
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Nested Exec Read Only Has Zero Operations

Nested EXEC Safe Requires Approval Before Provider Operation
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Nested Exec Requires Approval    safe

Nested EXEC Delete Safe Requires Approval Before Provider Operation
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Nested Exec Requires Approval    delete_safe

Nested EXEC Full Access Executes Through Effectful Tools
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Nested Exec Full Access Executes

IaC Recovery Read Only Refuses Before Ledger Writes
    [Documentation]    Preview is enabled; run, destroy and recover are refused without creating ledger state.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Iac Read Only Does Not Write Ledger

Nested EXEC Safe Rejects Tampered And Replayed Approval State
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Approval State Tampering And Replay Prevent Operations    safe

Nested EXEC Delete Safe Rejects Tampered And Replayed Approval State
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Approval State Tampering And Replay Prevent Operations    delete_safe

Reconnecting HTTP Clients Keep Safe Approval Audit Request Scoped
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Reconnecting Clients Keep Approval Audit Request Scoped    safe

Reconnecting HTTP Clients Keep Delete Safe Approval Audit Request Scoped
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Reconnecting Clients Keep Approval Audit Request Scoped    delete_safe

Concurrent HTTP Clients Keep Safe Approval Audit Request Scoped
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Concurrent Clients Keep Approval Audit Request Scoped    safe

Concurrent HTTP Clients Keep Delete Safe Approval Audit Request Scoped
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Concurrent Clients Keep Approval Audit Request Scoped    delete_safe
