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
    Should StackQL Exec Inline Equal Stderr
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
    Should StackQL Exec Inline Equal Stderr
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
    Should StackQL Exec Inline Equal Stderr
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
    Should StackQL Exec Inline Equal Stderr
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
    Should StackQL Exec Inline Equal Stderr
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
    Should StackQL Exec Inline Equal Stderr
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

Split Part Simple Invocation Working
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |------------------------|---------------|----------------------------------------------------------------------------------------|----------------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}name${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}id${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}network${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}network_region${SPACE}|
    ...    |------------------------|---------------|----------------------------------------------------------------------------------------|----------------|
    ...    |${SPACE}selected-allow-rdesk${SPACE}${SPACE}${SPACE}|${SPACE}8888888888888${SPACE}|${SPACE}https://www.googleapis.com/compute/v1/projects/testing-project/global/networks/default${SPACE}|${SPACE}global${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------|---------------|----------------------------------------------------------------------------------------|----------------|
    ...    |${SPACE}default-allow-ssh${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}777777777777${SPACE}|${SPACE}https://www.googleapis.com/compute/v1/projects/testing-project/global/networks/default${SPACE}|${SPACE}global${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------|---------------|----------------------------------------------------------------------------------------|----------------|
    ...    |${SPACE}default-allow-rdp${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}6666666666${SPACE}|${SPACE}https://www.googleapis.com/compute/v1/projects/testing-project/global/networks/default${SPACE}|${SPACE}global${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------|---------------|----------------------------------------------------------------------------------------|----------------|
    ...    |${SPACE}default-allow-internal${SPACE}|${SPACE}5555555555555${SPACE}|${SPACE}https://www.googleapis.com/compute/v1/projects/testing-project/global/networks/default${SPACE}|${SPACE}global${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------|---------------|----------------------------------------------------------------------------------------|----------------|
    ...    |${SPACE}default-allow-icmp${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}4444444444444${SPACE}|${SPACE}https://www.googleapis.com/compute/v1/projects/testing-project/global/networks/default${SPACE}|${SPACE}global${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------|---------------|----------------------------------------------------------------------------------------|----------------|
    ...    |${SPACE}default-allow-https${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}33333333${SPACE}|${SPACE}https://www.googleapis.com/compute/v1/projects/testing-project/global/networks/default${SPACE}|${SPACE}global${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------|---------------|----------------------------------------------------------------------------------------|----------------|
    ...    |${SPACE}default-allow-http${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}22222222222${SPACE}|${SPACE}https://www.googleapis.com/compute/v1/projects/testing-project/global/networks/default${SPACE}|${SPACE}global${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------|---------------|----------------------------------------------------------------------------------------|----------------|
    ...    |${SPACE}allow-spark-ui${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}111111111111${SPACE}|${SPACE}https://www.googleapis.com/compute/v1/projects/testing-project/global/networks/default${SPACE}|${SPACE}global${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------|---------------|----------------------------------------------------------------------------------------|----------------|
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, id, network, split_part(network, '/', 8) as network_region from google.compute.firewalls where project \= 'testing-project' order by id desc;
    ...    ${outputStr}
    ...    ${CURDIR}/tmp/Split-Part-Simple-Invocation-Working.tmp

Split Part Negative Index Invocation Working
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |------------------------|---------------|----------------------------------------------------------------------------------------|----------------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}name${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}id${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}network${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}network_region${SPACE}|
    ...    |------------------------|---------------|----------------------------------------------------------------------------------------|----------------|
    ...    |${SPACE}selected-allow-rdesk${SPACE}${SPACE}${SPACE}|${SPACE}8888888888888${SPACE}|${SPACE}https://www.googleapis.com/compute/v1/projects/testing-project/global/networks/default${SPACE}|${SPACE}global${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------|---------------|----------------------------------------------------------------------------------------|----------------|
    ...    |${SPACE}default-allow-ssh${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}777777777777${SPACE}|${SPACE}https://www.googleapis.com/compute/v1/projects/testing-project/global/networks/default${SPACE}|${SPACE}global${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------|---------------|----------------------------------------------------------------------------------------|----------------|
    ...    |${SPACE}default-allow-rdp${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}6666666666${SPACE}|${SPACE}https://www.googleapis.com/compute/v1/projects/testing-project/global/networks/default${SPACE}|${SPACE}global${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------|---------------|----------------------------------------------------------------------------------------|----------------|
    ...    |${SPACE}default-allow-internal${SPACE}|${SPACE}5555555555555${SPACE}|${SPACE}https://www.googleapis.com/compute/v1/projects/testing-project/global/networks/default${SPACE}|${SPACE}global${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------|---------------|----------------------------------------------------------------------------------------|----------------|
    ...    |${SPACE}default-allow-icmp${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}4444444444444${SPACE}|${SPACE}https://www.googleapis.com/compute/v1/projects/testing-project/global/networks/default${SPACE}|${SPACE}global${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------|---------------|----------------------------------------------------------------------------------------|----------------|
    ...    |${SPACE}default-allow-https${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}33333333${SPACE}|${SPACE}https://www.googleapis.com/compute/v1/projects/testing-project/global/networks/default${SPACE}|${SPACE}global${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------|---------------|----------------------------------------------------------------------------------------|----------------|
    ...    |${SPACE}default-allow-http${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}22222222222${SPACE}|${SPACE}https://www.googleapis.com/compute/v1/projects/testing-project/global/networks/default${SPACE}|${SPACE}global${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------|---------------|----------------------------------------------------------------------------------------|----------------|
    ...    |${SPACE}allow-spark-ui${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}111111111111${SPACE}|${SPACE}https://www.googleapis.com/compute/v1/projects/testing-project/global/networks/default${SPACE}|${SPACE}global${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------|---------------|----------------------------------------------------------------------------------------|----------------|
    Should Horrid Query StackQL Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, id, network, split_part(network, '/', -3) as network_region from google.compute.firewalls where project \= 'testing-project' order by id desc;
    ...    ${outputStr}
    ...    ${CURDIR}/tmp/Split-Part-Negative-Index-Invocation-Working.tmp

Create Table Scenario Working
    ${inputStr} =    Catenate
    ...    create table phystab_one(t_id int, z text);
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    DDL Execution Completed
    Should Stackql Exec Inline Equal Both Streams
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    ${EMPTY}
    ...    ${outputStr}
    ...    stderr=${CURDIR}/tmp/Create-Table-Scenario-Working.tmp

Create Static Materialized View Scenario Working
    ${inputStr} =    Catenate
    ...    create materialized view mv_one as select 1 as one;
    ...    select * from mv_one;
    ...    drop materialized view mv_one;
    ...    select * from mv_one;
    ...    create materialized view mv_one as select 1 as one;
    ...    select * from mv_one;
    ...    refresh materialized view mv_one;
    ...    select * from mv_one;
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |-----|
    ...    |${SPACE}one${SPACE}|
    ...    |-----|
    ...    |${SPACE}${SPACE}${SPACE}1${SPACE}|
    ...    |-----|
    ...    |-----|
    ...    |${SPACE}one${SPACE}|
    ...    |-----|
    ...    |${SPACE}${SPACE}${SPACE}1${SPACE}|
    ...    |-----|
    ...    |-----|
    ...    |${SPACE}one${SPACE}|
    ...    |-----|
    ...    |${SPACE}${SPACE}${SPACE}1${SPACE}|
    ...    |-----|
    ${stdErrStr} =    Catenate    SEPARATOR=\n
    ...    DDL Execution Completed
    ...    DDL Execution Completed
    ...    could not locate table 'mv_one'
    ...    DDL Execution Completed
    ...    refresh materialized view completed
    Should Stackql Exec Inline Equal Both Streams
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    ${outputStr}
    ...    ${stdErrStr}
    ...    stdout=${CURDIR}/tmp/Create-Static-Materialized-Scenario-Working.tmp
    ...    stderr=${CURDIR}/tmp/Create-Static-Materialized-Scenario-Working-stderr.tmp

Create Dynamic Materialized View Scenario Working
    ${inputStr} =    Catenate
    ...    create materialized view silly_mv as select * from google.compute.firewalls where project = 'testing-project';
    ...    select name, id from silly_mv order by name desc, id desc;
    ...    drop materialized view silly_mv;
    ...    select name, id from silly_mv order by name desc, id desc;
    ...    create materialized view silly_mv as select * from google.compute.firewalls where project = 'testing-project';
    ...    select name, id from silly_mv order by name desc, id desc;
    ...    refresh materialized view silly_mv;
    ...    select name, id from silly_mv order by name desc, id desc;
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |------------------------|---------------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}name${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}id${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}selected-allow-rdesk${SPACE}${SPACE}${SPACE}|${SPACE}8888888888888${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-ssh${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}777777777777${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-rdp${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}6666666666${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-internal${SPACE}|${SPACE}5555555555555${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-icmp${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}4444444444444${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-https${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}33333333${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-http${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}22222222222${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}allow-spark-ui${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}111111111111${SPACE}|
    ...    |------------------------|---------------|
    ...    |------------------------|---------------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}name${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}id${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}selected-allow-rdesk${SPACE}${SPACE}${SPACE}|${SPACE}8888888888888${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-ssh${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}777777777777${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-rdp${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}6666666666${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-internal${SPACE}|${SPACE}5555555555555${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-icmp${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}4444444444444${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-https${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}33333333${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-http${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}22222222222${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}allow-spark-ui${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}111111111111${SPACE}|
    ...    |------------------------|---------------|
    ...    |------------------------|---------------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}name${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}id${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}selected-allow-rdesk${SPACE}${SPACE}${SPACE}|${SPACE}8888888888888${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-ssh${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}777777777777${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-rdp${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}6666666666${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-internal${SPACE}|${SPACE}5555555555555${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-icmp${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}4444444444444${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-https${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}33333333${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-http${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}22222222222${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}allow-spark-ui${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}111111111111${SPACE}|
    ...    |------------------------|---------------|
    ${stdErrStr} =    Catenate    SEPARATOR=\n
    ...    DDL Execution Completed
    ...    DDL Execution Completed
    ...    could not locate table 'silly_mv'
    ...    DDL Execution Completed
    ...    refresh materialized view completed
    Should Stackql Exec Inline Equal Both Streams
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    ${outputStr}
    ...    ${stdErrStr}
    ...    stdout=${CURDIR}/tmp/Create-Dynamic-Materialized-Scenario-Working.tmp
    ...    stderr=${CURDIR}/tmp/Create-Dynamic-Materialized-Scenario-Working-stderr.tmp

Create and Interrogate Materialized View With Aliasing and Name Collision
    ${inputStr} =    Catenate
    ...    create materialized view vw_aws_usr as select Arn, UserName, UserId, region from aws.iam.users where region = 'us-east-1';
    ...    select u1.UserName, u2.UserId, u2.Arn, u1.region from aws.iam.users u1 inner join vw_aws_usr u2 on u1.Arn = u2.Arn where u1.region = 'us-east-1' and u2.region = 'us-east-1' order by u1.UserName desc;
    ...    drop materialized view vw_aws_usr;
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |----------|-----------------------|--------------------------------------------------------------------------------|-----------|
    ...    |${SPACE}UserName${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}UserId${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}Arn${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}region${SPACE}${SPACE}${SPACE}|
    ...    |----------|-----------------------|--------------------------------------------------------------------------------|-----------|
    ...    |${SPACE}Jackie${SPACE}${SPACE}${SPACE}|${SPACE}AIDIODR4TAW7CSEXAMPLE${SPACE}|${SPACE}arn:aws:iam::123456789012:user/division_abc/subdivision_xyz/engineering/Jackie${SPACE}|${SPACE}us-east-1${SPACE}|
    ...    |----------|-----------------------|--------------------------------------------------------------------------------|-----------|
    ...    |${SPACE}Andrew${SPACE}${SPACE}${SPACE}|${SPACE}AID2MAB8DPLSRHEXAMPLE${SPACE}|${SPACE}arn:aws:iam::123456789012:user/division_abc/subdivision_xyz/engineering/Andrew${SPACE}|${SPACE}us-east-1${SPACE}|
    ...    |----------|-----------------------|--------------------------------------------------------------------------------|-----------|
    ${stdErrStr} =    Catenate    SEPARATOR=\n
    ...    DDL Execution Completed
    ...    DDL Execution Completed
    Should Stackql Exec Inline Equal Both Streams
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    ${outputStr}
    ...    ${stdErrStr}
    ...    stdout=${CURDIR}/tmp/Create-and-Interrogate-Materialized-View-With-Aliasing-and-Name-Collision.tmp
    ...    stderr=${CURDIR}/tmp/Create-and-Interrogate-Materialized-View-With-Aliasing-and-Name-Collision-stderr.tmp

Create and Interrogate Materialized View With Union
    ${inputStr} =    Catenate
    ...    create materialized view vw_aws_usr as select Arn, UserName, UserId, region from aws.iam.users where region = 'us-east-1' union all select 'prefixed' || Arn, UserName, 'prefixed' || UserId, region from aws.iam.users where region = 'us-east-1';
    ...    select * from vw_aws_usr order by Arn desc;
    ...    drop materialized view vw_aws_usr;
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |----------------------------------------------------------------------------------------|----------|-------------------------------|-----------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}Arn${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}UserName${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}UserId${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}region${SPACE}${SPACE}${SPACE}|
    ...    |----------------------------------------------------------------------------------------|----------|-------------------------------|-----------|
    ...    |${SPACE}prefixedarn:aws:iam::123456789012:user/division_abc/subdivision_xyz/engineering/Jackie${SPACE}|${SPACE}Jackie${SPACE}${SPACE}${SPACE}|${SPACE}prefixedAIDIODR4TAW7CSEXAMPLE${SPACE}|${SPACE}us-east-1${SPACE}|
    ...    |----------------------------------------------------------------------------------------|----------|-------------------------------|-----------|
    ...    |${SPACE}prefixedarn:aws:iam::123456789012:user/division_abc/subdivision_xyz/engineering/Andrew${SPACE}|${SPACE}Andrew${SPACE}${SPACE}${SPACE}|${SPACE}prefixedAID2MAB8DPLSRHEXAMPLE${SPACE}|${SPACE}us-east-1${SPACE}|
    ...    |----------------------------------------------------------------------------------------|----------|-------------------------------|-----------|
    ...    |${SPACE}arn:aws:iam::123456789012:user/division_abc/subdivision_xyz/engineering/Jackie${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}Jackie${SPACE}${SPACE}${SPACE}|${SPACE}AIDIODR4TAW7CSEXAMPLE${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}us-east-1${SPACE}|
    ...    |----------------------------------------------------------------------------------------|----------|-------------------------------|-----------|
    ...    |${SPACE}arn:aws:iam::123456789012:user/division_abc/subdivision_xyz/engineering/Andrew${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}Andrew${SPACE}${SPACE}${SPACE}|${SPACE}AID2MAB8DPLSRHEXAMPLE${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}us-east-1${SPACE}|
    ...    |----------------------------------------------------------------------------------------|----------|-------------------------------|-----------|
    ${stdErrStr} =    Catenate    SEPARATOR=\n
    ...    DDL Execution Completed
    ...    DDL Execution Completed
    Should Stackql Exec Inline Equal Both Streams
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    ${outputStr}
    ...    ${stdErrStr}
    ...    stdout=${CURDIR}/tmp/Create-and-Interrogate-Materialized-View-With-Union.tmp
    ...    stderr=${CURDIR}/tmp/Create-and-Interrogate-Materialized-View-With-Union-stderr.tmp

Create and Interrogate Materialized View With Parenthesized Select and Union
    ${inputStr} =    Catenate
    ...    create materialized view vw_aws_usr as (select Arn, UserName, UserId, region from aws.iam.users where region = 'us-east-1' union all select 'prefixed' || Arn, UserName, 'prefixed' || UserId, region from aws.iam.users where region = 'us-east-1');
    ...    select * from vw_aws_usr order by Arn desc;
    ...    drop materialized view vw_aws_usr;
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |----------------------------------------------------------------------------------------|----------|-------------------------------|-----------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}Arn${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}UserName${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}UserId${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}region${SPACE}${SPACE}${SPACE}|
    ...    |----------------------------------------------------------------------------------------|----------|-------------------------------|-----------|
    ...    |${SPACE}prefixedarn:aws:iam::123456789012:user/division_abc/subdivision_xyz/engineering/Jackie${SPACE}|${SPACE}Jackie${SPACE}${SPACE}${SPACE}|${SPACE}prefixedAIDIODR4TAW7CSEXAMPLE${SPACE}|${SPACE}us-east-1${SPACE}|
    ...    |----------------------------------------------------------------------------------------|----------|-------------------------------|-----------|
    ...    |${SPACE}prefixedarn:aws:iam::123456789012:user/division_abc/subdivision_xyz/engineering/Andrew${SPACE}|${SPACE}Andrew${SPACE}${SPACE}${SPACE}|${SPACE}prefixedAID2MAB8DPLSRHEXAMPLE${SPACE}|${SPACE}us-east-1${SPACE}|
    ...    |----------------------------------------------------------------------------------------|----------|-------------------------------|-----------|
    ...    |${SPACE}arn:aws:iam::123456789012:user/division_abc/subdivision_xyz/engineering/Jackie${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}Jackie${SPACE}${SPACE}${SPACE}|${SPACE}AIDIODR4TAW7CSEXAMPLE${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}us-east-1${SPACE}|
    ...    |----------------------------------------------------------------------------------------|----------|-------------------------------|-----------|
    ...    |${SPACE}arn:aws:iam::123456789012:user/division_abc/subdivision_xyz/engineering/Andrew${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}Andrew${SPACE}${SPACE}${SPACE}|${SPACE}AID2MAB8DPLSRHEXAMPLE${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}us-east-1${SPACE}|
    ...    |----------------------------------------------------------------------------------------|----------|-------------------------------|-----------|
    ${stdErrStr} =    Catenate    SEPARATOR=\n
    ...    DDL Execution Completed
    ...    DDL Execution Completed
    Should Stackql Exec Inline Equal Both Streams
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    ${outputStr}
    ...    ${stdErrStr}
    ...    stdout=${CURDIR}/tmp/Create-and-Interrogate-Materialized-View-With-Union.tmp
    ...    stderr=${CURDIR}/tmp/Create-and-Interrogate-Materialized-View-With-Union-stderr.tmp

Create Changing Dynamic Materialized View Scenario Working
    ${inputStr} =    Catenate
    ...    create materialized view silly_changing_mv as select * from google.compute.firewalls where project = 'changing-project';
    ...    select name, id from silly_changing_mv order by name desc, id desc;
    ...    drop materialized view silly_changing_mv;
    ...    select name, id from silly_changing_mv order by name desc, id desc;
    ...    create materialized view silly_changing_mv as select * from google.compute.firewalls where project = 'changing-project';
    ...    select name, id from silly_changing_mv order by name desc, id desc;
    ...    refresh materialized view silly_changing_mv;
    ...    select name, id from silly_changing_mv order by name desc, id desc;
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |------------------------|---------------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}name${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}id${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}selected-allow-rdesk${SPACE}${SPACE}${SPACE}|${SPACE}8888888888888${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-ssh${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}777777777777${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-rdp${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}6666666666${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-internal${SPACE}|${SPACE}5555555555555${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-icmp${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}4444444444444${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-https${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}33333333${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-http${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}22222222222${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}allow-spark-ui${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}111111111111${SPACE}|
    ...    |------------------------|---------------|
    ...    |------------------------|---------------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}name${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}id${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}selected-allow-rdesk${SPACE}${SPACE}${SPACE}|${SPACE}8888888888888${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-ssh${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}777777777777${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-rdp${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}6666666666${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-internal${SPACE}|${SPACE}5555555555555${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-icmp${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}4444444444444${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-https${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}33333333${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}default-allow-http${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}22222222222${SPACE}|
    ...    |------------------------|---------------|
    ...    |${SPACE}allow-spark-ui${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}111111111111${SPACE}|
    ...    |------------------------|---------------|
    ...    |--------------------------------|---------------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}name${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}id${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|---------------|
    ...    |${SPACE}altered-selected-allow-rdesk${SPACE}${SPACE}${SPACE}|${SPACE}8888888888888${SPACE}|
    ...    |--------------------------------|---------------|
    ...    |${SPACE}altered-default-allow-ssh${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}777777777777${SPACE}|
    ...    |--------------------------------|---------------|
    ...    |${SPACE}altered-default-allow-rdp${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}6666666666${SPACE}|
    ...    |--------------------------------|---------------|
    ...    |${SPACE}altered-default-allow-internal${SPACE}|${SPACE}5555555555555${SPACE}|
    ...    |--------------------------------|---------------|
    ...    |${SPACE}altered-default-allow-icmp${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}4444444444444${SPACE}|
    ...    |--------------------------------|---------------|
    ...    |${SPACE}altered-default-allow-https${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}33333333${SPACE}|
    ...    |--------------------------------|---------------|
    ...    |${SPACE}altered-default-allow-http${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}22222222222${SPACE}|
    ...    |--------------------------------|---------------|
    ...    |${SPACE}altered-allow-spark-ui${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}111111111111${SPACE}|
    ...    |--------------------------------|---------------|
    ${stdErrStr} =    Catenate    SEPARATOR=\n
    ...    DDL Execution Completed
    ...    DDL Execution Completed
    ...    could not locate table 'silly_changing_mv'
    ...    DDL Execution Completed
    ...    refresh materialized view completed
    Should Stackql Exec Inline Equal Both Streams
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    ${outputStr}
    ...    ${stdErrStr}
    ...    stdout=${CURDIR}/tmp/Create-Changing-Dynamic-Materialized-Scenario-Working.tmp
    ...    stderr=${CURDIR}/tmp/Create-Changing-Dynamic-Materialized-Scenario-Working-stderr.tmp

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
    Should Stackql Exec Inline Contain Stderr
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
    Should Stackql Exec Inline Equal Stderr
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    {"digitalocean": { "username_var": "DUMMY_DIGITALOCEAN_USERNAME", "password_var": "DUMMY_DIGITALOCEAN_PASSWORD", "type": "basic", "valuePrefix": "TOTALLY_CONTRIVED "}}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    INSERT INTO digitalocean.droplets.droplets ( data__name, data__region, data__size, data__image, data__backups, data__ipv6, data__monitoring, data__tags ) SELECT 'some.example.com', 'nyc3', 's-1vcpu-1gb', 'ubuntu-20-04-x64', true, true, true, '["env:prod", "web"]' ;
    ...    The operation was despatched successfully
    ...    stderr=${CURDIR}/tmp/Digitalocean-Insert-Droplet.tmp

Transaction Rollback Digitalocean Insert Droplet
    ${nativeOutputStr} =    Catenate    SEPARATOR=\n
    ...    OK
    ...    mutating statement queued
    ...    Rollback OK
    ${dockerOutputStr} =    Catenate    SEPARATOR=\n
    ...    Rollback OK
    ${outputStr} =    Set Variable If    "${EXECUTION_PLATFORM}" == "docker"     ${dockerOutputStr}    ${nativeOutputStr}
    Should Stackql Exec Inline Equal Stderr
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    {"digitalocean": { "username_var": "DUMMY_DIGITALOCEAN_USERNAME", "password_var": "DUMMY_DIGITALOCEAN_PASSWORD", "type": "basic", "valuePrefix": "TOTALLY_CONTRIVED "}}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    begin; INSERT INTO digitalocean.droplets.droplets ( data__name, data__region, data__size, data__image, data__backups, data__ipv6, data__monitoring, data__tags ) SELECT 'some.example.com', 'nyc3', 's-1vcpu-1gb', 'ubuntu-20-04-x64', true, true, true, '["env:prod", "web"]' ; rollback;
    ...    ${outputStr}
    ...    stderr=${CURDIR}/tmp/Digitalocean-Insert-Droplet.tmp

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
    ${nativeOutputStr} =    Catenate    SEPARATOR=\n
    ...    OK
    ...    mutating statement queued
    ...    mutating statement queued
    ...    mutating statement queued
    ...    insert over HTTP error: 500 Internal Server Error
    ...    UNDO required: Undo the insert on digitalocean.droplets.droplets
    ...    UNDO required: Undo the insert on digitalocean.droplets.droplets
    ${dockerOutputStr} =    Catenate    SEPARATOR=\n
    ...    UNDO required: Undo the insert on digitalocean.droplets.droplets
    ${outputStr} =    Set Variable If    "${EXECUTION_PLATFORM}" == "docker"     ${dockerOutputStr}    ${nativeOutputStr}
    Should Stackql Exec Inline Equal Stderr
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    {"digitalocean": { "username_var": "DUMMY_DIGITALOCEAN_USERNAME", "password_var": "DUMMY_DIGITALOCEAN_PASSWORD", "type": "basic", "valuePrefix": "TOTALLY_CONTRIVED "}}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    ${outputStr}
    ...    stderr=${CURDIR}/tmp/Transaction-Abort-Attempted-Commit-Digitalocean-Insert-Droplet.tmp

Transaction Rollback Eager Idealised Google Admin Directory User
    ${inputStr} =    Catenate
    ...    begin; 
    ...    insert into googleadmin.directory.users(data__primaryEmail)
    ...    values ('somejimbo@grubit.com');
    ...    rollback;
    ${nativeOutputStr} =    Catenate    SEPARATOR=\n
    ...    OK
    ...    The operation was despatched successfully
    ...    Rollback OK
    ${dockerOutputStr} =    Catenate    SEPARATOR=\n
    ...    Rollback OK
    ${outputStr} =    Set Variable If    "${EXECUTION_PLATFORM}" == "docker"     ${dockerOutputStr}    ${nativeOutputStr}
    Should Stackql Exec Inline Equal Stderr
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    {"digitalocean": { "username_var": "DUMMY_DIGITALOCEAN_USERNAME", "password_var": "DUMMY_DIGITALOCEAN_PASSWORD", "type": "basic", "valuePrefix": "TOTALLY_CONTRIVED "}}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    ${outputStr}
    ...    stackql_rollback_eager=True
    ...    stderr=${CURDIR}/tmp/Transaction-Rollback-Eager-Idealised-Google-Admin-Directory-User.tmp

Transaction Rollback Failure Eager Idealised Google Admin Directory User
    ${inputStr} =    Catenate
    ...    begin; 
    ...    insert into googleadmin.directory.users(data__primaryEmail)
    ...    values ('joeblow@grubit.com');
    ...    rollback;
    ${stderrOutputStr} =    Catenate    SEPARATOR=\n
    ...    OK
    ...    The operation was despatched successfully
    ...    undo over HTTP error: 404 Not Found
    ...    Rollback failed
    Should Stackql Exec Inline Equal Both Streams
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    {"digitalocean": { "username_var": "DUMMY_DIGITALOCEAN_USERNAME", "password_var": "DUMMY_DIGITALOCEAN_PASSWORD", "type": "basic", "valuePrefix": "TOTALLY_CONTRIVED "}}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    ${EMPTY}
    ...    ${stderrOutputStr}
    ...    stackql_rollback_eager=True
    ...    stderr=${CURDIR}/tmp/Transaction-Rollback-Failure-Eager-Idealised-Google-Admin-Directory-User.tmp

Recently Active Logic Multi Backend
    ${sqliteInputStr} =    Catenate
    ...    select 
    ...    UserName, 
    ...    CASE WHEN ( 
    ...      strftime('%Y-%m-%d %H:%M:%SZ', PasswordLastUsed) 
    ...      > ( datetime('now', '-20 days' ) ) ) 
    ...     then 'true' else 'false' end as active 
    ...    from aws.iam.users 
    ...    WHERE region = 'us-east-1' and PasswordLastUsed is not null
    ...    order by UserName asc;
    ${postgresInputStr} =    Catenate
    ...    select 
    ...    UserName,
    ...    CASE WHEN ( 
    ...      TO_TIMESTAMP(PasswordLastUsed, 'YYYY-MM-DDTHH:MI:SSZ') 
    ...      > (now() - interval '7 days' ) )
    ...     then 'true' else 'false' end as active 
    ...    from aws.iam.users 
    ...    WHERE region = 'us-east-1' and PasswordLastUsed is not null
    ...    order by UserName asc;
    ${inputStr} =    Set Variable If    "${SQL_BACKEND}" == "postgres_tcp"     ${postgresInputStr}    ${sqliteInputStr}
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |----------|--------|
    ...    |${SPACE}UserName${SPACE}|${SPACE}active${SPACE}|
    ...    |----------|--------|
    ...    |${SPACE}Andrew${SPACE}${SPACE}${SPACE}|${SPACE}false${SPACE}${SPACE}|
    ...    |----------|--------|
    ...    |${SPACE}Jackie${SPACE}${SPACE}${SPACE}|${SPACE}true${SPACE}${SPACE}${SPACE}|
    ...    |----------|--------|
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Recently-Active-Logic-Multi-Backend.tmp    

Server Parameter in Projection
    ${inputStr} =    Catenate
    ...    select UserName, region from aws.iam.users WHERE region = 'us-east-1' order by UserName desc;
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |----------|-----------|
    ...    |${SPACE}UserName${SPACE}|${SPACE}${SPACE}region${SPACE}${SPACE}${SPACE}|
    ...    |----------|-----------|
    ...    |${SPACE}Jackie${SPACE}${SPACE}${SPACE}|${SPACE}us-east-1${SPACE}|
    ...    |----------|-----------|
    ...    |${SPACE}Andrew${SPACE}${SPACE}${SPACE}|${SPACE}us-east-1${SPACE}|
    ...    |----------|-----------|
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Server-Parameter-in-Projection.tmp  

Server Parameter in Select Star
    ${inputStr} =    Catenate
    ...    select * from aws.ec2.volumes where region = 'ap-southeast-1' order by volumeId asc;
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |------------------|---------------|--------------------------|-----------|--------------|------|----------|--------------------|------------|----------------|------|------------|-----------|--------|------------|-----------------------|------------|
    ...    |${SPACE}AvailabilityZone${SPACE}|${SPACE}attachmentSet${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}createTime${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}encrypted${SPACE}|${SPACE}fastRestored${SPACE}|${SPACE}iops${SPACE}|${SPACE}kmsKeyId${SPACE}|${SPACE}multiAttachEnabled${SPACE}|${SPACE}outpostArn${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}region${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}size${SPACE}|${SPACE}snapshotId${SPACE}|${SPACE}${SPACE}status${SPACE}${SPACE}${SPACE}|${SPACE}tagSet${SPACE}|${SPACE}throughput${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}volumeId${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}volumeType${SPACE}|
    ...    |------------------|---------------|--------------------------|-----------|--------------|------|----------|--------------------|------------|----------------|------|------------|-----------|--------|------------|-----------------------|------------|
    ...    |${SPACE}null${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}2022-05-02T23:09:30.171Z${SPACE}|${SPACE}false${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}100${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}false${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}ap-southeast-1${SPACE}|${SPACE}${SPACE}${SPACE}10${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}available${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}vol-00100000000000000${SPACE}|${SPACE}gp2${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------|---------------|--------------------------|-----------|--------------|------|----------|--------------------|------------|----------------|------|------------|-----------|--------|------------|-----------------------|------------|
    ...    |${SPACE}null${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}2022-05-11T04:45:40.627Z${SPACE}|${SPACE}false${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}100${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}false${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}ap-southeast-1${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}8${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}available${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}vol-00200000000000000${SPACE}|${SPACE}gp2${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------|---------------|--------------------------|-----------|--------------|------|----------|--------------------|------------|----------------|------|------------|-----------|--------|------------|-----------------------|------------|
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Server-Parameter-in-Select-Star.tmp  

Left Outer Join Users
    ${sqliteInputStr} =    Catenate
    ...    select 
    ...    aid.UserName as aws_user_name
    ...    ,json_extract(gad.name, '$.fullName') as gcp_user_name
    ...    ,lower( substr(aid.UserName, 1, 5) ) as aws_fuzz_name 
    ...    ,lower( substr(json_extract(gad.name, '$.fullName'), 1, 5) ) as gcp_fuzz_name
    ...    from 
    ...      aws.iam.users aid 
    ...    LEFT OUTER JOIN 
    ...      googleadmin.directory.users gad 
    ...    ON 
    ...    lower(substr(aid.UserName, 1, 5) ) = lower(substr(json_extract(gad.name, '$.fullName'), 1, 5) ) 
    ...    WHERE 
    ...      aid.region = 'us-east-1' 
    ...    AND 
    ...      gad.domain = 'grubit.com'
    ...    ORDER BY 
    ...      aws_user_name DESC
    ...    ;
    ${postgresInputStr} =    Catenate
    ...    select 
    ...       aid.UserName as aws_user_name
    ...      ,json_extract_path_text(gad.name, 'fullName') as gcp_user_name
    ...      ,lower(substr(aid.UserName, 1, 5)) as aws_fuzz_name 
    ...      ,lower(substr(json_extract_path_text(gad.name, 'fullName'), 1, 5)) as gcp_fuzz_name
    ...    from 
    ...      aws.iam.users aid 
    ...    LEFT OUTER JOIN 
    ...      googleadmin.directory.users gad 
    ...    ON 
    ...      lower(substr(aid.UserName, 1, 5)) = lower(substr(json_extract_path_text(gad.name, 'fullName'), 1, 5)) 
    ...    WHERE 
    ...      aid.region = 'us-east-1' 
    ...    AND 
    ...      gad.domain = 'grubit.com'
    ...    ORDER BY 
    ...      aws_user_name DESC
    ...    ;
    ${inputStr} =    Set Variable If    "${SQL_BACKEND}" == "postgres_tcp"     ${postgresInputStr}    ${sqliteInputStr}
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |---------------|----------------|---------------|---------------|
    ...    |${SPACE}aws_user_name${SPACE}|${SPACE}gcp_user_name${SPACE}${SPACE}|${SPACE}aws_fuzz_name${SPACE}|${SPACE}gcp_fuzz_name${SPACE}|
    ...    |---------------|----------------|---------------|---------------|
    ...    |${SPACE}Jackie${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}Jackie${SPACE}Citizen${SPACE}|${SPACE}jacki${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}jacki${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |---------------|----------------|---------------|---------------|
    ...    |${SPACE}Andrew${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}andre${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |---------------|----------------|---------------|---------------|
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Left-Outer-Join-Users.tmp

Left Outer Join Network Infra
    ${inputStr} =    Catenate
    ...    select 
    ...    nw.name as network_name, 
    ...    sn.name as subnetwork_name, 
    ...    split_part(sn.network, '/', 10) as sn_fuzz  
    ...    from 
    ...    google.compute.networks nw 
    ...    LEFT OUTER JOIN 
    ...    google.compute.subnetworks sn  
    ...    on 
    ...    lower(nw.name) = lower(split_part(sn.network, '/', 10))    
    ...    where nw.project = 'testing-project' and sn.region = 'australia-southeast1' 
    ...    and 
    ...    sn.project = 'testing-project' 
    ...    order by 
    ...    network_name, subnetwork_name
    ...    ;
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |------------------------------|-----------------|---------------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}network_name${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}subnetwork_name${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}sn_fuzz${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------------|-----------------|---------------|
    ...    |${SPACE}demo-disk-xx5${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}demo-disk-xx5${SPACE}${SPACE}${SPACE}|${SPACE}demo-disk-xx5${SPACE}|
    ...    |------------------------------|-----------------|---------------|
    ...    |${SPACE}k8s-01-vpc${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------------|-----------------|---------------|
    ...    |${SPACE}kr-vpc-01${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}aus-sn-01${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}kr-vpc-01${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------------|-----------------|---------------|
    ...    |${SPACE}kr-vpc-01${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}aus-sn-02${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}kr-vpc-01${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------------|-----------------|---------------|
    ...    |${SPACE}kubernetes-the-hard-way-vpc2${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------------|-----------------|---------------|
    ...    |${SPACE}testing-network-01${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |------------------------------|-----------------|---------------|
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Left-Outer-Join-Network-Infra.tmp

Left Inner Join Users
    ${sqliteInputStr} =    Catenate
    ...    select 
    ...    aid.UserName as aws_user_name
    ...    ,json_extract(gad.name, '$.fullName') as gcp_user_name
    ...    ,lower( substr(aid.UserName, 1, 5) ) as aws_fuzz_name 
    ...    ,lower( substr(json_extract(gad.name, '$.fullName'), 1, 5) ) as gcp_fuzz_name
    ...    from 
    ...      aws.iam.users aid 
    ...    JOIN 
    ...      googleadmin.directory.users gad 
    ...    ON 
    ...    lower(substr(aid.UserName, 1, 5) ) = lower(substr(json_extract(gad.name, '$.fullName'), 1, 5) ) 
    ...    WHERE 
    ...      aid.region = 'us-east-1' 
    ...    AND 
    ...      gad.domain = 'grubit.com'
    ...    ORDER BY 
    ...      aws_user_name DESC
    ...    ;
    ${postgresInputStr} =    Catenate
    ...    select 
    ...       aid.UserName as aws_user_name
    ...      ,json_extract_path_text(gad.name, 'fullName') as gcp_user_name
    ...      ,lower(substr(aid.UserName, 1, 5)) as aws_fuzz_name 
    ...      ,lower(substr(json_extract_path_text(gad.name, 'fullName'), 1, 5)) as gcp_fuzz_name
    ...    from 
    ...      aws.iam.users aid 
    ...    JOIN 
    ...      googleadmin.directory.users gad 
    ...    ON 
    ...      lower(substr(aid.UserName, 1, 5)) = lower(substr(json_extract_path_text(gad.name, 'fullName'), 1, 5)) 
    ...    WHERE 
    ...      aid.region = 'us-east-1' 
    ...    AND 
    ...      gad.domain = 'grubit.com'
    ...    ORDER BY 
    ...      aws_user_name DESC
    ...    ;
    ${inputStr} =    Set Variable If    "${SQL_BACKEND}" == "postgres_tcp"     ${postgresInputStr}    ${sqliteInputStr}
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |---------------|----------------|---------------|---------------|
    ...    |${SPACE}aws_user_name${SPACE}|${SPACE}gcp_user_name${SPACE}${SPACE}|${SPACE}aws_fuzz_name${SPACE}|${SPACE}gcp_fuzz_name${SPACE}|
    ...    |---------------|----------------|---------------|---------------|
    ...    |${SPACE}Jackie${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}Jackie${SPACE}Citizen${SPACE}|${SPACE}jacki${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}jacki${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |---------------|----------------|---------------|---------------|
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Left-Inner-Join-Users.tmp

Google Admin Directory Small Response Also De Facto Credentials Path Env Var
    Set Environment Variable    GOOGLE_APPLICATION_CREDENTIALS    ${GOOGLE_APPLICATION_CREDENTIALS}
    ${inputStr} =    Catenate
    ...    select 
    ...    json_extract(name, '$.fullName') as fullName, 
    ...    primaryEmail, 
    ...    isAdmin 
    ...    from googleadmin.directory.users 
    ...    where domain = 'grubit.com'
    ...    order by primaryEmail desc;
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |----------------|--------------------------|---------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}fullName${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}primaryEmail${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}isAdmin${SPACE}|
    ...    |----------------|--------------------------|---------|
    ...    |${SPACE}Joe${SPACE}Blow${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}joeblow@grubit.com${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}true${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |----------------|--------------------------|---------|
    ...    |${SPACE}Jackie${SPACE}Citizen${SPACE}|${SPACE}jackiecitizen@grubit.com${SPACE}|${SPACE}true${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |----------------|--------------------------|---------|
    ...    |${SPACE}Info${SPACE}Contact${SPACE}${SPACE}${SPACE}|${SPACE}info@grubit.com${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}false${SPACE}${SPACE}${SPACE}|
    ...    |----------------|--------------------------|---------|
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
    ...    stdout=${CURDIR}/tmp/Google-Admin-Directory-Small-Response-Also-De-Facto-Credentials-Path-Env-Var.tmp

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

Aggregated List JSON Path on additionalProperties Verify 
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |---------------------|------------------|---------------------------------------------------------------------------------------------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}name${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}id${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}zone${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |---------------------|------------------|---------------------------------------------------------------------------------------------|
    ...    |${SPACE}testing-project-014${SPACE}|${SPACE}1000000000000006${SPACE}|${SPACE}https://www.googleapis.com/compute/v1/projects/testing-project/zones/australia-southeast1-a${SPACE}|
    ...    |---------------------|------------------|---------------------------------------------------------------------------------------------|
    ...    |${SPACE}testing-project-013${SPACE}|${SPACE}1000000000000005${SPACE}|${SPACE}https://www.googleapis.com/compute/v1/projects/testing-project/zones/australia-southeast1-a${SPACE}|
    ...    |---------------------|------------------|---------------------------------------------------------------------------------------------|
    ...    |${SPACE}testing-project-004${SPACE}|${SPACE}1000000000000004${SPACE}|${SPACE}https://www.googleapis.com/compute/v1/projects/testing-project/zones/australia-southeast1-a${SPACE}|
    ...    |---------------------|------------------|---------------------------------------------------------------------------------------------|
    ...    |${SPACE}testing-project-003${SPACE}|${SPACE}1000000000000003${SPACE}|${SPACE}https://www.googleapis.com/compute/v1/projects/testing-project/zones/australia-southeast1-a${SPACE}|
    ...    |---------------------|------------------|---------------------------------------------------------------------------------------------|
    ...    |${SPACE}testing-project-002${SPACE}|${SPACE}1000000000000002${SPACE}|${SPACE}https://www.googleapis.com/compute/v1/projects/testing-project/zones/australia-southeast1-a${SPACE}|
    ...    |---------------------|------------------|---------------------------------------------------------------------------------------------|
    ...    |${SPACE}testing-project-001${SPACE}|${SPACE}1000000000000001${SPACE}|${SPACE}https://www.googleapis.com/compute/v1/projects/testing-project/zones/australia-southeast1-a${SPACE}|
    ...    |---------------------|------------------|---------------------------------------------------------------------------------------------|
    ...    |${SPACE}instance-1${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}222222222222${SPACE}|${SPACE}https://www.googleapis.com/compute/v1/projects/testing-project/zones/australia-southeast1-b${SPACE}|
    ...    |---------------------|------------------|---------------------------------------------------------------------------------------------|
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, id, zone from google.compute.instances where project \= 'testing-project' order by name desc;
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Aggregated-List-JSON-Path-on-additionalProperties-Verify.tmp

Google Asset List Aggregate Verify 
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |------------------------------------------------|-------------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}assetType${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}asset_count${SPACE}|
    ...    |------------------------------------------------|-------------|
    ...    |${SPACE}compute.googleapis.com/Route${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}43${SPACE}|
    ...    |------------------------------------------------|-------------|
    ...    |${SPACE}serviceusage.googleapis.com/Service${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}40${SPACE}|
    ...    |------------------------------------------------|-------------|
    ...    |${SPACE}compute.googleapis.com/Subnetwork${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}38${SPACE}|
    ...    |------------------------------------------------|-------------|
    ...    |${SPACE}iam.googleapis.com/ServiceAccountKey${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}12${SPACE}|
    ...    |------------------------------------------------|-------------|
    ...    |${SPACE}storage.googleapis.com/Bucket${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}7${SPACE}|
    ...    |------------------------------------------------|-------------|
    ...    |${SPACE}compute.googleapis.com/Instance${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}7${SPACE}|
    ...    |------------------------------------------------|-------------|
    ...    |${SPACE}compute.googleapis.com/Firewall${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}7${SPACE}|
    ...    |------------------------------------------------|-------------|
    ...    |${SPACE}compute.googleapis.com/Disk${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}7${SPACE}|
    ...    |------------------------------------------------|-------------|
    ...    |${SPACE}iam.googleapis.com/ServiceAccount${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}6${SPACE}|
    ...    |------------------------------------------------|-------------|
    ...    |${SPACE}pubsub.googleapis.com/Topic${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}4${SPACE}|
    ...    |------------------------------------------------|-------------|
    ...    |${SPACE}compute.googleapis.com/Network${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}4${SPACE}|
    ...    |------------------------------------------------|-------------|
    ...    |${SPACE}cloudkms.googleapis.com/CryptoKeyVersion${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}4${SPACE}|
    ...    |------------------------------------------------|-------------|
    ...    |${SPACE}logging.googleapis.com/LogSink${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}2${SPACE}|
    ...    |------------------------------------------------|-------------|
    ...    |${SPACE}logging.googleapis.com/LogBucket${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}2${SPACE}|
    ...    |------------------------------------------------|-------------|
    ...    |${SPACE}cloudkms.googleapis.com/KeyRing${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}2${SPACE}|
    ...    |------------------------------------------------|-------------|
    ...    |${SPACE}cloudkms.googleapis.com/CryptoKey${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}2${SPACE}|
    ...    |------------------------------------------------|-------------|
    ...    |${SPACE}firestore.googleapis.com/Database${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}1${SPACE}|
    ...    |------------------------------------------------|-------------|
    ...    |${SPACE}compute.googleapis.com/Project${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}1${SPACE}|
    ...    |------------------------------------------------|-------------|
    ...    |${SPACE}compute.googleapis.com/HealthCheck${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}1${SPACE}|
    ...    |------------------------------------------------|-------------|
    ...    |${SPACE}cloudresourcemanager.googleapis.com/Project${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}1${SPACE}|
    ...    |------------------------------------------------|-------------|
    ...    |${SPACE}cloudbilling.googleapis.com/ProjectBillingInfo${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}1${SPACE}|
    ...    |------------------------------------------------|-------------|
    ...    |${SPACE}bigquery.googleapis.com/Dataset${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}1${SPACE}|
    ...    |------------------------------------------------|-------------|
    ...    |${SPACE}appengine.googleapis.com/Application${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}1${SPACE}|
    ...    |------------------------------------------------|-------------|
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    SELECT assetType, count(*) as asset_count FROM google.cloudasset.assets WHERE parentType \= 'projects' and parent \= 'testing-project' GROUP BY assetType order by count(*) desc, assetType desc;
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Google-Asset-List-Aggregate-Verify.tmp

Transaction Commit Eager Show and Lazy Digitalocean Insert Droplet
    Should Stackql Exec Inline Equal Both Streams
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    {"digitalocean": { "username_var": "DUMMY_DIGITALOCEAN_USERNAME", "password_var": "DUMMY_DIGITALOCEAN_PASSWORD", "type": "basic", "valuePrefix": "TOTALLY_CONTRIVED "}}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    begin; INSERT INTO digitalocean.droplets.droplets ( data__name, data__region, data__size, data__image, data__backups, data__ipv6, data__monitoring, data__tags ) SELECT 'some.example.com', 'nyc3', 's-1vcpu-1gb', 'ubuntu-20-04-x64', true, true, true, '["env:prod", "web"]' ; show services in digitalocean like 'droplets'; commit;
    ...    |-----------------------|----------|-----------------------------|\n|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}id${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}name${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}title${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|\n|-----------------------|----------|-----------------------------|\n|${SPACE}droplets:v23.03.00127${SPACE}|${SPACE}droplets${SPACE}|${SPACE}DigitalOcean${SPACE}API${SPACE}-${SPACE}Droplets${SPACE}|\n|-----------------------|----------|-----------------------------|
    ...    OK\nmutating${SPACE}statement${SPACE}queued\nThe${SPACE}operation${SPACE}was${SPACE}despatched${SPACE}successfully\nOK
    ...    stdout=${CURDIR}/tmp/Digitalocean-Insert-Droplet.tmp

Registry Pull Google Provider Specific Version Prerelease
    Should Stackql Exec Inline Contain Stderr
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
    Should Stackql Exec Inline Contain Stderr
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

Basic Count Star From View Returns Expected Result
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |----------|
    ...    |${SPACE}count(*)${SPACE}|
    ...    |----------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}3${SPACE}|
    ...    |----------|
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    SELECT count(*) FROM stackql_repositories ;
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Basic-Count-Star-From-View-Returns-Expected-Result.tmp

Basic Aliased Count Star From View Returns Expected Result
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |------------------|
    ...    |${SPACE}repository_count${SPACE}|
    ...    |------------------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}3${SPACE}|
    ...    |------------------|
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    SELECT count(*) as repository_count FROM stackql_repositories ;
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Basic-Aliased-Count-Star-From-View-Returns-Expected-Result.tmp

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


Select Expression Function Expression Alias Reference Alongside Wildcard Returns Results
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    This is a genuine case of difference. Postgres does not support aliased colummns in where clauses.
    ${inputStr} =    CATENATE    select *, JSON_EXTRACT(sourceRanges, '$[0]') sr from google.compute.firewalls where project = 'testing-project' and sr = '0.0.0.0/0';
    Should Stackql Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select *, JSON_EXTRACT(sourceRanges, '$[0]') sr from google.compute.firewalls where project \= 'testing-project' and sr \= '0.0.0.0/0';
    ...    default-allow-ssh
    ...    stdout=${CURDIR}/tmp/Select-Expression-Function-Expression-Alias-Reference-Alongside-Wildcard-Returns-Results.tmp

Select Expression Function Expression Alias Reference Alongside Projection Returns Expected Results
    Pass Execution If    "${SQL_BACKEND}" == "postgres_tcp"    This is a genuine case of difference. Postgres does not support aliased colummns in where clauses.
    ${inputStr} =    Catenate
    ...    select name, direction, denied, allowed, JSON_EXTRACT(sourceRanges, '$[0]') sr  
    ...    from google.compute.firewalls 
    ...    where project = 'testing-project' and sr = '0.0.0.0/0' and denied is null and allowed is not null 
    ...    order by name desc;
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |---------------------|-----------|--------|-----------------------------------------|-----------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}name${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}direction${SPACE}|${SPACE}denied${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}allowed${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}sr${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |---------------------|-----------|--------|-----------------------------------------|-----------|
    ...    |${SPACE}default-allow-ssh${SPACE}${SPACE}${SPACE}|${SPACE}INGRESS${SPACE}${SPACE}${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}|${SPACE}\[{"IPProtocol":"tcp","ports":["22"]}]${SPACE}${SPACE}${SPACE}|${SPACE}0.0.0.0/0${SPACE}|
    ...    |---------------------|-----------|--------|-----------------------------------------|-----------|
    ...    |${SPACE}default-allow-rdp${SPACE}${SPACE}${SPACE}|${SPACE}INGRESS${SPACE}${SPACE}${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}|${SPACE}\[{"IPProtocol":"tcp","ports":["3389"]}]${SPACE}|${SPACE}0.0.0.0/0${SPACE}|
    ...    |---------------------|-----------|--------|-----------------------------------------|-----------|
    ...    |${SPACE}default-allow-icmp${SPACE}${SPACE}|${SPACE}INGRESS${SPACE}${SPACE}${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}|${SPACE}\[{"IPProtocol":"icmp"}]${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}0.0.0.0/0${SPACE}|
    ...    |---------------------|-----------|--------|-----------------------------------------|-----------|
    ...    |${SPACE}default-allow-https${SPACE}|${SPACE}INGRESS${SPACE}${SPACE}${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}|${SPACE}\[{"IPProtocol":"tcp","ports":["443"]}]${SPACE}${SPACE}|${SPACE}0.0.0.0/0${SPACE}|
    ...    |---------------------|-----------|--------|-----------------------------------------|-----------|
    ...    |${SPACE}default-allow-http${SPACE}${SPACE}|${SPACE}INGRESS${SPACE}${SPACE}${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}|${SPACE}\[{"IPProtocol":"tcp","ports":["80"]}]${SPACE}${SPACE}${SPACE}|${SPACE}0.0.0.0/0${SPACE}|
    ...    |---------------------|-----------|--------|-----------------------------------------|-----------|
    ...    |${SPACE}allow-spark-ui${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}INGRESS${SPACE}${SPACE}${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}|${SPACE}\[{"IPProtocol":"tcp","ports":["4040"]}]${SPACE}|${SPACE}0.0.0.0/0${SPACE}|
    ...    |---------------------|-----------|--------|-----------------------------------------|-----------|
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Select-Expression-Function-Expression-Alias-Reference-Alongside-Projection-Returns-Results.tmp

Table Valued Function Plus Projection Returns Expected Results
    ${sqliteInputStr} =    Catenate
    ...    select fw.id, fw.name, json_each.value as source_range, json_each.value = '0.0.0.0/0' as is_entire_network 
    ...    from google.compute.firewalls fw, json_each(sourceRanges) 
    ...    where project = 'testing-project' 
    ...    order by name desc, source_range desc;
    ${postgresInputStr} =    Catenate
    ...    select fw.id, fw.name, rd.value as source_range, case when rd.value = '0.0.0.0/0' then 1 else 0 end as is_entire_network 
    ...    from google.compute.firewalls fw, json_array_elements_text(sourceRanges) as rd
    ...    where project = 'testing-project' 
    ...    order by name desc, source_range desc;
    ${inputStr} =    Set Variable If    "${SQL_BACKEND}" == "postgres_tcp"     ${postgresInputStr}    ${sqliteInputStr}
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |---------------|------------------------|--------------|-------------------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}id${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}name${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}source_range${SPACE}|${SPACE}is_entire_network${SPACE}|
    ...    |---------------|------------------------|--------------|-------------------|
    ...    |${SPACE}8888888888888${SPACE}|${SPACE}selected-allow-rdesk${SPACE}${SPACE}${SPACE}|${SPACE}10.128.0.0/9${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}0${SPACE}|
    ...    |---------------|------------------------|--------------|-------------------|
    ...    |${SPACE}8888888888888${SPACE}|${SPACE}selected-allow-rdesk${SPACE}${SPACE}${SPACE}|${SPACE}10.0.0.0/16${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}0${SPACE}|
    ...    |---------------|------------------------|--------------|-------------------|
    ...    |${SPACE}${SPACE}777777777777${SPACE}|${SPACE}default-allow-ssh${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}0.0.0.0/0${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}1${SPACE}|
    ...    |---------------|------------------------|--------------|-------------------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}6666666666${SPACE}|${SPACE}default-allow-rdp${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}0.0.0.0/0${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}1${SPACE}|
    ...    |---------------|------------------------|--------------|-------------------|
    ...    |${SPACE}5555555555555${SPACE}|${SPACE}default-allow-internal${SPACE}|${SPACE}10.128.0.0/9${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}0${SPACE}|
    ...    |---------------|------------------------|--------------|-------------------|
    ...    |${SPACE}4444444444444${SPACE}|${SPACE}default-allow-icmp${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}0.0.0.0/0${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}1${SPACE}|
    ...    |---------------|------------------------|--------------|-------------------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}33333333${SPACE}|${SPACE}default-allow-https${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}0.0.0.0/0${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}1${SPACE}|
    ...    |---------------|------------------------|--------------|-------------------|
    ...    |${SPACE}${SPACE}${SPACE}22222222222${SPACE}|${SPACE}default-allow-http${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}0.0.0.0/0${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}1${SPACE}|
    ...    |---------------|------------------------|--------------|-------------------|
    ...    |${SPACE}${SPACE}111111111111${SPACE}|${SPACE}allow-spark-ui${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}0.0.0.0/0${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}1${SPACE}|
    ...    |---------------|------------------------|--------------|-------------------|
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Table-Valued-Function-Plus-Projection-Returns-Expected-Results.tmp

Embedded Materialized View Projection Returns Expected Results
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |--------------------------------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}gossip${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|
    ...    |${SPACE}stackql${SPACE}is${SPACE}open${SPACE}to${SPACE}extension${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|
    ...    |${SPACE}stackql${SPACE}is${SPACE}not${SPACE}opinionated${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|
    ...    |${SPACE}stackql${SPACE}wants${SPACE}to${SPACE}hear${SPACE}from${SPACE}you${SPACE}|
    ...    |--------------------------------|
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select gossip from stackql_gossip order by category desc;
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Embedded-Materialized-View-Projection-Returns-Expected-Results.tmp

Embedded Materialized View Star Returns Expected Results
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |--------------------------------|-----------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}gossip${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}category${SPACE}${SPACE}|
    ...    |--------------------------------|-----------|
    ...    |${SPACE}stackql${SPACE}is${SPACE}open${SPACE}to${SPACE}extension${SPACE}${SPACE}${SPACE}|${SPACE}tech${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|-----------|
    ...    |${SPACE}stackql${SPACE}is${SPACE}not${SPACE}opinionated${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}opinion${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|-----------|
    ...    |${SPACE}stackql${SPACE}wants${SPACE}to${SPACE}hear${SPACE}from${SPACE}you${SPACE}|${SPACE}community${SPACE}|
    ...    |--------------------------------|-----------|
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select * from stackql_gossip order by category desc;
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Embedded-Materialized-View-Star-Returns-Expected-Results.tmp

Embedded Table Projection Returns Expected Results
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |--------------------------------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}note${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|
    ...    |${SPACE}v0.5.418${SPACE}introduced${SPACE}table${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |${SPACE}valued${SPACE}functions,${SPACE}for${SPACE}example${SPACE}${SPACE}|
    ...    |${SPACE}json_each.${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|
    ...    |${SPACE}stackql${SPACE}supports${SPACE}the${SPACE}postgres${SPACE}${SPACE}|
    ...    |${SPACE}wire${SPACE}protocol.${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select note from stackql_notes order by priority desc;
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Embedded-Table-Projection-Returns-Expected-Results.tmp

Embedded Table Star Returns Expected Results
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |--------------------------------|----------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}note${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}priority${SPACE}|
    ...    |--------------------------------|----------|
    ...    |${SPACE}v0.5.418${SPACE}introduced${SPACE}table${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}1000${SPACE}|
    ...    |${SPACE}valued${SPACE}functions,${SPACE}for${SPACE}example${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |${SPACE}json_each.${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|----------|
    ...    |${SPACE}stackql${SPACE}supports${SPACE}the${SPACE}postgres${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}10${SPACE}|
    ...    |${SPACE}wire${SPACE}protocol.${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|----------|
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select * from stackql_notes order by priority desc;
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Embedded-Table-Star-Returns-Expected-Results.tmp

Embedded Table Join Materialized View Projection Returns Expected Results
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |--------------------------------|------------------------------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}note${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}gossip${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|------------------------------|
    ...    |${SPACE}v0.5.418${SPACE}introduced${SPACE}table${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}stackql${SPACE}is${SPACE}not${SPACE}opinionated${SPACE}${SPACE}${SPACE}|
    ...    |${SPACE}valued${SPACE}functions,${SPACE}for${SPACE}example${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |${SPACE}json_each.${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|------------------------------|
    ...    |${SPACE}stackql${SPACE}supports${SPACE}the${SPACE}postgres${SPACE}${SPACE}|${SPACE}stackql${SPACE}is${SPACE}open${SPACE}to${SPACE}extension${SPACE}|
    ...    |${SPACE}wire${SPACE}protocol.${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|------------------------------|
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select note, gossip from stackql_notes sn inner join stackql_gossip sg on case when sn.priority \= 1000 then 'opinion' else 'tech' end \= sg.category order by sn.priority desc;
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Embedded-Table-Join-Materialized-View-Projection-Returns-Expected-Results.tmp

Embedded Table Join Materialized View Aliased Projection Returns Expected Results
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |--------------------------------|------------------------------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}n${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}g${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|------------------------------|
    ...    |${SPACE}v0.5.418${SPACE}introduced${SPACE}table${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}stackql${SPACE}is${SPACE}not${SPACE}opinionated${SPACE}${SPACE}${SPACE}|
    ...    |${SPACE}valued${SPACE}functions,${SPACE}for${SPACE}example${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |${SPACE}json_each.${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|------------------------------|
    ...    |${SPACE}stackql${SPACE}supports${SPACE}the${SPACE}postgres${SPACE}${SPACE}|${SPACE}stackql${SPACE}is${SPACE}open${SPACE}to${SPACE}extension${SPACE}|
    ...    |${SPACE}wire${SPACE}protocol.${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|------------------------------|
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select note as n, gossip as g from stackql_notes sn inner join stackql_gossip sg on case when sn.priority \= 1000 then 'opinion' else 'tech' end \= sg.category order by sn.priority desc;
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Embedded-Table-Join-Materialized-View-Aliased-Projection-Returns-Expected-Results.tmp

Complex Dynamic and Embedded Static Join Returns Expected Results
    ${sqliteInputStr} =    Catenate
    ...    select 
    ...    fw.id, 
    ...    fw.name, 
    ...    json_each.value as source_range, 
    ...    json_each.value = '0.0.0.0/0' as is_permissive,
    ...    note,
    ...    gossip
    ...    from google.compute.firewalls fw
    ...    inner join stackql_notes sn 
    ...    on case when json_each.value = '0.0.0.0/0' then 10 else 1000 end = sn.priority 
    ...    inner join stackql_gossip sg 
    ...    on case when sn.priority = 1000 then 'opinion' else 'tech' end = sg.category
    ...    , json_each(sourceRanges) 
    ...    where project = 'testing-project'
    ...    order by name desc, source_range desc
    ...    ;
    ${postgresInputStr} =    Catenate
    ...    select 
    ...   fw.id, 
    ...   fw.name, 
    ...   fw.source_range as source_range, 
    ...   case when fw.source_range = '0.0.0.0/0' then 1 else 0 end as is_permissive,
    ...   sn.note,
    ...   gossip
    ...   from
    ...   (select id, name, sr.value as source_range from google.compute.firewalls, json_array_elements_text(sourceRanges) sr where project = 'testing-project') fw
    ...   inner join stackql_notes sn 
    ...   on case when fw.source_range = '0.0.0.0/0' then 10 else 1000 end = sn.priority 
    ...   inner join stackql_gossip sg 
    ...   on case when sn.priority = 1000 then 'opinion' else 'tech' end = sg.category
    ...   order by name desc, source_range desc
    ...   ;
    ${inputStr} =    Set Variable If    "${SQL_BACKEND}" == "postgres_tcp"     ${postgresInputStr}    ${sqliteInputStr}
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |---------------|------------------------|--------------|---------------|--------------------------------|------------------------------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}id${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}name${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}source_range${SPACE}|${SPACE}is_permissive${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}note${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}gossip${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |---------------|------------------------|--------------|---------------|--------------------------------|------------------------------|
    ...    |${SPACE}8888888888888${SPACE}|${SPACE}selected-allow-rdesk${SPACE}${SPACE}${SPACE}|${SPACE}10.128.0.0/9${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}0${SPACE}|${SPACE}v0.5.418${SPACE}introduced${SPACE}table${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}stackql${SPACE}is${SPACE}not${SPACE}opinionated${SPACE}${SPACE}${SPACE}|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}valued${SPACE}functions,${SPACE}for${SPACE}example${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}json_each.${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |---------------|------------------------|--------------|---------------|--------------------------------|------------------------------|
    ...    |${SPACE}8888888888888${SPACE}|${SPACE}selected-allow-rdesk${SPACE}${SPACE}${SPACE}|${SPACE}10.0.0.0/16${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}0${SPACE}|${SPACE}v0.5.418${SPACE}introduced${SPACE}table${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}stackql${SPACE}is${SPACE}not${SPACE}opinionated${SPACE}${SPACE}${SPACE}|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}valued${SPACE}functions,${SPACE}for${SPACE}example${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}json_each.${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |---------------|------------------------|--------------|---------------|--------------------------------|------------------------------|
    ...    |${SPACE}${SPACE}777777777777${SPACE}|${SPACE}default-allow-ssh${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}0.0.0.0/0${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}1${SPACE}|${SPACE}stackql${SPACE}supports${SPACE}the${SPACE}postgres${SPACE}${SPACE}|${SPACE}stackql${SPACE}is${SPACE}open${SPACE}to${SPACE}extension${SPACE}|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}wire${SPACE}protocol.${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |---------------|------------------------|--------------|---------------|--------------------------------|------------------------------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}6666666666${SPACE}|${SPACE}default-allow-rdp${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}0.0.0.0/0${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}1${SPACE}|${SPACE}stackql${SPACE}supports${SPACE}the${SPACE}postgres${SPACE}${SPACE}|${SPACE}stackql${SPACE}is${SPACE}open${SPACE}to${SPACE}extension${SPACE}|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}wire${SPACE}protocol.${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |---------------|------------------------|--------------|---------------|--------------------------------|------------------------------|
    ...    |${SPACE}5555555555555${SPACE}|${SPACE}default-allow-internal${SPACE}|${SPACE}10.128.0.0/9${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}0${SPACE}|${SPACE}v0.5.418${SPACE}introduced${SPACE}table${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}stackql${SPACE}is${SPACE}not${SPACE}opinionated${SPACE}${SPACE}${SPACE}|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}valued${SPACE}functions,${SPACE}for${SPACE}example${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}json_each.${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |---------------|------------------------|--------------|---------------|--------------------------------|------------------------------|
    ...    |${SPACE}4444444444444${SPACE}|${SPACE}default-allow-icmp${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}0.0.0.0/0${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}1${SPACE}|${SPACE}stackql${SPACE}supports${SPACE}the${SPACE}postgres${SPACE}${SPACE}|${SPACE}stackql${SPACE}is${SPACE}open${SPACE}to${SPACE}extension${SPACE}|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}wire${SPACE}protocol.${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |---------------|------------------------|--------------|---------------|--------------------------------|------------------------------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}33333333${SPACE}|${SPACE}default-allow-https${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}0.0.0.0/0${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}1${SPACE}|${SPACE}stackql${SPACE}supports${SPACE}the${SPACE}postgres${SPACE}${SPACE}|${SPACE}stackql${SPACE}is${SPACE}open${SPACE}to${SPACE}extension${SPACE}|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}wire${SPACE}protocol.${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |---------------|------------------------|--------------|---------------|--------------------------------|------------------------------|
    ...    |${SPACE}${SPACE}${SPACE}22222222222${SPACE}|${SPACE}default-allow-http${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}0.0.0.0/0${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}1${SPACE}|${SPACE}stackql${SPACE}supports${SPACE}the${SPACE}postgres${SPACE}${SPACE}|${SPACE}stackql${SPACE}is${SPACE}open${SPACE}to${SPACE}extension${SPACE}|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}wire${SPACE}protocol.${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |---------------|------------------------|--------------|---------------|--------------------------------|------------------------------|
    ...    |${SPACE}${SPACE}111111111111${SPACE}|${SPACE}allow-spark-ui${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}0.0.0.0/0${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}1${SPACE}|${SPACE}stackql${SPACE}supports${SPACE}the${SPACE}postgres${SPACE}${SPACE}|${SPACE}stackql${SPACE}is${SPACE}open${SPACE}to${SPACE}extension${SPACE}|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}wire${SPACE}protocol.${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |---------------|------------------------|--------------|---------------|--------------------------------|------------------------------|
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Complex-Dynamic-and-Embedded-Static-Join-Returns-Expected-Results.tmp

Function Expression And Where Clause Function Expression Predicate Alongside Wildcard Returns Results
    ${sqliteInputStr} =    CATENATE    select *, JSON_EXTRACT(sourceRanges, '$[0]') sr from google.compute.firewalls where project = 'testing-project' and JSON_EXTRACT(sourceRanges, '$[0]') = '0.0.0.0/0';
    ${postgresInputStr} =    CATENATE    select *, json_extract_path_text(sourceRanges, '0') sr from google.compute.firewalls where project = 'testing-project' and json_extract_path_text(sourceRanges, '0') = '0.0.0.0/0';
    ${inputStr} =    Set Variable If    "${SQL_BACKEND}" == "postgres_tcp"     ${postgresInputStr}    ${sqliteInputStr}
    Should Stackql Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    default-allow-ssh
    ...    stdout=${CURDIR}/tmp/Function-Expression-And-Where-Clause-Function-Expression-Predicate-Alongside-Wildcard-Returns-Results.tmp

Function Expression And Where Clause Function Expression Predicate Alongside Projection Returns Expected Results
    ${sqliteInputStr} =    Catenate
    ...    select name, direction, denied, allowed, JSON_EXTRACT(sourceRanges, '$[0]') sr  
    ...    from google.compute.firewalls 
    ...    where project = 'testing-project' and sr = '0.0.0.0/0' and denied is null and allowed is not null 
    ...    order by name desc;
    ${postgresInputStr} =    Catenate
    ...    select name, direction, denied, allowed, json_extract_path_text(sourceRanges, '0') sr  
    ...    from google.compute.firewalls 
    ...    where project = 'testing-project' and json_extract_path_text(sourceRanges, '0') = '0.0.0.0/0' and denied is null and allowed is not null 
    ...    order by name desc;
    ${inputStr} =    Set Variable If    "${SQL_BACKEND}" == "postgres_tcp"     ${postgresInputStr}    ${sqliteInputStr}
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |---------------------|-----------|--------|-----------------------------------------|-----------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}name${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}direction${SPACE}|${SPACE}denied${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}allowed${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}sr${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |---------------------|-----------|--------|-----------------------------------------|-----------|
    ...    |${SPACE}default-allow-ssh${SPACE}${SPACE}${SPACE}|${SPACE}INGRESS${SPACE}${SPACE}${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}|${SPACE}\[{"IPProtocol":"tcp","ports":["22"]}]${SPACE}${SPACE}${SPACE}|${SPACE}0.0.0.0/0${SPACE}|
    ...    |---------------------|-----------|--------|-----------------------------------------|-----------|
    ...    |${SPACE}default-allow-rdp${SPACE}${SPACE}${SPACE}|${SPACE}INGRESS${SPACE}${SPACE}${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}|${SPACE}\[{"IPProtocol":"tcp","ports":["3389"]}]${SPACE}|${SPACE}0.0.0.0/0${SPACE}|
    ...    |---------------------|-----------|--------|-----------------------------------------|-----------|
    ...    |${SPACE}default-allow-icmp${SPACE}${SPACE}|${SPACE}INGRESS${SPACE}${SPACE}${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}|${SPACE}\[{"IPProtocol":"icmp"}]${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}0.0.0.0/0${SPACE}|
    ...    |---------------------|-----------|--------|-----------------------------------------|-----------|
    ...    |${SPACE}default-allow-https${SPACE}|${SPACE}INGRESS${SPACE}${SPACE}${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}|${SPACE}\[{"IPProtocol":"tcp","ports":["443"]}]${SPACE}${SPACE}|${SPACE}0.0.0.0/0${SPACE}|
    ...    |---------------------|-----------|--------|-----------------------------------------|-----------|
    ...    |${SPACE}default-allow-http${SPACE}${SPACE}|${SPACE}INGRESS${SPACE}${SPACE}${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}|${SPACE}\[{"IPProtocol":"tcp","ports":["80"]}]${SPACE}${SPACE}${SPACE}|${SPACE}0.0.0.0/0${SPACE}|
    ...    |---------------------|-----------|--------|-----------------------------------------|-----------|
    ...    |${SPACE}allow-spark-ui${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}INGRESS${SPACE}${SPACE}${SPACE}|${SPACE}null${SPACE}${SPACE}${SPACE}|${SPACE}\[{"IPProtocol":"tcp","ports":["4040"]}]${SPACE}|${SPACE}0.0.0.0/0${SPACE}|
    ...    |---------------------|-----------|--------|-----------------------------------------|-----------|
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Function-Expression-And-Where-Clause-Function-Expression-Predicate-Alongside-Projection-Returns-Expected-Results.tmp

Insert All Simple Patterns Into Embedded Table Then Projection Returns Expected Results
    ${inputStr} =    Catenate
    ...    insert into stackql_notes(note, priority) values ('this is a test', 2000);
    ...    insert into stackql_notes(note, priority) select gossip, 3000 from stackql_gossip;
    ...    insert into stackql_notes(note, priority) select name, 1000 as pr from google.compute.firewalls where project = 'testing-project';
    ...    select note from stackql_notes order by priority desc, note desc;
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |--------------------------------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}note${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|
    ...    |${SPACE}stackql${SPACE}wants${SPACE}to${SPACE}hear${SPACE}from${SPACE}you${SPACE}|
    ...    |--------------------------------|
    ...    |${SPACE}stackql${SPACE}is${SPACE}open${SPACE}to${SPACE}extension${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|
    ...    |${SPACE}stackql${SPACE}is${SPACE}not${SPACE}opinionated${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|
    ...    |${SPACE}this${SPACE}is${SPACE}a${SPACE}test${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|
    ...    |${SPACE}v0.5.418${SPACE}introduced${SPACE}table${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |${SPACE}valued${SPACE}functions,${SPACE}for${SPACE}example${SPACE}${SPACE}|
    ...    |${SPACE}json_each.${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|
    ...    |${SPACE}selected-allow-rdesk${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|
    ...    |${SPACE}default-allow-ssh${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|
    ...    |${SPACE}default-allow-rdp${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|
    ...    |${SPACE}default-allow-internal${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|
    ...    |${SPACE}default-allow-icmp${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|
    ...    |${SPACE}default-allow-https${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|
    ...    |${SPACE}default-allow-http${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|
    ...    |${SPACE}allow-spark-ui${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|
    ...    |${SPACE}stackql${SPACE}supports${SPACE}the${SPACE}postgres${SPACE}${SPACE}|
    ...    |${SPACE}wire${SPACE}protocol.${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------|
    Should Stackql Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Insert-All-Simple-Patterns-Into-Embedded-Table-Then-Projection-Returns-Expected-Results.tmp

Table Lifecycle Returns Expected Results
    ${inputStr} =    Catenate
    ...    create table my_silly_table(id int, name text, magnitude numeric);
    ...    insert into my_silly_table(id, name, magnitude) values (1, 'one', 1.0); 
    ...    insert into my_silly_table(id, name, magnitude) values (2, 'two', 2.0); 
    ...    insert into my_silly_table(id, name, magnitude) values (3, 'three', 3.0); 
    ...    select name, magnitude from my_silly_table order by magnitude desc;
    ...    drop table my_silly_table;
    ...    select name, magnitude from my_silly_table order by magnitude desc;
    ...    create table my_silly_table(id int, name text, magnitude numeric);
    ...    insert into my_silly_table(id, name, magnitude) values (11, 'eleven', 11.0); 
    ...    insert into my_silly_table(id, name, magnitude) values (12, 'twelve', 12.0); 
    ...    insert into my_silly_table(id, name, magnitude) values (13, 'thirteen', 13.0);
    ...    select name, magnitude from my_silly_table order by magnitude desc;
    ...    drop table my_silly_table;
    ...    select name, magnitude from my_silly_table order by magnitude desc;
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |-------|-----------|
    ...    |${SPACE}name${SPACE}${SPACE}|${SPACE}magnitude${SPACE}|
    ...    |-------|-----------|
    ...    |${SPACE}three${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}3${SPACE}|
    ...    |-------|-----------|
    ...    |${SPACE}two${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}2${SPACE}|
    ...    |-------|-----------|
    ...    |${SPACE}one${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}1${SPACE}|
    ...    |-------|-----------|
    ...    |----------|-----------|
    ...    |${SPACE}${SPACE}${SPACE}name${SPACE}${SPACE}${SPACE}|${SPACE}magnitude${SPACE}|
    ...    |----------|-----------|
    ...    |${SPACE}thirteen${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}13${SPACE}|
    ...    |----------|-----------|
    ...    |${SPACE}twelve${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}12${SPACE}|
    ...    |----------|-----------|
    ...    |${SPACE}eleven${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}11${SPACE}|
    ...    |----------|-----------|
    ${stdErrStr} =    Catenate    SEPARATOR=\n
    ...    DDL Execution Completed
    ...    insert into table completed
    ...    insert into table completed
    ...    insert into table completed
    ...    DDL Execution Completed
    ...    could not locate table 'my_silly_table'
    ...    DDL Execution Completed
    ...    insert into table completed
    ...    insert into table completed
    ...    insert into table completed
    ...    DDL Execution Completed
    ...    could not locate table 'my_silly_table'
    Should Stackql Exec Inline Equal Both Streams
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    ${outputStr}
    ...    ${stdErrStr}
    ...    stdout=${CURDIR}/tmp/Table-Lifecycle-Returns-Expected-Results.tmp
    ...    stderr=${CURDIR}/tmp/Table-Lifecycle-Returns-Expected-Results-stderr.tmp

Table Lifecycle Plus Update Returns Expected Results
    ${inputStr} =    Catenate
    ...    create table my_silly_table(id int, name text, magnitude numeric);
    ...    insert into my_silly_table(id, name, magnitude) values (1, 'one', 1.0); 
    ...    insert into my_silly_table(id, name, magnitude) values (2, 'two', 2.0);
    ...    select name, magnitude from my_silly_table order by magnitude desc;
    ...    update my_silly_table set magnitude = 1.5 where id = 1;
    ...    select name, magnitude from my_silly_table order by magnitude desc;
    ...    drop table my_silly_table;
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |------|-----------|
    ...    |${SPACE}name${SPACE}|${SPACE}magnitude${SPACE}|
    ...    |------|-----------|
    ...    |${SPACE}two${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}2${SPACE}|
    ...    |------|-----------|
    ...    |${SPACE}one${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}1${SPACE}|
    ...    |------|-----------|
    ...    |------|-----------|
    ...    |${SPACE}name${SPACE}|${SPACE}magnitude${SPACE}|
    ...    |------|-----------|
    ...    |${SPACE}two${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}2${SPACE}|
    ...    |------|-----------|
    ...    |${SPACE}one${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}1.5${SPACE}|
    ...    |------|-----------|
    ${stdErrStr} =    Catenate    SEPARATOR=\n
    ...    DDL Execution Completed
    ...    insert into table completed
    ...    insert into table completed
    ...    exec completed
    ...    DDL Execution Completed
    Should Stackql Exec Inline Equal Both Streams
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    ${outputStr}
    ...    ${stdErrStr}
    ...    stdout=${CURDIR}/tmp/Table-Lifecycle-Plus-Update-Returns-Expected-Results.tmp
    ...    stderr=${CURDIR}/tmp/Table-Lifecycle-Plus-Update-Returns-Expected-Results-stderr.tmp

Table Lifecycle Plus Delete Returns Expected Results
    ${inputStr} =    Catenate
    ...    create table my_silly_table(id int, name text, magnitude numeric);
    ...    insert into my_silly_table(id, name, magnitude) values (1, 'one', 1.0); 
    ...    insert into my_silly_table(id, name, magnitude) values (2, 'two', 2.0);
    ...    insert into my_silly_table(id, name, magnitude) values (3, 'three', 3.0);
    ...    select name, magnitude from my_silly_table order by magnitude desc;
    ...    delete from my_silly_table where id = 3;
    ...    select name, magnitude from my_silly_table order by magnitude desc;
    ...    delete from my_silly_table;
    ...    select name, magnitude from my_silly_table order by magnitude desc;
    ...    drop table my_silly_table;
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |-------|-----------|
    ...    |${SPACE}name${SPACE}${SPACE}|${SPACE}magnitude${SPACE}|
    ...    |-------|-----------|
    ...    |${SPACE}three${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}3${SPACE}|
    ...    |-------|-----------|
    ...    |${SPACE}two${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}2${SPACE}|
    ...    |-------|-----------|
    ...    |${SPACE}one${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}1${SPACE}|
    ...    |-------|-----------|
    ...    |------|-----------|
    ...    |${SPACE}name${SPACE}|${SPACE}magnitude${SPACE}|
    ...    |------|-----------|
    ...    |${SPACE}two${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}2${SPACE}|
    ...    |------|-----------|
    ...    |${SPACE}one${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}1${SPACE}|
    ...    |------|-----------|
    ...    |------|-----------|
    ...    |${SPACE}name${SPACE}|${SPACE}magnitude${SPACE}|
    ...    |------|-----------|
    ${stdErrStr} =    Catenate    SEPARATOR=\n
    ...    DDL Execution Completed
    ...    insert into table completed
    ...    insert into table completed
    ...    insert into table completed
    ...    exec completed
    ...    exec completed
    ...    DDL Execution Completed
    Should Stackql Exec Inline Equal Both Streams
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${inputStr}
    ...    ${outputStr}
    ...    ${stdErrStr}
    ...    stdout=${CURDIR}/tmp/Table-Lifecycle-Plus-Delete-Returns-Expected-Results.tmp
    ...    stderr=${CURDIR}/tmp/Table-Lifecycle-Plus-Delete-Returns-Expected-Results-stderr.tmp

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
