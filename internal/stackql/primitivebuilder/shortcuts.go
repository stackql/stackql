package primitivebuilder

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/metadatavisitors"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/provider"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"github.com/stackql/stackql/internal/stackql/util"
	"github.com/stackql/stackql/pkg/prettyprint"
)

func NewUpdateableValsPrimitive(handlerCtx handler.HandlerContext, vals map[*sqlparser.ColName]interface{}) (primitive.IPrimitive, error) {
	return primitive.NewLocalPrimitive(
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			keys := make(map[string]map[string]interface{})
			row := make(map[string]interface{})
			rawRow := make(map[int]interface{})
			var columnOrder []string
			i := 0
			lookupMap := make(map[string]*sqlparser.ColName)
			for k, _ := range vals {
				columnOrder = append(columnOrder, k.Name.GetRawVal())
				lookupMap[k.Name.GetRawVal()] = k
			}
			sort.Strings(columnOrder)
			for _, rk := range columnOrder {
				k := lookupMap[rk]
				v := vals[k]
				row[k.Name.GetRawVal()] = v
				rawRow[i] = v
				i++
			}
			keys["0"] = row
			rawRows := map[int]map[int]interface{}{
				0: rawRow,
			}
			return util.PrepareResultSet(internaldto.NewPrepareResultSetPlusRawDTO(nil, keys, columnOrder, nil, nil, nil, rawRows))
		}), nil
}

func NewInsertableValsPrimitive(handlerCtx handler.HandlerContext, vals map[int]map[int]interface{}) (primitive.IPrimitive, error) {
	return primitive.NewLocalPrimitive(
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			keys := make(map[string]map[string]interface{})
			row := make(map[string]interface{})
			var rowKeys []int
			var colKeys []int
			var columnOrder []string
			for k, _ := range vals {
				rowKeys = append(rowKeys, k)
			}
			for _, v := range vals {
				for ck, _ := range v {
					colKeys = append(colKeys, ck)
				}
				break
			}
			sort.Ints(rowKeys)
			sort.Ints(colKeys)
			for _, ck := range colKeys {
				columnOrder = append(columnOrder, "val_"+strconv.Itoa(ck))
			}
			for idx := range colKeys {
				col := vals[0][idx]
				colName := columnOrder[idx]
				row[colName] = col
			}
			keys["0"] = row
			return util.PrepareResultSet(internaldto.NewPrepareResultSetPlusRawDTO(nil, keys, columnOrder, nil, nil, nil, vals))
		}), nil
}

func NewShowInstructionExecutor(node *sqlparser.Show, prov provider.IProvider, tbl tablemetadata.ExtendedTableMetadata, handlerCtx handler.HandlerContext, commentDirectives sqlparser.CommentDirectives, tableFilter func(openapistackql.ITable) (openapistackql.ITable, error)) internaldto.ExecutorOutput {
	extended := strings.TrimSpace(strings.ToUpper(node.Extended)) == "EXTENDED"
	nodeTypeUpperCase := strings.ToUpper(node.Type)
	var keys map[string]map[string]interface{}
	var columnOrder []string
	var err error
	var filter func(interface{}) (openapistackql.ITable, error)
	logging.GetLogger().Infoln(fmt.Sprintf("filter type = %T", filter))
	switch nodeTypeUpperCase {
	case "AUTH":
		logging.GetLogger().Infoln(fmt.Sprintf("Show For node.Type = '%s'", node.Type))
		if err == nil {
			authCtx, err := handlerCtx.GetAuthContext(prov.GetProviderString())
			if err == nil {
				var authMeta *openapistackql.AuthMetadata
				authMeta, err = prov.ShowAuth(authCtx)
				if err == nil {
					keys = map[string]map[string]interface{}{
						"1": authMeta.ToMap(),
					}
					columnOrder = authMeta.GetHeaders()
				}
			}
		}
	case "INSERT":
		ppCtx := prettyprint.NewPrettyPrintContext(
			handlerCtx.GetRuntimeContext().OutputFormat == constants.PrettyTextStr,
			constants.DefaultPrettyPrintIndent,
			constants.DefaultPrettyPrintBaseIndent,
			"'",
			logging.GetLogger(),
		)
		if err != nil {
			return util.GenerateSimpleErroneousOutput(err)
		}
		meth, err := tbl.GetMethod()
		if err != nil {
			tblName, tblErr := tbl.GetStackQLTableName()
			if tblErr != nil {
				return util.GenerateSimpleErroneousOutput(fmt.Errorf("Cannot find matching operation, possible causes include missing required parameters or an unsupported method for the resource, to find required parameters for supported methods run SHOW METHODS: %s", err.Error()))
			}
			return util.GenerateSimpleErroneousOutput(fmt.Errorf("Cannot find matching operation, possible causes include missing required parameters or an unsupported method for the resource, to find required parameters for supported methods run SHOW METHODS IN %s: %s", tblName, err.Error()))
		}
		svc, err := tbl.GetService()
		if err != nil {
			return util.GenerateSimpleErroneousOutput(err)
		}
		pp := prettyprint.NewPrettyPrinter(ppCtx)
		ppPlaceholder := prettyprint.NewPrettyPrinter(ppCtx)
		requiredOnly := commentDirectives != nil && commentDirectives.IsSet("REQUIRED")
		insertStmt, err := metadatavisitors.ToInsertStatement(node.Columns, meth, svc, extended, pp, ppPlaceholder, requiredOnly)
		tableName, _ := tbl.GetTableName()
		if err != nil {
			return util.GenerateSimpleErroneousOutput(fmt.Errorf("error creating insert statement for %s: %s", tableName, err.Error()))
		}
		stmtStr := fmt.Sprintf(insertStmt, tableName)
		keys = map[string]map[string]interface{}{
			"1": {
				"insert_statement": stmtStr,
			},
		}
	case "METHODS":
		var rsc *openapistackql.Resource
		rsc, err = prov.GetResource(node.OnTable.Qualifier.GetRawVal(), node.OnTable.Name.GetRawVal(), handlerCtx.GetRuntimeContext())
		methods := rsc.GetMethodsMatched()
		var filter func(openapistackql.ITable) (openapistackql.ITable, error)
		if tbl == nil {
			logging.GetLogger().Infoln(fmt.Sprintf("table and therefore filter not found for AST, shall procede nil filter"))
		} else {
			filter = tbl.GetTableFilter()
		}
		methods, err = filterMethods(methods, filter)
		if err != nil {
			return util.PrepareResultSet(internaldto.NewPrepareResultSetDTO(nil, keys, columnOrder, nil, err, nil))
		}
		mOrd, err := methods.OrderMethods()
		if err != nil {
			return util.PrepareResultSet(internaldto.NewPrepareResultSetDTO(nil, keys, columnOrder, nil, err, nil))
		}
		methodKeys := make(map[string]map[string]interface{})
		for i, k := range mOrd {
			method := k
			methMap := method.ToPresentationMap(extended)
			methodKeys[fmt.Sprintf("%06d", i)] = methMap
			columnOrder = method.GetColumnOrder(extended)
		}
		keys = methodKeys
	case "PROVIDERS":
		keys, err = handlerCtx.GetSupportedProviders(extended)
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		rv := util.PrepareResultSet(internaldto.NewPrepareResultSetDTO(nil, keys, columnOrder, nil, err, nil))
		if len(keys) == 0 {
			rv = util.EmptyProtectResultSet(
				rv,
				[]string{"name", "version"},
			)
		}
		return rv
	case "RESOURCES":
		svcName := node.OnTable.Name.GetRawVal()
		if svcName == "" {
			return prepareErroneousResultSet(keys, columnOrder, fmt.Errorf("no service designated from which to resolve resources"))
		}
		var resources map[string]*openapistackql.Resource
		resources, err = prov.GetResourcesRedacted(svcName, handlerCtx.GetRuntimeContext(), extended)
		if err != nil {
			return prepareErroneousResultSet(keys, columnOrder, err)
		}
		columnOrder = openapistackql.GetResourcesHeader(extended)
		var filter func(openapistackql.ITable) (openapistackql.ITable, error)
		if err != nil {
			logging.GetLogger().Infoln(fmt.Sprintf("table and therefore filter not found for AST, shall procede nil filter"))
		} else {
			filter = tableFilter
		}
		resources, err = filterResources(resources, filter)
		if err != nil {
			return prepareErroneousResultSet(keys, columnOrder, err)
		}
		keys = make(map[string]map[string]interface{})
		for k, v := range resources {
			keys[k] = v.ToMap(extended)
		}
	case "SERVICES":
		logging.GetLogger().Infoln(fmt.Sprintf("Show For node.Type = '%s': Displaying services for provider = '%s'", node.Type, prov.GetProviderString()))
		var services map[string]*openapistackql.ProviderService
		services, err = prov.GetProviderServicesRedacted(handlerCtx.GetRuntimeContext(), extended)
		if err != nil {
			return prepareErroneousResultSet(keys, columnOrder, err)
		}
		columnOrder = openapistackql.GetServicesHeader(extended)
		services, err = filterServices(services, tableFilter, handlerCtx.GetRuntimeContext().UseNonPreferredAPIs)
		if err != nil {
			return prepareErroneousResultSet(keys, columnOrder, err)
		}
		keys = convertProviderServicesToMap(services, extended)
	}
	return util.PrepareResultSet(internaldto.NewPrepareResultSetDTO(nil, keys, columnOrder, nil, err, nil))
}

func filterResources(resources map[string]*openapistackql.Resource, tableFilter func(openapistackql.ITable) (openapistackql.ITable, error)) (map[string]*openapistackql.Resource, error) {
	var err error
	if tableFilter != nil {
		filteredResources := make(map[string]*openapistackql.Resource)
		for k, rsc := range resources {
			filteredResource, filterErr := tableFilter(rsc)
			if filterErr == nil && filteredResource != nil {
				filteredResources[k] = filteredResource.(*openapistackql.Resource)
			}
			if filterErr != nil {
				err = filterErr
			}
		}
		resources = filteredResources
	}
	return resources, err
}

func getProviderServiceMap(item openapistackql.ProviderService, extended bool) map[string]interface{} {
	retVal := map[string]interface{}{
		"id":    item.ID,
		"name":  item.Name,
		"title": item.Title,
	}
	if extended {
		retVal["description"] = item.Description
		retVal["version"] = item.Version
	}
	return retVal
}

func convertProviderServicesToMap(services map[string]*openapistackql.ProviderService, extended bool) map[string]map[string]interface{} {
	retVal := make(map[string]map[string]interface{})
	for k, v := range services {
		retVal[k] = getProviderServiceMap(*v, extended)
	}
	return retVal
}

func filterServices(services map[string]*openapistackql.ProviderService, tableFilter func(openapistackql.ITable) (openapistackql.ITable, error), useNonPreferredAPIs bool) (map[string]*openapistackql.ProviderService, error) {
	var err error
	if tableFilter != nil {
		filteredServices := make(map[string]*openapistackql.ProviderService)
		for k, svc := range services {
			if useNonPreferredAPIs || svc.Preferred {
				filteredService, filterErr := tableFilter(svc)
				if filterErr == nil && filteredService != nil {
					filteredServices[k] = (filteredService.(*openapistackql.ProviderService))
				}
				if filterErr != nil {
					err = filterErr
				}
			}
		}
		services = filteredServices
	}
	return services, err
}

func filterMethods(methods openapistackql.Methods, tableFilter func(openapistackql.ITable) (openapistackql.ITable, error)) (openapistackql.Methods, error) {
	var err error
	if tableFilter != nil {
		filteredMethods := make(openapistackql.Methods)
		for k, m := range methods {
			filteredMethod, filterErr := tableFilter(&m)
			if filterErr == nil && filteredMethod != nil {
				filteredMethods[k] = m
			}
			if filterErr != nil {
				err = filterErr
			}
		}
		methods = filteredMethods
	}
	return methods, err
}

func prepareErroneousResultSet(rowMap map[string]map[string]interface{}, columnOrder []string, err error) internaldto.ExecutorOutput {
	return util.PrepareResultSet(
		internaldto.NewPrepareResultSetDTO(
			nil,
			rowMap,
			columnOrder,
			nil,
			err,
			nil,
		),
	)
}

func NewDescribeTableInstructionExecutor(handlerCtx handler.HandlerContext, tbl tablemetadata.ExtendedTableMetadata, extended bool, full bool) internaldto.ExecutorOutput {
	schema, err := tbl.GetSelectableObjectSchema()
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	columnOrder := openapistackql.GetDescribeHeader(extended)
	descriptionMap := schema.ToDescriptionMap(extended)
	keys := make(map[string]map[string]interface{})
	for k, v := range descriptionMap {
		switch val := v.(type) {
		case map[string]interface{}:
			keys[k] = val
		}
	}
	return util.PrepareResultSet(internaldto.NewPrepareResultSetDTO(nil, keys, columnOrder, util.DescribeRowSort, err, nil))
}

func NewDescribeViewInstructionExecutor(handlerCtx handler.HandlerContext, tbl tablemetadata.ExtendedTableMetadata, nonControlColumns []internaldto.ColumnMetadata, extended bool, full bool) internaldto.ExecutorOutput {
	columnOrder := openapistackql.GetDescribeHeader(extended)
	descriptionMap := columnsToFlatDescriptionMap(nonControlColumns, extended)
	keys := make(map[string]map[string]interface{})
	for k, v := range descriptionMap {
		switch val := v.(type) {
		case map[string]interface{}:
			keys[k] = val
		}
	}
	return util.PrepareResultSet(internaldto.NewPrepareResultSetDTO(nil, keys, columnOrder, util.DescribeRowSort, nil, nil))
}

func columnsToFlatDescriptionMap(colz []internaldto.ColumnMetadata, extended bool) map[string]interface{} {
	retVal := make(map[string]interface{})
	for _, col := range colz {
		colName := col.GetIdentifier()
		colMap := make(map[string]interface{})
		colMap["name"] = colName
		colMap["type"] = col.GetType()
		if extended {
			colMap["description"] = ""
		}
		retVal[colName] = colMap
	}
	return retVal
}

func NewLocalSelectExecutor(handlerCtx handler.HandlerContext, node *sqlparser.Select, rowSort func(map[string]map[string]interface{}) []string, colz []map[string]interface{}) (primitive.IPrimitive, error) {
	return primitive.NewLocalPrimitive(
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			var columnOrder []string
			keys := make(map[string]map[string]interface{})
			row := make(map[string]interface{})
			for idx := range colz {
				col := colz[idx]
				if col != nil {
					var alias string
					var val interface{}
					for k, v := range col {
						alias = k
						val = v
						break
					}
					if alias == "" {
						alias = "val_" + strconv.Itoa(idx)
					}
					row[alias] = val
					columnOrder = append(columnOrder, alias)
				}
			}
			keys["0"] = row
			return util.PrepareResultSet(internaldto.NewPrepareResultSetDTO(nil, keys, columnOrder, rowSort, nil, nil))
		}), nil
}
