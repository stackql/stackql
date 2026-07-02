*** Settings ***
Resource          ${CURDIR}/stackql.resource
Test Teardown     Stackql Per Test Teardown
Documentation     Functional coverage for the any-sdk schema_driven_xml_v0.1.0 response
...               transform (PR 107), exercised through the no-auth stackql_native_test
...               provider against the local native_test flask mock. Each archetype
...               (ec2 / query / rest-xml) is projected per-row with schema-driven typing.
...               Multi-word columns whose snake alias differs from the wire name
...               (e.g. cidr_block <- cidrBlock, volume_id <- volumeId) now project their
...               VALUE via any-sdk GetWireName (issue 108); the assertions below check the
...               extracted value, not just the column name.

*** Test Cases ***
Schema Driven Xml Ec2 Archetype Projects Typed Rows
    [Documentation]    ec2 envelope is skipped and <volumeSet><item> rows are projected with declared types.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select size, state, encrypted from stackql_native_test.xml_ec2.volumes order by size;
    ...    available

Schema Driven Xml Ec2 Projects Boolean And Integer Types
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select size, encrypted from stackql_native_test.xml_ec2.volumes where state \= 'in-use';
    ...    false

Schema Driven Xml Snake Aliases Multi Word Column Value
    [Documentation]    snake_case_aliases renames wire cidrBlock -> cidr_block; the VALUE now
    ...    projects via GetWireName (issue 108) instead of NULL.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select volume_id, cidr_block from stackql_native_test.xml_ec2.volumes order by volume_id;
    ...    10.0.0.0/24

Schema Driven Xml Snake Aliases Multi Word Column Value Second Column
    [Documentation]    a second multi-word column (volume_id <- volumeId) also projects its value.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select volume_id, cidr_block from stackql_native_test.xml_ec2.volumes order by volume_id;
    ...    vol-1

Schema Driven Xml Query Archetype Skips Result Wrapper
    [Documentation]    query archetype skips the extra <DescribeStacksResult> wrapper and projects <member> rows.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select id, region from stackql_native_test.xml_query.stacks order by id;
    ...    us-east-1

Schema Driven Xml Empty Self Closing List Yields Zero Rows
    [Documentation]    a self-closing <Stacks/> element projects zero rows rather than erroring.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select count(*) as cnt from stackql_native_test.xml_query.stacks_empty;
    ...    0

Schema Driven Xml Rest Xml Singleton Yields One Row
    [Documentation]    rest-xml singleton (no list envelope) is unwrapped to exactly one row.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select id, name from stackql_native_test.xml_restxml.hostedzone;
    ...    example.com

Xml Name Override Projects Row Values
    [Documentation]    Schema properties carry botocore member names (VolumeId) with
    ...    xml: name overrides for the wire elements (volumeId) - the AWS provider
    ...    shape. The walker extracts by the override and the value projects under
    ...    the snake alias of the member name.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select volume_id, size from stackql_native_test.xml_ec2.volumes_alias order by volume_id;
    ...    vol-a1

Xml Name Override Complex Value Is Json
    [Documentation]    A nested wire element (<attachmentSet>) landing under a
    ...    string-typed column serialises as JSON (never Go map notation), so users
    ...    can JSON_EXTRACT it.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select attachments from stackql_native_test.xml_ec2.volumes_alias where volume_id \= 'vol-a1';
    ...    "instanceId"

Xml Where On Snake Column Post Filters
    [Documentation]    A WHERE key that is a response column (not a request parameter)
    ...    post-filters the materialised rows; the snake alias must resolve in the
    ...    symbol table (stackql symtab display-name registration fix).
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select state from stackql_native_test.xml_ec2.volumes where cidr_block \= '10.0.1.0/24';
    ...    in-use

Xml Singleton Unwrap Projects Row
    [Documentation]    CreateVpc-style response nests the row under a named wrapper
    ...    ({vpc: {...}}); the walker unwraps one level when the payload root does not
    ...    carry the row fields (this is what makes INSERT ... RETURNING project).
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select vpc_id, cidr_block from stackql_native_test.xml_ec2.vpc_singleton;
    ...    vpc-fixture-1

Xml Empty Response Body Yields Zero Rows
    [Documentation]    A 200 with an empty body (S3 CreateBucket-style) yields zero
    ...    rows, not an mxj EOF transform error.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select count(*) as num from stackql_native_test.xml_ec2.volumes_empty_body;
    ...    0
