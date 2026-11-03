package parserutil

import (
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

func IsInsertRowsDynamic(node *sqlparser.Insert) bool {
	switch rowsNode := node.Rows.(type) {
	case *sqlparser.Select:
		return IsSelectDynamic(rowsNode)
	case sqlparser.Values:
		return false
	default:
		return true
	}
}

func IsSelectDynamic(node *sqlparser.Select) bool {
	if node.Where != nil {
		return true
	}
	return false
}
