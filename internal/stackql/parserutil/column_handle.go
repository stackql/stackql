package parserutil

import (
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
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
