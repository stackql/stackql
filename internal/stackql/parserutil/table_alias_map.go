package parserutil

import (
	"vitess.io/vitess/go/vt/sqlparser"
)

type TableAliasMap map[string]sqlparser.TableExpr
