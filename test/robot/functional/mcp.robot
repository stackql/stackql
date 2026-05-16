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
    Start Process                         ${STACKQL_EXE}
    ...                                   mcp
    ...                                   \-\-mcp.server.type\=http
    ...                                   \-\-mcp.config
    ...                                   {"server": {"transport": "http", "address": "127.0.0.1:9915"}, "enabled_tools": ["server_info"] }
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
    ...                                   {"server": {"transport": "http", "address": "127.0.0.1:9916", "read_only": true} }
    ...                                   \-\-registry
    ...                                   ${REGISTRY_NO_VERIFY_CFG_JSON_STR}
    ...                                   \-\-auth
    ...                                   ${AUTH_CFG_STR}
    ...                                   \-\-tls.allowInsecure
    ...                                   stdout=${CURDIR}${/}tmp${/}Stackql-MCP-Server-ReadOnly.txt
    ...                                   stderr=${CURDIR}${/}tmp${/}Stackql-MCP-Server-ReadOnly-stderr.txt
    Sleep         5s

Parse MCP JSON Output
    [Arguments]    ${input}
    ${parsed}=    Evaluate
    ...    json.loads('''${input}''')    json
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
    Dictionary Should Contain Key    ${lifecycle_obj}    messages
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
    Should Contain    ${mutation_result.stderr}    read-only
    ${lifecycle_result}=    Run Process          ${STACKQL_MCP_CLIENT_EXE}
    ...                  exec
    ...                  \-\-client\-type\=http
    ...                  \-\-url\=http://127.0.0.1:9916
    ...                  \-\-exec.action      run_lifecycle_operation
    ...                  \-\-exec.args        {"sql":"EXEC google.compute.instances.start @project \= 'mutable\-project', @zone \= 'us\-central1\-a', @instance \= 'demo';"}
    ...                  stdout=${CURDIR}${/}tmp${/}MCP-ReadOnly-lifecycle.txt
    ...                  stderr=${CURDIR}${/}tmp${/}MCP-ReadOnly-lifecycle-stderr.txt
    Should Not Be Equal As Integers    ${lifecycle_result.rc}    0
    Should Contain    ${lifecycle_result.stderr}    read-only
