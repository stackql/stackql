package parserutil

import (
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

type ComparisonTableMap map[*sqlparser.ComparisonExpr]sqlparser.TableExpr
