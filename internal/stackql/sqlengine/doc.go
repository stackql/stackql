// Package sqlengine owns StackQL's database connections and local SQL execution.
// Callers depend on SQLEngine; backend implementations and SQLite driver
// registration remain private to this package.
//
// Omni supplies provider rows without receiving a database connection. StackQL
// uses this package to stage those finished rows, evaluate the remaining SQL,
// and maintain local tables, views, and catalogue metadata.
package sqlengine
