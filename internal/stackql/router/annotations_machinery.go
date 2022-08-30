package router

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
)

func obtainAnnotationCtx(
	sqlEngine sqlengine.SQLEngine,
	tbl *taxonomy.ExtendedTableMetadata,
	parameters map[string]interface{},
) (taxonomy.AnnotationCtx, error) {
	schema, mediaType, err := tbl.GetResponseSchemaAndMediaType()
	if err != nil {
		return nil, err
	}
	itemObjS, selectItemsKey, err := schema.GetSelectSchema(tbl.LookupSelectItemsKey(), mediaType)
	unsuitableSchemaMsg := "schema unsuitable for select query"
	if err != nil {
		return nil, fmt.Errorf(unsuitableSchemaMsg)
	}
	tbl.SelectItemsKey = selectItemsKey
	provStr, _ := tbl.GetProviderStr()
	svcStr, _ := tbl.GetServiceStr()
	if itemObjS == nil {
		return nil, fmt.Errorf(unsuitableSchemaMsg)
	}
	name := itemObjS.GetSelectionName()
	hIds := dto.NewHeirarchyIdentifiers(provStr, svcStr, name, "")
	return taxonomy.NewStaticStandardAnnotationCtx(itemObjS, hIds, tbl, parameters), nil
}
