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
