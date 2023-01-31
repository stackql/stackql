package astformat

import (
	"vitess.io/vitess/go/vt/sqlparser"
)

func DefaultSelectExprsFormatter(buf *sqlparser.TrackedBuffer, node sqlparser.SQLNode) {
	switch node := node.(type) {
	case sqlparser.ColIdent:
		formatColIdent(node, buf)
		return

	default:
		node.Format(buf)
		return
	}
}
