package router

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/sql_system"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
)

func obtainAnnotationCtx(
	sqlSystem sql_system.SQLSystem,
	tbl tablemetadata.ExtendedTableMetadata,
	parameters map[string]interface{},
	namespaceCollection tablenamespace.TableNamespaceCollection,
) (taxonomy.AnnotationCtx, error) {
	_, isView := tbl.GetView()
	_, isSQLDataSource := tbl.GetSQLDataSource()
	if isView || isSQLDataSource {
		// TODO: upgrade this flow; nil == YUCK!!!
		return taxonomy.NewStaticStandardAnnotationCtx(nil, tbl.GetHeirarchyObjects().GetHeirarchyIds(), tbl, parameters), nil
	}
	schema, mediaType, err := tbl.GetResponseSchemaAndMediaType()
	if err != nil {
		return nil, err
	}
	itemObjS, selectItemsKey, err := schema.GetSelectSchema(tbl.LookupSelectItemsKey(), mediaType)
	unsuitableSchemaMsg := "schema unsuitable for select query"
	if err != nil {
		return nil, fmt.Errorf(unsuitableSchemaMsg)
	}
	tbl.SetSelectItemsKey(selectItemsKey)
	provStr, _ := tbl.GetProviderStr()
	svcStr, _ := tbl.GetServiceStr()
	rscStr, _ := tbl.GetResourceStr()
	methodStr, _ := tbl.GetMethodStr()
	if itemObjS == nil {
		return nil, fmt.Errorf(unsuitableSchemaMsg)
	}
	name := itemObjS.GetSelectionName()
	tbl, err = tbl.WithResponseSchemaStr(name)
	if err != nil {
		return nil, fmt.Errorf("%s: %s", unsuitableSchemaMsg, err.Error())
	}
	tn, err := tbl.GetTableName()
	if err == nil && namespaceCollection.GetAnalyticsCacheTableNamespaceConfigurator().IsAllowed(tn) {
		name, _ = tbl.GetResponseSchemaStr()
	}
	hIds := internaldto.NewHeirarchyIdentifiers(provStr, svcStr, rscStr, methodStr).WithResponseSchemaStr(name)
	viewDTO, isView := sqlSystem.GetViewByName(hIds.GetTableName())
	if isView {
		hIds = hIds.WithView(viewDTO)
	}
	return taxonomy.NewStaticStandardAnnotationCtx(itemObjS, hIds, tbl, parameters), nil
}
