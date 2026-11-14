*** Settings ***
Documentation     SQL AssumeRole adapter regressions against an isolated Flask STS and SigV4 verifier.
Library           STSRoleLibrary.py
Test Setup        Prepare STS    ${STACKQL_EXE}    ${EXECUTION_PLATFORM}
Test Teardown     Stop STS

*** Variables ***
${STACKQL_EXE}          ${EXECDIR}/build/stackql
${EXECUTION_PLATFORM}   native

*** Test Cases ***
AWS Role Settings Reach STS And Returned Credentials Sign SQL Requests
    Role Auth Fields Reach Sts And Sign Downstream

AWS Role ARN And External ID Support Environment Indirection
    Role Auth Environment Indirection

AWS Role Expiry Refreshes Between SQL Statements
    Role Expiry Refreshes Between Sql Statements

AWS STS Rejection Never Dispatches Provider SQL
    Rejected Sts Never Dispatches Provider Query

AWS Expired STS Response Never Dispatches Provider SQL
    Expired Sts Never Dispatches Provider Query

AWS Static Session Credentials Bypass STS
    Static Session Credentials Do Not Call Sts

AWS Unset Role ARN Fails Before STS Dispatch
    Unset Role Configuration Fails Before Sts

AWS Repeated SQL Reuses Unexpired Role Credentials
    Repeated Sql Reuses Unexpired Role Credentials

AWS Union Branches Share Role Acquisition
    Union Branches Share Role Acquisition

AWS Canceled SQL Does Not Dispatch With Incomplete Credentials
    Canceled Sql Stops Before Downstream Dispatch

AWS SQL Auth Rotation Replaces The Cached Role Source
    Sql Auth Rotation Replaces Role Source
