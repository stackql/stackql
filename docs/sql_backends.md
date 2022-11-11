
# SQL Backends

For `stackql`, an `SQL backend` is an abstraction that executes SQL statements.  The workload is is light on for updates and even lighter for read-write contention.  

At present, traditional RDBMS systems such as `sqlite` and `postgres` are used to implment SQL backends.  However, there is no reason that a non-traditional, analytics optimised system can not be used at least in part.

## Backends

### SQLite

The default implementation is **embedded** SQLite.  SQLite does **not** have a wire protocol or TCP-native version.

### Postgres

#### Postgres over TCP

- [Using golang SQL driver interfaces](https://github.com/jackc/pgx/wiki/Getting-started-with-pgx-through-database-sql#hello-world-from-postgresql).
- [PGX native (improved performance)](https://github.com/jackc/pgx/wiki/Getting-started-with-pgx).

#### Embedded Postgres

https://github.com/fergusstrange/embedded-postgres

#### Postgres integration bugs

Postgres failing tests checklist:

- [ ] Google IAM Policy Agg                                                 
- [ ] Google Join Plus String Concatenated Select Expressions               
- [ ] Okta Users Select Simple Paginated                                    
- [ ] AWS EC2 Volumes Select Simple                                         
- [ ] GitHub SAML Identities Select GraphQL                                 
- [ ] GitHub Repository With Functions Select                               
- [ ] Join GCP Okta Cross Provider JSON Dependent Keyword in Table Name     
- [ ] K8S Nodes Select Leveraging JSON Path                                 
- [ ] Data Flow Sequential Join Select With Functions Github                       
- [ ] Shell Session Simple                                                  
- [ ] PG Session Postgres Client Typed Queries                              
- [ ] PG Session Postgres Client V2 Typed Queries                       

## Technical notes

### Golang SQL drivers

How to use drivers:

- https://go.dev/doc/database/open-handle

List of drivers:

- https://github.com/golang/go/wiki/SQLDrivers

### Data Source Name (DSN) strings

- [SQLite as per golang](https://github.com/mattn/go-sqlite3#dsn-examples).
- [Postgres URI](https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING).
