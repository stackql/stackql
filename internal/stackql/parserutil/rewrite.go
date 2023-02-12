package parserutil

import (
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

func naiveRewriteComparisonExpr(ex *sqlparser.ComparisonExpr) {
	ex.Left = &sqlparser.SQLVal{Type: sqlparser.IntVal, Val: []byte("1")}
	ex.Right = &sqlparser.SQLVal{Type: sqlparser.IntVal, Val: []byte("1")}
	// ex.Operator =  ex.Operator
	// ex.Escape =   ex.Escape
}

func NaiveRewriteComparisonExprs(m map[*sqlparser.ComparisonExpr]struct{}) {
	for k, _ := range m {
		naiveRewriteComparisonExpr(k)
	}
}
