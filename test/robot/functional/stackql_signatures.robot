*** Settings ***
Library           Process
Library           OperatingSystem
Library           Collections
Suite Setup       Prepare Signed Registry Fixtures
Suite Teardown    Remove Directory    ${SIGNED_FIXTURES}    recursive=True
Test Tags         provider-signatures

*** Variables ***
${SIGNATURE_REPO}    ${CURDIR}${/}..${/}..${/}..
${SIGNATURE_EXE}     ${SIGNATURE_REPO}${/}build${/}stackql
${SIGNATURE_GENERATOR}    ${EMPTY}
${EXECUTION_PLATFORM}    native
${USE_STACKQL_PREINSTALLED}    false

*** Test Cases ***
Valid Embedded Signatures Work With Both SDKs
    FOR    ${provider}    IN    fixture    stackql_unstable_fixture
        ${result}=    Read Signed Registry    valid    ${provider}
        Should Be Equal As Integers    ${result.rc}    0    ${result.stderr}
        Should Be Empty    ${result.stderr}
        Should Contain    ${result.stdout}    members
    END

Tampered Provider Is Rejected By Both SDKs
    Signed Registry Must Fail    tampered-provider    provider.yaml

Tampered Service Is Rejected By Both SDKs
    Signed Registry Must Fail    tampered-service    orgs.yaml

Unknown Signing Key Is Rejected By Both SDKs
    Signed Registry Must Fail    unknown    unknown key ID

Missing Embedded Signature Is Rejected By Both SDKs
    Signed Registry Must Fail    unsigned    x-stackql-signature

Verification Remains Optional For Both SDKs
    FOR    ${provider}    IN    fixture    stackql_unstable_fixture
        ${result}=    Read Signed Registry    opt-out    ${provider}
        Should Be Equal As Integers    ${result.rc}    0    ${result.stderr}
        Should Be Empty    ${result.stderr}
        Should Contain    ${result.stdout}    members
    END

*** Keywords ***
Prepare Signed Registry Fixtures
    ${directory}=    Evaluate    tempfile.mkdtemp(prefix="stackql-signatures-")    tempfile
    Set Suite Variable    ${SIGNED_FIXTURES}    ${directory}
    @{generator}=    Create List    go    run    ./test/tools/signing-fixtures
    IF    $SIGNATURE_GENERATOR
        @{generator}=    Create List    ${SIGNATURE_GENERATOR}
    END
    ${result}=    Run Process    @{generator}    -out    ${directory}
    ...    cwd=${SIGNATURE_REPO}    stdout=${directory}${/}generator.stdout    stderr=${directory}${/}generator.stderr
    Should Be Equal As Integers    ${result.rc}    0    ${result.stderr}
    ${exe}=    Set Variable    ${SIGNATURE_EXE}
    IF    str($USE_STACKQL_PREINSTALLED).lower() == 'true'
        ${exe}=    Set Variable    stackql
    ELSE IF    ${{os.name == 'nt'}}
        ${exe}=    Set Variable    ${SIGNATURE_EXE}.exe
    END
    Set Suite Variable    ${SIGNED_EXE}    ${exe}

Read Signed Registry
    [Arguments]    ${variant}    ${provider}
    ${config}=    Get File    ${SIGNED_FIXTURES}${/}${variant}.json
    ${approot}=    Set Variable    ${SIGNED_FIXTURES}${/}app-${variant}-${provider}
    @{command}=    Create List    ${SIGNED_EXE}
    IF    $EXECUTION_PLATFORM == 'docker'
        ${config}=    Evaluate    json.loads($config)    json
        Set To Dictionary    ${config}    url=file:///signatures/${variant}    localDocRoot=/signatures/${variant}
        ${config}=    Evaluate    json.dumps($config)    json
        ${approot}=    Set Variable    /tmp/stackql-signatures
        ${image}=    Get Environment Variable    STACKQL_IMAGE_NAME    stackql/stackql
        ${user}=    Evaluate    str(os.getuid()) + ':' + str(os.getgid())    os
        @{command}=    Create List    docker    run    --rm    --network=none    --entrypoint=stackql
        ...    --user=${user}    --volume=${SIGNED_FIXTURES}:/signatures:ro    ${image}
    END
    ${result}=    Run Process    @{command}    exec
    ...    SHOW RESOURCES IN ${provider}.orgs;
    ...    --registry\=${config}    --preview\={"unstable":true}    --output\=csv
    ...    --approot\=${approot}
    ...    cwd=${SIGNATURE_REPO}    timeout=30s    on_timeout=terminate
    ...    stdout=${SIGNED_FIXTURES}${/}${variant}-${provider}.stdout
    ...    stderr=${SIGNED_FIXTURES}${/}${variant}-${provider}.stderr
    RETURN    ${result}

Signed Registry Must Fail
    [Arguments]    ${variant}    ${message}
    FOR    ${provider}    IN    fixture    stackql_unstable_fixture
        ${result}=    Read Signed Registry    ${variant}    ${provider}
        Should Be Empty    ${result.stdout}
        Should Contain    ${result.stderr}    ${message}
        Should Contain    ${result.stderr}    pqsig:
    END
