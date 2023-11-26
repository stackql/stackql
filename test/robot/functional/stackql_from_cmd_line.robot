*** Settings ***
Resource          ${CURDIR}/stackql.resource

*** Test Cases *** 
Positive Control
    Should contain    ''    ''

Get Providers
    Should StackQL Exec Contain    ${SHOW_PROVIDERS_STR}   okta
    Should StackQL Novel Exec Contain    ${SHOW_PROVIDERS_STR}   v0.3.1
    Should StackQL Exec Contain JSON output    ${SHOW_PROVIDERS_STR}   okta

Get Providers No Config
    Should StackQL No Cfg Exec Contain    ${SHOW_PROVIDERS_STR}   name

Get Okta Services
    Should StackQL Exec Contain    ${SHOW_OKTA_SERVICES_FILTERED_STR}    Application${SPACE}API

Get Okta Application Resources
    Should StackQL Exec Contain    ${SHOW_OKTA_APPLICATION_RESOURCES_FILTERED_STR}    grants    groups

Describe GitHub Repos Pages
    Should StackQL Novel Exec Contain    ${DESCRIBE_GITHUB_REPOS_PAGES}    https_certificate    url

Describe AWS EC2 Instances Exemplifies Deep XPath for schema
    Should StackQL NoVerify Only Exec Contain    ${DESCRIBE_AWS_EC2_INSTANCES}
                                                 ...  architecture    bootMode    subnetId
                                                 ...  stdout=${CURDIR}/tmp/describe-aws-ec2-instances.tmp

Describe AWS EC2 Default KMS Key ID Exemplifies Top Level XPath for schema
    Should StackQL NoVerify Only Exec Contain    ${DESCRIBE_AWS_EC2_DEFAULT_KMS_KEY_ID}
                                                 ...  kmsKeyId
                                                 ...  stdout=${CURDIR}/tmp/describe-aws-ec2-default-kms-key-id.tmp

Describe allOf Object Returns Expected Result
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |---------------------------------------|---------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}name${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}type${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}id${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}@odata.type${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}_@odata.type${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}aboutMe${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}accountEnabled${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}boolean${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}activities${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}ageGroup${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}agreementAcceptances${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}appRoleAssignments${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}assignedLicenses${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}assignedPlans${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}authentication${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}object${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}authorizationInfo${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}object${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}birthday${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}businessPhones${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}calendar${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}object${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}calendarGroups${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}calendarView${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}calendars${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}chats${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}city${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}companyName${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}consentProvidedForMinor${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}contactFolders${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}contacts${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}country${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}createdDateTime${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}createdObjects${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}creationType${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}customSecurityAttributes${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}object${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}deletedDateTime${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}department${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}deviceEnrollmentLimit${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}integer${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}deviceManagementTroubleshootingEvents${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}directReports${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}displayName${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}drive${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}object${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}drives${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}employeeExperience${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}object${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}employeeHireDate${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}employeeId${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}employeeLeaveDateTime${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}employeeOrgData${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}object${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}employeeType${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}events${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}extensions${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}externalUserState${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}externalUserStateChangeDateTime${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}faxNumber${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}followedSites${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}givenName${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}hireDate${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}identities${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}imAddresses${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}inferenceClassification${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}object${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}insights${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}object${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}interests${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}isResourceAccount${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}boolean${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}jobTitle${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}joinedTeams${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}lastPasswordChangeDateTime${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}legalAgeGroupClassification${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}licenseAssignmentStates${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}licenseDetails${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}mail${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}mailFolders${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}mailNickname${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}mailboxSettings${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}object${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}managedAppRegistrations${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}managedDevices${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}manager${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}object${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}memberOf${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}messages${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}mobilePhone${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}mySite${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}oauth2PermissionGrants${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}officeLocation${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}onPremisesDistinguishedName${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}onPremisesDomainName${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}onPremisesExtensionAttributes${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}object${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}onPremisesImmutableId${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}onPremisesLastSyncDateTime${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}onPremisesProvisioningErrors${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}onPremisesSamAccountName${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}onPremisesSecurityIdentifier${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}onPremisesSyncEnabled${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}boolean${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}onPremisesUserPrincipalName${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}onenote${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}object${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}onlineMeetings${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}otherMails${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}outlook${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}object${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}ownedDevices${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}ownedObjects${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}passwordPolicies${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}passwordProfile${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}object${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}pastProjects${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}people${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}photo${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}object${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}photos${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}planner${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}object${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}postalCode${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}preferredDataLocation${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}preferredLanguage${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}preferredName${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}presence${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}object${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}print${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}object${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}provisionedPlans${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}proxyAddresses${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}registeredDevices${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}responsibilities${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}schools${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}scopedRoleMemberOf${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}securityIdentifier${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}serviceProvisioningErrors${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}settings${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}object${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}showInAddressList${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}boolean${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}signInActivity${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}object${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}signInSessionsValidFromDateTime${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}skills${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}state${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}streetAddress${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}surname${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}teamwork${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}object${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}todo${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}object${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}transitiveMemberOf${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}usageLocation${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}userPrincipalName${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    ...    |${SPACE}userType${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |---------------------------------------|---------|
    Should StackQL Current Exec Equal    
    ...    describe stackql_test.users.users;
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Describe-allOf-Object-Returns-Expected-Result.tmp
    ...    stderr=${CURDIR}/tmp/Describe-allOf-Object-Returns-Expected-Result-stderr.tmp

Describe anyOf Object Returns Expected Result
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |--------------------------------------|---------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}name${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}type${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}id${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}description${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}@odata.type${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}_@odata.type${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}acceptedSenders${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}address${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}allowExternalSenders${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}boolean${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}appRoleAssignments${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}assignedLabels${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}assignedLicenses${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}autoSubscribeNewMembers${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}boolean${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}calendar${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}calendarView${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}classification${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}conversations${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}coordinates${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}createdDateTime${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}createdOnBehalfOf${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}deletedDateTime${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}displayName${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}drive${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}drives${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}events${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}expirationDateTime${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}extensions${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}groupLifecyclePolicies${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}groupTypes${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}hasMembersWithLicenseErrors${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}boolean${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}hideFromAddressLists${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}boolean${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}hideFromOutlookClients${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}boolean${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}isArchived${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}boolean${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}isAssignableToRole${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}boolean${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}isSubscribedByMail${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}boolean${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}licenseProcessingState${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}locationEmailAddress${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}locationType${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}locationUri${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}mail${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}mailEnabled${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}boolean${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}mailNickname${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}memberOf${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}members${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}membersWithLicenseErrors${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}membershipRule${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}membershipRuleProcessingState${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}microsoft.graph.location_@odata.type${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}microsoft.graph.location_displayName${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}onPremisesDomainName${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}onPremisesLastSyncDateTime${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}onPremisesNetBiosName${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}onPremisesProvisioningErrors${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}onPremisesSamAccountName${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}onPremisesSecurityIdentifier${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}onPremisesSyncEnabled${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}boolean${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}onenote${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}owners${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}permissionGrants${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}photo${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}photos${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}planner${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}preferredDataLocation${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}preferredLanguage${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}proxyAddresses${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}rejectedSenders${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}renewedDateTime${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}securityEnabled${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}boolean${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}securityIdentifier${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}serviceProvisioningErrors${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}settings${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}sites${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}team${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}theme${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}threads${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}transitiveMemberOf${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}transitiveMembers${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}uniqueId${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}uniqueIdType${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}unseenCount${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}integer${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}visibility${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    Should StackQL Current Exec Equal    
    ...    describe stackql_test.totally_contrived.any_of_explore;
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Describe-anyOf-Object-Returns-Expected-Result.tmp
    ...    stderr=${CURDIR}/tmp/Describe-anyOf-Object-Returns-Expected-Result-stderr.tmp

Describe oneOf Object Returns Expected Result
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |--------------------------------------|---------|
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}name${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}type${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}id${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}description${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}@odata.type${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}_@odata.type${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}acceptedSenders${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}address${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}allowExternalSenders${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}boolean${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}appRoleAssignments${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}assignedLabels${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}assignedLicenses${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}autoSubscribeNewMembers${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}boolean${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}calendar${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}calendarView${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}classification${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}conversations${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}coordinates${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}createdDateTime${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}createdOnBehalfOf${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}deletedDateTime${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}displayName${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}drive${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}drives${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}events${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}expirationDateTime${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}extensions${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}groupLifecyclePolicies${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}groupTypes${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}hasMembersWithLicenseErrors${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}boolean${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}hideFromAddressLists${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}boolean${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}hideFromOutlookClients${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}boolean${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}isArchived${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}boolean${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}isAssignableToRole${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}boolean${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}isSubscribedByMail${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}boolean${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}licenseProcessingState${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}locationEmailAddress${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}locationType${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}locationUri${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}mail${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}mailEnabled${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}boolean${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}mailNickname${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}memberOf${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}members${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}membersWithLicenseErrors${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}membershipRule${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}membershipRuleProcessingState${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}microsoft.graph.location_@odata.type${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}microsoft.graph.location_displayName${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}onPremisesDomainName${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}onPremisesLastSyncDateTime${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}onPremisesNetBiosName${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}onPremisesProvisioningErrors${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}onPremisesSamAccountName${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}onPremisesSecurityIdentifier${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}onPremisesSyncEnabled${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}boolean${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}onenote${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}owners${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}permissionGrants${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}photo${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}photos${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}planner${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}preferredDataLocation${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}preferredLanguage${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}proxyAddresses${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}rejectedSenders${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}renewedDateTime${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}securityEnabled${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}boolean${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}securityIdentifier${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}serviceProvisioningErrors${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}settings${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}sites${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}team${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}theme${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}threads${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}transitiveMemberOf${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}transitiveMembers${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}array${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}uniqueId${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}uniqueIdType${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}unseenCount${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}integer${SPACE}|
    ...    |--------------------------------------|---------|
    ...    |${SPACE}visibility${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}string${SPACE}${SPACE}|
    ...    |--------------------------------------|---------|
    Should StackQL Current Exec Equal    
    ...    describe stackql_test.totally_contrived.one_of_explore;
    ...    ${outputStr}
    ...    stdout=${CURDIR}/tmp/Describe-oneOf-Object-Returns-Expected-Result.tmp
    ...    stderr=${CURDIR}/tmp/Describe-oneOf-Object-Returns-Expected-Result-stderr.tmp

Show Methods GitHub
    Should StackQL Novel Exec Equal    ${SHOW_METHODS_GITHUB_REPOS_REPOS}   ${SHOW_METHODS_GITHUB_REPOS_REPOS_EXPECTED}

Show Methods including server params AWS 
    ${outputStr} =    Catenate    SEPARATOR=\n
    ...    |---------------|----------------|---------|
    ...    |${SPACE}${SPACE}MethodName${SPACE}${SPACE}${SPACE}|${SPACE}RequiredParams${SPACE}|${SPACE}SQLVerb${SPACE}|
    ...    |---------------|----------------|---------|
    ...    |${SPACE}vpcs_Describe${SPACE}|${SPACE}region${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}SELECT${SPACE}${SPACE}|
    ...    |---------------|----------------|---------|
    ...    |${SPACE}vpc_Create${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}region${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}INSERT${SPACE}${SPACE}|
    ...    |---------------|----------------|---------|
    ...    |${SPACE}vpc_Delete${SPACE}${SPACE}${SPACE}${SPACE}|${SPACE}VpcId,${SPACE}region${SPACE}${SPACE}|${SPACE}DELETE${SPACE}${SPACE}|
    ...    |---------------|----------------|---------|
    Should StackQL Current Exec Equal    
    ...    show methods in aws.ec2.vpcs;
    ...    ${outputStr}

Show Insert Google Container Clusters
    Should StackQL Exec Contain    
    ...    SHOW INSERT INTO google.container."projects.zones.clusters";
    ...    ${SHOW_INSERT_GOOGLE_CONTAINER_CLUSTERS} 
    ...    stackql_H=True
    ...    stdout=${CURDIR}/tmp/Show-Insert-Google-Container-Clusters.tmp
    
JSONNET Plus Env Vars
    ${varList}=    Create List    project=stackql-demo    region=australia-southeast1
    Should StackQL Exec Equal    
    ...    ""
    ...    ${JSONNET_PLUS_ENV_VARS_EXPECTED} 
    ...    stackql_H=True 
    ...    stackql_dryrun=True
    ...    stackql_i=${JSONNET_PLUS_ENV_VARS_QUERY_FILE}
    ...    stackql_vars=${varList}
    ...    stackql_iqldata=${JSONNET_PLUS_ENV_VARS_VAR_FILE}
    ...    stdout=${CURDIR}/tmp/JSONNET-Plus-Env-Vars.tmp

Show Extended Insert Google BQ Datasets
    Should StackQL Exec Contain    
    ...    SHOW EXTENDED INSERT INTO google.bigquery.datasets;
    ...    ${SHOW_INSERT_GOOGLE_BIGQUERY_DATASET} 
    ...    stackql_H=True
    ...    stdout=${CURDIR}/tmp/Show-Extended-Insert-Google-BQ-Datasets.tmp   

Show Insert Google BQ Datasets
    Should StackQL Exec Contain    
    ...    SHOW INSERT INTO google.bigquery.datasets;
    ...    ${SHOW_INSERT_GOOGLE_BIGQUERY_DATASET} 
    ...    stackql_H=True
    ...    stdout=${CURDIR}/tmp/Show-Insert-Google-BQ-Datasets.tmp

*** Keywords ***
Should StackQL Exec Equal
    [Arguments]    ${_EXEC_CMD_STR}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}    &{kwargs}
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    &{kwargs}
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_CANONICAL_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    &{kwargs}
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_DEPRECATED_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    &{kwargs}

Should StackQL Current Exec Equal
    [Arguments]    ${_EXEC_CMD_STR}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}    &{kwargs}
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    &{kwargs}

Should StackQL Novel Exec Equal
    [Arguments]    ${_EXEC_CMD_STR}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}    &{kwargs}
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    &{kwargs}
    Should StackQL Exec Inline Equal
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_CANONICAL_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    &{kwargs}

Should StackQL Exec Contain
    [Arguments]    ${_EXEC_CMD_STR}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}    &{kwargs}
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    &{kwargs}
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_CANONICAL_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    &{kwargs}
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_DEPRECATED_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    &{kwargs}

Should StackQL Exec Contain JSON output
    [Arguments]    ${_EXEC_CMD_STR}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}    &{kwargs}
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    \-o\=json
    ...    &{kwargs}
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_CANONICAL_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    \-o\=json
    ...    &{kwargs}
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_DEPRECATED_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    \-o\=json
    ...    &{kwargs}

Should StackQL Novel Exec Contain
    [Arguments]    ${_EXEC_CMD_STR}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}    &{kwargs}
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    &{kwargs}
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_CANONICAL_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    &{kwargs}

Should StackQL NoVerify Only Exec Contain
    [Arguments]    ${_EXEC_CMD_STR}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}    &{kwargs}
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    &{kwargs}

Should StackQL No Cfg Exec Contain
    [Arguments]    ${_EXEC_CMD_STR}    ${_EXEC_CMD_EXPECTED_OUTPUT}    @{varargs}    &{kwargs}
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NULL}
    ...    ${EMPTY}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    ${_EXEC_CMD_STR}
    ...    ${_EXEC_CMD_EXPECTED_OUTPUT}
    ...    &{kwargs}


