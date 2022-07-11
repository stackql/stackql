package resultutil

import (
	openapistackql_util "github.com/stackql/go-openapistackql/pkg/util"
	"vitess.io/vitess/go/sqltypes"
	querypb "vitess.io/vitess/go/vt/proto/query"
)

func TransformRow(row []interface{}) []sqltypes.Value {
	rowVals := make([]sqltypes.Value, len(row))
	for j := range row {
		rvj, _ := sqltypes.NewValue(querypb.Type_TEXT, openapistackql_util.InterfaceToBytes(row[j], false))
		rowVals[j] = rvj
	}
	return rowVals
}
