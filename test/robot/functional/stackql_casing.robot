*** Settings ***
Resource          ${CURDIR}/stackql.resource
Test Teardown     Stackql Per Test Teardown
Documentation     Functional coverage for the any-sdk casing engine, exercised through the
...               no-auth stackql_native_test provider against the local native_test flask
...               echo mock. Three behaviours are wired end-to-end: (1) config.snake_case_aliases
...               renames response columns to snake_case and the multi-word column VALUE
...               projects via any-sdk GetWireName (issue 108); (2) a snake_case WHERE key is
...               reverse-resolved to its PascalCase wire parameter via the native-casing param
...               set (issue 109); (3) PascalCase wire request parameters are transmitted as
...               declared. Both snake and wire WHERE forms are accepted. The tail of the
...               suite covers any-sdk #131 / #119 surface parity: hyphenated and
...               acronym-headed wire names, snake body keys, presentation under
...               SHOW, body-less EXEC, wire-spelled projections and the wire-spelling
...               backward-compatibility block.

*** Test Cases ***
Snake Case Aliases Multi Word Response Column Projects Value
    [Documentation]    Wire field VpcId is exposed as snake_case column vpc_id and its VALUE
    ...    (echoed back by the mock) now projects via GetWireName (issue 108) instead of NULL.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select vpc_id from stackql_native_test.casing.echo where VpcId \= 'vpc-77';
    ...    vpc-77

Snake Case Aliases Single Word Column Projects Value
    [Documentation]    echoed_query (snake alias == wire name) projects its value normally.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select echoed_query from stackql_native_test.casing.echo where VpcId \= 'abc123';
    ...    abc123

Snake Case Where Key Resolves To Wire Parameter
    [Documentation]    A snake_case WHERE key (vpc_id) reaches the wire as its PascalCase
    ...    parameter VpcId (issue 109), reverse-resolved via the native-casing param set.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select echoed_query from stackql_native_test.casing.echo where vpc_id \= 'v1';
    ...    VpcId\=v1

Snake Case Multiple Where Keys Resolve To Wire
    [Documentation]    Two snake_case WHERE keys both reverse-resolve to their wire parameters.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select echoed_query from stackql_native_test.casing.echo where vpc_id \= 'v1' and subnet_id \= 's9';
    ...    SubnetId\=s9

Pascal Case Wire Parameter Transmitted As Declared
    [Documentation]    The VpcId query parameter reaches the wire with its declared PascalCase name.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select echoed_query from stackql_native_test.casing.echo where VpcId \= 'abc123';
    ...    VpcId\=abc123

Multiple Pascal Case Wire Parameters Transmitted
    [Documentation]    Two PascalCase wire parameters in one WHERE both reach the wire.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select echoed_query from stackql_native_test.casing.echo where VpcId \= 'abc123' and SubnetId \= 'sub-9';
    ...    SubnetId\=sub-9

Describe Extended Shows Snake Case Aliases
    [Documentation]    DESCRIBE surfaces the same snake aliases as SELECT: wire VpcId
    ...    renders as vpc_id (any-sdk ToDescriptionMap parity fix).
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    describe extended stackql_native_test.casing.echo;
    ...    vpc_id

Select Star Projects Snake Aliased Values
    [Documentation]    SELECT * expands to snake-aliased columns (any-sdk GetAllColumns
    ...    parity fix). Before the fix the wire-cased identifiers resolved as string
    ...    literals on a case-sensitive backend and every value projected as its own
    ...    column name; the assertion checks the echoed VALUE, not the header.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select * from stackql_native_test.casing.echo where VpcId \= 'star-val-1';
    ...    star-val-1

Snake Case Where Key Satisfies Required Wire Parameter
    [Documentation]    Method routing accepts a snake key for a REQUIRED wire param
    ...    (any-sdk parameterMatch reverse-casing fix): echo_strict requires VpcId and
    ...    the SQL supplies vpc_id; the echoed wire query proves both routing and
    ...    request construction re-keyed it.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select echoed_query from stackql_native_test.casing.echo_strict where vpc_id \= 'req-9';
    ...    VpcId\=req-9

Base Fallback Body Sent When No Body Params
    [Documentation]    A method with request.base '{}' and no SQL-supplied body fields
    ...    sends the base bytes verbatim (the aws-json no-input pattern); the mock
    ...    echoes the received body.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select echoed_body, ok from stackql_native_test.casing.echo_post;
    ...    {}

Hyphenated Wire Header Resolves From Snake Where Key
    [Documentation]    any-sdk #119: ToSnake treats '-' as a word boundary, so the
    ...    kebab-case wire header openai-organization is addressable as
    ...    openai_organization (method nativeCasing kebab); the mock echoes the header.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select echoed_org from stackql_native_test.casing.echo_header where openai_organization \= 'org-x';
    ...    org-x

Hyphenated Wire Header Quoted Wire Spelling Unchanged
    [Documentation]    The quoted wire spelling keeps working alongside the snake alias.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select echoed_org from stackql_native_test.casing.echo_header where "openai-organization" \= 'org-y';
    ...    org-y

Hyphenated Wire Header Misspelt Snake Key Rejected
    [Documentation]    Negative: a near-miss snake key is not fuzzily matched to the header.
    Should Stackql Exec Inline Contain Stderr
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select echoed_org from stackql_native_test.casing.echo_header where openai_organizatoin \= 'x';
    ...    could not locate symbol

Acronym Headed Wire Parameter Resolves From Snake Where Key
    [Documentation]    any-sdk #131: ip_protocol resolves through the declared wire
    ...    name IPProtocol (the mechanical FromSnake form IpProtocol does not exist).
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select echoed_query from stackql_native_test.casing.echo where ip_protocol \= 'tcp';
    ...    IPProtocol\=tcp

Exec Required Path Parameter Satisfied By Snake Alias
    [Documentation]    @binary_id satisfies the REQUIRED wire path parameter BinaryId
    ...    (required exec args are now resolved via the method, like optional ones);
    ...    the echoed row proves the path and query both reached the wire.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select binary_id, echoed_query from (exec stackql_native_test.casing.echo_bodyless.exec_by_id @binary_id \= 'b3', @vpc_id \= 'v3');
    ...    | b3${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}| VpcId\=v3${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|

Insert Snake Body Key Maps To Camel Wire Property
    [Documentation]    data__storage_class maps to the declared wire property
    ...    storageClass under nativeCasing pascal (declared-name resolution, not
    ...    the mechanical StorageClass); RETURNING surfaces the echoed body.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    insert into stackql_native_test.casing.echo (data__storage_class) values ('NEARLINE') returning echoed_body;
    ...    {"storageClass":"NEARLINE"}

Insert Wire Body Key Sends Wire Property Unchanged
    [Documentation]    data__storageClass sends the same wire body as the snake form.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    insert into stackql_native_test.casing.echo (data__storageClass) values ('NEARLINE') returning echoed_body;
    ...    {"storageClass":"NEARLINE"}

Show Methods Presents Snake Names For Native Casing Method
    [Documentation]    SHOW METHODS renders the required wire parameter VpcId as vpc_id
    ...    when the method declares nativeCasing and the provider enables snake aliases.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    show methods in stackql_native_test.casing.echo_strict;
    ...    | select${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}| vpc_id${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}| SELECT${SPACE}${SPACE}|

Show Methods Keeps Wire Names Without Native Casing
    [Documentation]    A method with no request.nativeCasing still presents wire names.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    show methods in stackql_native_test.casing.echo_wire;
    ...    | select${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}| VpcId${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}| SELECT${SPACE}${SPACE}|

Show Insert Renders Snake Body Keys
    [Documentation]    SHOW INSERT presents the camelCase wire body key storageClass as
    ...    data__storage_class for a nativeCasing method.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    show insert into stackql_native_test.casing.echo;
    ...    data__storage_class

Bodyless Exec With Metadata Only Request Block Dispatches Wire Args
    [Documentation]    A body-less method carrying request.nativeCasing (no schema)
    ...    dispatches: GetRequestBodySchema reports nil, nil rather than an error.
    Should Stackql Exec Inline Equal Stderr
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    exec stackql_native_test.casing.echo_bodyless.exec_by_id @BinaryId \= 'b1', @VpcId \= 'v1';
    ...    The operation was despatched successfully

Bodyless Exec With Metadata Only Request Block Dispatches Snake Args
    Should Stackql Exec Inline Equal Stderr
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    exec stackql_native_test.casing.echo_bodyless.exec_by_id @binary_id \= 'b2', @vpc_id \= 'v2';
    ...    The operation was despatched successfully

Bodyless Exec Rejects Payload Without Panic
    [Documentation]    Negative: supplying @@json to a method with no request body
    ...    schema errors cleanly (the nil schema is guarded) instead of dereferencing nil.
    Should Stackql Exec Inline Contain Stderr
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    exec stackql_native_test.casing.echo_bodyless.exec_by_id @BinaryId \= 'b4' @@json \= '{"a": 1}';
    ...    has no request body schema

Wire Spelled Projection Selects Snake Display Column
    [Documentation]    SELECT VpcId under snake_case_aliases selects the vpc_id backend
    ...    column (the descriptor is re-keyed to the display name) and keeps the
    ...    wire spelling as the output header.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select VpcId from stackql_native_test.casing.echo where VpcId \= 'vpc-77';
    ...    | vpc-77 |

Wire Spelled Order By Sorts Snake Display Column
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select VpcId from stackql_native_test.casing.echo where VpcId \= 'vpc-78' order by VpcId;
    ...    | vpc-78 |

Unknown Snake Projection Still Rejected
    [Documentation]    Negative: a column absent from the schema is not conjured by the
    ...    display-name re-keying; the backend rejects it in its own dialect.
    ${expected} =    Set Variable If    "${SQL_BACKEND}" == "postgres_tcp"    column "vpc_idz" does not exist    no such column: vpc_idz
    Should Stackql Exec Inline Contain Stderr
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select vpc_idz from stackql_native_test.casing.echo where VpcId \= 'vpc-77';
    ...    ${expected}

Backward Compat Wire Parameter In Where And Wire Column In Projection
    [Documentation]    any-sdk #131 compatibility block: wire spellings in WHERE,
    ...    projection and ORDER BY produce the same echoed wire shape as the snake forms.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select VpcId, echoed_query from stackql_native_test.casing.echo where IPProtocol \= 'udp' and VpcId \= 'bc-1' order by VpcId;
    ...    IPProtocol\=udp&VpcId\=bc-1

Backward Compat Wire Body Key In Insert
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    insert into stackql_native_test.casing.echo (data__DisplayName) values ('wire-name') returning echoed_body;
    ...    {"DisplayName":"wire-name"}

Backward Compat Wire Exec Argument
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select binary_id, echoed_query from (exec stackql_native_test.casing.echo_bodyless.exec_by_id @BinaryId \= 'bw', @VpcId \= 'vw');
    ...    VpcId\=vw
