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

MCP HTTPS Server JSON DTO Greet
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${greet}=    Run Process
    ...    ${STACKQL_MCP_CLIENT_EXE}
    ...    exec
    ...    \-\-client\-type\=http
    ...    \-\-url\=https://127.0.0.1:9004
    ...    \-\-client\-cfg
    ...    { "apply_tls_globally": true, "insecure_skip_verify": true, "ca_file": "test/server/mtls/credentials/pg_server_cert.pem", "promote_leaf_to_ca": true }
    ...    \-\-exec.action
    ...    greet
    ...    \-\-exec.args
    ...    {"name":"JSON TEST"}
    ...    stdout=${CURDIR}${/}tmp${/}MCP-HTTPS-greet.txt
    ...    stderr=${CURDIR}${/}tmp${/}MCP-HTTPS-greet-stderr.txt
    Should Be Equal As Integers    ${greet.rc}    0
    Should Contain    ${greet.stdout}    Hi JSON TEST

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
    Dictionary Should Contain Key    ${srvinfo_obj}    name
    Dictionary Should Contain Key    ${srvinfo_obj}    info
    Dictionary Should Contain Key    ${srvinfo_obj}    is_read_only

MCP HTTPS Server JSON DTO DB Identity
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${dbident}=    Run Process
    ...    ${STACKQL_MCP_CLIENT_EXE}
    ...    exec
    ...    \-\-client\-type\=http
    ...    \-\-url\=https://127.0.0.1:9004
    ...    \-\-client\-cfg
    ...    { "apply_tls_globally": true, "insecure_skip_verify": true, "ca_file": "test/server/mtls/credentials/pg_server_cert.pem", "promote_leaf_to_ca": true }
    ...    \-\-exec.action
    ...    db_identity
    ...    stdout=${CURDIR}${/}tmp${/}MCP-HTTPS-db-identity.txt
    ...    stderr=${CURDIR}${/}tmp${/}MCP-HTTPS-db-identity-stderr.txt
    Should Be Equal As Integers    ${dbident.rc}    0
    ${dbident_obj}=    Parse MCP JSON Output    ${dbident.stdout}
    Dictionary Should Contain Key    ${dbident_obj}    identity

MCP HTTPS Server JSON DTO Query V3 JSON
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${query_json}=    Run Process
    ...    ${STACKQL_MCP_CLIENT_EXE}
    ...    exec
    ...    \-\-client\-type\=http
    ...    \-\-url\=https://127.0.0.1:9004
    ...    \-\-client\-cfg
    ...    { "apply_tls_globally": true, "insecure_skip_verify": true, "ca_file": "test/server/mtls/credentials/pg_server_cert.pem", "promote_leaf_to_ca": true }
    ...    \-\-exec.action
    ...    query_v3
    ...    \-\-exec.args
    ...    {"sql":"show providers;","format":"json"}
    ...    stdout=${CURDIR}${/}tmp${/}MCP-HTTPS-query-v2-json.txt
    ...    stderr=${CURDIR}${/}tmp${/}MCP-HTTPS-query-v2-json-stderr.txt
    Should Be Equal As Integers    ${query_json.rc}    0
    ${query_obj}=    Parse MCP JSON Output    ${query_json.stdout}
    Should Be Equal    ${query_obj["format"]}    json
    Dictionary Should Contain Key    ${query_obj}    rows
    ${row_count}=    Get From Dictionary    ${query_obj}    row_count
    Should Be True    ${row_count} > 0

MCP HTTPS Server Query Exec Text
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    # Future proofing: raw text format reserved; may gain structured hints later.
    ${ns_query_text}=    Run Process
    ...    ${STACKQL_MCP_CLIENT_EXE}
    ...    exec
    ...    \-\-client\-type\=http
    ...    \-\-url\=https://127.0.0.1:9004
    ...    \-\-client\-cfg
    ...    { "apply_tls_globally": true, "insecure_skip_verify": true, "ca_file": "test/server/mtls/credentials/pg_server_cert.pem", "promote_leaf_to_ca": true }
    ...    \-\-exec.action
    ...    query.exec_text
    ...    \-\-exec.args
    ...    {"sql":"SELECT 1 as foo"}
    ...    stdout=${CURDIR}${/}tmp${/}MCP-HTTPS-query-exec-text.txt
    ...    stderr=${CURDIR}${/}tmp${/}MCP-HTTPS-query-exec-text-stderr.txt
    Should Be Equal As Integers    ${ns_query_text.rc}    0
    Should Contain     ${ns_query_text.stdout}   foo

MCP HTTPS Server JSON DTO Query Exec JSON
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${ns_query_json}=    Run Process
    ...    ${STACKQL_MCP_CLIENT_EXE}
    ...    exec
    ...    \-\-client\-type\=http
    ...    \-\-url\=https://127.0.0.1:9004
    ...    \-\-client\-cfg
    ...    { "apply_tls_globally": true, "insecure_skip_verify": true, "ca_file": "test/server/mtls/credentials/pg_server_cert.pem", "promote_leaf_to_ca": true }
    ...    \-\-exec.action
    ...    query.exec_json
    ...    \-\-exec.args
    ...    {"sql":"SELECT 1 as foo","row_limit":5}
    ...    stdout=${CURDIR}${/}tmp${/}MCP-HTTPS-query-exec-json.txt
    ...    stderr=${CURDIR}${/}tmp${/}MCP-HTTPS-query-exec-json-stderr.txt
    Should Be Equal As Integers    ${ns_query_json.rc}    0
    ${ns_query_json_obj}=    Parse MCP JSON Output    ${ns_query_json.stdout}
    Should Be Equal    ${ns_query_json_obj["format"]}    json
    ${ns_row_count}=    Get From Dictionary    ${ns_query_json_obj}    row_count
    Should Be True    ${ns_row_count} >= 0

MCP HTTPS Server JSON DTO Meta Get Foreign Keys
    [Documentation]     Future proofing: foreign key discovery not yet implemented; placeholder.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${meta_fk}=    Run Process
    ...    ${STACKQL_MCP_CLIENT_EXE}
    ...    exec
    ...    \-\-client\-type\=http
    ...    \-\-url\=https://127.0.0.1:9004
    ...    \-\-client\-cfg
    ...    { "apply_tls_globally": true, "insecure_skip_verify": true, "ca_file": "test/server/mtls/credentials/pg_server_cert.pem", "promote_leaf_to_ca": true }
    ...    \-\-exec.action
    ...    meta.get_foreign_keys
    ...    \-\-exec.args
    ...    {"provider":"google","service":"cloudresourcemanager","resource":"projects"}
    ...    stdout=${CURDIR}${/}tmp${/}MCP-HTTPS-meta-get-foreign-keys.txt
    ...    stderr=${CURDIR}${/}tmp${/}MCP-HTTPS-meta-get-foreign-keys-stderr.txt
    Should Be Equal As Integers    ${meta_fk.rc}    0
    ${meta_fk_obj}=    Parse MCP JSON Output    ${meta_fk.stdout}
    Dictionary Should Contain Key    ${meta_fk_obj}    text

MCP HTTPS Server JSON DTO Meta Find Relationships
    [Documentation]     Future proofing: relationship graph inference pending; placeholder output.
    Pass Execution If    "%{IS_SKIP_MCP_TEST=false}" == "true"    Some platforms do not have the MCP client available
    ${meta_rels}=    Run Process
    ...    ${STACKQL_MCP_CLIENT_EXE}
    ...    exec
    ...    \-\-client\-type\=http
    ...    \-\-url\=https://127.0.0.1:9004
    ...    \-\-client\-cfg
    ...    { "apply_tls_globally": true, "insecure_skip_verify": true, "ca_file": "test/server/mtls/credentials/pg_server_cert.pem", "promote_leaf_to_ca": true }
    ...    \-\-exec.action
    ...    meta.find_relationships
    ...    \-\-exec.args
    ...    {"provider":"google","service":"cloudresourcemanager","resource":"projects"}
    ...    stdout=${CURDIR}${/}tmp${/}MCP-HTTPS-meta-find-relationships.txt
    ...    stderr=${CURDIR}${/}tmp${/}MCP-HTTPS-meta-find-relationships-stderr.txt
    Should Be Equal As Integers    ${meta_rels.rc}    0
    ${meta_rels_obj}=    Parse MCP JSON Output    ${meta_rels.stdout}
    Dictionary Should Contain Key    ${meta_rels_obj}    text
