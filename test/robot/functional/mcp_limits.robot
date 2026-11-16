*** Settings ***
Documentation     Issue 828: real native binary, isolated registry/staging files and ephemeral loopback provider/MCP ports.
...               Run only this suite with STACKQL_EXE pointing to a freshly built binary.
Library           MCPLimitsLibrary.py
Suite Setup       Prepare Native MCP Limits
Suite Teardown    Stop MCP Limits

*** Variables ***
${STACKQL_EXE}     ${EXECDIR}${/}build${/}stackql

*** Keywords ***
Prepare Native MCP Limits
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Prepare MCP Limits    ${STACKQL_EXE}

*** Test Cases ***
MCP Stdio Configured Concurrency Bounds Slow Provider Calls
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Configured Concurrency Bounds Provider Requests    stdio

MCP HTTP Configured Concurrency Bounds Slow Provider Calls
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Configured Concurrency Bounds Provider Requests    http

MCP Stdio Deadline Cancels Provider Call And Session Survives
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Configured Deadline Cancels Provider And Session Survives    stdio

MCP HTTP Deadline Cancels Provider Call And Session Survives
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Configured Deadline Cancels Provider And Session Survives    http
