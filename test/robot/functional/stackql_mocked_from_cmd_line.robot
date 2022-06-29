*** Variables ***
${LOCAL_LIB_HOME}    ../lib

*** Settings ***
Library    Process
Library    OperatingSystem
Library    String
Library    ${LOCAL_LIB_HOME}/StackQLInterfaces.py

*** Settings ***
Variables         ${CURDIR}/../variables/stackql_context.py
Suite Setup       Prepare StackQL Environment
Suite Teardown    Terminate All Processes

*** Test Cases *** 
Google Container Agg Desc
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SELECT_CONTAINER_SUBNET_AGG_DESC}
    ...    ${SELECT_CONTAINER_SUBNET_AGG_DESC_EXPECTED}

Google Container Agg Asc
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SELECT_CONTAINER_SUBNET_AGG_ASC}
    ...    ${SELECT_CONTAINER_SUBNET_AGG_ASC_EXPECTED}

Google IAM Policy Agg
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    \-\-infile\=${GET_IAM_POLICY_AGG_ASC_INPUT_FILE}
    ...    ${GET_IAM_POLICY_AGG_ASC_EXPECTED}
    ...    \-o\=csv


Google Select Project IAM Policy
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_EXPERIMENTAL_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY}
    ...    ${SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_EXPECTED}

Google Select Project IAM Policy Filtered And Verify Like Filtering
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_EXPERIMENTAL_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_LIKE_FILTERED}
    ...    ${SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_FILTERED_EXPECTED}

Google Select Project IAM Policy Filtered And Verify Where Filtering
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_EXPERIMENTAL_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_COMPARISON_FILTERED}
    ...    ${SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_FILTERED_EXPECTED}

Google Join Plus String Concatenated Select Expressions
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SELECT_GOOGLE_JOIN_CONCATENATED_SELECT_EXPRESSIONS}
    ...    ${SELECT_GOOGLE_JOIN_CONCATENATED_SELECT_EXPRESSIONS_EXPECTED}
    ...    ${CURDIR}/tmp/Google-Join-Plus-String-Concatenated-Select-Expressions.tmp

Google AcceleratorTypes SQL verb pre changeover
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SELECT_ACCELERATOR_TYPES_DESC}
    ...    ${SELECT_ACCELERATOR_TYPES_DESC_EXPECTED}

Google Machine Types Select Paginated
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SELECT_MACHINE_TYPES_DESC}
    ...    ${SELECT_MACHINE_TYPES_DESC_EXPECTED}
    ...    ${CURDIR}/tmp/Google-Machine-Types-Select-Paginated.tmp

Google AcceleratorTypes SQL verb post changeover
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_SQL_VERB_CONTRIVED_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SELECT_ACCELERATOR_TYPES_DESC}
    ...    ${SELECT_ACCELERATOR_TYPES_DESC_EXPECTED}

Okta Apps Select Simple
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SELECT_OKTA_APPS}
    ...    ${SELECT_OKTA_APPS_ASC_EXPECTED}

AWS Volumes Select Simple
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SELECT_AWS_VOLUMES}
    ...    ${SELECT_AWS_VOLUMES_ASC_EXPECTED}

AWS Volume Insert Simple
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${CREATE_AWS_VOLUME}
    ...    The operation completed successfully

GitHub Pages Select Top Level Object
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SELECT_GITHUB_REPOS_PAGES_SINGLE}
    ...    ${SELECT_GITHUB_REPOS_PAGES_SINGLE_EXPECTED}

GitHub Scim Users Select
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SELECT_GITHUB_SCIM_USERS}
    ...    ${SELECT_GITHUB_SCIM_USERS_EXPECTED}

GitHub Branch Names Paginated Select
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SELECT_GITHUB_BRANCHES_NAMES_DESC}
    ...    ${SELECT_GITHUB_BRANCHES_NAMES_DESC_EXPECTED}
    ...    ${CURDIR}/tmp/GitHub-Branch-Names-Paginated-Select.tmp

GitHub Tags Paginated Count
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SELECT_GITHUB_TAGS_COUNT}
    ...    ${SELECT_GITHUB_TAGS_COUNT_EXPECTED}

GitHub Repository IDs Select
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SELECT_GITHUB_REPOS_IDS_ASC}
    ...    ${SELECT_GITHUB_REPOS_IDS_ASC_EXPECTED}

Filter on Implicit Selectable Object
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SELECT_GITHUB_REPOS_FILTERED_SINGLE}
    ...    ${SELECT_GITHUB_REPOS_FILTERED_SINGLE_EXPECTED}

Join GCP Okta Cross Provider
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SELECT_CONTRIVED_GCP_OKTA_JOIN}
    ...    ${SELECT_CONTRIVED_GCP_OKTA_JOIN_EXPECTED}

Join GCP Okta Cross Provider JSON Dependent Keyword in Table Name
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SELECT_CONTRIVED_GCP_GITHUB_JSON_DEPENDENT_JOIN}
    ...    ${SELECT_CONTRIVED_GCP_GITHUB_JSON_DEPENDENT_JOIN_EXPECTED}

Join GCP Three Way
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SELECT_CONTRIVED_GCP_THREE_WAY_JOIN}
    ...    ${SELECT_CONTRIVED_GCP_THREE_WAY_JOIN_EXPECTED}

Join GCP Self
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SELECT_CONTRIVED_GCP_SELF_JOIN}
    ...    ${SELECT_CONTRIVED_GCP_SELF_JOIN_EXPECTED}

K8S Nodes Select Leveraging JSON Path
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SELECT_K8S_NODES_ASC}
    ...    ${SELECT_K8S_NODES_ASC_EXPECTED}

Registry List All
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_MOCKED_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${REGISTRY_LIST} 
    ...    ${REGISTRY_LIST_EXPECTED}

Registry List Google Provider
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_MOCKED_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${REGISTRY_GOOGLE_PROVIDER_LIST} 
    ...    ${REGISTRY_GOOGLE_PROVIDER_LIST_EXPECTED}


Data Flow Sequential Join Paginated Select Github 
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SELECT_GITHUB_JOIN_DATA_FLOW_SEQUENTIAL} 
    ...    ${SELECT_GITHUB_JOIN_DATA_FLOW_SEQUENTIAL_EXPECTED}

Basic Query mTLS Returns OK
    Should PG Client Inline Contain
    ...    ${CURDIR}
    ...    ${PSQL_EXE}
    ...    ${PSQL_MTLS_CONN_STR}
    ...    ${SELECT_CONTAINER_SUBNET_AGG_ASC}
    ...    ipCidrRange

Basic Query unencrypted Returns OK
    Should PG Client Inline Contain
    ...    ${CURDIR}
    ...    ${PSQL_EXE}
    ...    ${PSQL_UNENCRYPTED_CONN_STR}
    ...    ${SELECT_CONTAINER_SUBNET_AGG_ASC}
    ...    ipCidrRange

Erroneous mTLS Config Plus Basic Query Returns Error
    Should PG Client Error Inline Contain
    ...    ${CURDIR}
    ...    ${PSQL_EXE}
    ...    ${PSQL_MTLS_INVALID_CONN_STR}
    ...    ${SELECT_CONTAINER_SUBNET_AGG_ASC}
    ...    error

*** Keywords ***
Start Mock Server
    [Arguments]    ${_JSON_INIT_FILE_PATH}    ${_MOCKSERVER_JAR}    ${_MOCKSERVER_PORT}
    ${process} =    Start Process    java    \-Dfile.encoding\=UTF-8
    ...  \-Dmockserver.initializationJsonPath\=${_JSON_INIT_FILE_PATH}
    ...  \-jar    ${_MOCKSERVER_JAR}
    ...  \-serverPort    ${_MOCKSERVER_PORT}    \-logLevel    INFO
    Sleep    5s
    [Return]    ${process}


Prepare StackQL Environment
    Set Environment Variable    OKTA_SECRET_KEY    ${OKTA_SECRET_STR}
    Set Environment Variable    GITHUB_SECRET_KEY    ${GITHUB_SECRET_STR}
    Set Environment Variable    K8S_SECRET_KEY    ${K8S_SECRET_STR}
    Start Mock Server    ${JSON_INIT_FILE_PATH_GOOGLE}    ${MOCKSERVER_JAR}    ${MOCKSERVER_PORT_GOOGLE}
    Start Mock Server    ${JSON_INIT_FILE_PATH_OKTA}    ${MOCKSERVER_JAR}    ${MOCKSERVER_PORT_OKTA}
    Start Mock Server    ${JSON_INIT_FILE_PATH_GITHUB}    ${MOCKSERVER_JAR}    ${MOCKSERVER_PORT_GITHUB}
    Start Mock Server    ${JSON_INIT_FILE_PATH_AWS}    ${MOCKSERVER_JAR}    ${MOCKSERVER_PORT_AWS}
    Start Mock Server    ${JSON_INIT_FILE_PATH_K8S}    ${MOCKSERVER_JAR}    ${MOCKSERVER_PORT_K8S}
    Start Mock Server    ${JSON_INIT_FILE_PATH_REGISTRY}    ${MOCKSERVER_JAR}    ${MOCKSERVER_PORT_REGISTRY}
    Start StackQL PG Server mTLS    ${PG_SRV_PORT_MTLS}    ${PG_SRV_MTLS_CFG_STR}
    Start StackQL PG Server unencrypted    ${PG_SRV_PORT_UNENCRYPTED}

Start StackQL PG Server mTLS
    [Arguments]    ${_SRV_PORT_MTLS}    ${_MTLS_CFG_STR}
    ${process} =    Start Process    ${STACKQL_EXE}
                    ...  srv    \-\-registry\=${REGISTRY_NO_VERIFY_CFG_STR}
                    ...  \-\-auth\=${AUTH_CFG_STR}
                    ...  \-\-tls\.allowInsecure\=true
                    ...  \-\-pgsrv\.address\=0.0.0.0 
                    ...  \-\-pgsrv\.port\=${_SRV_PORT_MTLS} 
                    ...  \-\-pgsrv\.tls    ${_MTLS_CFG_STR}
    Sleep    15s
    [Return]    ${process}


Start StackQL PG Server unencrypted
    [Arguments]    ${_SRV_PORT_UNENCRYPTED}
    ${process} =    Start Process    ${STACKQL_EXE}
                    ...  srv    \-\-registry\=${REGISTRY_NO_VERIFY_CFG_STR}
                    ...  \-\-auth\=${AUTH_CFG_STR}
                    ...  \-\-tls\.allowInsecure\=true
                    ...  \-\-pgsrv\.address\=0.0.0.0 
                    ...  \-\-pgsrv\.port\=${_SRV_PORT_UNENCRYPTED}
    Sleep    15s
    [Return]    ${process}





