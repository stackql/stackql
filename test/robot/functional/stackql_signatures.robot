*** Settings ***
Library           Process
Library           OperatingSystem
Suite Setup       Prepare Signed Registry Fixtures
Suite Teardown    Remove Directory    ${SIGNED_FIXTURES}    recursive=True
Test Tags         provider-signatures

*** Variables ***
${SIGNATURE_REPO}    ${CURDIR}${/}..${/}..${/}..
${SIGNATURE_EXE}     ${SIGNATURE_REPO}${/}build${/}stackql

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
    ${result}=    Run Process    go    run    ./test/tools/signing-fixtures    -out    ${directory}
    ...    cwd=${SIGNATURE_REPO}    stdout=${directory}${/}generator.stdout    stderr=${directory}${/}generator.stderr
    Should Be Equal As Integers    ${result.rc}    0    ${result.stderr}
    ${exe}=    Set Variable    ${SIGNATURE_EXE}
    IF    ${{os.name == 'nt'}}
        ${exe}=    Set Variable    ${SIGNATURE_EXE}.exe
    END
    Set Suite Variable    ${SIGNED_EXE}    ${exe}

Read Signed Registry
    [Arguments]    ${variant}    ${provider}
    ${config}=    Get File    ${SIGNED_FIXTURES}${/}${variant}.json
    ${result}=    Run Process    ${SIGNED_EXE}    exec
    ...    SHOW RESOURCES IN ${provider}.orgs;
    ...    --registry\=${config}    --preview\={"unstable":true}    --output\=csv
    ...    --approot\=${SIGNED_FIXTURES}${/}app-${variant}-${provider}
    ...    cwd=${SIGNATURE_REPO}    timeout=30s    on_timeout=kill
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
