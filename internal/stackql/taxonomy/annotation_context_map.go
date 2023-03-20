package taxonomy

import (
	"fmt"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

type AnnotationCtxMap map[sqlparser.SQLNode]AnnotationCtx

func (am AnnotationCtxMap) AssignParams() error {
	rv := make(map[string]interface{})
	for k, v := range am {
		for p, pVal := range v.GetParameters() {
			switch pVal.(type) { //nolint:gocritic // low impact
			case *sqlparser.ColName:
			}
			aliasedName, ok := k.(*sqlparser.AliasedTableExpr)
			if !ok {
				continue
			}
			tableAlias := aliasedName.As.GetRawVal()
			nk := fmt.Sprintf("%s.%s", tableAlias, p)
			rv[nk] = pVal
		}
	}
	return nil
}

func (am AnnotationCtxMap) GetStringParams() map[string]interface{} {
	rv := make(map[string]interface{})
	for k, v := range am {
		for p, pVal := range v.GetParameters() {
			aliasedName, ok := k.(*sqlparser.AliasedTableExpr)
			if !ok {
				continue
			}
			tableAlias := aliasedName.As.GetRawVal()
			nk := fmt.Sprintf("%s.%s", tableAlias, p)
			rv[nk] = pVal
		}
	}
	return rv
}
