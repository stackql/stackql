package primitivebuilder

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/primitivegraph"

	"github.com/stackql/go-openapistackql/openapistackql"
)

type Builder interface {
	Build() error

	GetRoot() primitivegraph.PrimitiveNode

	GetTail() primitivegraph.PrimitiveNode
}

func castItemsArray(iArr interface{}) ([]map[string]interface{}, error) {
	switch iArr := iArr.(type) {
	case []map[string]interface{}:
		return iArr, nil
	case []interface{}:
		var rv []map[string]interface{}
		for i := range iArr {
			item, ok := iArr[i].(map[string]interface{})
			if !ok {
				if iArr[i] != nil {
					item = map[string]interface{}{openapistackql.AnonymousColumnName: iArr[i]}
				} else {
					item = nil
				}
			}
			rv = append(rv, item)
		}
		return rv, nil
	default:
		return nil, fmt.Errorf("cannot accept items array of type = '%T'", iArr)
	}
}
