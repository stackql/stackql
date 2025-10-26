package obtain_context //nolint:revive,cyclop,stylecheck // TODO: allow

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/sql_system"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
)

func ObtainAnnotationCtx(
	sqlSystem sql_system.SQLSystem,
	tbl tablemetadata.ExtendedTableMetadata,
	parameters map[string]interface{},
	namespaceCollection tablenamespace.Collection,
) (taxonomy.AnnotationCtx, error) {
	_, isView := tbl.GetView()
	_, isSQLDataSource := tbl.GetSQLDataSource()
	_, isSubquery := tbl.GetSubquery()
	isPGInternalObject := tbl.GetHeirarchyObjects().IsPGInternalObject()
	if isView || isSQLDataSource || isSubquery || isPGInternalObject {
		// TODO: upgrade this flow; nil == YUCK!!!
		return taxonomy.NewStaticStandardAnnotationCtx(
			nil, tbl.GetHeirarchyObjects().GetHeirarchyIDs(),
			tbl, parameters, false,
		), nil
	}
	schema, mediaType, err := tbl.GetResponseSchemaAndMediaType()
	if err != nil {
		return nil, err
	}
	// TODO: emulate in address space
	itemObjS, selectItemsKey, err := schema.GetSelectSchema(tbl.LookupSelectItemsKey(), mediaType)
	unsuitableSchemaMsg := "schema unsuitable for select query"
	if err != nil {
		//nolint:govet // fine with this
		return nil, fmt.Errorf(unsuitableSchemaMsg)
	}
	tbl.SetSelectItemsKey(selectItemsKey)
	provStr, _ := tbl.GetProviderStr()
	svcStr, _ := tbl.GetServiceStr()
	rscStr, _ := tbl.GetResourceStr()
	methodStr, _ := tbl.GetMethodStr()
	if itemObjS == nil {
		//nolint:govet // fine with this
		return nil, fmt.Errorf(unsuitableSchemaMsg)
	}
	name := itemObjS.GetSelectionName()
	tbl, err = tbl.WithResponseSchemaStr(name)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", unsuitableSchemaMsg, err)
	}
	tn, err := tbl.GetTableName()
	if err == nil && namespaceCollection.GetAnalyticsCacheTableNamespaceConfigurator().IsAllowed(tn) {
		name, _ = tbl.GetResponseSchemaStr()
	}
	hIDs := internaldto.NewHeirarchyIdentifiers(provStr, svcStr, rscStr, methodStr).WithResponseSchemaStr(name)
	viewDTO, isView := sqlSystem.GetViewByName(hIDs.GetTableName())
	// TODO: match on params
	if isView {
		hIDs = hIDs.WithView(viewDTO)
	}
	return taxonomy.NewStaticStandardAnnotationCtx(itemObjS, hIDs, tbl, parameters, false), nil
}
