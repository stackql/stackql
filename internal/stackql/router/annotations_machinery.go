package router

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/internaldto"
	"github.com/stackql/stackql/internal/stackql/sqldialect"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
)

func obtainAnnotationCtx(
	sqlDialect sqldialect.SQLDialect,
	tbl tablemetadata.ExtendedTableMetadata,
	parameters map[string]interface{},
	namespaceCollection tablenamespace.TableNamespaceCollection,
) (taxonomy.AnnotationCtx, error) {
	if tbl.IsView() {
		// TODO: upgrade this flow
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
	isView := sqlDialect.ViewExists(hIds.GetTableName())
	hIds = hIds.WithIsView(isView)
	return taxonomy.NewStaticStandardAnnotationCtx(itemObjS, hIds, tbl, parameters), nil
}
