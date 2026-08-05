*** Settings ***
Resource          ${CURDIR}/stackql.resource
Test Teardown     Stackql Per Test Teardown
Documentation     OCI request signing (oci_signing_v1) via any-sdk, against the local
...               native_test mock. The mock cracks the draft-cavage Authorization
...               header into columns (GET) and enforces the six-header body-verb
...               contract including the x-content-sha256 digest (POST). Credentials
...               are a checked-in throwaway RSA key, never a real tenancy.

*** Variables ***
${OCI_AUTH_RAW_ENV}        {"stackql_oci_testing":{"type":"oci_signing_v1","tenancy_ocid_env_var":"STACKQL_OCI_TESTING_TENANCY_OCID","user_ocid_env_var":"STACKQL_OCI_TESTING_USER_OCID","fingerprint_env_var":"STACKQL_OCI_TESTING_FINGERPRINT","private_key_path_env_var":"STACKQL_OCI_TESTING_PRIVATE_KEY_PATH"}}
${OCI_AUTH_PARTIAL_RAW}    {"stackql_oci_testing":{"type":"oci_signing_v1","tenancy_ocid":"ocid1.tenancy.oc1..onlytenancy"}}
${OCI_TEST_TENANCY}        ocid1.tenancy.oc1..stackqltesttenancy
${OCI_TEST_USER}           ocid1.user.oc1..stackqltestuser
${OCI_TEST_FINGERPRINT}    12:34:56:78:9a:bc:de:f0:12:34:56:78:9a:bc:de:f0
${OCI_TEST_KEY_REL}        src${/}stackql_oci_testing${/}v0.1.0${/}credentials${/}oci_test_key.pem

*** Test Cases ***
OCI Signing Raw Env GET Signs KeyId From Env Credentials
    Set OCI Raw Env Credentials
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${OCI_AUTH_RAW_ENV}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, auth_key_id, auth_algorithm from stackql_oci_testing.signing.buckets;
    ...    ${OCI_TEST_TENANCY}/${OCI_TEST_USER}/${OCI_TEST_FINGERPRINT}

OCI Signing Raw Env GET Signs Exactly Date Request-Target Host
    Set OCI Raw Env Credentials
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${OCI_AUTH_RAW_ENV}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, auth_signed_headers from stackql_oci_testing.signing.buckets;
    ...    date (request-target) host${SPACE}|

OCI Signing Raw Env GET Uses Rsa Sha256
    Set OCI Raw Env Credentials
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${OCI_AUTH_RAW_ENV}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, auth_scheme, auth_algorithm from stackql_oci_testing.signing.buckets;
    ...    rsa-sha256

OCI Signing Raw Env POST Signs Body Verb Header Set
    [Documentation]    The mock 400s unless the signed header list is exactly
    ...                "date (request-target) host content-length content-type x-content-sha256"
    ...                and x-content-sha256 matches the body digest, so a successful
    ...                despatch proves the six-header contract.
    Set OCI Raw Env Credentials
    Should Stackql Exec Inline Equal Stderr
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${OCI_AUTH_RAW_ENV}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    insert into stackql_oci_testing.signing.buckets(data__name) values ('bucket-new');
    ...    The operation was despatched successfully

OCI Signing Config File Variant GET Signs KeyId From Config
    [Documentation]    Config-file credentials (distinct tenancy/user/fingerprint from
    ...                the raw-env cases) prove the file, not the environment, was read.
    Pass Execution If    "${EXECUTION_PLATFORM}" == "docker"    config file and key paths are host-local; the raw variant covers docker
    ${key_path} =    Set Variable    ${REPOSITORY_ROOT}${/}test${/}registry${/}${OCI_TEST_KEY_REL}
    ${key_path_fwd} =    Replace String    ${key_path}    \\    /
    ${cfg_content} =    Catenate    SEPARATOR=\n
    ...    [DEFAULT]
    ...    user=ocid1.user.oc1..stackqlcfguser
    ...    fingerprint=fe:dc:ba:98:76:54:32:10:fe:dc:ba:98:76:54:32:10
    ...    tenancy=ocid1.tenancy.oc1..stackqlcfgtenancy
    ...    region=us-ashburn-1
    ...    key_file=${key_path_fwd}
    Create File    ${CURDIR}${/}tmp${/}oci-test-config    ${cfg_content}
    ${cfg_path_fwd} =    Replace String    ${CURDIR}${/}tmp${/}oci-test-config    \\    /
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    {"stackql_oci_testing":{"type":"oci_signing_v1","config_file_path":"${cfg_path_fwd}"}}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, auth_key_id from stackql_oci_testing.signing.buckets;
    ...    ocid1.tenancy.oc1..stackqlcfgtenancy/ocid1.user.oc1..stackqlcfguser/fe:dc:ba:98:76:54:32:10:fe:dc:ba:98:76:54:32:10

OCI Signing Partial Raw Credentials Fail Fast
    [Documentation]    Any raw credential present makes all four required; the
    ...                composition error surfaces before any request is signed.
    Should Stackql Exec Inline Contain Stderr
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${OCI_AUTH_PARTIAL_RAW}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name from stackql_oci_testing.signing.buckets;
    ...    cannot compose OCI signing credentials

*** Keywords ***
Set OCI Raw Env Credentials
    Set Environment Variable    STACKQL_OCI_TESTING_TENANCY_OCID    ${OCI_TEST_TENANCY}
    Set Environment Variable    STACKQL_OCI_TESTING_USER_OCID    ${OCI_TEST_USER}
    Set Environment Variable    STACKQL_OCI_TESTING_FINGERPRINT    ${OCI_TEST_FINGERPRINT}
    ${key_path} =    Set Variable If    "${EXECUTION_PLATFORM}" == "docker"
    ...    /opt/stackql/registry/${OCI_TEST_KEY_REL}
    ...    ${REPOSITORY_ROOT}${/}test${/}registry${/}${OCI_TEST_KEY_REL}
    Set Environment Variable    STACKQL_OCI_TESTING_PRIVATE_KEY_PATH    ${key_path}
