*** Settings ***
Resource          ${CURDIR}/stackql.resource
Library           Collections
Documentation     Offline inventory of the dependencies shipped in the StackQL binary.

*** Variables ***
@{DEPENDENCY_COLUMNS}    name    version    license    description
@{EXTENDED_DEPENDENCY_COLUMNS}
...    name    version    license    description
...    source_url    license_url    purl    checksum    original_name    original_version

*** Test Cases ***
Show Dependencies Default And Extended
    ${ordinary}=    Dependency CSV    SHOW DEPENDENCIES;
    ${extended}=    Dependency CSV    SHOW EXTENDED DEPENDENCIES;
    Lists Should Be Equal    ${ordinary}[0]    ${DEPENDENCY_COLUMNS}
    Lists Should Be Equal    ${extended}[0]    ${EXTENDED_DEPENDENCY_COLUMNS}
    ${projected}=    Evaluate    [row[:4] for row in $extended[1:]]
    Lists Should Be Equal    ${ordinary}[1:]    ${projected}
    ${ordered}=    Evaluate    sorted($ordinary[1:], key=lambda row: (row[0], row[1]))
    Lists Should Be Equal    ${ordinary}[1:]    ${ordered}
    Should Be True    len($ordinary) > 1
    ${names}=    Evaluate    [row[0] for row in $ordinary[1:]]
    List Should Not Contain Value    ${names}    github.com/google/licensecheck
    List Should Not Contain Value    ${names}    github.com/stretchr/testify

Show Dependencies Like Has Identical Default And Extended Rows
    FOR    ${pattern}    IN    %stackql%    %VITESS%    %CHZYER%    github.com/stackql/readlin_
        ${ordinary}=    Dependency CSV    SHOW DEPENDENCIES LIKE '${pattern}';
        ${extended}=    Dependency CSV    SHOW EXTENDED DEPENDENCIES LIKE '${pattern}';
        Lists Should Be Equal    ${ordinary}[0]    ${DEPENDENCY_COLUMNS}
        Lists Should Be Equal    ${extended}[0]    ${EXTENDED_DEPENDENCY_COLUMNS}
        ${projected}=    Evaluate    [row[:4] for row in $extended[1:]]
        Lists Should Be Equal    ${ordinary}[1:]    ${projected}
        Should Be True    len($ordinary) > 1
    END

Show Dependencies Effective And Original Names Match
    ${effective}=    Dependency JSON    SHOW EXTENDED DEPENDENCIES LIKE 'github.com/stackql/readline';
    ${original}=    Dependency JSON    SHOW EXTENDED DEPENDENCIES LIKE 'github.com/chzyer/readline';
    Lists Should Be Equal    ${effective}    ${original}
    Length Should Be    ${effective}    1
    Should Be Equal    ${effective}[0][name]    github.com/stackql/readline
    Should Be Equal    ${effective}[0][original_name]    github.com/chzyer/readline
    Should Be Equal    ${effective}[0][license]    MIT
    Should Start With    ${effective}[0][checksum]    h1:
    Should Contain    ${effective}[0][license_url]    ${effective}[0][version]
    Should Be Equal    ${effective}[0][purl]    pkg:golang/github.com/stackql/readline@${effective}[0][version]

Show Dependencies Description And Case Insensitive Wildcards
    ${upper}=    Dependency JSON    SHOW EXTENDED DEPENDENCIES LIKE '%VITESS%';
    ${lower}=    Dependency JSON    SHOW EXTENDED DEPENDENCIES LIKE '%vitess%';
    Lists Should Be Equal    ${upper}    ${lower}
    Length Should Be    ${upper}    1
    Should Be Equal    ${upper}[0][name]    github.com/stackql/stackql-parser
    Should Be Equal    ${upper}[0][original_name]    ${NONE}
    Should Be Equal    ${upper}[0][original_version]    ${NONE}

Show Dependencies No Matches Preserve Columns
    FOR    ${pattern}    IN    no-such-component    github.com/stackql/readlin__    %can''t-match%    ${EMPTY}
        ${ordinary}=    Dependency CSV    SHOW DEPENDENCIES LIKE '${pattern}';
        ${extended}=    Dependency CSV    SHOW EXTENDED DEPENDENCIES LIKE '${pattern}';
        Length Should Be    ${ordinary}    1
        Length Should Be    ${extended}    1
        Lists Should Be Equal    ${ordinary}[0]    ${DEPENDENCY_COLUMNS}
        Lists Should Be Equal    ${extended}[0]    ${EXTENDED_DEPENDENCY_COLUMNS}
        ${json}=    Dependency JSON    SHOW EXTENDED DEPENDENCIES LIKE '${pattern}';
        Should Be Equal    ${json}    ${NONE}
    END

Show Dependencies Percent Matches Unfiltered Inventory
    ${all}=    Dependency JSON    SHOW DEPENDENCIES;
    ${wildcard}=    Dependency JSON    SHOW DEPENDENCIES LIKE '%';
    Lists Should Be Equal    ${all}    ${wildcard}

Show Dependencies PostgreSQL Wire Metadata And Nulls
    [Tags]    dependency-wire
    FOR    ${extended}    IN    ${EMPTY}    EXTENDED${SPACE}
        ${columns}=    Set Variable If    '${extended}' == ''    ${DEPENDENCY_COLUMNS}    ${EXTENDED_DEPENDENCY_COLUMNS}
        ${descriptions}=    Evaluate    [{'name': name, 'type_code': 25} for name in $columns]
        Should PG Client Column Descriptions Equal
        ...    ${POSTGRES_URL_UNENCRYPTED_CONN}
        ...    SHOW ${extended}DEPENDENCIES LIKE '%readline%';
        ...    ${descriptions}
        Should PG Client Column Descriptions Equal
        ...    ${POSTGRES_URL_UNENCRYPTED_CONN}
        ...    SHOW ${extended}DEPENDENCIES LIKE 'no-such-component';
        ...    ${descriptions}
    END
    ${expected}=    Dependency JSON    SHOW EXTENDED DEPENDENCIES LIKE '%VITESS%';
    ${queries}=    Create List    SHOW EXTENDED DEPENDENCIES LIKE '%VITESS%';
    Should PG Client V2 Session Inline Equal    ${POSTGRES_URL_UNENCRYPTED_CONN}    ${queries}    ${expected}
    ${queries}=    Create List    SHOW EXTENDED DEPENDENCIES LIKE 'no-such-component';
    ${empty}=    Create List
    Should PG Client V2 Session Inline Equal    ${POSTGRES_URL_UNENCRYPTED_CONN}    ${queries}    ${empty}

*** Keywords ***
Dependency Output
    [Arguments]    ${query}    ${format}
    ${result}=    Run StackQL Exec Command No Errors
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${query}
    ...    -o\=${format}
    ...    stdout=${CURDIR}/tmp/Show-Dependencies.tmp
    ...    stderr=${CURDIR}/tmp/Show-Dependencies-stderr.tmp
    ...    timeout=30s
    ...    on_timeout=kill
    Should Be Empty    ${result.stderr}
    RETURN    ${result.stdout}

Dependency CSV
    [Arguments]    ${query}
    ${output}=    Dependency Output    ${query}    csv
    ${rows}=    Evaluate    list(csv.reader(io.StringIO($output)))    csv,io
    RETURN    ${rows}

Dependency JSON
    [Arguments]    ${query}
    ${output}=    Dependency Output    ${query}    json
    ${rows}=    Evaluate    json.loads($output)    json
    RETURN    ${rows}
