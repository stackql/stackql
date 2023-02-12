package sqltypeutil

import (
	"github.com/stackql/stackql-parser/go/sqltypes"
)

func InterfaceToSQLType(val interface{}) (sqltypes.Value, error) {
	switch t := val.(type) {
	case string:
		return sqltypes.InterfaceToValue([]byte(t))
	case bool:
		var v int64
		if t {
			v = 1
		}
		return sqltypes.InterfaceToValue(v)
	}
	return sqltypes.InterfaceToValue(val)
}
