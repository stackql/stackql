package astformat

import (
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

func DefaultSelectExprsFormatter(buf *sqlparser.TrackedBuffer, node sqlparser.SQLNode) {
	switch node := node.(type) {
	case sqlparser.ColIdent:
		formatColIdent(node, buf)
		return
	case sqlparser.TableName:
		buf.WriteString(node.GetRawVal())
		return
	default:
		node.Format(buf)
		return
	}
}
