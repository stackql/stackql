package parserutil

import (
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

// ColTableMap maps a ColumnarReference (column-like input)
// to a "Table Expression"; this may be a simple (single table) or
// composite (eg: subquery, union) object.
type ColTableMap map[ColumnarReference]sqlparser.TableExpr
