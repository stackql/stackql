package parserutil

import (
	"vitess.io/vitess/go/vt/sqlparser"
)

type ComparisonTableMap map[*sqlparser.ComparisonExpr]sqlparser.TableExpr
