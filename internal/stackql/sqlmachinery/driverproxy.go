package sqlmachinery

import "database/sql"

type Querier interface {
	Query(string, ...interface{}) (*sql.Rows, error)
}

type ExecQuerier interface {
	Exec(string, ...interface{}) (sql.Result, error)
}
