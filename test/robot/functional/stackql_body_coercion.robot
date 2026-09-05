*** Settings ***
Resource          ${CURDIR}/stackql.resource
Test Teardown     Stackql Per Test Teardown
Documentation     Schema-driven request body coercion (any-sdk #129, stackql #725) via the
...               no-auth stackql_native_test.coerce service. Body values are converted to
...               the schema-declared type at marshal time: string -> number / integer /
...               boolean, numeric -> string for string-typed properties. The mock echoes
...               the raw received body, surfaced through RETURNING, so each case asserts
...               the exact wire bytes. Unparseable or fractional-for-integer values fail
...               before any request is sent.

*** Test Cases ***
Update Set String Literals Coerced To Number And Boolean
    [Documentation]    SET values arrive as strings on the update path; the schema types
    ...    number / boolean make them unquoted on the wire (the stackql #725 repro).
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    update stackql_native_test.coerce.settings set data__backup_period_in_hours \= '24', data__enabled \= 'true' where service_id \= 'svc-1' returning echoed_body;
    ...    {"backupPeriodInHours":24,"enabled":true}

Replace Set String Literals Coerced To Number And Boolean
    [Documentation]    The PUT (REPLACE) path sends the same coerced shape.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    replace stackql_native_test.coerce.settings set data__backup_period_in_hours \= '24', data__enabled \= 'true' where service_id \= 'svc-2' returning echoed_body;
    ...    {"backupPeriodInHours":24,"enabled":true}

Update Numeric Literal For String Property Sent Quoted
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    update stackql_native_test.coerce.settings set data__label \= 12 where service_id \= 'svc-1' returning echoed_body;
    ...    {"label":"12"}

Insert Typed Literals Unchanged
    [Documentation]    INSERT already carried typed values; coercion is a no-op on them.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    insert into stackql_native_test.coerce.settings (data__backup_period_in_hours, data__enabled, data__label, data__retry_count) select 24, true, 'x', 3 returning echoed_body;
    ...    {"backupPeriodInHours":24,"enabled":true,"label":"x","retryCount":3}

Exec Json Payload Number Property Validates
    [Documentation]    IsFloat accepts OpenAPI type number, so a JSON number no longer
    ...    fails validation as "expected float64 but is float64".
    Should Stackql Exec Inline Equal Stderr
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    exec stackql_native_test.coerce.settings.update @serviceId \= 'svc-3' @@json \= '{"backupPeriodInHours": 24}';
    ...    The operation was despatched successfully

Exec Json Payload Integral Value For Integer Property Validates
    [Documentation]    stackql #725 follow-through: JSON numbers unmarshal as float64, so
    ...    an integral value must satisfy a type integer property.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select echoed_body from (exec stackql_native_test.coerce.settings.update @serviceId \= 'svc-3' @@json \= '{"backupPeriodInHours": 24, "retryCount": 3}');
    ...    {"backupPeriodInHours":24,"retryCount":3}

Exec Json Payload Fractional Value For Integer Property Rejected
    Should Stackql Exec Inline Contain Stderr
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    exec stackql_native_test.coerce.settings.update @serviceId \= 'svc-3' @@json \= '{"retryCount": 3.5}';
    ...    key 'retryCount' expected to contain element of type

Update Unparseable String For Number Property Fails Before Request
    [Documentation]    Negative: the coercion error names the wire key and schema type;
    ...    no request reaches the mock.
    Should Stackql Exec Inline Contain Stderr
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    update stackql_native_test.coerce.settings set data__backup_period_in_hours \= 'abc' where service_id \= 'svc-1' returning echoed_body;
    ...    request body key 'backupPeriodInHours': value 'abc' cannot be coerced to schema type 'number'

Update Fractional String For Integer Property Fails Before Request
    Should Stackql Exec Inline Contain Stderr
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    update stackql_native_test.coerce.settings set data__retry_count \= '1.5' where service_id \= 'svc-1' returning echoed_body;
    ...    request body key 'retryCount': value '1.5' cannot be coerced to schema type 'integer'
