*** Settings ***
Resource          ${CURDIR}/stackql.resource
Suite Setup       Prepare StackQL Environment
Suite Teardown    Terminate All Processes    kill=True

