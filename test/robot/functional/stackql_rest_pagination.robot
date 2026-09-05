*** Settings ***
Resource          ${CURDIR}/stackql.resource
Test Teardown     Stackql Per Test Teardown
Documentation     any-sdk page_number REST pagination (issue 684) via the no-auth
...               stackql_native_test.paged service. The mock serves three pages of
...               two items; each row carries wire_page so traversal is asserted
...               from STDOUT. The unterminated resource omits total_pages.
...               Also: Link-header pagination (any-sdk #123) via the linked service and
...               requestToken.encoding none (any-sdk #121) via the cursor service.

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

Link Header Pagination Response Token Only Traverses All Pages
    [Documentation]    any-sdk #123: responseToken {key: link, location: header} alone
    ...                drives traversal; the Link URL replaces the request URL. DB6 is
    ...                only served on page 3 and rows carry the page they came from.
    [Timeout]    120 seconds
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, wire_page from stackql_native_test.linked.items order by name;
    ...    | DB6${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}3 |

Link Header Pagination With Query Request Token Follows Link Verbatim
    [Documentation]    A declared query requestToken (fromName) is ignored for the Link
    ...                scheme: the mock sees ?fromName=DB3 from the Link URL, never a
    ...                percent-encoded URL stuffed into fromName.
    [Timeout]    120 seconds
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, wire_page, wire_query from stackql_native_test.linked.items_query_token order by name;
    ...    | DB4${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}2 | fromName\=DB3 |

Link Header Pagination Algorithm On Differently Named Header Traverses
    [Documentation]    algorithm link_header_next reads the next URL from X-Next-Page.
    [Timeout]    120 seconds
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, wire_page from stackql_native_test.linked.items_xnext order by name;
    ...    | DB6${SPACE}${SPACE}|${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}3 |

Link Header Pagination Stops When Final Page Has No Link
    [Documentation]    Negative: page 3 carries no Link, so exactly 6 rows and the run
    ...                completes (the timeout guards against the old worker-goroutine hang).
    [Timeout]    120 seconds
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select count(*) as linked_row_tally from stackql_native_test.linked.items;
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}6 |

Link Header Pagination Ignores Prev Only Link
    [Documentation]    Negative: a Link header carrying only rel="prev" yields one page.
    [Timeout]    120 seconds
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select count(*) as prev_only_row_tally from stackql_native_test.linked.items_prev_only;
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}2 |

Cursor Request Token Encoding None Sends Cursor Verbatim
    [Documentation]    any-sdk #121: requestToken.encoding none appends the base64-padded
    ...                cursor as issued (page=page_AAAA==), so all three pages traverse.
    [Timeout]    120 seconds
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, wire_query from stackql_native_test.cursor.items order by name;
    ...    | C6${SPACE}${SPACE}${SPACE}| page\=page_BBBB\=\= |

Cursor Request Token Default Encoding Unchanged
    [Documentation]    Negative (default byte-identical): without encoding the cursor is
    ...                percent-escaped, the mock 400s the second page, and only the first
    ...                page of rows returns with the mock's message on stderr.
    [Timeout]    120 seconds
    Should Stackql Exec Inline Contain Both Streams
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select count(*) as default_encoding_row_tally from stackql_native_test.cursor.items_default_encoding;
    ...    |${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}${SPACE}2 |
    ...    The page token is invalid

Cursor Request Token Encoding None Keeps Other Parameters Escaped
    [Documentation]    Only the cursor is verbatim: a second query parameter containing a
    ...                space is still percent-encoded alongside it.
    [Timeout]    120 seconds
    Should StackQL Exec Inline Contain
    ...    ${STACKQL_EXE}
    ...    ${OKTA_SECRET_STR}
    ...    ${GITHUB_SECRET_STR}
    ...    ${K8S_SECRET_STR}
    ...    ${REGISTRY_NO_VERIFY_CFG_STR}
    ...    ${AUTH_CFG_STR}
    ...    ${SQL_BACKEND_CFG_STR_CANONICAL}
    ...    select name, wire_query from stackql_native_test.cursor.items where group_by \= 'project id' order by name;
    ...    group_by\=project+id&page\=page_AAAA\=\=
