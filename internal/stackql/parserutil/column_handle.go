package parserutil

import (
	"vitess.io/vitess/go/vt/sqlparser"
)

type ColumnHandle struct {
	Alias           string
	Expr            sqlparser.Expr
	Name            string
	Qualifier       string
	DecoratedColumn string
	IsColumn        bool
	Type            sqlparser.ValType
	Val             *sqlparser.SQLVal
}
