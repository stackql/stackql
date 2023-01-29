
# High Level Design

## Architecture

The architecture of `stackql` can be regarded as an amalgam of:

1. an SQL parser and rewriter.
2. an ORM between arbitrary APIs and a traditional SQL RDBMS.  In practice all APIs are HTTP(S) APIs at this time.
3. a planner for DAGs defined by calls to APIs and their dependencies.
4. an executor for (3).
5. a handle to an SQL RDBMS, so that the application can support SQL semantics. 

---

`stackql` generalizes the idea of infrastructure / computing resources into a `provider`, `service`, `resource` hierarchy that can be queried with SQL semantics, plus some imperative operations which are not canonical SQL.  Potentially any infrastructure: computing, orchestration, storage, SAAS, PAAS offerings etc can be managed with `stackql`, although the primary driver is cloud infrastructure management.  Multi-provider queries are a first class citizen in `stackql`.

---

Considering query execution in a bottom-up manner from backend execution to frontend source code processing, the strategic design for `stackql` is:

  - Backend **Execution** of queries through `Primitive` interfaces that encapsulate access and mutation operations against arbitrary APIs.  `Primitive`s may act on any particular API, eg: http, SDK, IPC, specific wire protocol.  Potentially variegated (eg: part http API, part SDK).
  - A `Plan` object includes a [DAG](https://en.wikipedia.org/wiki/Directed_acyclic_graph) of `Primitive`s.  `Plan`s may be optimized and cached a la [vitess](https://github.com/vitessio/vitess).  Logically, the `Plan`, once initialized, is matured in the following sequential phases:
    1. **Intermediate Code Generation**; `//TODO` for now no formal language is defined.  Simply objects and function pointers of `stackql`, encapsulated in `Primitives`.
    2. **Code Optimization**; parallelization of independent operations, removal of redundant operations.
    3. **Code Generation**; final calls against whatever backend, eg HTTP API. 
  - **Semantic Analysis** of queries is a phase that accepts an AST as input and:
    - creates a symbol table.
    - analyzes provider hierarchies and API(s) required to complete the query.  Typically these would be sourced by downloading and cacheing provider discovery documents.
    - performs type checking, scope (label) analysis.
    - creates a `Planbuilder` object and decorates it during analysis.
    - **may** generate some primitives.
    - generates, at the very least, a `Plan` stub.
  - **Lexical and Syntax analysis**; using the machinery from Vitess, which is a lex / yacc style grammar, processed with golang libraries to emulate lex and yacc.  The [sqlparser](https://github.com/stackql/vitess/blob/feature/stackql-develop/go/vt/sqlparser) module, originally from [vitess](https://github.com/vitessio/vitess) contains the implementation.  The output is an AST.

The semantic analysis and latter phases are sensitive to the type and structure of provider backends.

---

## Components

At a high level, the component relationships are such as represented in Figure A1.
For brevity, the graph is simplified and much is omitted.
Database connection details are initialised from user supplied context and then attached to the `HandlerContext`, which is passed through as required.

![High Level Component Architecture](/docs/images/components-HLDD.drawio.svg)

**Figure A1**: High Level Component Architecture.

## Database and tables

The application leverages an RDBMS to implement SQL semantics.
The existing default implementation is an embedded `SQLite3` binary, accessibled via `CGO`.
By default the database instance is in memory, but can be persistent, specified via runtime arguments.

For each API response type, a database table can be lazily created (eagerly is slower, but we would not rule out some future opt-in scenarios for persistent databases).  
Table nomenclature includes dot (`.`) separated namespacing for provider, service, resource, schema type and `generation`; this latter being a positive integer version for the table itself.  Generation supports either analytics on extinct records or multiple versions of an API.

Each table contains control columns to identify the query and session to which records belong.
This will support audit, concurrency, and garbage collection.

### Garbage Collection

Garbage collection is not implemented at this point in time.
In the short term, a manual collect all option will be implemented.

**Table GC1**: Proposed database object lifetimes.
| Object | Proposed Minimum Lifetime | Proposed Maximum Lifetime |
| --- | ----------- | ----------- |
| Database |  |  |
| Tables | Same as API document lifetime | Same as API document lifetime |
| Records | Query or Session (latter possibly useful for SELECT) | Session or some arbitrary limit, on a best-effort basis |
