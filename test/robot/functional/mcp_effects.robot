*** Settings ***
Documentation     Issue 818: nested EXEC through the real native MCP binary with a counted POST operation.
...               Fixture registry, application root and database are unique per test; loopback ports are ephemeral.
...               SELECT tool remains read-only in all modes; effectful SQL uses mutation/lifecycle tools.
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
