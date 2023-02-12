package parserutil

import (
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

type ColumnUsageMetadata struct {
	ColName *sqlparser.ColName
	ColVal  *sqlparser.SQLVal
}
