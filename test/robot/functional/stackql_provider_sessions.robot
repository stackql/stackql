*** Settings ***
Documentation     Explicit provider sessions, credential rotation and UNSET through the CLI against loopback mocks.
Resource          ${CURDIR}/stackql.resource
Library           ProviderSessionLibrary.py
Suite Setup       Prepare Session Provider    ${STACKQL_EXE}    ${EXECUTION_PLATFORM}    ${SQL_BACKEND_CFG_STR_CANONICAL}
Suite Teardown    Stop Session Provider

*** Test Cases ***
Provider Aliases Keep Their Credentials And Scope Across Session Edits
    Provider Aliases Survive Session Edits

AWS Aliases Union Buckets With Independent Signing Credentials
    AWS Aliases Union Buckets With Independent Signing Credentials

OS Secrets Rotate Without Recreating The Session
    Secret Credentials Rotate And Reset

Auth Dto Fields Can Be Set And Unset In Memory
    Auth Dto Changes And Unsets

Unset Alias Auth Fails Before Any Provider Request
    Unset Alias Auth Fails Before Request

Explicit Session Scope Supports Set And Unset
    Explicit Session Scope

Global Configuration Scope Fails Before Provider Execution
    Unsupported Configuration Scope

Dollar Targets Configure Runtime And Plain Targets Store Variables
    Dollar Targets Configure Runtime And Plain Targets Store Variables

Scalar Variables Preserve Functions Operators And Types
    Scalar Variables Preserve Sql Expressions

Variable Predicates Resolve Before Provider Dispatch
    Variable Predicate Resolves Before Provider Dispatch

Missing Unset And Secret Variables Fail Before Dispatch
    Variable Errors Prevent Provider Dispatch

Secret Dereferencing Requires Trusted Runtime Opt In
    Secret Variables Require Runtime Opt In

Persistent Secret Stores Require Operator Opt In
    Persistent Secret Stores Require Operator Opt In

Flat File Secret Stores Are Rejected Even With Opt In
    Flat File Secret Stores Are Rejected Even With Opt In

OS Secrets Survive New Sessions And Namespaces Are Isolated
    OS Secrets Survive New Sessions And Namespaces Are Isolated

In Memory Auth Values Support SQL With Explicit Opt In
    In Memory Auth Values Support SQL With Explicit Opt In

Structured OS Secret Fields Support SQL With Explicit Opt In
    Structured OS Secret Fields Support SQL With Explicit Opt In

Scalar Variables Support Conversion Concatenation And Digest Chains
    Scalar Variables Support Conversion Concatenation And Digest Chains

Conversion And Digest Chains Bind Provider Parameters
    Conversion And Digest Chains Bind Provider Parameters

Variables Resolve Relation Column Alias Order And Limit Identifiers
    Variables Resolve Relation Column Alias Order And Limit Identifiers

Quoted Variables And Identifier Values Cannot Inject SQL
    Quoted Variables And Identifier Values Cannot Inject SQL

Memory Secrets Rotate Delete And Expire With Session
    Memory Secrets Rotate Delete And Expire With Session

Independent SQL Sessions Isolate Variables Secrets And Runtime Config
    Independent SQL Sessions Isolate Variables Secrets And Runtime Config

Variable Identifiers Work In DDL And Table Mutations
    Variable Identifiers Work In DDL And Table Mutations

Variable Identifiers Work In CTEs Joins And Subqueries
    Variable Identifiers Work In CTEs Joins And Subqueries

String Boolean Integer And Null Variables Filter Provider Rows
    String Boolean Integer And Null Variables Filter Provider Rows
