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
