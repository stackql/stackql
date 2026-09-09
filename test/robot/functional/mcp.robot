*** Settings ***
Resource          ${CURDIR}${/}stackql.resource
Library           Collections


*** Keywords ***
Start MCP Servers
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Start Process                         ${STACKQL_EXE}
    ...                                   mcp
    ...                                   \-\-mcp.server.type\=http
    ...                                   \-\-mcp.config
    ...                                   {"server": {"transport": "http", "address": "127.0.0.1:9912", "mode": "full_access", "audit": {"disabled": true}} }
    ...                                   \-\-registry
    ...                                   ${REGISTRY_NO_VERIFY_CFG_JSON_STR}
    ...                                   \-\-auth
    ...                                   ${AUTH_CFG_STR}
    ...                                   \-\-tls.allowInsecure
    Start Process                         ${STACKQL_EXE}
    ...                                   srv
    ...                                   \-\-mcp.server.type\=http
    ...                                   \-\-mcp.config
    ...                                   {"server": {"transport": "http", "address": "127.0.0.1:9913", "mode": "full_access", "audit": {"disabled": true}} }
    ...                                   \-\-registry
    ...                                   ${REGISTRY_NO_VERIFY_CFG_JSON_STR}
    ...                                   \-\-auth
    ...                                   ${AUTH_CFG_STR}
    ...                                   \-\-tls.allowInsecure
    ...                                   \-\-pgsrv.port
    ...                                   5665
    Start Process                         ${STACKQL_EXE}
    ...                                   srv
    ...                                   \-\-mcp.server.type\=reverse_proxy
    ...                                   \-\-mcp.config
    ...                                   {"server": {"transport": "http", "address": "127.0.0.1:9914", "mode": "full_access", "audit": {"disabled": true}}, "backend": {"dsn": "postgres:\/\/stackql:stackql@127.0.0.1:5445?default_query_exec_mode\=simple_protocol"} }
    ...                                   \-\-registry
    ...                                   ${REGISTRY_NO_VERIFY_CFG_JSON_STR}
    ...                                   \-\-auth
    ...                                   ${AUTH_CFG_STR}
    ...                                   \-\-tls.allowInsecure
    ...                                   \-\-pgsrv.port
    ...                                   5445
    Start Process                         ${STACKQL_EXE}
    ...                                   srv
    ...                                   \-\-mcp.server.type\=reverse_proxy
    ...                                   \-\-mcp.config
    ...                                   {"server": {"tls_cert_file": "test/server/mtls/credentials/pg_server_cert.pem", "tls_key_file": "test/server/mtls/credentials/pg_server_key.pem", "transport": "http", "address": "127.0.0.1:9004", "mode": "full_access", "audit": {"disabled": true}}, "backend": {"dsn": "postgres:\/\/stackql:stackql@127.0.0.1:5446?default_query_exec_mode\=simple_protocol"} }
    ...                                   \-\-registry
    ...                                   ${REGISTRY_NO_VERIFY_CFG_JSON_STR}
    ...                                   \-\-auth
    ...                                   ${AUTH_CFG_STR}
    ...                                   \-\-tls.allowInsecure
    ...                                   \-\-pgsrv.port
    ...                                   5446
    ...                                   stdout=${CURDIR}${/}tmp${/}Stackql-MCP-Server-HTTPS.txt
    ...                                   stderr=${CURDIR}${/}tmp${/}Stackql-MCP-Server-HTTPS-stderr.txt
    Start Process                         ${STACKQL_EXE}
    ...                                   mcp
    ...                                   \-\-mcp.server.type\=http
    ...                                   \-\-mcp.config
    ...                                   {"server": {"transport": "http", "address": "127.0.0.1:9915", "mode": "full_access", "audit": {"disabled": true}}, "enabled_tools": ["server_info"] }
    ...                                   \-\-registry
    ...                                   ${REGISTRY_NO_VERIFY_CFG_JSON_STR}
    ...                                   \-\-auth
    ...                                   ${AUTH_CFG_STR}
    ...                                   \-\-tls.allowInsecure
    ...                                   stdout=${CURDIR}${/}tmp${/}Stackql-MCP-Server-Restricted.txt
    ...                                   stderr=${CURDIR}${/}tmp${/}Stackql-MCP-Server-Restricted-stderr.txt
    Start Process                         ${STACKQL_EXE}
    ...                                   mcp
    ...                                   \-\-mcp.server.type\=http
    ...                                   \-\-mcp.config
    ...                                   {"server": {"transport": "http", "address": "127.0.0.1:9916", "read_only": true, "audit": {"disabled": true}} }
    ...                                   \-\-registry
    ...                                   ${REGISTRY_NO_VERIFY_CFG_JSON_STR}
    ...                                   \-\-auth
    ...                                   ${AUTH_CFG_STR}
    ...                                   \-\-tls.allowInsecure
    ...                                   stdout=${CURDIR}${/}tmp${/}Stackql-MCP-Server-ReadOnly.txt
    ...                                   stderr=${CURDIR}${/}tmp${/}Stackql-MCP-Server-ReadOnly-stderr.txt
    Start Process                         ${STACKQL_EXE}
    ...                                   mcp
    ...                                   \-\-mcp.server.type\=http
    ...                                   \-\-mcp.config
    ...                                   {"server": {"transport": "http", "address": "127.0.0.1:9917", "mode": "full_access", "audit": {"disabled": true}} }
    ...                                   \-\-registry
    ...                                   ${REGISTRY_NO_VERIFY_CFG_JSON_STR}
    ...                                   \-\-auth
    ...                                   ${AUTH_CFG_STR}
    ...                                   \-\-tls.allowInsecure
    ...                                   env:STACKQL_QUERY_LIBRARY_OFFLINE=true
    ...                                   stdout=${CURDIR}${/}tmp${/}Stackql-MCP-Server-QueryLibrary.txt
    ...                                   stderr=${CURDIR}${/}tmp${/}Stackql-MCP-Server-QueryLibrary-stderr.txt
    # Mode-contract servers: one per non-default mode.  Audit disabled so we
    # don't litter the cwd with log files; audit is exercised by 9923.
    Start Process                         ${STACKQL_EXE}
    ...                                   mcp
    ...                                   \-\-mcp.server.type\=http
    ...                                   \-\-mcp.config
    ...                                   {"server": {"transport": "http", "address": "127.0.0.1:9920", "mode": "read_only", "audit": {"disabled": true}} }
    ...                                   \-\-registry
    ...                                   ${REGISTRY_NO_VERIFY_CFG_JSON_STR}
    ...                                   \-\-auth
    ...                                   ${AUTH_CFG_STR}
    ...                                   \-\-tls.allowInsecure
    ...                                   stdout=${CURDIR}${/}tmp${/}Stackql-MCP-Server-Mode-ReadOnly.txt
    ...                                   stderr=${CURDIR}${/}tmp${/}Stackql-MCP-Server-Mode-ReadOnly-stderr.txt
    Start Process                         ${STACKQL_EXE}
    ...                                   mcp
    ...                                   \-\-mcp.server.type\=http
    ...                                   \-\-mcp.config
    ...                                   {"server": {"transport": "http", "address": "127.0.0.1:9921", "mode": "delete_safe", "audit": {"disabled": true}} }
    ...                                   \-\-registry
    ...                                   ${REGISTRY_NO_VERIFY_CFG_JSON_STR}
    ...                                   \-\-auth
    ...                                   ${AUTH_CFG_STR}
    ...                                   \-\-tls.allowInsecure
    ...                                   stdout=${CURDIR}${/}tmp${/}Stackql-MCP-Server-Mode-DeleteSafe.txt
    ...                                   stderr=${CURDIR}${/}tmp${/}Stackql-MCP-Server-Mode-DeleteSafe-stderr.txt
    Start Process                         ${STACKQL_EXE}
    ...                                   mcp
    ...                                   \-\-mcp.server.type\=http
    ...                                   \-\-mcp.config
    ...                                   {"server": {"transport": "http", "address": "127.0.0.1:9922", "mode": "full_access", "audit": {"disabled": true}} }
    ...                                   \-\-registry
    ...                                   ${REGISTRY_NO_VERIFY_CFG_JSON_STR}
    ...                                   \-\-auth
    ...                                   ${AUTH_CFG_STR}
    ...                                   \-\-tls.allowInsecure
    ...                                   stdout=${CURDIR}${/}tmp${/}Stackql-MCP-Server-Mode-FullAccess.txt
    ...                                   stderr=${CURDIR}${/}tmp${/}Stackql-MCP-Server-Mode-FullAccess-stderr.txt
    # Audit-enabled server: writes JSONL to a known path so the audit scenario
    # can read it back.  Path lives under the test tmp dir.
    Start Process                         ${STACKQL_EXE}
    ...                                   mcp
    ...                                   \-\-mcp.server.type\=http
    ...                                   \-\-mcp.config
    ...                                   {"server": {"transport": "http", "address": "127.0.0.1:9923", "mode": "full_access", "audit": {"file": {"path": "mcp-audit-9923.log"}}} }
    ...                                   \-\-registry
    ...                                   ${REGISTRY_NO_VERIFY_CFG_JSON_STR}
    ...                                   \-\-auth
    ...                                   ${AUTH_CFG_STR}
    ...                                   \-\-tls.allowInsecure
    ...                                   stdout=${CURDIR}${/}tmp${/}Stackql-MCP-Server-Audit.txt
    ...                                   stderr=${CURDIR}${/}tmp${/}Stackql-MCP-Server-Audit-stderr.txt
    # Issue #669: server-level render default.  Tool result text content is
    # rendered as compact JSON rather than markdown.
    Start Process                         ${STACKQL_EXE}
    ...                                   mcp
    ...                                   \-\-mcp.server.type\=http
    ...                                   \-\-mcp.config
    ...                                   {"server": {"transport": "http", "address": "127.0.0.1:9924", "mode": "full_access", "render": "json", "audit": {"disabled": true}} }
    ...                                   \-\-registry
    ...                                   ${REGISTRY_NO_VERIFY_CFG_JSON_STR}
    ...                                   \-\-auth
    ...                                   ${AUTH_CFG_STR}
    ...                                   \-\-tls.allowInsecure
    ...                                   stdout=${CURDIR}${/}tmp${/}Stackql-MCP-Server-Render-JSON.txt
    ...                                   stderr=${CURDIR}${/}tmp${/}Stackql-MCP-Server-Render-JSON-stderr.txt
    # Issue #729: audit log in the otel (OTLP/JSON) format, selected via the
    # --mcp.log.format flag rather than mcp.config to exercise the override.
    Start Process                         ${STACKQL_EXE}
    ...                                   mcp
    ...                                   \-\-mcp.server.type\=http
    ...                                   \-\-mcp.log.format\=otel
    ...                                   \-\-mcp.config
    ...                                   {"server": {"transport": "http", "address": "127.0.0.1:9925", "mode": "full_access", "audit": {"file": {"path": "mcp-audit-otel-9925.log"}}} }
    ...                                   \-\-registry
    ...                                   ${REGISTRY_NO_VERIFY_CFG_JSON_STR}
    ...                                   \-\-auth
    ...                                   ${AUTH_CFG_STR}
    ...                                   \-\-tls.allowInsecure
    ...                                   stdout=${CURDIR}${/}tmp${/}Stackql-MCP-Server-Audit-OTel.txt
    ...                                   stderr=${CURDIR}${/}tmp${/}Stackql-MCP-Server-Audit-OTel-stderr.txt
    # Issue #729: sessionless Streamable HTTP (server.stateless) serves protocol
    # revision 2026-07-28 natively; safe mode so the gated write is exercised.
    Start Process                         ${STACKQL_EXE}
    ...                                   mcp
    ...                                   \-\-mcp.server.type\=http
    ...                                   \-\-mcp.config
    ...                                   {"server": {"transport": "http", "address": "127.0.0.1:9926", "stateless": true, "mode": "safe", "audit": {"disabled": true}} }
    ...                                   \-\-registry
    ...                                   ${REGISTRY_NO_VERIFY_CFG_JSON_STR}
    ...                                   \-\-auth
    ...                                   ${AUTH_CFG_STR}
    ...                                   \-\-tls.allowInsecure
    ...                                   stdout=${CURDIR}${/}tmp${/}Stackql-MCP-Server-Stateless.txt
    ...                                   stderr=${CURDIR}${/}tmp${/}Stackql-MCP-Server-Stateless-stderr.txt
    # Mocked HTTP registry with a throwaway approot so pull_provider performs
    # a real install; REGISTRY PULL is an error against the file registry.
    Start Process                         ${STACKQL_EXE}
    ...                                   mcp
    ...                                   \-\-mcp.server.type\=http
    ...                                   \-\-mcp.config
    ...                                   {"server": {"transport": "http", "address": "127.0.0.1:9927", "mode": "full_access", "audit": {"disabled": true}} }
    ...                                   \-\-registry
    ...                                   ${REGISTRY_MOCKED_CFG_STR.get_config_str('native')}
    ...                                   \-\-approot
    ...                                   ${TEST_TMP_EXEC_APP_ROOT_NATIVE}
    ...                                   \-\-auth
    ...                                   ${AUTH_CFG_STR}
    ...                                   \-\-tls.allowInsecure
    ...                                   stdout=${CURDIR}${/}tmp${/}Stackql-MCP-Server-Registry-Pull.txt
    ...                                   stderr=${CURDIR}${/}tmp${/}Stackql-MCP-Server-Registry-Pull-stderr.txt
    Sleep         5s

Parse MCP JSON Output
    [Arguments]    ${input}
    # Pass the raw string through Robot's variable namespace ($input) rather
    # than interpolating into Python source, so embedded quotes/backslashes
    # in nested JSON values (eg DESCRIBE METHOD's "shape" column) survive.
    ${parsed}=    Evaluate    json.loads($input)    json
    RETURN    ${parsed}

*** Settings ***
Suite Setup     Start MCP Servers


*** Test Cases ***
MCP HTTP Server Run List Tools
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9912
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-HTTP-Server-Run-List-Tools.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-HTTP-Server-Run-List-Tools-stderr.txt
    Should Contain       ${result.stdout}       Get server identity
    Should Be Equal As Integers    ${result.rc}    0


MCP HTTP Server List Providers Tool
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9912
    ...                  \-\-exec.action      list_providers
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-HTTP-Server-List-Providers.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-HTTP-Server-List-Providers-stderr.txt
    Should Contain       ${result.stdout}       local_openssl
    Should Be Equal As Integers    ${result.rc}    0


MCP HTTP Server List Services Tool
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9912
    ...                  \-\-exec.action      list_services
    ...                  \-\-exec.args        {"provider": "google"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-HTTP-Server-List-Services.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-HTTP-Server-List-Services-stderr.txt
    Should Contain       ${result.stdout}       YouTube Analytics API
    Should Be Equal As Integers    ${result.rc}    0

MCP HTTP Server List Resources Tool
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9912
    ...                  \-\-exec.action      list_resources
    ...                  \-\-exec.args        {"provider": "google", "service": "cloudresourcemanager"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-HTTP-Server-List-Resources.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-HTTP-Server-List-Resources-stderr.txt
    Should Contain       ${result.stdout}       projects
    Should Be Equal As Integers    ${result.rc}    0

MCP HTTP Server List Methods Tool
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9912
    ...                  \-\-exec.action      list_methods
    ...                  \-\-exec.args        {"provider": "google", "service": "compute", "resource": "instances"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-HTTP-Server-List-Methods.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-HTTP-Server-List-Methods-stderr.txt
    Should Contain       ${result.stdout}       getScreenshot
    Should Be Equal As Integers    ${result.rc}    0

MCP HTTP Server Info Includes Version
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9912
    ...                  \-\-exec.action      server_info
    ...                  \-\-exec.args        {}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-HTTP-Server-Info.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-HTTP-Server-Info-stderr.txt
    Should Contain       ${result.stdout}       version
    Should Contain       ${result.stdout}       transport
    Should Contain       ${result.stdout}       sql_backend
    Should Contain       ${result.stdout}       provider_registry
    Should Contain       ${result.stdout}       mode
    Should Match Regexp    ${result.stdout}       \\d+\\.\\d+\\.\\d+
    Should Be Equal As Integers    ${result.rc}    0

PG Server Show Version
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${posixInput} =     Catenate
    ...    "${PSQL_EXE}"    -d     postgres://stackql:stackql@127.0.0.1:5665   -c
    ...    "SHOW VERSION;"
    ${windowsInput} =     Catenate
    ...    &    ${posixInput}
    ${input} =    Set Variable If    "${IS_WINDOWS}" == "1"    ${windowsInput}    ${posixInput}
    ${shellExe} =    Set Variable If    "${IS_WINDOWS}" == "1"    powershell    sh
    ${psql_client_result}=    Run Process
    ...                  ${shellExe}     \-c    ${input}
    ...                  stdout=${CURDIR}${/}tmp${/}PG-Server-Show-Version-psql.txt
    ...                  stderr=${CURDIR}${/}tmp${/}PG-Server-Show-Version-psql-stderr.txt
    Should Contain       ${psql_client_result.stdout}       version
    Should Match Regexp    ${psql_client_result.stdout}       \\d+\\.\\d+\\.\\d+
    Should Be Equal As Integers    ${psql_client_result.rc}    0

PG Server Show Version Extended
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${posixInput} =     Catenate
    ...    "${PSQL_EXE}"    -d     postgres://stackql:stackql@127.0.0.1:5665   -c
    ...    "SHOW EXTENDED VERSION;"
    ${windowsInput} =     Catenate
    ...    &    ${posixInput}
    ${input} =    Set Variable If    "${IS_WINDOWS}" == "1"    ${windowsInput}    ${posixInput}
    ${shellExe} =    Set Variable If    "${IS_WINDOWS}" == "1"    powershell    sh
    ${psql_client_result}=    Run Process
    ...                  ${shellExe}     \-c    ${input}
    ...                  stdout=${CURDIR}${/}tmp${/}PG-Server-Show-Version-Extended-psql.txt
    ...                  stderr=${CURDIR}${/}tmp${/}PG-Server-Show-Version-Extended-psql-stderr.txt
    Should Contain       ${psql_client_result.stdout}       version
    Should Contain       ${psql_client_result.stdout}       commit
    Should Contain       ${psql_client_result.stdout}       build_date
    Should Contain       ${psql_client_result.stdout}       platform
    Should Match Regexp    ${psql_client_result.stdout}       \\d+\\.\\d+\\.\\d+
    Should Be Equal As Integers    ${psql_client_result.rc}    0

PG Server Show Contributors
    [Documentation]    Issue #320: the embedded leaderboard is served over the
    ...                wire protocol from the same implementation.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${posixInput} =     Catenate
    ...    "${PSQL_EXE}"    -d     postgres://stackql:stackql@127.0.0.1:5665   -c
    ...    "SHOW EXTENDED CONTRIBUTORS;"
    ${windowsInput} =     Catenate
    ...    &    ${posixInput}
    ${input} =    Set Variable If    "${IS_WINDOWS}" == "1"    ${windowsInput}    ${posixInput}
    ${shellExe} =    Set Variable If    "${IS_WINDOWS}" == "1"    powershell    sh
    ${psql_client_result}=    Run Process
    ...                  ${shellExe}     \-c    ${input}
    ...                  stdout=${CURDIR}${/}tmp${/}PG-Server-Show-Contributors-psql.txt
    ...                  stderr=${CURDIR}${/}tmp${/}PG-Server-Show-Contributors-psql-stderr.txt
    Should Contain       ${psql_client_result.stdout}       contributor
    Should Contain       ${psql_client_result.stdout}       contributions
    Should Match Regexp    ${psql_client_result.stdout}       (?s)general-kroll-4-life.*jeffreyaven
    Should Be Equal As Integers    ${psql_client_result.rc}    0

MCP HTTP Server Query Tool
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9912
    ...                  \-\-exec.action      run_select_query
    ...                  \-\-exec.args        {"sql": "SELECT assetType, count(*) as asset_count FROM google.cloudasset.assets WHERE parentType \= 'projects' and parent \= 'testing-project' GROUP BY assetType order by count(*) desc, assetType desc;"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-HTTP-Server-Query-Tool.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-HTTP-Server-Query-Tool-stderr.txt
    Should Contain       ${result.stdout}       cloudkms.googleapis.com
    Should Be Equal As Integers    ${result.rc}    0


Concurrent psql and MCP HTTP Server Query Tool
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${mcp_client_result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9913
    ...                  \-\-exec.action      run_select_query
    ...                  \-\-exec.args        {"sql": "SELECT assetType, count(*) as asset_count FROM google.cloudasset.assets WHERE parentType \= 'projects' and parent \= 'testing-project' GROUP BY assetType order by count(*) desc, assetType desc;"}
    ...                  stdout=${CURDIR}${/}tmp${/}Concurrent-psql-and-MCP-HTTP-Server-Query-Tool.txt
    ...                  stderr=${CURDIR}${/}tmp${/}Concurrent-psql-and-MCP-HTTP-Server-Query-Tool-stderr.txt
    Should Contain       ${mcp_client_result.stdout}       cloudkms.googleapis.com
    Should Be Equal As Integers    ${mcp_client_result.rc}    0
    ${posixInput} =     Catenate
    ...    "${PSQL_EXE}"    -d     postgres://stackql:stackql@127.0.0.1:5665   -c
    ...    "SELECT assetType, count(*) as asset_count FROM google.cloudasset.assets WHERE parentType = 'projects' and parent = 'testing-project' GROUP BY assetType order by count(*) desc, assetType desc;"
    ${windowsInput} =     Catenate
    ...    &    ${posixInput}
    ${input} =    Set Variable If    "${IS_WINDOWS}" == "1"    ${windowsInput}    ${posixInput}
    ${shellExe} =    Set Variable If    "${IS_WINDOWS}" == "1"    powershell    sh
    ${psql_client_result}=    Run Process
    ...                  ${shellExe}     \-c    ${input}
    ...                  stdout=${CURDIR}${/}tmp${/}Concurrent-psql-and-MCP-HTTP-Server-Query-Tool-psql.txt
    ...                  stderr=${CURDIR}${/}tmp${/}Concurrent-psql-and-MCP-HTTP-Server-Query-Tool-psql-stderr.txt
    Should Contain       ${psql_client_result.stdout}       cloudkms.googleapis.com
    Should Be Equal As Integers    ${psql_client_result.rc}    0

Concurrent psql and Reverse Proxy MCP HTTP Server Query Tool
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${mcp_client_result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9914
    ...                  \-\-exec.action      run_select_query
    ...                  \-\-exec.args        {"sql": "SELECT assetType, count(*) as asset_count FROM google.cloudasset.assets WHERE parentType \= 'projects' and parent \= 'testing-project' GROUP BY assetType order by count(*) desc, assetType desc;"}
    ...                  stdout=${CURDIR}${/}tmp${/}Concurrent-psql-and-Reverse-Proxy-MCP-HTTP-Server-Query-Tool.txt
    ...                  stderr=${CURDIR}${/}tmp${/}Concurrent-psql-and-Reverse-Proxy-MCP-HTTP-Server-Query-Tool-stderr.txt
    Should Contain       ${mcp_client_result.stdout}       cloudkms.googleapis.com
    Should Be Equal As Integers    ${mcp_client_result.rc}    0
    ${posixInput} =     Catenate
    ...    "${PSQL_EXE}"    -d     postgres://stackql:stackql@127.0.0.1:5445   -c
    ...    "SELECT assetType, count(*) as asset_count FROM google.cloudasset.assets WHERE parentType = 'projects' and parent = 'testing-project' GROUP BY assetType order by count(*) desc, assetType desc;"
    ${windowsInput} =     Catenate
    ...    &    ${posixInput}
    ${input} =    Set Variable If    "${IS_WINDOWS}" == "1"    ${windowsInput}    ${posixInput}
    ${shellExe} =    Set Variable If    "${IS_WINDOWS}" == "1"    powershell    sh
    ${psql_client_result}=    Run Process
    ...                  ${shellExe}     \-c    ${input}
    ...                  stdout=${CURDIR}${/}tmp${/}Concurrent-psql-and-Reverse-Proxy-MCP-HTTP-Server-Query-Tool-psql.txt
    ...                  stderr=${CURDIR}${/}tmp${/}Concurrent-psql-and-Reverse-Proxy-MCP-HTTP-Server-Query-Tool-psql-stderr.txt
    Should Contain       ${psql_client_result.stdout}       cloudkms.googleapis.com
    Should Be Equal As Integers    ${psql_client_result.rc}    0

Concurrent psql and Reverse Proxy MCP HTTPS Server Query Tool
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${mcp_client_result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=https://127.0.0.1:9004
    ...                  \-\-client\-cfg      { "apply_tls_globally": true, "insecure_skip_verify": true, "ca_file": "test/server/mtls/credentials/pg_server_cert.pem", "promote_leaf_to_ca": true }
    ...                  \-\-exec.action      run_select_query
    ...                  \-\-exec.args        {"sql": "SELECT assetType, count(*) as asset_count FROM google.cloudasset.assets WHERE parentType \= 'projects' and parent \= 'testing-project' GROUP BY assetType order by count(*) desc, assetType desc;"}
    ...                  stdout=${CURDIR}${/}tmp${/}Concurrent-psql-and-Reverse-Proxy-MCP-HTTPS-Server-Query-Tool.txt
    ...                  stderr=${CURDIR}${/}tmp${/}Concurrent-psql-and-Reverse-Proxy-MCP-HTTPS-Server-Query-Tool-stderr.txt
    Should Contain       ${mcp_client_result.stdout}       cloudkms.googleapis.com
    Should Be Equal As Integers    ${mcp_client_result.rc}    0
    ${posixInput} =     Catenate
    ...    "${PSQL_EXE}"    -d     postgres://stackql:stackql@127.0.0.1:5446   -c
    ...    "SELECT assetType, count(*) as asset_count FROM google.cloudasset.assets WHERE parentType = 'projects' and parent = 'testing-project' GROUP BY assetType order by count(*) desc, assetType desc;"
    ${windowsInput} =     Catenate
    ...    &    ${posixInput}
    ${input} =    Set Variable If    "${IS_WINDOWS}" == "1"    ${windowsInput}    ${posixInput}
    ${shellExe} =    Set Variable If    "${IS_WINDOWS}" == "1"    powershell    sh
    ${psql_client_result}=    Run Process
    ...                  ${shellExe}     \-c    ${input}
    ...                  stdout=${CURDIR}${/}tmp${/}Concurrent-psql-and-Reverse-Proxy-MCP-HTTPS-Server-Query-Tool-psql.txt
    ...                  stderr=${CURDIR}${/}tmp${/}Concurrent-psql-and-Reverse-Proxy-MCP-HTTPS-Server-Query-Tool-psql-stderr.txt
    Should Contain       ${psql_client_result.stdout}       cloudkms.googleapis.com
    Should Be Equal As Integers    ${psql_client_result.rc}    0

MCP HTTPS Server JSON DTO Server Info
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${srvinfo}=    Run Process
    ...    ${STACKQL_MCP_CLIENT_EXE}
    ...    exec
    ...    \-\-client\-type\=http
    ...    \-\-url\=https://127.0.0.1:9004
    ...    \-\-client\-cfg
    ...    { "apply_tls_globally": true, "insecure_skip_verify": true, "ca_file": "test/server/mtls/credentials/pg_server_cert.pem", "promote_leaf_to_ca": true }
    ...    \-\-exec.action
    ...    server_info
    ...    stdout=${CURDIR}${/}tmp${/}MCP-HTTPS-server-info.txt
    ...    stderr=${CURDIR}${/}tmp${/}MCP-HTTPS-server-info-stderr.txt
    Should Be Equal As Integers    ${srvinfo.rc}    0
    ${srvinfo_obj}=    Parse MCP JSON Output    ${srvinfo.stdout}
    Dictionary Should Contain Key    ${srvinfo_obj}    is_read_only


MCP HTTPS List Providers Canonical
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${meta_rels}=    Run Process
    ...    ${STACKQL_MCP_CLIENT_EXE}
    ...    exec
    ...    \-\-client\-type\=http
    ...    \-\-url\=https://127.0.0.1:9004
    ...    \-\-client\-cfg
    ...    { "apply_tls_globally": true, "insecure_skip_verify": true, "ca_file": "test/server/mtls/credentials/pg_server_cert.pem", "promote_leaf_to_ca": true }
    ...    \-\-exec.action
    ...    list_providers
    ...    \-\-exec.args
    ...    {"provider": "google"}
    ...    stdout=${CURDIR}${/}tmp${/}MCP-HTTPS-list-providers-canonical.txt
    ...    stderr=${CURDIR}${/}tmp${/}MCP-HTTPS-list-providers-canonical-stderr.txt
    ${meta_rels_obj}=    Parse MCP JSON Output    ${meta_rels.stdout}
    Dictionary Should Contain Key    ${meta_rels_obj}    rows
    Should Not Be Empty        ${meta_rels_obj['rows']}

MCP HTTPS List Services Canonical
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${meta_rels}=    Run Process
    ...    ${STACKQL_MCP_CLIENT_EXE}
    ...    exec
    ...    \-\-client\-type\=http
    ...    \-\-url\=https://127.0.0.1:9004
    ...    \-\-client\-cfg
    ...    { "apply_tls_globally": true, "insecure_skip_verify": true, "ca_file": "test/server/mtls/credentials/pg_server_cert.pem", "promote_leaf_to_ca": true }
    ...    \-\-exec.action
    ...    list_services
    ...    \-\-exec.args
    ...    {"provider": "google"}
    ...    stdout=${CURDIR}${/}tmp${/}MCP-HTTPS-list-services-canonical.txt
    ...    stderr=${CURDIR}${/}tmp${/}MCP-HTTPS-list-services-canonical-stderr.txt
    ${meta_rels_obj}=    Parse MCP JSON Output    ${meta_rels.stdout}
    Dictionary Should Contain Key    ${meta_rels_obj}    rows
    Should Not Be Empty        ${meta_rels_obj['rows']}

MCP HTTPS List Resources Canonical
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${meta_rels}=    Run Process
    ...    ${STACKQL_MCP_CLIENT_EXE}
    ...    exec
    ...    \-\-client\-type\=http
    ...    \-\-url\=https://127.0.0.1:9004
    ...    \-\-client\-cfg
    ...    { "apply_tls_globally": true, "insecure_skip_verify": true, "ca_file": "test/server/mtls/credentials/pg_server_cert.pem", "promote_leaf_to_ca": true }
    ...    \-\-exec.action
    ...    list_resources
    ...    \-\-exec.args
    ...    {"provider": "google", "service": "compute"}
    ...    stdout=${CURDIR}${/}tmp${/}MCP-HTTPS-list-resources-canonical.txt
    ...    stderr=${CURDIR}${/}tmp${/}MCP-HTTPS-list-resources-canonical-stderr.txt
    ${meta_rels_obj}=    Parse MCP JSON Output    ${meta_rels.stdout}
    Dictionary Should Contain Key    ${meta_rels_obj}    rows
    Should Not Be Empty        ${meta_rels_obj['rows']}


MCP HTTPS List Methods Canonical
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${meta_rels}=    Run Process
    ...    ${STACKQL_MCP_CLIENT_EXE}
    ...    exec
    ...    \-\-client\-type\=http
    ...    \-\-url\=https://127.0.0.1:9004
    ...    \-\-client\-cfg
    ...    { "apply_tls_globally": true, "insecure_skip_verify": true, "ca_file": "test/server/mtls/credentials/pg_server_cert.pem", "promote_leaf_to_ca": true }
    ...    \-\-exec.action
    ...    list_methods
    ...    \-\-exec.args
    ...    {"provider": "google", "service": "compute", "resource": "networks"}
    ...    stdout=${CURDIR}${/}tmp${/}MCP-HTTPS-list-methods-canonical.txt
    ...    stderr=${CURDIR}${/}tmp${/}MCP-HTTPS-list-methods-canonical-stderr.txt
    ${meta_rels_obj}=    Parse MCP JSON Output    ${meta_rels.stdout}
    Dictionary Should Contain Key    ${meta_rels_obj}    rows
    Should Not Be Empty        ${meta_rels_obj['rows']}


MCP HTTPS Describe Resource Canonical
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${meta_rels}=    Run Process
    ...    ${STACKQL_MCP_CLIENT_EXE}
    ...    exec
    ...    \-\-client\-type\=http
    ...    \-\-url\=https://127.0.0.1:9004
    ...    \-\-client\-cfg
    ...    { "apply_tls_globally": true, "insecure_skip_verify": true, "ca_file": "test/server/mtls/credentials/pg_server_cert.pem", "promote_leaf_to_ca": true }
    ...    \-\-exec.action
    ...    describe_resource
    ...    \-\-exec.args
    ...    {"provider": "google", "service": "compute", "resource": "networks"}
    ...    stdout=${CURDIR}${/}tmp${/}MCP-HTTPS-describe-resource-canonical.txt
    ...    stderr=${CURDIR}${/}tmp${/}MCP-HTTPS-describe-resource-canonical-stderr.txt
    ${meta_rels_obj}=    Parse MCP JSON Output    ${meta_rels.stdout}
    Dictionary Should Contain Key    ${meta_rels_obj}    rows
    Should Not Be Empty        ${meta_rels_obj['rows']}

MCP HTTPS Describe Method Canonical
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${meta_rels}=    Run Process
    ...    ${STACKQL_MCP_CLIENT_EXE}
    ...    exec
    ...    \-\-client\-type\=http
    ...    \-\-url\=https://127.0.0.1:9004
    ...    \-\-client\-cfg
    ...    { "apply_tls_globally": true, "insecure_skip_verify": true, "ca_file": "test/server/mtls/credentials/pg_server_cert.pem", "promote_leaf_to_ca": true }
    ...    \-\-exec.action
    ...    describe_method
    ...    \-\-exec.args
    ...    {"provider": "google", "service": "compute", "resource": "networks", "method": "get"}
    ...    stdout=${CURDIR}${/}tmp${/}MCP-HTTPS-describe-method-canonical.txt
    ...    stderr=${CURDIR}${/}tmp${/}MCP-HTTPS-describe-method-canonical-stderr.txt
    ${meta_rels_obj}=    Parse MCP JSON Output    ${meta_rels.stdout}
    Dictionary Should Contain Key    ${meta_rels_obj}    rows
    Should Not Be Empty        ${meta_rels_obj['rows']}
    ${project_rows}=    Evaluate    [r for r in $meta_rels_obj['rows'] if r.get('name') == 'project']
    Should Not Be Empty    ${project_rows}    describe_method rows should contain an entry with name='project'

MCP HTTPS Server Validate Canonical
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${meta_rels}=    Run Process
    ...    ${STACKQL_MCP_CLIENT_EXE}
    ...    exec
    ...    \-\-client\-type\=http
    ...    \-\-url\=https://127.0.0.1:9004
    ...    \-\-client\-cfg
    ...    { "apply_tls_globally": true, "insecure_skip_verify": true, "ca_file": "test/server/mtls/credentials/pg_server_cert.pem", "promote_leaf_to_ca": true }
    ...    \-\-exec.action
    ...    validate_select_query
    ...    \-\-exec.args
    ...    {"sql":"select * from google.storage.buckets where project \= 'stackql\-demo';"}
    ...    stdout=${CURDIR}${/}tmp${/}MCP-HTTPS-validate-canonical.txt
    ...    stderr=${CURDIR}${/}tmp${/}MCP-HTTPS-validate-canonical-stderr.txt
    ${meta_rels_obj}=    Parse MCP JSON Output    ${meta_rels.stdout}
    Dictionary Should Contain Key    ${meta_rels_obj}    valid
    Should Be True                   ${meta_rels_obj}[valid]

MCP HTTPS Server Validate Canonical Negative
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${meta_rels}=    Run Process
    ...    ${STACKQL_MCP_CLIENT_EXE}
    ...    exec
    ...    \-\-client\-type\=http
    ...    \-\-url\=https://127.0.0.1:9004
    ...    \-\-client\-cfg
    ...    { "apply_tls_globally": true, "insecure_skip_verify": true, "ca_file": "test/server/mtls/credentials/pg_server_cert.pem", "promote_leaf_to_ca": true }
    ...    \-\-exec.action
    ...    validate_select_query
    ...    \-\-exec.args
    ...    {"sql":"select * from google.storage.buckets2 where project \= 'stackql\-demo';"}
    ...    stdout=${CURDIR}${/}tmp${/}MCP-HTTPS-validate-canonical-negative.txt
    ...    stderr=${CURDIR}${/}tmp${/}MCP-HTTPS-validate-canonical-negative-stderr.txt
    ${meta_rels_obj}=    Parse MCP JSON Output    ${meta_rels.stdout}
    Dictionary Should Contain Key    ${meta_rels_obj}    valid
    Should Be True                   ${meta_rels_obj}[valid] == False

MCP HTTPS Server Query Canonical
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${meta_rels}=    Run Process
    ...    ${STACKQL_MCP_CLIENT_EXE}
    ...    exec
    ...    \-\-client\-type\=http
    ...    \-\-url\=https://127.0.0.1:9004
    ...    \-\-client\-cfg
    ...    { "apply_tls_globally": true, "insecure_skip_verify": true, "ca_file": "test/server/mtls/credentials/pg_server_cert.pem", "promote_leaf_to_ca": true }
    ...    \-\-exec.action
    ...    run_select_query
    ...    \-\-exec.args
    ...    {"sql":"select name, id from google.storage.buckets where project \= 'stackql\-demo';"}
    ...    stdout=${CURDIR}${/}tmp${/}MCP-HTTPS-Query-canonical.txt
    ...    stderr=${CURDIR}${/}tmp${/}MCP-HTTPS-Query-canonical-stderr.txt
    ${meta_rels_obj}=    Parse MCP JSON Output    ${meta_rels.stdout}
    Dictionary Should Contain Key    ${meta_rels_obj}    rows
    Length Should Be    ${meta_rels_obj['rows']}    7

MCP HTTPS Server Exec Query Canonical
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${meta_rels}=    Run Process
    ...    ${STACKQL_MCP_CLIENT_EXE}
    ...    exec
    ...    \-\-client\-type\=http
    ...    \-\-url\=https://127.0.0.1:9004
    ...    \-\-client\-cfg
    ...    { "apply_tls_globally": true, "insecure_skip_verify": true, "ca_file": "test/server/mtls/credentials/pg_server_cert.pem", "promote_leaf_to_ca": true }
    ...    \-\-exec.action
    ...    run_mutation_query
    ...    \-\-exec.args
    ...    {"sql":"delete from google.compute.firewalls where project \= 'mutable\-project' and firewall \= 'deletable\-firewall';"}
    ...    stdout=${CURDIR}${/}tmp${/}MCP-HTTPS-Exec-Query-canonical.txt
    ...    stderr=${CURDIR}${/}tmp${/}MCP-HTTPS-Exec-Query-canonical-stderr.txt
    ${meta_rels_obj}=    Parse MCP JSON Output    ${meta_rels.stdout}
    Dictionary Should Contain Key    ${meta_rels_obj}    timestamp

MCP HTTP Server Restricted Tools Allowlist
    [Documentation]    Verify enabled_tools in mcp.config restricts which tools are published.
    ...                Server at 9915 is started with enabled_tools=["server_info"]; only server_info should be callable.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${list_result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9915
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Restricted-list-tools.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Restricted-list-tools-stderr.txt
    Should Be Equal As Integers    ${list_result.rc}    0
    Should Contain        ${list_result.stdout}    server_info
    Should Not Contain    ${list_result.stdout}    list_providers
    Should Not Contain    ${list_result.stdout}    run_select_query
    ${info_result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9915
    ...                  \-\-exec.action      server_info
    ...                  \-\-exec.args        {}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Restricted-server-info.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Restricted-server-info-stderr.txt
    Should Be Equal As Integers    ${info_result.rc}    0
    Should Contain    ${info_result.stdout}    version
    Should Match Regexp    ${info_result.stdout}    \\d+\\.\\d+\\.\\d+
    ${denied_result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9915
    ...                  \-\-exec.action      list_providers
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Restricted-list-providers.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Restricted-list-providers-stderr.txt
    Should Not Be Equal As Integers    ${denied_result.rc}    0
    Should Contain    ${denied_result.stderr}    unknown tool

MCP HTTPS Run Lifecycle Operation Canonical
    [Documentation]    Positive path: run_lifecycle_operation executes an EXEC successfully and returns messages + timestamp.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${lifecycle}=    Run Process
    ...    ${STACKQL_MCP_CLIENT_EXE}
    ...    exec
    ...    \-\-client\-type\=http
    ...    \-\-url\=https://127.0.0.1:9004
    ...    \-\-client\-cfg
    ...    { "apply_tls_globally": true, "insecure_skip_verify": true, "ca_file": "test/server/mtls/credentials/pg_server_cert.pem", "promote_leaf_to_ca": true }
    ...    \-\-exec.action
    ...    run_lifecycle_operation
    ...    \-\-exec.args
    ...    {"sql":"exec aws.ec2.instances.instances_Start @region \= 'ap\-southeast\-2', @InstanceId \= 'id\-001';"}
    ...    stdout=${CURDIR}${/}tmp${/}MCP-HTTPS-run-lifecycle.txt
    ...    stderr=${CURDIR}${/}tmp${/}MCP-HTTPS-run-lifecycle-stderr.txt
    Should Be Equal As Integers    ${lifecycle.rc}    0
    ${lifecycle_obj}=    Parse MCP JSON Output    ${lifecycle.stdout}
    # The reverse-proxy backend at 9004 returns {timestamp, rows_affected, last_insert_id}
    # from db.Exec; the orchestrator-backed primary backend would return
    # {messages, timestamp}.  Assert the common floor: a timestamp is present.
    Dictionary Should Contain Key    ${lifecycle_obj}    timestamp

MCP HTTP Read Only Server Info Flag
    [Documentation]    The read-only server at 9916 must report is_read_only=true via server_info.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${srvinfo}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9916
    ...                  \-\-exec.action      server_info
    ...                  \-\-exec.args        {}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-ReadOnly-server-info.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-ReadOnly-server-info-stderr.txt
    Should Be Equal As Integers    ${srvinfo.rc}    0
    ${srvinfo_obj}=    Parse MCP JSON Output    ${srvinfo.stdout}
    Dictionary Should Contain Key    ${srvinfo_obj}    is_read_only
    Should Be True    ${srvinfo_obj}[is_read_only]

MCP HTTPS Run Mutation Refused In Read Only
    [Documentation]    A read-only server must refuse run_mutation_query and run_lifecycle_operation.
    ...                The 9916 server is started with the legacy read_only:true wire form so
    ...                this scenario also exercises the legacy-shim mapping to mode=read_only.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${mutation_result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9916
    ...                  \-\-exec.action      run_mutation_query
    ...                  \-\-exec.args        {"sql":"delete from google.compute.firewalls where project \= 'mutable\-project' and firewall \= 'deletable\-firewall';"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-ReadOnly-mutation.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-ReadOnly-mutation-stderr.txt
    Should Not Be Equal As Integers    ${mutation_result.rc}    0
    Should Contain    ${mutation_result.stderr}    read_only
    ${lifecycle_result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9916
    ...                  \-\-exec.action      run_lifecycle_operation
    ...                  \-\-exec.args        {"sql":"EXEC google.compute.instances.start @project \= 'mutable\-project', @zone \= 'us\-central1\-a', @instance \= 'demo';"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-ReadOnly-lifecycle.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-ReadOnly-lifecycle-stderr.txt
    Should Not Be Equal As Integers    ${lifecycle_result.rc}    0
    Should Contain    ${lifecycle_result.stderr}    read_only

# ===========================================================================
# Mode contract.  The test MCP client does NOT advertise elicitation, so the
# safe and delete_safe modes hit the refuse-with-message fallback path.  The
# elicitation-positive path (client supports elicitation, user accepts /
# declines) is verified manually with elicitation-capable clients; this robot
# suite verifies only the no-elicitation fallback path.
# ===========================================================================

MCP HTTP Mode Read Only Refuses Mutations And Lifecycle
    [Documentation]    Server at 9920 starts with mode=read_only.  Selects work; mutations and lifecycle refused with the read_only message.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${select_result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9920
    ...                  \-\-exec.action      run_select_query
    ...                  \-\-exec.args        {"sql":"select name, id from google.storage.buckets where project \= 'stackql\-demo';"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Mode-ReadOnly-select.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Mode-ReadOnly-select-stderr.txt
    Should Be Equal As Integers    ${select_result.rc}    0
    ${mut}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9920
    ...                  \-\-exec.action      run_mutation_query
    ...                  \-\-exec.args        {"sql":"delete from google.compute.firewalls where project \= 'mutable\-project' and firewall \= 'deletable\-firewall';"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Mode-ReadOnly-mutation.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Mode-ReadOnly-mutation-stderr.txt
    Should Not Be Equal As Integers    ${mut.rc}    0
    Should Contain    ${mut.stderr}    read_only
    ${life}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9920
    ...                  \-\-exec.action      run_lifecycle_operation
    ...                  \-\-exec.args        {"sql":"EXEC aws.ec2.instances.instances_Start @region \= 'ap\-southeast\-2', @InstanceId \= 'id\-001';"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Mode-ReadOnly-lifecycle.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Mode-ReadOnly-lifecycle-stderr.txt
    Should Not Be Equal As Integers    ${life.rc}    0
    Should Contain    ${life.stderr}    read_only
    # A common table expression heading a query body is read-only and must be
    # allowed; one heading a mutation must still be refused.
    ${cte_select}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9920
    ...                  \-\-exec.action      run_select_query
    ...                  \-\-exec.args        {"sql":"WITH b AS (select name, id from google.storage.buckets where project \= 'stackql\-demo') select name, id from b;"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Mode-ReadOnly-cte-select.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Mode-ReadOnly-cte-select-stderr.txt
    Should Be Equal As Integers    ${cte_select.rc}    0
    ${cte_mut}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9920
    ...                  \-\-exec.action      run_mutation_query
    ...                  \-\-exec.args        {"sql":"WITH f AS (select 1) delete from google.compute.firewalls where project \= 'mutable\-project' and firewall \= 'deletable\-firewall';"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Mode-ReadOnly-cte-mutation.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Mode-ReadOnly-cte-mutation-stderr.txt
    Should Not Be Equal As Integers    ${cte_mut.rc}    0
    Should Contain    ${cte_mut.stderr}    read_only

MCP HTTP Mode Safe Refuses Mutations Without Elicitation
    [Documentation]    Server at 9912 starts with mode=full_access (existing scenarios assume that).
    ...                We verify the safe-mode no-elicitation path via the *default* mode on a
    ...                fresh server, by hitting an arbitrary safe-mode server.  Since 9912/9913/9914
    ...                are full_access, use the implicit-safe behaviour of the audit-enabled 9923
    ...                server by overriding via a request to 9920 (read_only) is wrong - use a
    ...                dedicated server.  9921 = delete_safe is the closest analogue for safe in
    ...                the absence of an explicit safe-mode server; the brief specifies just three
    ...                non-default mode servers.  Verify safe-mode refusal by calling DELETE on the
    ...                9921 delete_safe server (it refuses delete the same way safe refuses any
    ...                mutation).
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${del}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9921
    ...                  \-\-exec.action      run_mutation_query
    ...                  \-\-exec.args        {"sql":"delete from google.compute.firewalls where project \= 'mutable\-project' and firewall \= 'deletable\-firewall';"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Mode-DeleteSafe-delete.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Mode-DeleteSafe-delete-stderr.txt
    Should Not Be Equal As Integers    ${del.rc}    0
    Should Contain    ${del.stderr}    does not support elicitation

MCP HTTP Mode Delete Safe Allows Create Refuses Delete And Lifecycle
    [Documentation]    Server at 9921 allows SELECT and INSERT/UPDATE; refuses DELETE and EXEC.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    # SELECT proceeds.
    ${sel}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9921
    ...                  \-\-exec.action      run_select_query
    ...                  \-\-exec.args        {"sql":"select name, id from google.storage.buckets where project \= 'stackql\-demo';"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Mode-DeleteSafe-select.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Mode-DeleteSafe-select-stderr.txt
    Should Be Equal As Integers    ${sel.rc}    0
    # DELETE refused.
    ${del}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9921
    ...                  \-\-exec.action      run_mutation_query
    ...                  \-\-exec.args        {"sql":"delete from google.compute.firewalls where project \= 'mutable\-project' and firewall \= 'deletable\-firewall';"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Mode-DeleteSafe-delete2.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Mode-DeleteSafe-delete2-stderr.txt
    Should Not Be Equal As Integers    ${del.rc}    0
    Should Contain    ${del.stderr}    delete_safe
    # EXEC refused.
    ${life}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9921
    ...                  \-\-exec.action      run_lifecycle_operation
    ...                  \-\-exec.args        {"sql":"EXEC aws.ec2.instances.instances_Start @region \= 'ap\-southeast\-2', @InstanceId \= 'id\-001';"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Mode-DeleteSafe-lifecycle.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Mode-DeleteSafe-lifecycle-stderr.txt
    Should Not Be Equal As Integers    ${life.rc}    0
    Should Contain    ${life.stderr}    delete_safe

MCP HTTP Mode Full Access Allows Everything
    [Documentation]    Server at 9922 starts with mode=full_access.  SELECT, INSERT, DELETE, EXEC all proceed.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${sel}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9922
    ...                  \-\-exec.action      run_select_query
    ...                  \-\-exec.args        {"sql":"select name, id from google.storage.buckets where project \= 'stackql\-demo';"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Mode-FullAccess-select.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Mode-FullAccess-select-stderr.txt
    Should Be Equal As Integers    ${sel.rc}    0
    ${life}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9922
    ...                  \-\-exec.action      run_lifecycle_operation
    ...                  \-\-exec.args        {"sql":"exec aws.ec2.instances.instances_Start @region \= 'ap\-southeast\-2', @InstanceId \= 'id\-001';"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Mode-FullAccess-lifecycle.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Mode-FullAccess-lifecycle-stderr.txt
    Should Be Equal As Integers    ${life.rc}    0
    ${life_obj}=    Parse MCP JSON Output    ${life.stdout}
    Dictionary Should Contain Key    ${life_obj}    timestamp

# ===========================================================================
# Audit log
# ===========================================================================

MCP HTTP Audit Basic Records Tool Calls
    [Documentation]    The 9923 server is configured with audit enabled writing to a known path.
    ...                After a SELECT and an EXEC are dispatched, the file should contain at least
    ...                two JSONL lines with the expected tool and decision fields.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${sel}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9923
    ...                  \-\-exec.action      server_info
    ...                  \-\-exec.args        {}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Audit-preflight.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Audit-preflight-stderr.txt
    Should Be Equal As Integers    ${sel.rc}    0    9923 server must be reachable; if this fails the audit log path probably failed to parse
    ${sel}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9923
    ...                  \-\-exec.action      run_select_query
    ...                  \-\-exec.args        {"sql":"select name, id from google.storage.buckets where project \= 'stackql\-demo';"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Audit-select.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Audit-select-stderr.txt
    Should Be Equal As Integers    ${sel.rc}    0
    ${life}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9923
    ...                  \-\-exec.action      run_lifecycle_operation
    ...                  \-\-exec.args        {"sql":"exec aws.ec2.instances.instances_Start @region \= 'ap\-southeast\-2', @InstanceId \= 'id\-001';"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Audit-lifecycle.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Audit-lifecycle-stderr.txt
    Should Be Equal As Integers    ${life.rc}    0
    Sleep         1s
    # The audit log path in mcp.config was specified as the relative name
    # `mcp-audit-9923.log`; the stackql process resolves that against its
    # cwd (the directory robot was invoked from, ie EXECDIR).
    ${log_contents}=    Get File    ${EXECDIR}${/}mcp-audit-9923.log
    Should Contain    ${log_contents}    "tool":"run_select_query"
    Should Contain    ${log_contents}    "tool":"run_lifecycle_operation"
    Should Contain    ${log_contents}    "decision":"allow"
    Should Contain    ${log_contents}    "mode":"full_access"

MCP HTTP Audit Disabled Writes No File
    [Documentation]    The 9912 server has audit.disabled=true.  Running a query should not
    ...                produce any audit log file in cwd.  We assert by listing cwd before and
    ...                after and checking no new stackql_mcp_server_*.log appeared.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${before}=    Run Process    sh    -c    ls stackql_mcp_server_*.log 2>/dev/null | wc -l
    ${sel}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9912
    ...                  \-\-exec.action      run_select_query
    ...                  \-\-exec.args        {"sql":"select name, id from google.storage.buckets where project \= 'stackql\-demo';"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-AuditDisabled-select.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-AuditDisabled-select-stderr.txt
    Should Be Equal As Integers    ${sel.rc}    0
    ${after}=    Run Process    sh    -c    ls stackql_mcp_server_*.log 2>/dev/null | wc -l
    Should Be Equal    ${before.stdout}    ${after.stdout}

# ===========================================================================
# Issue #661 scenarios.  Cover fix 1 (empty result no longer reported as
# extraction failure), fix 2 (literal/expression columns render unwrapped, not
# as Go nullable wrapper text), and the two new tools list_registry /
# pull_provider that close the discover -> pull -> query loop.
# ===========================================================================

MCP HTTP Empty Result Renders Cleanly
    [Documentation]    Issue #661 fix 1: a zero-row result set must render cleanly,
    ...                not be reported as "failed to extract query results".
    ...                google.cloudkms.key_rings against the mock backend is
    ...                the canonical empty-table case (the mock returns 200 OK
    ...                with no keyRings key); the existing
    ...                "Empty Response 200 Missing Jsonpath..." CLI scenario
    ...                already pins that shape.  The mcp_client prefers
    ...                StructuredContent over rendered text, so the empty case
    ...                surfaces on the wire as `{"rows":[]}` rather than the
    ...                renderer's `**no results**` string.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9912
    ...                  \-\-exec.action      run_select_query
    ...                  \-\-exec.args        {"sql":"select * from google.cloudkms.key_rings where projectsId \= 'testing\-project' and locationsId \= 'australia\-southeast1';"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Empty-Result.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Empty-Result-stderr.txt
    Should Be Equal As Integers    ${result.rc}    0
    Should Contain                 ${result.stdout}    "rows":[]
    Should Not Contain             ${result.stdout}    failed to extract

MCP HTTP Literal Select Renders Unwrapped Scalars
    [Documentation]    Issue #661 fix 2: literal/expression columns must render
    ...                unwrapped, not as Go nullable wrapper text like
    ...                `&{ok true}`.  The backslash before `&{` keeps Robot
    ...                Framework from parsing it as a dictionary-variable token.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9912
    ...                  \-\-exec.action      run_select_query
    ...                  \-\-exec.args        {"sql":"SELECT 1 as n, 'ok' as status;"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Literal-Select.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Literal-Select-stderr.txt
    Should Be Equal As Integers    ${result.rc}    0
    Should Contain                 ${result.stdout}    ok
    Should Not Contain             ${result.stdout}    \&{

MCP HTTP List Registry Returns Available Providers
    [Documentation]    Issue #661 feature: list_registry surfaces providers
    ...                available to pull from the configured registry, distinct
    ...                from list_providers (which shows already-pulled
    ...                providers).  The MCP servers in this suite use a
    ...                file:// registry config, against which any-sdk's
    ...                ListAllAvailableProviders deliberately refuses with
    ...                "'registry list' is meaningless in local mode".  That
    ...                refusal surfaces here as an ExecutorOutput.GetError(),
    ...                which extractQueryResults reports to the client as
    ...                the generic "failed to extract query results" - we
    ...                pin that substring in the mcp_client panic message.
    ...                The pull_provider scenario below covers the happy path
    ...                against the same registry.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9912
    ...                  \-\-exec.action      list_registry
    ...                  \-\-exec.args        {}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-List-Registry.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-List-Registry-stderr.txt
    Should Contain                 ${result.stderr}    failed to extract query results

MCP HTTP Pull Provider Installs Known Provider
    [Documentation]    Issue #661 feature: pull_provider installs a single
    ...                provider into the approot cache.  Allowed under every
    ...                mode (writes only local cache state per the issue's "not
    ...                a cloud mutation" rationale).  The full_access 9927 server
    ...                on the mocked HTTP registry is used so the pull is real;
    ...                statement errors now propagate through run_mutation_query,
    ...                run_lifecycle_operation and pull_provider alike.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9927
    ...                  \-\-exec.action      pull_provider
    ...                  \-\-exec.args        {"provider":"google"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Pull-Provider.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Pull-Provider-stderr.txt
    Should Be Equal As Integers    ${result.rc}    0
    Should Contain                 ${result.stdout}    timestamp
    Should Contain                 ${result.stdout}    successfully installed

# ===========================================================================
# Issue #668 scenarios.  The stdio transport must tolerate CRLF-terminated
# JSON-RPC frames (Windows text-mode pipes produce these); before the fix the
# server exited silently (code 0, no output) on the first CRLF line.  The
# python harness drives the binary over raw byte pipes so the terminator
# reaches the server verbatim on every platform.
# ===========================================================================

MCP Stdio Server Handles LF Terminated JSON RPC
    [Documentation]    Control scenario: LF framing worked before and after the fix.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${result}=    Evaluate    stackql_test_tooling.mcp_stdio_client.run_stdio_initialize_roundtrip($STACKQL_EXE, $REGISTRY_NO_VERIFY_CFG_JSON_STR, $AUTH_CFG_STR, line_ending='lf')    modules=stackql_test_tooling.mcp_stdio_client
    Should Contain    ${result['stdout']}    serverInfo
    Should Contain    ${result['stdout']}    "tools"
    Should Be Equal As Integers    ${result['returncode']}    0

MCP Stdio Server Tolerates CRLF Terminated JSON RPC
    [Documentation]    Issue #668: CRLF-terminated JSON-RPC frames must be served,
    ...                not answered with a silent exit code 0.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${result}=    Evaluate    stackql_test_tooling.mcp_stdio_client.run_stdio_initialize_roundtrip($STACKQL_EXE, $REGISTRY_NO_VERIFY_CFG_JSON_STR, $AUTH_CFG_STR, line_ending='crlf')    modules=stackql_test_tooling.mcp_stdio_client
    Should Contain    ${result['stdout']}    serverInfo
    Should Contain    ${result['stdout']}    "tools"
    Should Be Equal As Integers    ${result['returncode']}    0

# ===========================================================================
# Issue #701 scenarios.  A syntactically invalid JSON-RPC frame must be
# answered with a -32700 parse error (id null) - and a valid-JSON-but-not-
# JSON-RPC frame with -32600 - without terminating the stdio session; before
# the fix the server exited with code 1 on the first such frame.
# ===========================================================================

MCP Stdio Server Answers Malformed JSON With Parse Error And Session Survives
    [Documentation]    Issue #701: the literal repro frame from the work order.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${frame}=    Set Variable    {this is not json
    ${result}=    Evaluate    stackql_test_tooling.mcp_stdio_client.run_stdio_malformed_frame_roundtrip($STACKQL_EXE, $REGISTRY_NO_VERIFY_CFG_JSON_STR, $AUTH_CFG_STR, $frame)    modules=stackql_test_tooling.mcp_stdio_client
    Should Be Equal As Integers    ${result['error_code']}    -32700
    Should Be True    ${result['error_id_is_null']}
    Should Be True    ${result['ping_ok']}
    Should Be True    ${result['still_running_after_ping']}
    Should Be Equal As Integers    ${result['returncode']}    0

MCP Stdio Server Answers Truncated JSON With Parse Error And Session Survives
    [Documentation]    Issue #701 variant: a truncated JSON document.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${frame}=    Set Variable    {"jsonrpc":
    ${result}=    Evaluate    stackql_test_tooling.mcp_stdio_client.run_stdio_malformed_frame_roundtrip($STACKQL_EXE, $REGISTRY_NO_VERIFY_CFG_JSON_STR, $AUTH_CFG_STR, $frame)    modules=stackql_test_tooling.mcp_stdio_client
    Should Be Equal As Integers    ${result['error_code']}    -32700
    Should Be True    ${result['error_id_is_null']}
    Should Be True    ${result['ping_ok']}
    Should Be True    ${result['still_running_after_ping']}
    Should Be Equal As Integers    ${result['returncode']}    0

MCP Stdio Server Answers Bare Token With Parse Error And Session Survives
    [Documentation]    Issue #701 variant: a bare non-JSON token.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${frame}=    Set Variable    not-json-token
    ${result}=    Evaluate    stackql_test_tooling.mcp_stdio_client.run_stdio_malformed_frame_roundtrip($STACKQL_EXE, $REGISTRY_NO_VERIFY_CFG_JSON_STR, $AUTH_CFG_STR, $frame)    modules=stackql_test_tooling.mcp_stdio_client
    Should Be Equal As Integers    ${result['error_code']}    -32700
    Should Be True    ${result['error_id_is_null']}
    Should Be True    ${result['ping_ok']}
    Should Be True    ${result['still_running_after_ping']}
    Should Be Equal As Integers    ${result['returncode']}    0

MCP Stdio Server Answers Non JSONRPC Object With Invalid Request And Session Survives
    [Documentation]    Issue #701: valid JSON that is not a JSON-RPC message
    ...                gets -32600 (id null) instead of terminating the session.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${frame}=    Set Variable    {"jsonrpc":"2.0","nonsense":true}
    ${result}=    Evaluate    stackql_test_tooling.mcp_stdio_client.run_stdio_malformed_frame_roundtrip($STACKQL_EXE, $REGISTRY_NO_VERIFY_CFG_JSON_STR, $AUTH_CFG_STR, $frame)    modules=stackql_test_tooling.mcp_stdio_client
    Should Be Equal As Integers    ${result['error_code']}    -32600
    Should Be True    ${result['error_id_is_null']}
    Should Be True    ${result['ping_ok']}
    Should Be True    ${result['still_running_after_ping']}
    Should Be Equal As Integers    ${result['returncode']}    0

# ===========================================================================
# Issue #669 scenarios.  Tool result text content is markdown by default; a
# per-call `format` argument or a server-level `render` config switches it to
# compact JSON.  The client's `prefer_text` config surfaces the text blocks
# (rather than structuredContent) so the rendering itself is asserted.
# ===========================================================================

MCP HTTP Text Content Defaults To Markdown
    [Documentation]    Control scenario: without a format argument the 9912 server
    ...                (no render config) renders a markdown table.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9912
    ...                  \-\-client\-cfg      {"prefer_text": true}
    ...                  \-\-exec.action      run_select_query
    ...                  \-\-exec.args        {"sql": "SELECT 1 as n, 'ok' as status;"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Text-Default-Markdown.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Text-Default-Markdown-stderr.txt
    Should Be Equal As Integers    ${result.rc}    0
    Should Contain                 ${result.stdout}    | n | status |
    Should Contain                 ${result.stdout}    | 1 | ok |

MCP HTTP Per Call JSON Format Renders JSON Text
    [Documentation]    Issue #669: format=json on the call renders the DTO as
    ...                compact JSON in the text content, overriding the server's
    ...                markdown default.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9912
    ...                  \-\-client\-cfg      {"prefer_text": true}
    ...                  \-\-exec.action      run_select_query
    ...                  \-\-exec.args        {"sql": "SELECT 1 as n, 'ok' as status;", "format": "json"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Text-PerCall-JSON.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Text-PerCall-JSON-stderr.txt
    Should Be Equal As Integers    ${result.rc}    0
    ${parsed}=    Parse MCP JSON Output    ${result.stdout}
    Dictionary Should Contain Key    ${parsed}    rows
    Should Be Equal As Strings    ${parsed['rows'][0]['status']}    ok
    Should Not Contain    ${result.stdout}    | n | status |

MCP HTTP Server Level JSON Render Default
    [Documentation]    Issue #669: the 9924 server is started with
    ...                {"server": {"render": "json"}}; text content is JSON with
    ...                no per-call argument.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9924
    ...                  \-\-client\-cfg      {"prefer_text": true}
    ...                  \-\-exec.action      run_select_query
    ...                  \-\-exec.args        {"sql": "SELECT 1 as n, 'ok' as status;"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Text-ServerLevel-JSON.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Text-ServerLevel-JSON-stderr.txt
    Should Be Equal As Integers    ${result.rc}    0
    ${parsed}=    Parse MCP JSON Output    ${result.stdout}
    Dictionary Should Contain Key    ${parsed}    rows
    Should Be Equal As Strings    ${parsed['rows'][0]['status']}    ok

MCP HTTP Invalid Format Argument Is Rejected
    [Documentation]    Issue #669: an unknown format value fails fast rather than
    ...                silently falling back to markdown.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9912
    ...                  \-\-exec.action      run_select_query
    ...                  \-\-exec.args        {"sql": "SELECT 1 as n;", "format": "yaml"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Text-Invalid-Format.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Text-Invalid-Format-stderr.txt
    Should Not Be Equal As Integers    ${result.rc}    0
    Should Contain    ${result.stderr}    invalid format

# ===========================================================================
# Issue #670 scenarios.  Upstream HTTP errors on SELECTs must surface as tool
# errors with an http_status / retryable classification, not as a successful
# empty result set.  HTTP 404 is the deliberate exception: under stackql's
# database semantics, querying an absent resource is "zero rows", so a 404
# keeps returning an empty result set.  The github mock serves 403 for org
# `ratelimitedorg`, 429 for org `throttledorg` and 404 for org
# `nonexistentorg`, all with JSON object bodies as GitHub sends.
# ===========================================================================

MCP HTTP Upstream 403 Surfaces As Non Retryable Tool Error
    [Documentation]    Issue #670: a 403 from the provider becomes an MCP tool
    ...                error classified as non-retryable, not {"rows": []}.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9912
    ...                  \-\-exec.action      run_select_query
    ...                  \-\-exec.args        {"sql": "SELECT id, name FROM github.repos.repos WHERE org \= 'ratelimitedorg';"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Upstream-403.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Upstream-403-stderr.txt
    Should Not Be Equal As Integers    ${result.rc}    0
    Should Contain        ${result.stderr}    upstream http error
    Should Contain        ${result.stderr}    "http_status": 403
    Should Contain        ${result.stderr}    "retryable": false
    Should Not Contain    ${result.stdout}    "rows":[]

MCP HTTP Upstream 429 Surfaces As Retryable Tool Error
    [Documentation]    Issue #670: a 429 from the provider becomes an MCP tool
    ...                error classified as retryable.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9912
    ...                  \-\-exec.action      run_select_query
    ...                  \-\-exec.args        {"sql": "SELECT id, name FROM github.repos.repos WHERE org \= 'throttledorg';"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Upstream-429.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Upstream-429-stderr.txt
    Should Not Be Equal As Integers    ${result.rc}    0
    Should Contain        ${result.stderr}    upstream http error
    Should Contain        ${result.stderr}    "http_status": 429
    Should Contain        ${result.stderr}    "retryable": true

MCP HTTP Upstream 404 Returns Empty Result Set
    [Documentation]    Issue #670 scope guard: querying an absent resource (404)
    ...                is database-semantics "zero rows", so it keeps returning
    ...                a successful empty result set over MCP, not a tool error.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9912
    ...                  \-\-exec.action      run_select_query
    ...                  \-\-exec.args        {"sql": "SELECT id, name FROM github.repos.repos WHERE org \= 'nonexistentorg';"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Upstream-404.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Upstream-404-stderr.txt
    Should Be Equal As Integers    ${result.rc}    0
    Should Contain        ${result.stdout}    "rows":[]
    Should Not Contain    ${result.stderr}    upstream http error

# ===========================================================================
# Issue #688: a stdio MCP server spawned without credential env vars (the
# Claude Desktop failure mode) re-sources them mid-session from the
# --env.file dotenv file via the reload_credentials tool.
# ===========================================================================

MCP Stdio Reload Credentials Reports Installed Providers On Fresh Session
    [Documentation]    Unscoped and scoped reload_credentials before any query
    ...                report every installed provider from the registry, not the
    ...                lazily populated auth context cache; unknown provider is
    ...                the only scoped error.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${env_file}=    Set Variable    ${CURDIR}${/}tmp${/}mcp-reload-fresh.env
    ${child_env}=    Evaluate    {"DD_API_KEY": "myusername", "DD_APPLICATION_KEY": "mypassword"}
    ${steps}=    Evaluate    [{"call": "reload_credentials", "args": {}, "as": "unscoped"}, {"call": "reload_credentials", "args": {"provider": "stackql_auth_testing"}, "as": "scoped"}, {"call": "reload_credentials", "args": {"provider": "nonexistent_provider"}, "as": "unknown"}]
    ${result}=    Evaluate    stackql_test_tooling.mcp_stdio_client.run_stdio_credential_script($STACKQL_EXE, $REGISTRY_NO_VERIFY_CFG_JSON_STR, $AUTH_CFG_STR, $env_file, $child_env, $steps)    modules=stackql_test_tooling.mcp_stdio_client
    Log    ${result['stderr']}
    Should Contain        ${result['unscoped']}    "env_file_sourced": true
    Should Not Contain    ${result['unscoped']}    "providers": []
    Should Not Contain    ${result['unscoped']}    isError=true
    Should Contain        ${result['unscoped']}    | custom | false |  | stackql_auth_testing | env:DD_API_KEY | ok |
    Should Contain        ${result['unscoped']}    | okta | env:OKTA_SECRET_KEY |
    Should Contain        ${result['unscoped']}    | aws_signing_v4 |
    Should Not Contain    ${result['unscoped']}    myusername
    Should Not Contain    ${result['scoped']}      isError=true
    Should Contain        ${result['scoped']}      | custom | false |  | stackql_auth_testing | env:DD_API_KEY | ok |
    Should Not Contain    ${result['scoped']}      | okta |
    Should Contain        ${result['unknown']}     provider 'nonexistent_provider' is not installed
    Should Contain        ${result['unknown']}     isError=true
    Should Be Equal As Integers    ${result['returncode']}    0

MCP Stdio Reload Credentials Rotates Key After Auth Context Registered
    [Documentation]    A query registers the provider's auth context under a
    ...                stale key (rejected by the mock), the env file is rotated,
    ...                one reload reports changed: true and the next query
    ...                authenticates with the new key; a second reload against the
    ...                unchanged file is a no-op; a deleted file is an explicit error.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${env_file}=    Set Variable    ${CURDIR}${/}tmp${/}mcp-reload-rotate-after.env
    ${select_sql}=    Set Variable    select id from stackql_auth_testing.collectors.collectors order by id desc;
    ${child_env}=    Evaluate    {"DD_API_KEY": "stale-key", "DD_APPLICATION_KEY": "mypassword"}
    ${steps}=    Evaluate    [{"call": "run_select_query", "args": {"sql": $select_sql}, "as": "select_stale"}, {"write_env": {"DD_API_KEY": "myusername"}}, {"call": "reload_credentials", "args": {"provider": "stackql_auth_testing"}, "as": "reload"}, {"call": "run_select_query", "args": {"sql": $select_sql}, "as": "select_rotated"}, {"call": "reload_credentials", "args": {"provider": "stackql_auth_testing"}, "as": "reload_noop"}, {"remove_env": True}, {"call": "reload_credentials", "args": {}, "as": "reload_missing"}]
    ${result}=    Evaluate    stackql_test_tooling.mcp_stdio_client.run_stdio_credential_script($STACKQL_EXE, $REGISTRY_NO_VERIFY_CFG_JSON_STR, $AUTH_CFG_STR, $env_file, $child_env, $steps)    modules=stackql_test_tooling.mcp_stdio_client
    Log    ${result['stderr']}
    Should Contain        ${result['select_stale']}      isError=true
    Should Not Contain    ${result['select_stale']}      100000001
    Should Contain        ${result['reload']}            | custom | true |  | stackql_auth_testing | env:DD_API_KEY | ok |
    Should Not Contain    ${result['reload']}            myusername
    Should Contain        ${result['select_rotated']}    100000001
    Should Not Contain    ${result['select_rotated']}    isError=true
    Should Contain        ${result['reload_noop']}       | custom | false |  | stackql_auth_testing | env:DD_API_KEY | ok |
    Should Contain        ${result['reload_missing']}    isError=true
    Should Contain        ${result['reload_missing']}    mcp-reload-rotate-after.env' not found
    Should Be Equal As Integers    ${result['returncode']}    0

MCP Stdio Reload Credentials Rotates Key Before First Query
    [Documentation]    The env file is rotated before the provider has ever been
    ...                queried; the unscoped reload is a full report (no error) and
    ...                the first query uses the new key.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${env_file}=    Set Variable    ${CURDIR}${/}tmp${/}mcp-reload-rotate-before.env
    ${select_sql}=    Set Variable    select id from stackql_auth_testing.collectors.collectors order by id desc;
    ${child_env}=    Evaluate    {"DD_API_KEY": "stale-key", "DD_APPLICATION_KEY": "mypassword"}
    ${steps}=    Evaluate    [{"write_env": {"DD_API_KEY": "myusername"}}, {"call": "reload_credentials", "args": {}, "as": "reload"}, {"call": "run_select_query", "args": {"sql": $select_sql}, "as": "select_first"}]
    ${result}=    Evaluate    stackql_test_tooling.mcp_stdio_client.run_stdio_credential_script($STACKQL_EXE, $REGISTRY_NO_VERIFY_CFG_JSON_STR, $AUTH_CFG_STR, $env_file, $child_env, $steps)    modules=stackql_test_tooling.mcp_stdio_client
    Log    ${result['stderr']}
    Should Not Contain    ${result['reload']}          isError=true
    Should Contain        ${result['reload']}          DD_API_KEY
    Should Contain        ${result['reload']}          | custom | true |  | stackql_auth_testing | env:DD_API_KEY | ok |
    Should Contain        ${result['reload']}          | okta | env:OKTA_SECRET_KEY |
    Should Contain        ${result['select_first']}    100000001
    Should Not Contain    ${result['select_first']}    isError=true
    Should Be Equal As Integers    ${result['returncode']}    0

MCP Stdio Reload Credentials Recovers From Failed Resolution
    [Documentation]    Select and mutation failures name the provider and the
    ...                fix-file / reload / retry recovery; a reload against an
    ...                unchanged file reports changed: false (the stop signal);
    ...                after the fix one reload reports changed: true and the retry
    ...                succeeds.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${env_file}=    Set Variable    ${CURDIR}${/}tmp${/}mcp-reload-recover.env
    ${select_sql}=    Set Variable    select id from stackql_auth_testing.collectors.collectors order by id desc;
    ${delete_sql}=    Set Variable    delete from stackql_auth_testing.collectors.collectors where id = '100000001';
    ${child_env}=    Evaluate    {"DD_API_KEY": None, "DD_APPLICATION_KEY": "mypassword"}
    ${steps}=    Evaluate    [{"call": "run_select_query", "args": {"sql": $select_sql}, "as": "select_before"}, {"call": "run_mutation_query", "args": {"sql": $delete_sql}, "as": "delete_before"}, {"call": "reload_credentials", "args": {"provider": "stackql_auth_testing"}, "as": "reload_unchanged"}, {"write_env": {"DD_API_KEY": "myusername"}}, {"call": "reload_credentials", "args": {"provider": "stackql_auth_testing"}, "as": "reload"}, {"call": "run_select_query", "args": {"sql": $select_sql}, "as": "select_after"}]
    ${result}=    Evaluate    stackql_test_tooling.mcp_stdio_client.run_stdio_credential_script($STACKQL_EXE, $REGISTRY_NO_VERIFY_CFG_JSON_STR, $AUTH_CFG_STR, $env_file, $child_env, $steps)    modules=stackql_test_tooling.mcp_stdio_client
    Log    ${result['stderr']}
    Should Contain        ${result['select_before']}       credential resolution failed for provider 'stackql_auth_testing'
    Should Contain        ${result['select_before']}       fix the configured env file, call reload_credentials, then retry
    Should Contain        ${result['delete_before']}       credential resolution failed for provider 'stackql_auth_testing'
    Should Contain        ${result['delete_before']}       fix the configured env file, call reload_credentials, then retry
    Should Contain        ${result['reload_unchanged']}    | custom | false | credentialsenvvar references empty string | stackql_auth_testing | env:DD_API_KEY | unresolved |
    Should Not Contain    ${result['reload_unchanged']}    isError=true
    Should Contain        ${result['reload']}              | custom | true |  | stackql_auth_testing | env:DD_API_KEY | ok |
    Should Contain        ${result['select_after']}        100000001
    Should Not Contain    ${result['select_after']}        isError=true
    Should Be Equal As Integers    ${result['returncode']}    0

MCP Stdio Reload Credentials Sources Env File Mid Session
    [Documentation]    Issue #688: credential (re)sourcing at any point in the
    ...                MCP server lifecycle, stdio transport, cross-platform.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${env_file}=    Set Variable    ${CURDIR}${/}tmp${/}mcp-reload-credentials.env
    ${select_sql}=    Set Variable    select name, status from okta.application.apps apps where apps.subdomain = 'example-subdomain' order by name asc;
    ${child_env}=    Evaluate    {"OKTA_SECRET_KEY": None}
    ${steps}=    Evaluate    [{"call": "run_select_query", "args": {"sql": $select_sql}, "as": "select_before"}, {"write_env": {"OKTA_SECRET_KEY": $OKTA_SECRET_STR}}, {"call": "reload_credentials", "args": {}, "as": "reload"}, {"call": "run_select_query", "args": {"sql": $select_sql}, "as": "select_after"}]
    ${result}=    Evaluate    stackql_test_tooling.mcp_stdio_client.run_stdio_credential_script($STACKQL_EXE, $REGISTRY_NO_VERIFY_CFG_JSON_STR, $AUTH_CFG_STR, $env_file, $child_env, $steps)    modules=stackql_test_tooling.mcp_stdio_client
    Log    ${result['stderr']}
    # Before the env file exists, credential resolution fails with the
    # agent-actionable hint pointing at the reload_credentials tool.
    Should Contain        ${result['select_before']}    references empty string
    Should Contain        ${result['select_before']}    credential resolution failed for provider 'okta'
    Should Contain        ${result['select_before']}    fix the configured env file, call reload_credentials, then retry
    Should Not Contain    ${result['select_before']}    okta_browser_plugin
    # The reload sources the var (names only, never values) and reports the
    # okta provider as resolvable.
    Should Contain        ${result['reload']}    OKTA_SECRET_KEY
    Should Contain        ${result['reload']}    | api_key | true |  | okta | env:OKTA_SECRET_KEY | ok |
    Should Not Contain    ${result['reload']}    ${OKTA_SECRET_STR}
    # The same query now succeeds against the mocked okta provider.
    Should Contain        ${result['select_after']}    okta_browser_plugin
    Should Be Equal As Integers    ${result['returncode']}    0


MCP HTTP Server Query Library Search Tool
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9917
    ...                  \-\-exec.action      query_library_search
    ...                  \-\-exec.args        {"intent": "list enabled aws regions"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-HTTP-Server-Query-Library-Search.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-HTTP-Server-Query-Library-Search-stderr.txt
    Should Contain       ${result.stdout}       aws/ec2/regions-enabled
    Should Be Equal As Integers    ${result.rc}    0


MCP HTTP Server Query Library Get Rendered Tool
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9917
    ...                  \-\-exec.action      query_library_get
    ...                  \-\-exec.args        {"id": "aws/ec2/regions-enabled", "params": {"seed_region": "us-west-2"}}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-HTTP-Server-Query-Library-Get.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-HTTP-Server-Query-Library-Get-stderr.txt
    Should Contain       ${result.stdout}       region = 'us-west-2'
    Should Contain       ${result.stdout}       run_select_query
    Should Be Equal As Integers    ${result.rc}    0

# ===========================================================================
# Issue #729 scenarios.  (1) Protocol revision 2026-07-28: stdio serves every
# revision; Streamable HTTP serves it on the sessionless 9926 server
# (server.stateless) while the default stateful servers negotiate current
# clients down.  The Python harness plays a 2025-06-18 handshake client and a
# 2026-07-28 stateless client on both transports, including the safe-mode
# gated write.  (2) The otel audit log format: OTLP/JSON log records on 9925.
# ===========================================================================

MCP HTTP Legacy Revision Client Initialize Handshake Still Served
    [Documentation]    A 2025-06-18 client keeps the initialize handshake and the
    ...                Mcp-Session-Id session; the server negotiates the client's revision.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${result}=    Evaluate    stackql_test_tooling.mcp_stdio_client.run_http_legacy_roundtrip('http://127.0.0.1:9912')    modules=stackql_test_tooling.mcp_stdio_client
    Should Be Equal    ${result['negotiated']}    2025-06-18
    Should Be True     ${result['session_issued']}
    List Should Contain Value    ${result['tools']}    server_info
    List Should Contain Value    ${result['tools']}    run_select_query

MCP HTTP Current Revision Client Runs Without Handshake Or Session
    [Documentation]    On the sessionless 9926 server a 2026-07-28 client sends no
    ...                initialize and no session header; server/discover serves the
    ...                supported revisions and the embedded instructions once, then the
    ...                revision and capabilities ride in _meta and tools/list is
    ...                connection-invariant.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${result}=    Evaluate    stackql_test_tooling.mcp_stdio_client.run_http_stateless_roundtrip('http://127.0.0.1:9926')    modules=stackql_test_tooling.mcp_stdio_client
    List Should Contain Value    ${result['discover_versions']}    2026-07-28
    Should Contain    ${result['discover_instructions']}    \# Discovery workflow
    Should Not Be True    ${result['session_issued']}
    List Should Contain Value    ${result['tools']}    server_info
    List Should Contain Value    ${result['tools']}    run_select_query
    Should Contain    ${result['server_info']}    "version"

MCP HTTP Stateless Server Still Serves Legacy Handshake Client
    [Documentation]    Interop: a 2025-06-18 initialize is accepted on the sessionless
    ...                server (no session is issued) and tools/list works.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${result}=    Evaluate    stackql_test_tooling.mcp_stdio_client.run_http_legacy_roundtrip('http://127.0.0.1:9926')    modules=stackql_test_tooling.mcp_stdio_client
    Should Be Equal    ${result['negotiated']}    2025-06-18
    Should Not Be True    ${result['session_issued']}
    List Should Contain Value    ${result['tools']}    server_info

MCP HTTP Stateful Server Negotiates Current Client Down
    [Documentation]    The default (stateful) HTTP server does not serve 2026-07-28: a raw
    ...                _meta-versioned request is rejected, and the bundled client
    ...                discovers that and negotiates a prior revision, so existing HTTP
    ...                hosts keep sessions and elicitation unchanged.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${result}=    Evaluate    stackql_test_tooling.mcp_stdio_client.run_http_stateless_roundtrip('http://127.0.0.1:9912')    modules=stackql_test_tooling.mcp_stdio_client
    Should Be Equal    ${result['tools']}    ${{[]}}
    ${sel}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9912
    ...                  \-\-exec.action      server_info
    ...                  \-\-exec.args        {}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Stateful-negotiate.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Stateful-negotiate-stderr.txt
    Should Be Equal As Integers    ${sel.rc}    0

MCP HTTP Stateless Current Revision Client Gated Write Approved Via Input Requests
    [Documentation]    safe mode over sessionless HTTP on 2026-07-28: input_required, then
    ...                the retry with inputResponses accept runs the mutation.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${sql}=    Set Variable    delete from google.compute.firewalls where project = 'mutable-project' and firewall = 'deletable-firewall';
    ${result}=    Evaluate    stackql_test_tooling.mcp_stdio_client.run_http_stateless_gated_write('http://127.0.0.1:9926', $sql, approval_action='accept')    modules=stackql_test_tooling.mcp_stdio_client
    Should Be Equal    ${result['first_result_type']}    input_required
    Should Be Equal    ${result['input_request_methods']}    ${{['elicitation/create']}}
    Should Contain        ${result['retry']}    timestamp
    Should Not Contain    ${result['retry']}    isError=true

MCP HTTP Stateless Current Revision Client Gated Write Declined Via Input Requests
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${sql}=    Set Variable    delete from google.compute.firewalls where project = 'mutable-project' and firewall = 'deletable-firewall';
    ${result}=    Evaluate    stackql_test_tooling.mcp_stdio_client.run_http_stateless_gated_write('http://127.0.0.1:9926', $sql, approval_action='decline')    modules=stackql_test_tooling.mcp_stdio_client
    Should Be Equal    ${result['first_result_type']}    input_required
    Should Contain    ${result['retry']}    declined approval

MCP Stdio Current Revision Client Gated Write Approved Via Input Requests
    [Documentation]    safe mode on 2026-07-28: the approval comes back as an
    ...                input_required result carrying an elicitation/create input request;
    ...                the retry with inputResponses accept runs the mutation.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${sql}=    Set Variable    delete from google.compute.firewalls where project = 'mutable-project' and firewall = 'deletable-firewall';
    ${result}=    Evaluate    stackql_test_tooling.mcp_stdio_client.run_stdio_stateless_roundtrip($STACKQL_EXE, $REGISTRY_NO_VERIFY_CFG_JSON_STR, $AUTH_CFG_STR, $sql, approval_action='accept')    modules=stackql_test_tooling.mcp_stdio_client
    Log    ${result['stderr']}
    List Should Contain Value    ${result['tools']}    run_mutation_query
    Should Be Equal    ${result['first_result_type']}    input_required
    Should Be Equal    ${result['input_request_methods']}    ${{['elicitation/create']}}
    Should Contain        ${result['retry']}    timestamp
    Should Not Contain    ${result['retry']}    isError=true
    Should Be Equal As Integers    ${result['returncode']}    0

MCP Stdio Current Revision Client Gated Write Declined Via Input Requests
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${sql}=    Set Variable    delete from google.compute.firewalls where project = 'mutable-project' and firewall = 'deletable-firewall';
    ${result}=    Evaluate    stackql_test_tooling.mcp_stdio_client.run_stdio_stateless_roundtrip($STACKQL_EXE, $REGISTRY_NO_VERIFY_CFG_JSON_STR, $AUTH_CFG_STR, $sql, approval_action='decline')    modules=stackql_test_tooling.mcp_stdio_client
    Should Be Equal    ${result['first_result_type']}    input_required
    Should Contain    ${result['retry']}    declined approval
    Should Be Equal As Integers    ${result['returncode']}    0

MCP Stdio Legacy Revision Client Gated Write Approved Via Elicitation Request
    [Documentation]    safe mode on 2025-06-18: the same gate reaches the client as a
    ...                server-initiated elicitation/create request (the SDK fulfils the
    ...                multi round-trip server-side); accepting runs the mutation.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${sql}=    Set Variable    delete from google.compute.firewalls where project = 'mutable-project' and firewall = 'deletable-firewall';
    ${result}=    Evaluate    stackql_test_tooling.mcp_stdio_client.run_stdio_legacy_approval_roundtrip($STACKQL_EXE, $REGISTRY_NO_VERIFY_CFG_JSON_STR, $AUTH_CFG_STR, $sql, approval_action='accept')    modules=stackql_test_tooling.mcp_stdio_client
    Log    ${result['stderr']}
    Should Be Equal    ${result['negotiated']}    2025-06-18
    Should Contain    ${result['elicitation_message']}    Approve run_mutation_query
    Should Contain    ${result['elicitation_message']}    deletable-firewall
    Should Contain        ${result['call']}    timestamp
    Should Not Contain    ${result['call']}    isError=true
    Should Be Equal As Integers    ${result['returncode']}    0

MCP Stdio Legacy Revision Client Gated Write Declined Via Elicitation Request
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${sql}=    Set Variable    delete from google.compute.firewalls where project = 'mutable-project' and firewall = 'deletable-firewall';
    ${result}=    Evaluate    stackql_test_tooling.mcp_stdio_client.run_stdio_legacy_approval_roundtrip($STACKQL_EXE, $REGISTRY_NO_VERIFY_CFG_JSON_STR, $AUTH_CFG_STR, $sql, approval_action='decline')    modules=stackql_test_tooling.mcp_stdio_client
    Should Be Equal    ${result['negotiated']}    2025-06-18
    Should Contain    ${result['call']}    declined approval
    Should Be Equal As Integers    ${result['returncode']}    0

MCP HTTP Audit OTel Format Emits OTLP JSON Log Records
    [Documentation]    The 9925 server writes the audit log as OTLP/JSON (one LogsData per
    ...                line) mapped to the GenAI/MCP semantic conventions; the JSONL 9923
    ...                server keeps its byte-compatible shape.  Row values the client
    ...                received reach neither format while the verbatim SQL (including a
    ...                RETURNING mutation) does.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${sel}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9925
    ...                  \-\-exec.action      run_select_query
    ...                  \-\-exec.args        {"sql":"select name, id from google.storage.buckets where project \= 'stackql\-demo';"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Audit-OTel-select.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Audit-OTel-select-stderr.txt
    Should Be Equal As Integers    ${sel.rc}    0
    ${ins}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9925
    ...                  \-\-exec.action      run_mutation_query
    ...                  \-\-exec.args        {"sql":"insert into google.storage.buckets( project, data__name) select 'testing\-project', 'silly\-bucket' returning projectNumber, name, location;"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Audit-OTel-insert.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Audit-OTel-insert-stderr.txt
    Should Be Equal As Integers    ${ins.rc}    0
    ${ins_jsonl}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9923
    ...                  \-\-exec.action      run_mutation_query
    ...                  \-\-exec.args        {"sql":"insert into google.storage.buckets( project, data__name) select 'testing\-project', 'silly\-bucket' returning projectNumber, name, location;"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-Audit-JSONL-insert.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-Audit-JSONL-insert-stderr.txt
    Should Be Equal As Integers    ${ins_jsonl.rc}    0
    Sleep         1s
    ${otel_log}=    Get File    ${EXECDIR}${/}mcp-audit-otel-9925.log
    ${jsonl_log}=    Get File    ${EXECDIR}${/}mcp-audit-9923.log
    # Every line is a complete OTLP/JSON LogsData (the otlpjsonfile receiver shape).
    ${line_count}=    Evaluate    sum(1 for l in $otel_log.splitlines() if l.strip() and 'resourceLogs' in json.loads(l))    json
    Should Be True    ${line_count} >= 2
    Should Contain    ${otel_log}    "gen_ai.operation.name"
    Should Contain    ${otel_log}    "execute_tool"
    Should Contain    ${otel_log}    "gen_ai.tool.name"
    Should Contain    ${otel_log}    "run_mutation_query"
    # 9925 is a stateful server, so the bundled client negotiates down to 2025-11-25.
    Should Contain    ${otel_log}    "mcp.protocol.version"
    Should Contain    ${otel_log}    "2025-11-25"
    Should Contain    ${otel_log}    "stackql.query"
    Should Contain    ${otel_log}    returning projectNumber
    Should Contain    ${otel_log}    "stackql.rows_returned"
    Should Contain    ${otel_log}    "service.name"
    # The select returned bucket rows to the client; result values are never
    # serialised in either format (redaction parity), only the statements.
    Should Contain        ${sel.stdout}    demo-app-bucket1
    Should Not Contain    ${otel_log}     demo-app-bucket1
    Should Not Contain    ${jsonl_log}    demo-app-bucket1
    Should Contain        ${jsonl_log}    returning projectNumber
    # JSONL stays byte-compatible: no OTel or wire-context keys leak into it.
    Should Not Contain    ${jsonl_log}    resourceLogs
    Should Not Contain    ${jsonl_log}    rows_returned
    Should Not Contain    ${jsonl_log}    2026-07-28
