*** Settings ***
Documentation     Real-binary MCP cache policy and OAuth opt-out checks with per-run staging and ephemeral loopback ports.
Library           MCPCacheAuthLibrary.py
Suite Setup       Prepare Isolated Cache Auth Fixture
Suite Teardown    Stop Cache Auth Fixture

*** Variables ***
${STACKQL_EXE}     ${EXECDIR}${/}build${/}stackql

*** Keywords ***
Prepare Isolated Cache Auth Fixture
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    MCP tests are disabled on this platform
    Prepare Cache Auth Fixture    ${STACKQL_EXE}

*** Test Cases ***
MCP Private Catalogue Isolation And Immediately Stale Reads
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    MCP tests are disabled on this platform
    Cache Catalogues Are Private And Reads Immediately Stale

MCP OAuth Disabled Preserves Bearer And Anonymous Opt In After Cache Warming
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    MCP tests are disabled on this platform
    OAuth Disabled HTTP Preserves Auth After Cache Warming
