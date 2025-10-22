*** Settings ***
Resource          ${CURDIR}${/}stackql.resource


*** Keywords ***
Start MCP Servers
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Start Process                         ${STACKQL_EXE}
    ...                                   mcp
    ...                                   \-\-mcp.server.type\=http
    ...                                   \-\-mcp.config
    ...                                   {"server": {"transport": "http", "address": "127.0.0.1:9912"} }
    ...                                   \-\-registry
    ...                                   ${REGISTRY_NO_VERIFY_CFG_JSON_STR}
    ...                                   \-\-auth
    ...                                   ${AUTH_CFG_STR}
    ...                                   \-\-tls.allowInsecure
    Start Process                         ${STACKQL_EXE}
    ...                                   srv
    ...                                   \-\-mcp.server.type\=http
    ...                                   \-\-mcp.config
    ...                                   {"server": {"transport": "http", "address": "127.0.0.1:9913"} }
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
    ...                                   {"server": {"transport": "http", "address": "127.0.0.1:9914"}, "backend": {"dsn": "postgres:\/\/stackql:stackql@127.0.0.1:5445?default_query_exec_mode\=simple_protocol"} }
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
    ...                                   {"server": {"tls_cert_file": "test/server/mtls/credentials/pg_server_cert.pem", "tls_key_file": "test/server/mtls/credentials/pg_server_key.pem", "transport": "http", "address": "127.0.0.1:9004"}, "backend": {"dsn": "postgres:\/\/stackql:stackql@127.0.0.1:5446?default_query_exec_mode\=simple_protocol"} }
    ...                                   \-\-registry
    ...                                   ${REGISTRY_NO_VERIFY_CFG_JSON_STR}
    ...                                   \-\-auth
    ...                                   ${AUTH_CFG_STR}
    ...                                   \-\-tls.allowInsecure
    ...                                   \-\-pgsrv.port
    ...                                   5446
    ...                                   stdout=${CURDIR}${/}tmp${/}Stackql-MCP-Server-HTTPS.txt
    ...                                   stderr=${CURDIR}${/}tmp${/}Stackql-MCP-Server-HTTPS-stderr.txt
    Sleep         5s

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
    Should Contain       ${result.stdout}       Get server information
    Should Be Equal As Integers    ${result.rc}    0


MCP HTTP Server Verify Greeting Tool
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http 
    ...                  \-\-url\=http://127.0.0.1:9912
    ...                  \-\-exec.action      greet 
    ...                  \-\-exec.args        {"name": "JOE BLOW"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-HTTP-Server-Verify-Greeting-Tool.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-HTTP-Server-Verify-Greeting-Tool-stderr.txt
    Should Contain       ${result.stdout}       JOE BLOW
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

MCP HTTP Server Query Tool
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    Sleep         5s
    ${result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http 
    ...                  \-\-url\=http://127.0.0.1:9912
    ...                  \-\-exec.action      query_v2
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
    ...                  \-\-exec.action      query_v2
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
    ...                  \-\-exec.action      query_v2
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
    ...                  \-\-exec.action      query_v2
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
