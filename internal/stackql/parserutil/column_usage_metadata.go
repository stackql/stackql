package parserutil

import (
	"vitess.io/vitess/go/vt/sqlparser"
)

type ColumnUsageMetadata struct {
	ColName *sqlparser.ColName
	ColVal  *sqlparser.SQLVal
}
