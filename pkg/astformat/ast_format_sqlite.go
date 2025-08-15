package astformat

import (
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

func SQLiteSelectExprsFormatter(buf *sqlparser.TrackedBuffer, node sqlparser.SQLNode) {
	switch node := node.(type) {
	case sqlparser.ColIdent:
		formatColIdentCaseInsensitive(node, buf)
		return

	default:
		node.Format(buf)
		return
	}
}
