*** Settings ***
Resource          ${CURDIR}/stackql.resource
Test Teardown     Stackql Per Test Teardown

*** Test Cases *** 
Google Container Agg Desc
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
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
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_CONTAINER_SUBNET_AGG_ASC}
    ...    ${SELECT_CONTAINER_SUBNET_AGG_ASC_EXPECTED}

Google IAM Policy Agg
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Skipping postgres backend test due to unsupported function group_concat
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
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
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
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
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
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
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_COMPARISON_FILTERED}
    ...    ${SELECT_GOOGLE_CLOUDRESOURCEMANAGER_IAMPOLICY_FILTERED_EXPECTED}

Google Join Plus String Concatenated Select Expressions
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Skipping postgres backend test due to unsupported function json_extract
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
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
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
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
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
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
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
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
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_OKTA_APPS}
    ...    ${SELECT_OKTA_APPS_ASC_EXPECTED}

Okta Users Select Simple Paginated
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Skipping postgres backend test due to unsupported function json_extract
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_OKTA_USERS_ASC}
    ...    ${SELECT_OKTA_USERS_ASC_EXPECTED}
    ...    ${CURDIR}/tmp/Okta-Users-Select-Simple-Paginated.tmp

AWS EC2 Volumes Select Simple
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_AWS_VOLUMES}
    ...    ${SELECT_AWS_VOLUMES_ASC_EXPECTED}

AWS IAM Users Select Simple
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_AWS_IAM_USERS_ASC}
    ...    ${SELECT_AWS_IAM_USERS_ASC_EXPECTED}
    ...    ${CURDIR}/tmp/AWS-IAM-Users-Select-Simple.tmp

AWS S3 Buckets Select Simple
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_AWS_S3_BUCKETS}
    ...    ${SELECT_AWS_S3_BUCKETS_EXPECTED}
    ...    ${CURDIR}/tmp/AWS-S3-Buckets-Select-Simple.tmp

AWS S3 Objects Select Simple
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_AWS_S3_OBJECTS}
    ...    ${SELECT_AWS_S3_OBJECTS_EXPECTED}
    ...    ${CURDIR}/tmp/AWS-S3-Objects-Select-Simple.tmp

AWS S3 Objects Null Select Simple
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_AWS_S3_OBJECTS_NULL}
    ...    ${SELECT_AWS_S3_OBJECTS_NULL_EXPECTED}
    ...    ${CURDIR}/tmp/AWS-S3-Objects-Null-Select-Simple.tmp

AWS S3 Bucket Locations Top Level Property Select Simple
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_AWS_S3_BUCKET_LOCATIONS}
    ...    ${SELECT_AWS_S3_BUCKET_LOCATIONS_EXPECTED}
    ...    ${CURDIR}/tmp/AWS-S3-Bucket-Locations-Top-Level-Property-Select-Simple.tmp

AWS EC2 VPN Gateways Null Select Simple
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_AWS_EC2_VPN_GATEWAYS_NULL}
    ...    ${SELECT_AWS_EC2_VPN_GATEWAYS_NULL_EXPECTED}
    ...    ${CURDIR}/tmp/AWS-EC2-VPN-Gateways-Null-Select-Simple.tmp

AWS Cloud Control VPCs Select Simple
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_AWS_CLOUD_CONTROL_VPCS_DESC}
    ...    ${SELECT_AWS_CLOUD_CONTROL_VPCS_DESC_EXPECTED}

AWS Cloud Control Operations Select Simple
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_AWS_CLOUD_CONTROL_OPERATIONS_DESC}
    ...    ${SELECT_AWS_CLOUD_CONTROL_OPERATIONS_DESC_EXPECTED}
    ...    ${CURDIR}/tmp/AWS-Cloud-Control-Operations-Select-Simple.tmp

AWS EC2 Volume Insert Simple
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${CREATE_AWS_VOLUME}
    ...    The operation was despatched successfully

AWS EC2 Volume Update Simple
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${UPDATE_AWS_EC2_VOLUME}
    ...    The operation was despatched successfully

GitHub Orgs Org Update Simple
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${UPDATE_GITHUB_ORG}
    ...    The operation was despatched successfully

AWS Cloud Control Log Group Insert Simple
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${CREATE_AWS_CLOUD_CONTROL_LOG_GROUP}
    ...    The operation was despatched successfully

AWS Cloud Control Log Group Delete Simple
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${DELETE_AWS_CLOUD_CONTROL_LOG_GROUP}
    ...    The operation was despatched successfully

AWS Cloud Control Log Group Update Simple
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${UPDATE_AWS_CLOUD_CONTROL_REQUEST_LOG_GROUP}
    ...    The operation was despatched successfully

GitHub Pages Select Top Level Object
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
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
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_GITHUB_SCIM_USERS}
    ...    ${SELECT_GITHUB_SCIM_USERS_EXPECTED}

GitHub SAML Identities Select GraphQL
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: Fix this... Skipping postgres backend test due to unsupported function json_extract
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_GITHUB_SAML_IDENTITIES}
    ...    ${SELECT_GITHUB_SAML_IDENTITIES_EXPECTED}
    ...    ${CURDIR}/tmp/GitHub-SAML-Identities-Select-GraphQL.tmp

GitHub Branch Names Paginated Select
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
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
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
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
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_GITHUB_REPOS_IDS_ASC}
    ...    ${SELECT_GITHUB_REPOS_IDS_ASC_EXPECTED}

GitHub Analytics Simple Select Repositories Collaborators
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_ANALYTICS}
    ...    ${SELECT_ANALYTICS_CACHE_GITHUB_REPOSITORIES_COLLABORATORS_SIMPLE}
    ...    ${SELECT_ANALYTICS_CACHE_GITHUB_REPOSITORIES_COLLABORATORS_EXPECTED}
    ...    \-\-namespaces\=${NAMESPACES_TTL_SIMPLE}
    ...    stdout=${CURDIR}/tmp/GitHub-Analytics-Select-Repositories-Collaborators.tmp

GitHub Analytics Transparent Select Repositories Collaborators
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_ANALYTICS}
    ...    ${SELECT_ANALYTICS_CACHE_GITHUB_REPOSITORIES_COLLABORATORS_TRANSPARENT}
    ...    ${SELECT_ANALYTICS_CACHE_GITHUB_REPOSITORIES_COLLABORATORS_EXPECTED}
    ...    \-\-namespaces\=${NAMESPACES_TTL_TRANSPARENT}
    ...    stdout=${CURDIR}/tmp/GitHub-Analytics-Select-Repositories-Collaborators.tmp

GitHub Repository With Functions Select
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: Fix this... Skipping postgres backend test due to unsupported function split_part
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_GITHUB_REPOS_WITH_USEFUL_FUNCTIONS}
    ...    ${SELECT_GITHUB_REPOS_WITH_USEFUL_FUNCTIONS_EXPECTED}
    ...    ${CURDIR}/tmp/GitHub-Repository-With-Functions-Select.tmp

GitHub Join Input Params Select
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_GITHUB_JOIN_IN_PARAMS}
    ...    ${SELECT_GITHUB_JOIN_IN_PARAMS_EXPECTED}
    ...    ${CURDIR}/tmp/GitHub-Join-Input-Params-Select.tmp

Filter on Implicit Selectable Object
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
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
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_CONTRIVED_GCP_OKTA_JOIN}
    ...    ${SELECT_CONTRIVED_GCP_OKTA_JOIN_EXPECTED}

Join GCP Okta Cross Provider JSON Dependent Keyword in Table Name
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Skipping postgres backend test due to unsupported function json_extract
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
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
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
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
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
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
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_K8S_NODES_ASC}
    ...    ${SELECT_K8S_NODES_ASC_EXPECTED}

Google Compute Instance IAM Policy Select
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY}
    ...    ${SELECT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY_EXPECTED}

Google IAM Policy Show Insert
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SHOW_INSERT_GOOGLE_IAM_SERVICE_ACCOUNTS}
    ...    ${SHOW_INSERT_GOOGLE_IAM_SERVICE_ACCOUNTS_EXPECTED}


Google Compute Instance IAM Policy Show Insert Error
    Should Stackql Exec Inline Equal Stderr
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SHOW_INSERT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY_ERROR}
    ...    ${SHOW_INSERT_GOOGLE_COMPUTE_INSTANCE_IAM_POLICY_ERROR_EXPECTED}

Registry List All
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_MOCKED_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
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
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${REGISTRY_GOOGLE_PROVIDER_LIST} 
    ...    ${REGISTRY_GOOGLE_PROVIDER_LIST_EXPECTED}

Registry Pull Google Provider Specific Version
    Should Stackql Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_MOCKED_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    registry pull google v0.1.2 ; 
    ...    successfully installed

Basic Floating Point Projection Display Plus Bearer And User Password Auth Encoding
    Should Stackql Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select price_monthly, price_hourly from digitalocean.sizes.sizes where price_monthly \= 48.0 ;
    ...    0.07143
    ...    stdout=${CURDIR}/tmp/Basic-Floating-Point-Projection-Display-Plus-Bearer-And-User-Password-Auth-Encoding.tmp
   
Basic Floating Point Projection Display Plus Basic Auth Encoding
    Should Stackql Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    {"digitalocean": { "username": "myusername", "password": "mypassword", "type": "basic"}}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select price_monthly, price_hourly from digitalocean.sizes.sizes where price_monthly \= 48.0 ;
    ...    0.07143
    ...    stdout=${CURDIR}/tmp/Basic-Floating-Point-Projection-Display-Plus-Basic-Auth-Encoding.tmp  

Basic Floating Point Projection Display Plus Custom Basic Auth Encoding
    Should Stackql Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    {"digitalocean": { "username": "myusername", "password": "mypassword", "type": "basic", "valuePrefix": "CUSTOM "}}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select price_monthly, price_hourly from digitalocean.sizes.sizes where price_monthly \= 48.0 ;
    ...    0.07143
    ...    stdout=${CURDIR}/tmp/Basic-Floating-Point-Projection-Display-Plus-Custom-Basic-Auth-Encoding.tmp
   
Basic Floating Point Projection Display Plus Custom Env Var Basic Auth Encoding
    Should Stackql Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    {"digitalocean": { "username_var": "DUMMY_DIGITALOCEAN_USERNAME", "password_var": "DUMMY_DIGITALOCEAN_PASSWORD", "type": "basic", "valuePrefix": "TOTALLY_CONTRIVED "}}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select price_monthly, price_hourly from digitalocean.sizes.sizes where price_monthly \= 48.0 ;
    ...    0.07143
    ...    stdout=${CURDIR}/tmp/Basic-Floating-Point-Projection-Display-Plus-Custom-Env-Var-Basic-Auth-Encoding.tmp

Digitalocean Insert Droplet
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    {"digitalocean": { "username_var": "DUMMY_DIGITALOCEAN_USERNAME", "password_var": "DUMMY_DIGITALOCEAN_PASSWORD", "type": "basic", "valuePrefix": "TOTALLY_CONTRIVED "}}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    INSERT INTO digitalocean.droplets.droplets ( data__name, data__region, data__size, data__image, data__backups, data__ipv6, data__monitoring, data__tags ) SELECT 'some.example.com', 'nyc3', 's-1vcpu-1gb', 'ubuntu-20-04-x64', true, true, true, '["env:prod", "web"]' ;
    ...    The operation was despatched successfully
    ...    stdout=${CURDIR}/tmp/Digitalocean-Insert-Droplet.tmp

Transaction Rollback Digitalocean Insert Droplet
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    {"digitalocean": { "username_var": "DUMMY_DIGITALOCEAN_USERNAME", "password_var": "DUMMY_DIGITALOCEAN_PASSWORD", "type": "basic", "valuePrefix": "TOTALLY_CONTRIVED "}}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    begin; INSERT INTO digitalocean.droplets.droplets ( data__name, data__region, data__size, data__image, data__backups, data__ipv6, data__monitoring, data__tags ) SELECT 'some.example.com', 'nyc3', 's-1vcpu-1gb', 'ubuntu-20-04-x64', true, true, true, '["env:prod", "web"]' ; rollback;
    ...    OK\nmutating statement queued\nRollback OK
    ...    stdout=${CURDIR}/tmp/Digitalocean-Insert-Droplet.tmp

Transaction Abort Attempted Commit Digitalocean Insert Droplet
    ${inputStr} =    Catenate
    ...    begin; 
    ...    INSERT INTO digitalocean.droplets.droplets(
    ...    data__name, data__region, data__size, 
    ...    data__image, data__backups, data__ipv6,
    ...    data__monitoring, data__tags
    ...    ) 
    ...    SELECT 'some.example.com', 'nyc3', 's-1vcpu-1gb', 
    ...    'ubuntu-20-04-x64', true, true, true, 
    ...    '["env:prod", "web"]' ;
    ...    INSERT INTO digitalocean.droplets.droplets(
    ...    data__name, data__region, data__size, 
    ...    data__image, data__backups, data__ipv6,
    ...    data__monitoring, data__tags
    ...    ) 
    ...    SELECT 'some.example.com', 'nyc3', 's-1vcpu-1gb', 
    ...    'ubuntu-20-04-x64', true, true, true, 
    ...    '["env:prod", "web"]' ;
    ...    INSERT INTO digitalocean.droplets.droplets(
    ...    data__name, data__region, data__size, 
    ...    data__image, data__backups, data__ipv6,
    ...    data__monitoring, data__tags
    ...    ) 
    ...    SELECT 'error.example.com', 'nyc3', 's-1vcpu-1gb', 
    ...    'ubuntu-20-04-x64', true, true, true, 
    ...    '["env:prod", "web"]' ;
    ...    commit;
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    OK
    ...    mutating statement queued
    ...    mutating statement queued
    ...    mutating statement queued
    ...    UNDO required: Undo the insert on digitalocean.droplets.droplets
    ...    UNDO required: Undo the insert on digitalocean.droplets.droplets
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    {"digitalocean": { "username_var": "DUMMY_DIGITALOCEAN_USERNAME", "password_var": "DUMMY_DIGITALOCEAN_PASSWORD", "type": "basic", "valuePrefix": "TOTALLY_CONTRIVED "}}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Transaction-Abort-Attempted-Commit-Digitalocean-Insert-Droplet.tmp

Scalar Select Verify 
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |---|---|-------|
    ...    | 1 | 2 | three |
    ...    |---|---|-------|
    ...    | 1 | 2 | three |
    ...    |---|---|-------|
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    {"digitalocean": { "username_var": "DUMMY_DIGITALOCEAN_USERNAME", "password_var": "DUMMY_DIGITALOCEAN_PASSWORD", "type": "basic", "valuePrefix": "TOTALLY_CONTRIVED "}}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select 1 as "1", 2 as "2", 'three' as three;
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Scalar-Select-Verify.tmp

Transaction Commit Eager Show and Lazy Digitalocean Insert Droplet
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    {"digitalocean": { "username_var": "DUMMY_DIGITALOCEAN_USERNAME", "password_var": "DUMMY_DIGITALOCEAN_PASSWORD", "type": "basic", "valuePrefix": "TOTALLY_CONTRIVED "}}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    begin; INSERT INTO digitalocean.droplets.droplets ( data__name, data__region, data__size, data__image, data__backups, data__ipv6, data__monitoring, data__tags ) SELECT 'some.example.com', 'nyc3', 's-1vcpu-1gb', 'ubuntu-20-04-x64', true, true, true, '["env:prod", "web"]' ; show services in digitalocean like 'droplets'; commit;
    ...    OK\nmutating${SPACE}statement${SPACE}queued\n|-----------------------|----------|-----------------------------|\n|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}id${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}name${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}title${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|\n|-----------------------|----------|-----------------------------|\n|${SPACE}droplets:v23.03.00127${SPACE}|${SPACE}droplets${SPACE}|${SPACE}DigitalOcean${SPACE}API${SPACE}-${SPACE}Droplets${SPACE}|\n|-----------------------|----------|-----------------------------|\nThe${SPACE}operation${SPACE}was${SPACE}despatched${SPACE}successfully\nOK
    ...    stdout=${CURDIR}/tmp/Digitalocean-Insert-Droplet.tmp

Registry Pull Google Provider Specific Version Prerelease
    Should Stackql Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_MOCKED_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    registry pull google 'v0.1.1\-alpha01' ; 
    ...    successfully installed

Registry Pull Google Provider Implicit Latest Version
    Should Stackql Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_MOCKED_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    registry pull google ;
    ...    successfully installed


Data Flow Sequential Join Paginated Select Github 
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}    
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_GITHUB_JOIN_DATA_FLOW_SEQUENTIAL} 
    ...    ${SELECT_GITHUB_JOIN_DATA_FLOW_SEQUENTIAL_EXPECTED}
    ...    ${CURDIR}/tmp/Data-Flow-Sequential-Join-Paginated-Select-Github.tmp

Paginated and Data Flow Sequential Join Github Okta SAML 
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}    
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_GITHUB_OKTA_SAML_JOIN} 
    ...    ${SELECT_GITHUB_OKTA_SAML_JOIN_EXPECTED}
    ...    ${CURDIR}/tmp/Paginated-and-Data-Flow-Sequential-Join-Github-Okta-SAML.tmp

Data Flow Sequential Join Select With Functions Github 
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Skipping postgres backend test due to unsupported function instr
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_GITHUB_SCIM_JOIN_WITH_FUNCTIONS} 
    ...    ${SELECT_GITHUB_SCIM_JOIN_WITH_FUNCTIONS_EXPECTED}
    ...    ${CURDIR}/tmp/Data-Flow-Sequential-Join-Select-With-Functions-Github.tmp

Page Limited Select Github 
    Should Stackql Exec Inline Equal Page Limited
    ...    ${STACKQL_EXE}
    ...    2
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_GITHUB_ORGS_MEMBERS} 
    ...    ${SELECT_GITHUB_ORGS_MEMBERS_PAGE_LIMITED_EXPECTED}
    ...    stdout=${CURDIR}/tmp/Page-Limited-Select-Github.tmp

Basic Query mTLS Returns OK
    Should PG Client Inline Contain
    ...    ${CURDIR}
    ...    ${PSQL_EXE}
    ...    ${PSQL_MTLS_CONN_STR}
    ...    ${SELECT_CONTAINER_SUBNET_AGG_ASC}
    ...    ipCidrRange

Basic Error Query mTLS Returns Error Message
    Should PG Client StdErr Inline Contain
    ...    ${CURDIR}
    ...    ${PSQL_EXE}
    ...    ${PSQL_MTLS_CONN_STR}
    ...    select fake_name from github.repos.branches where owner \= 'dummyorg' and repo \= 'dummyapp.io' order by name desc;
    ...    column
    ...    stdout=${CURDIR}/tmp/Basic-Error-Query-mTLS-Returns-Error-Message.tmp
    ...    stderr=${CURDIR}/tmp/Basic-Error-Query-mTLS-Returns-Error-Message-stderr.tmp


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

Basic View Returns Results
    Should Stackql Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    SELECT * FROM stackql_repositories ;
    ...    dummyapp.io
    ...    stdout=${CURDIR}/tmp/Basic-View-Returns-Results.tmp

Basic Subquery Returns Results
    Should Stackql Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    SELECT * FROM (select id, name, url from github.repos.repos where org \= 'stackql') some_alias ;
    ...    dummyapp.io
    ...    stdout=${CURDIR}/tmp/Basic-Subquery-Returns-Results.tmp

Basic View of Union Returns Results
    Should Stackql Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select aws_region, volumeId, encrypted, size from aws_ec2_all_volumes ;
    ...    sa\-east\-1
    ...    stdout=${CURDIR}/tmp/Basic-View-of-Union-Returns-Results.tmp

Basic View Select Star of Union Returns Results
    Should Stackql Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select * from aws_ec2_all_volumes ;
    ...    sa\-east\-1
    ...    stdout=${CURDIR}/tmp/Basic-View-Select-Star-of-Union-Returns-Results.tmp

Basic Count of View of Union Returns Expected Result
    Should Stackql Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select count(volumeId) as ct from aws_ec2_all_volumes ;
    ...    34
    ...    stdout=${CURDIR}/tmp/Basic-Count-of-View-of-Union-Returns-Expected-Result.tmp

Basic View of Cloud Control Resource Returns Expected Result
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Skipping postgres backend test likely due to case sensitivity and incorrect XML property aliasing
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}    
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select Arn, BucketName, DomainName from aws_cc_bucket_detail ;
    ...    ${VIEW_SELECT_AWS_CLOUD_CONTROL_BUCKET_DETAIL_EXPECTED}
    ...    ${CURDIR}/tmp/Basic-View-of-Cloud-Control-Resource-Returns-Expected-Result.tmp

Parameterized View of Cloud Control Resource Returns Expected Result
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Skipping postgres backend test likely due to case sensitivity and incorrect XML property aliasing
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}    
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select Arn, BucketName, DomainName from aws_cc_bucket_unfiltered where data__Identifier = 'stackql-trial-bucket-01' ;
    ...    ${VIEW_SELECT_AWS_CLOUD_CONTROL_BUCKET_DETAIL_EXPECTED}
    ...    ${CURDIR}/tmp/Parameterized-View-of-Cloud-Control-Resource-Returns-Expected-Result.tmp

Basic View Select Star of Cloud Control Resource Returns Expected Result
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Skipping postgres backend test likely due to case sensitivity and incorrect XML property aliasing
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}    
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select * from aws_cc_bucket_detail ;
    ...    ${VIEW_SELECT_STAR_AWS_CLOUD_CONTROL_BUCKET_DETAIL_EXPECTED}
    ...    ${CURDIR}/tmp/Basic-View-Select-Star-of-Cloud-Control-Resource-Returns-Expected-Result.tmp

Select Star of EC2 Instances Returns Expected Result
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}    
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select * from aws.ec2.instances where region \= 'us-east-1' ;
    ...    vol-1234567890abcdef0
    ...    stdout=${CURDIR}/tmp/Select-Star-of-EC2-Instances-Returns-Expected-Result.tmp

# This also tests passing integers in request body parameters
Select Projection of CloudWatch Log Events Returns Expected Result
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Skipping postgres backend test likely due to case sensitivity and incorrect XML property aliasing
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}    
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select timestamp, message from aws.cloud_watch.log_events where region \= 'ap-southeast-1' and data__logGroupName \= 'LogGroupResourceExample' and data__logStreamName \= 'test-01' and data__startTime \= 1680528971190 and data__limit \= 2 ;
    ...    some rubbish 02
    ...    stdout=${CURDIR}/tmp/Select-Projection-of-CloudWatch-Log-Events-Returns-Expected-Result.tmp

Postgres Casting query returns some non error result
    Pass Execution If    "${SQL_BACKEND}" != "postgres_tcp"    This is a dashboard query regression test for postgres backends only
    Run Stackql Exec Command No Errors
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}    
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${QUERY_PARSER_TEST_POSTGRES_CASTING}
    ...    stdout=${CURDIR}/tmp/Postgres-Casting-query-returns-some-non-error-result.tmp    

Keyword quoting query returns some non error result
    Pass Execution If    "${SQL_BACKEND}" != "postgres_tcp"    This is a dashboard query regression test for postgres backends only
    Run Stackql Exec Command No Errors
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}    
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${QUERY_PARSER_TEST_KEYWORD_QUOTING}
    ...    stdout=${CURDIR}/tmp/Keyword-Quoting-query-returns-some-non-error-result.tmp  

Parameterized View Select Star of Cloud Control Resource Returns Expected Result
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Skipping postgres backend test likely due to case sensitivity and incorrect XML property aliasing
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}    
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select * from aws_cc_bucket_unfiltered where data__Identifier = 'stackql-trial-bucket-01' ;
    ...    ${VIEW_SELECT_STAR_AWS_CLOUD_CONTROL_BUCKET_DETAIL_EXPECTED}
    ...    ${CURDIR}/tmp/Parameterized-View-Select-Star-of-Cloud-Control-Resource-Returns-Expected-Result.tmp

Projection of Resource Level View of Cloud Control Resource Returns Expected Result
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Skipping postgres backend test likely due to case sensitivity and incorrect XML property aliasing
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}    
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${AWS_CLOUD_CONTROL_BUCKET_VIEW_DETAIL_PROJECTION}
    ...    ${AWS_CLOUD_CONTROL_BUCKET_VIEW_DETAIL_PROJECTION_EXPECTED}
    ...    ${CURDIR}/tmp/Projection-of-Resource-Level-View-of-Cloud-Control-Resource-Returns-Expected-Result.tmp

Star of Resource Level View of Cloud Control Resource Returns Expected Result
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Skipping postgres backend test likely due to case sensitivity and incorrect XML property aliasing
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}    
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${AWS_CLOUD_CONTROL_BUCKET_VIEW_DETAIL_STAR}
    ...    ${AWS_CLOUD_CONTROL_BUCKET_VIEW_DETAIL_STAR_EXPECTED}
    ...    ${CURDIR}/tmp/Star-of-Resource-Level-View-of-Cloud-Control-Resource-Returns-Expected-Result.tmp

Filtered Projection Resource Level View of Cloud Control Resource Returns Expected Result
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Skipping postgres backend test likely due to case sensitivity and incorrect XML property aliasing
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}    
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select BucketName, DomainName from aws.pseudo_s3.s3_bucket_listing where region \= 'ap\-southeast\-2' and BucketName \= 'stackql\-trial\-bucket\-01';
    ...    ${AWS_CC_VIEW_SELECT_PROJECTION_BUCKET_FILTERED_EXPECTED}
    ...    ${CURDIR}/tmp/Filtered-Projection-Resource-Level-View-of-Cloud-Control-Resource-Returns-Expected-Result.tmp

Filtered Star Resource Level View of Cloud Control Resource Returns Expected Result
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Skipping postgres backend test likely due to case sensitivity and incorrect XML property aliasing
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}    
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select * from aws.pseudo_s3.s3_bucket_listing where region \= 'ap\-southeast\-2' and BucketName \= 'stackql\-trial\-bucket\-01';
    ...    ${AWS_CC_VIEW_SELECT_STAR_BUCKET_FILTERED_EXPECTED}
    ...    ${CURDIR}/tmp/Filtered-Star-Resource-Level-View-of-Cloud-Control-Resource-Returns-Expected-Result.tmp

Filtered and Parameterised Projection Resource Level View of Cloud Control Resource Returns Expected Result
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Skipping postgres backend test likely due to case sensitivity and incorrect XML property aliasing
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}    
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select BucketName, DomainName from aws.pseudo_s3.s3_bucket_listing where data__Identifier = 'stackql\-trial\-bucket\-01' and region \= 'ap\-southeast\-2' and BucketName \= 'stackql\-trial\-bucket\-01';
    ...    ${AWS_CC_VIEW_SELECT_PROJECTION_BUCKET_COMPLEX_EXPECTED}
    ...    ${CURDIR}/tmp/Filtered-and-Parameterised-Projection-Resource-Level-View-of-Cloud-Control-Resource-Returns-Expected-Result.tmp

Filtered and Parameterised Star Resource Level View of Cloud Control Resource Returns Expected Result
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Skipping postgres backend test likely due to case sensitivity and incorrect XML property aliasing
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}    
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select * from aws.pseudo_s3.s3_bucket_listing where data__Identifier \= 'stackql\-trial\-bucket\-01' and region \= 'ap\-southeast\-2' and BucketName \= 'stackql\-trial\-bucket\-01';
    ...    ${AWS_CC_VIEW_SELECT_STAR_BUCKET_COMPLEX_EXPECTED}
    ...    ${CURDIR}/tmp/Filtered-and-Parameterised-Star-Resource-Level-View-of-Cloud-Control-Resource-Returns-Expected-Result.tmp

Describe View of Cloud Control Resource Returns Expected Result
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    TODO: FIX THIS... Skipping postgres backend test likely due to case sensitivity and incorrect XML property aliasing
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}    
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    describe aws.pseudo_s3.s3_bucket_listing;
    ...    RestrictPublicBuckets
    ...    stdout=${CURDIR}/tmp/Describe-View-of-Cloud-Control-Resource-Returns-Expected-Result.tmp

View Depth Expanded Limitation Respected
    Should Stackql Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}    
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    create view zz1 as select name from stackql_repositories; create view zz2 as select name from zz1; create view zz3 as select name from zz2; create view zz4 as select name from zz3; select * from zz4;
    ...    dummyapp.io
    ...    stdout=${CURDIR}/tmp/View-Depth-Limitation-Upheld-stdout.tmp
    ...    stderr=${CURDIR}/tmp/View-Depth-Limitation-Upheld-stderr.tmp

View Depth Limitation Enforced
    Should Stackql Exec Inline Contain Stderr
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}    
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    create view zz1 as select name from stackql_repositories; create view zz2 as select name from zz1; create view zz3 as select name from zz2; create view zz4 as select name from zz3; create view zz5 as select name from zz4; select * from zz5;
    ...    please do not cite views at too deep a level
    ...    stdout=${CURDIR}/tmp/View-Depth-Limitation-Upheld-stdout.tmp
    ...    stderr=${CURDIR}/tmp/View-Depth-Limitation-Upheld-stderr.tmp

Weird ID WSL bug query
    # ID cannot be handled as integer on WSL
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}    
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_SUMOLOGIC_COLLECTORS_IDS}
    ...    ${SELECT_SUMOLOGIC_COLLECTORS_IDS_EXPECTED}
    ...    ${CURDIR}/tmp/Weird-ID-WSL-bug-query.tmp


HTTP Log enabled regression test
    Should Horrid HTTP Log Enabled Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}    
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${SELECT_SUMOLOGIC_COLLECTORS_IDS}
    ...    ${SELECT_SUMOLOGIC_COLLECTORS_IDS_EXPECTED}
    ...    ${CURDIR}/tmp/HTTP-Log-enabled-regression-test.tmp

External Postgres Data Source Simple Ordered Query
    Pass Execution If    "${SHOULD_RUN_DOCKER_EXTERNAL_TESTS}" != "true"    Skipping docker tests in uncertain environment
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_PLUS_EXTERNAL_POSTGRES}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select role_name from pgi.information_schema.applicable_roles order by role_name desc;
    ...    ${SELECT_EXTERNAL_INFORMATION_SCHEMA_ORDERED_EXPECTED}
    ...    ${CURDIR}/tmp/External-Postgres-Data-Source-Simple-Ordered-Query.tmp

External Postgres Data Source Simple Filtered Query
    Pass Execution If    "${SHOULD_RUN_DOCKER_EXTERNAL_TESTS}" != "true"    Skipping docker tests in uncertain environment
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_PLUS_EXTERNAL_POSTGRES}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select role_name from pgi.information_schema.applicable_roles where role_name \= 'pg_database_owner';
    ...    ${SELECT_EXTERNAL_INFORMATION_SCHEMA_FILTERED_EXPECTED}
    ...    ${CURDIR}/tmp/External-Postgres-Data-Source-Simple-Filtered-Query.tmp

External Postgres Data Source Self Join Ordered Query
    Pass Execution If    "${SHOULD_RUN_DOCKER_EXTERNAL_TESTS}" != "true"    Skipping docker tests in uncertain environment
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_PLUS_EXTERNAL_POSTGRES}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select r1.role_name from pgi.information_schema.applicable_roles r1 inner join pgi.information_schema.applicable_roles r2 on r1.role_name \= r2.role_name order by r1.role_name desc;
    ...    ${SELECT_EXTERNAL_INFORMATION_SCHEMA_ORDERED_EXPECTED}
    ...    ${CURDIR}/tmp/External-Postgres-Data-Source-Self-Join-Ordered-Query.tmp

External Postgres Data Source Inner Join Ordered Query
    Pass Execution If    "${SHOULD_RUN_DOCKER_EXTERNAL_TESTS}" != "true"    Skipping docker tests in uncertain environment
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_PLUS_EXTERNAL_POSTGRES}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select rtg.table_catalog, rtg.table_schema, rtg.table_name, rtg.privilege_type, rtg.is_grantable, ar.is_grantable as role_is_grantable from pgi.information_schema.role_table_grants rtg inner join pgi.information_schema.applicable_roles ar on rtg.grantee \= ar.grantee where rtg.table_name \= 'pg_statistic' order by privilege_type desc;
    ...    ${SELECT_EXTERNAL_INFORMATION_SCHEMA_INNER_JOIN_EXPECTED}
    ...    ${CURDIR}/tmp/External-Postgres-Data-Source-Inner-Join-Ordered-Query.tmp
