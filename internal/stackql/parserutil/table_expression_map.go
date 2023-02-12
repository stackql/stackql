package parserutil

import "github.com/stackql/stackql-parser/go/vt/sqlparser"

type TableExprMap map[sqlparser.TableName]sqlparser.TableExpr

func (tem TableExprMap) GetByAlias(alias string) (sqlparser.TableExpr, bool) {
	for k, v := range tem {
		if k.GetRawVal() == alias {
			return v, true
		}
	}
	return nil, false
}

func (tm TableExprMap) SingleTableMap(filterTable sqlparser.TableName) TableExprMap {
	rv := make(TableExprMap)
	for k, v := range tm {
		if k == filterTable {
			rv[k] = v
		}
	}
	return rv
}

func (tm TableExprMap) ToStringMap() map[string]interface{} {
	rv := make(map[string]interface{})
	for k, v := range tm {
		rv[k.GetRawVal()] = v
	}
	return rv
}
