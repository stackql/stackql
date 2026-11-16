*** Settings ***
Documentation     Typed, bounded method discovery through SQL and MCP; no provider requests.
Resource          ${CURDIR}/stackql.resource
Library           MethodSchemaLibrary.py
Suite Setup       Prepare Method Schemas    ${STACKQL_EXE}    ${EXECUTION_PLATFORM}    ${SQL_BACKEND_CFG_STR_CANONICAL}
Suite Teardown    Stop Session Provider

*** Test Cases ***
Method Schema Default Is Typed Shallow And Omits Global Controls
    Json Schema Default Is Typed And Shallow

Method Schema Drill Down Preserves Descriptions And Enums
    Schema Drill Down Preserves Descriptions And Enums

Method Schema XML Request And Transformed Rows Have Nested Shape
    Xml Schema Reports Nested Request And Selected Row

Method Schema Rejects Invalid Controls And Missing Paths
    Invalid Schema Controls Are Rejected

MCP Describe Method Applies Schema Controls
    Mcp Method Schema Controls Reach Backend

MCP Describe Method Accepts Keyword Named Method
    Mcp Describe Method Named List

MCP Stdio Catalogue Cache Policy Is Private And Ordered
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${result}=    Evaluate    stackql_test_tooling.mcp_stdio_client.run_stdio_stateless_roundtrip($STACKQL_EXE, $REGISTRY_NO_VERIFY_CFG_JSON_STR, $AUTH_CFG_STR, "delete from example.service.resource;", approval_action="decline")    modules=stackql_test_tooling.mcp_stdio_client
    Should Be Equal As Integers    ${result['tool_cache_ttl_ms']}    30000
    Should Be Equal    ${result['tool_cache_scope']}    private
    Should Be Equal    ${result['tools']}    ${{sorted($result['tools'])}}
    Should Be Equal As Integers    ${result['returncode']}    0
