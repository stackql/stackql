*** Settings ***
Resource          ${CURDIR}/stackql.resource
Test Teardown     Stackql Per Test Teardown
Documentation     any-sdk page_number REST pagination (issue 684) via the no-auth
...               stackql_native_test.paged service. The mock serves three pages of
...               two items; each row carries wire_page so traversal is asserted
...               from STDOUT. The unterminated resource omits total_pages.

*** Test Cases ***
Page Number Pagination Traverses All Pages
    [Documentation]    paged-item-6 is only served on page 3, so its presence proves
    ...                the reader followed page 1 -> 2 -> 3 and stopped at total_pages.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, idx from stackql_native_test.paged.items order by idx;
    ...    paged-item-6

Page Number Pagination Requests Successive Pages On The Wire
    [Documentation]    The mock stamps each row with the page it served it on; rows
    ...                with wire_page 3 prove the third wire request carried page=3.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name from stackql_native_test.paged.items where wire_page \= 3 order by idx;
    ...    paged-item-5

Page Number Pagination Missing Terminator Stops After One Page
    [Documentation]    Negative case: no total_pages terminator, so exactly one page
    ...                (2 rows) is fetched - never an infinite loop.
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select count(*) as unterminated_row_tally from stackql_native_test.paged.items_unterminated;
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}2${SPACE}|
