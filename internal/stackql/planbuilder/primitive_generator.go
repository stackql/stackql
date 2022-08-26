package planbuilder

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/stackql/stackql/internal/stackql/asyncmonitor"
	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/httpbuild"
	"github.com/stackql/stackql/internal/stackql/httpmiddleware"
	"github.com/stackql/stackql/internal/stackql/iqlutil"
	"github.com/stackql/stackql/internal/stackql/metadatavisitors"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivecomposer"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/relational"
	"github.com/stackql/stackql/internal/stackql/symtab"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/stackql/pkg/prettyprint"
	"github.com/stackql/stackql/pkg/sqltypeutil"

	"github.com/stackql/go-openapistackql/openapistackql"

	log "github.com/sirupsen/logrus"

	"vitess.io/vitess/go/sqltypes"
	"vitess.io/vitess/go/vt/sqlparser"
)

type primitiveGenerator struct {
	Parent            *primitiveGenerator
	Children          []*primitiveGenerator
	PrimitiveComposer primitivecomposer.PrimitiveComposer
}

func newRootPrimitiveGenerator(ast sqlparser.SQLNode, handlerCtx *handler.HandlerContext, graph *primitivegraph.PrimitiveGraph) *primitiveGenerator {
	tblMap := make(taxonomy.TblMap)
	symTab := symtab.NewHashMapTreeSymTab()
	return &primitiveGenerator{
		PrimitiveComposer: primitivecomposer.NewPrimitiveComposer(nil, ast, handlerCtx.DrmConfig, handlerCtx.TxnCounterMgr, graph, tblMap, symTab, handlerCtx.SQLEngine),
	}
}

func (pb *primitiveGenerator) addChildPrimitiveGenerator(ast sqlparser.SQLNode, leaf symtab.SymTab) *primitiveGenerator {
	tables := pb.PrimitiveComposer.GetTables()
	switch node := ast.(type) {
	case sqlparser.Statement:
		log.Infoln(fmt.Sprintf("creating new table map for node = %v", node))
		tables = make(taxonomy.TblMap)
	}
	retVal := &primitiveGenerator{
		Parent: pb,
		PrimitiveComposer: primitivecomposer.NewPrimitiveComposer(
			pb.PrimitiveComposer,
			ast,
			pb.PrimitiveComposer.GetDRMConfig(),
			pb.PrimitiveComposer.GetTxnCounterManager(),
			pb.PrimitiveComposer.GetGraph(),
			tables,
			leaf,
			pb.PrimitiveComposer.GetSQLEngine(),
		),
	}
	pb.Children = append(pb.Children, retVal)
	pb.PrimitiveComposer.AddChild(retVal.PrimitiveComposer)
	return retVal
}

func (pb *primitiveGenerator) comparisonExprToFilterFunc(table openapistackql.ITable, parentNode *sqlparser.Show, expr *sqlparser.ComparisonExpr) (func(openapistackql.ITable) (openapistackql.ITable, error), error) {
	qualifiedName, ok := expr.Left.(*sqlparser.ColName)
	if !ok {
		return nil, fmt.Errorf("unexpected: %v", sqlparser.String(expr))
	}
	if !qualifiedName.Qualifier.IsEmpty() {
		return nil, fmt.Errorf("unsupported qualifier for column: %v", sqlparser.String(qualifiedName))
	}
	colName := qualifiedName.Name.GetRawVal()
	tableContainsKey := table.KeyExists(colName)
	if !tableContainsKey {
		return nil, fmt.Errorf("col name = '%s' not found in table name = '%s'", colName, table.GetName())
	}
	_, lhsValErr := table.GetKeyAsSqlVal(colName)
	if lhsValErr != nil {
		return nil, lhsValErr
	}
	var resolved sqltypes.Value
	var rhsStr string
	switch right := expr.Right.(type) {
	case *sqlparser.SQLVal:
		if right.Type != sqlparser.IntVal && right.Type != sqlparser.StrVal {
			return nil, fmt.Errorf("unexpected: %v", sqlparser.String(expr))
		}
		pv, err := sqlparser.NewPlanValue(right)
		if err != nil {
			return nil, err
		}
		rhsStr = string(right.Val)
		resolved, err = pv.ResolveValue(nil)
		if err != nil {
			return nil, err
		}
	case sqlparser.BoolVal:
		var resErr error
		resolved, resErr = sqltypeutil.InterfaceToSQLType(right == true)
		if resErr != nil {
			return nil, resErr
		}
	default:
		return nil, fmt.Errorf("unexpected: %v", sqlparser.String(right))
	}
	var retVal func(openapistackql.ITable) (openapistackql.ITable, error)
	if expr.Operator == sqlparser.LikeStr || expr.Operator == sqlparser.NotLikeStr {
		likeRegexp, err := regexp.Compile(iqlutil.TranslateLikeToRegexPattern(rhsStr))
		if err != nil {
			return nil, err
		}
		retVal = relational.ConstructLikePredicateFilter(colName, likeRegexp, expr.Operator == sqlparser.NotLikeStr)
		pb.PrimitiveComposer.SetColVisited(colName, true)
		return retVal, nil
	}
	operatorPredicate, preErr := relational.GetOperatorPredicate(expr.Operator)

	if preErr != nil {
		return nil, preErr
	}

	pb.PrimitiveComposer.SetColVisited(colName, true)
	return relational.ConstructTablePredicateFilter(colName, resolved, operatorPredicate), nil
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

func (pb *primitiveGenerator) inferProviderForShow(node *sqlparser.Show, handlerCtx *handler.HandlerContext) error {
	nodeTypeUpperCase := strings.ToUpper(node.Type)
	switch nodeTypeUpperCase {
	case "AUTH":
		prov, err := handlerCtx.GetProvider(node.OnTable.Name.GetRawVal())
		if err != nil {
			return err
		}
		pb.PrimitiveComposer.SetProvider(prov)
	case "INSERT":
		prov, err := handlerCtx.GetProvider(node.OnTable.QualifierSecond.GetRawVal())
		if err != nil {
			return err
		}
		pb.PrimitiveComposer.SetProvider(prov)

	case "METHODS":
		prov, err := handlerCtx.GetProvider(node.OnTable.QualifierSecond.GetRawVal())
		if err != nil {
			return err
		}
		pb.PrimitiveComposer.SetProvider(prov)
	case "PROVIDERS":
		// no provider, might create some dummy object dunno
	case "RESOURCES":
		prov, err := handlerCtx.GetProvider(node.OnTable.Qualifier.GetRawVal())
		if err != nil {
			return err
		}
		pb.PrimitiveComposer.SetProvider(prov)
	case "SERVICES":
		prov, err := handlerCtx.GetProvider(node.OnTable.Name.GetRawVal())
		if err != nil {
			return err
		}
		pb.PrimitiveComposer.SetProvider(prov)
	default:
		return fmt.Errorf("unsuported node type: '%s'", node.Type)
	}
	return nil
}

func (pb *primitiveGenerator) showInstructionExecutor(node *sqlparser.Show, handlerCtx *handler.HandlerContext) dto.ExecutorOutput {
	extended := strings.TrimSpace(strings.ToUpper(node.Extended)) == "EXTENDED"
	nodeTypeUpperCase := strings.ToUpper(node.Type)
	var keys map[string]map[string]interface{}
	var columnOrder []string
	var err error
	var filter func(interface{}) (openapistackql.ITable, error)
	log.Infoln(fmt.Sprintf("filter type = %T", filter))
	switch nodeTypeUpperCase {
	case "AUTH":
		log.Infoln(fmt.Sprintf("Show For node.Type = '%s'", node.Type))
		if err == nil {
			authCtx, err := handlerCtx.GetAuthContext(pb.PrimitiveComposer.GetProvider().GetProviderString())
			if err == nil {
				var authMeta *openapistackql.AuthMetadata
				authMeta, err = pb.PrimitiveComposer.GetProvider().ShowAuth(authCtx)
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
			handlerCtx.RuntimeContext.OutputFormat == constants.PrettyTextStr,
			constants.DefaultPrettyPrintIndent,
			constants.DefaultPrettyPrintBaseIndent,
			"'",
		)
		tbl, err := pb.PrimitiveComposer.GetTable(node)
		if err != nil {
			return util.GenerateSimpleErroneousOutput(err)
		}
		meth, err := tbl.GetMethod()
		if err != nil {
			tblName, tblErr := tbl.GetTableName()
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
		requiredOnly := pb.PrimitiveComposer.GetCommentDirectives() != nil && pb.PrimitiveComposer.GetCommentDirectives().IsSet("REQUIRED")
		insertStmt, err := metadatavisitors.ToInsertStatement(node.Columns, meth, svc, extended, pp, requiredOnly)
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
		rsc, err = pb.PrimitiveComposer.GetProvider().GetResource(node.OnTable.Qualifier.GetRawVal(), node.OnTable.Name.GetRawVal(), handlerCtx.RuntimeContext)
		methods := rsc.GetMethodsMatched()
		tbl, err := pb.PrimitiveComposer.GetTable(node.OnTable)
		var filter func(openapistackql.ITable) (openapistackql.ITable, error)
		if err != nil {
			log.Infoln(fmt.Sprintf("table and therefore filter not found for AST, shall procede nil filter"))
		} else {
			filter = tbl.TableFilter
		}
		methods, err = filterMethods(methods, filter)
		if err != nil {
			return util.PrepareResultSet(dto.NewPrepareResultSetDTO(nil, keys, columnOrder, nil, err, nil))
		}
		mOrd, err := methods.OrderMethods()
		if err != nil {
			return util.PrepareResultSet(dto.NewPrepareResultSetDTO(nil, keys, columnOrder, nil, err, nil))
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
		keys = handlerCtx.GetSupportedProviders(extended)
		rv := util.PrepareResultSet(dto.NewPrepareResultSetDTO(nil, keys, columnOrder, nil, err, nil))
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
		resources, err = pb.PrimitiveComposer.GetProvider().GetResourcesRedacted(svcName, handlerCtx.RuntimeContext, extended)
		if err != nil {
			return prepareErroneousResultSet(keys, columnOrder, err)
		}
		columnOrder = openapistackql.GetResourcesHeader(extended)
		var filter func(openapistackql.ITable) (openapistackql.ITable, error)
		if err != nil {
			log.Infoln(fmt.Sprintf("table and therefore filter not found for AST, shall procede nil filter"))
		} else {
			filter = pb.PrimitiveComposer.GetTableFilter()
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
		log.Infoln(fmt.Sprintf("Show For node.Type = '%s': Displaying services for provider = '%s'", node.Type, pb.PrimitiveComposer.GetProvider().GetProviderString()))
		var services map[string]*openapistackql.ProviderService
		services, err = pb.PrimitiveComposer.GetProvider().GetProviderServicesRedacted(handlerCtx.RuntimeContext, extended)
		if err != nil {
			return prepareErroneousResultSet(keys, columnOrder, err)
		}
		columnOrder = openapistackql.GetServicesHeader(extended)
		services, err = filterServices(services, pb.PrimitiveComposer.GetTableFilter(), handlerCtx.RuntimeContext.UseNonPreferredAPIs)
		if err != nil {
			return prepareErroneousResultSet(keys, columnOrder, err)
		}
		keys = convertProviderServicesToMap(services, extended)
	}
	return util.PrepareResultSet(dto.NewPrepareResultSetDTO(nil, keys, columnOrder, nil, err, nil))
}

func prepareErroneousResultSet(rowMap map[string]map[string]interface{}, columnOrder []string, err error) dto.ExecutorOutput {
	return util.PrepareResultSet(
		dto.NewPrepareResultSetDTO(
			nil,
			rowMap,
			columnOrder,
			nil,
			err,
			nil,
		),
	)
}

func (pb *primitiveGenerator) describeInstructionExecutor(handlerCtx *handler.HandlerContext, tbl *taxonomy.ExtendedTableMetadata, extended bool, full bool) dto.ExecutorOutput {
	schema, err := tbl.GetSelectableObjectSchema()
	if err != nil {
		return dto.NewErroneousExecutorOutput(err)
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
	return util.PrepareResultSet(dto.NewPrepareResultSetDTO(nil, keys, columnOrder, util.DescribeRowSort, err, nil))
}

func (pb *primitiveGenerator) insertExecutor(handlerCtx *handler.HandlerContext, node sqlparser.SQLNode, rowSort func(map[string]map[string]interface{}) []string) (primitive.IPrimitive, error) {
	switch node := node.(type) {
	case *sqlparser.Insert, *sqlparser.Update:
	default:
		return nil, fmt.Errorf("mutation executor: cannnot accomodate node of type '%T'", node)
	}
	tbl, err := pb.PrimitiveComposer.GetTable(node)
	if err != nil {
		return nil, err
	}
	prov, err := tbl.GetProvider()
	if err != nil {
		return nil, err
	}
	svc, err := tbl.GetService()
	if err != nil {
		return nil, err
	}
	m, err := tbl.GetMethod()
	if err != nil {
		return nil, err
	}
	_, _, err = tbl.GetResponseSchemaAndMediaType()
	if err != nil {
		return nil, err
	}
	insertPrimitive := primitive.NewHTTPRestPrimitive(
		prov,
		nil,
		nil,
		nil,
	)
	ex := func(pc primitive.IPrimitiveCtx) dto.ExecutorOutput {
		input, inputExists := insertPrimitive.GetInputFromAlias("")
		if !inputExists {
			return dto.NewErroneousExecutorOutput(fmt.Errorf("input does not exist"))
		}
		inputStream, err := input.ResultToMap()
		if err != nil {
			return dto.NewErroneousExecutorOutput(err)
		}
		rr, err := inputStream.Read()
		if err != nil {
			return dto.NewErroneousExecutorOutput(err)
		}
		inputMap, err := rr.GetMap()
		if err != nil {
			return dto.NewErroneousExecutorOutput(err)
		}
		httpArmoury, err := httpbuild.BuildHTTPRequestCtx(handlerCtx, node, prov, m, svc, inputMap, nil)
		if err != nil {
			return dto.NewErroneousExecutorOutput(err)
		}
		var target map[string]interface{}

		var zeroArityExecutors []func() dto.ExecutorOutput
		for _, r := range httpArmoury.GetRequestParams() {
			req := r
			zeroArityEx := func() dto.ExecutorOutput {
				// log.Infoln(fmt.Sprintf("req.BodyBytes = %s", string(req.BodyBytes)))
				// req.Context.SetBody(bytes.NewReader(req.BodyBytes))
				// log.Infoln(fmt.Sprintf("req.Context = %v", req.Context))
				response, apiErr := httpmiddleware.HttpApiCallFromRequest(*handlerCtx, prov, m, req.Request)
				if apiErr != nil {
					return dto.NewErroneousExecutorOutput(apiErr)
				}

				target, err = m.DeprecatedProcessResponse(response)
				handlerCtx.LogHTTPResponseMap(target)
				if err != nil {
					return dto.NewErroneousExecutorOutput(err)
				}
				pb.composeAsyncMonitor(handlerCtx, insertPrimitive, tbl)
				if err != nil {
					return dto.NewErroneousExecutorOutput(err)
				}
				log.Infoln(fmt.Sprintf("target = %v", target))
				items, ok := target[tbl.LookupSelectItemsKey()]
				keys := make(map[string]map[string]interface{})
				if ok {
					iArr, ok := items.([]interface{})
					if ok && len(iArr) > 0 {
						for i := range iArr {
							item, ok := iArr[i].(map[string]interface{})
							if ok {
								keys[strconv.Itoa(i)] = item
							}
						}
					}
				}
				msgs := dto.BackendMessages{}
				if err == nil {
					msgs.WorkingMessages = generateSuccessMessagesFromHeirarchy(tbl)
				} else {
					msgs.WorkingMessages = []string{err.Error()}
				}
				return dto.NewExecutorOutput(nil, target, nil, &msgs, err)
			}
			zeroArityExecutors = append(zeroArityExecutors, zeroArityEx)
		}
		resultSet := dto.NewErroneousExecutorOutput(fmt.Errorf("no executions detected"))
		msgs := dto.BackendMessages{}
		if !pb.PrimitiveComposer.IsAwait() {
			for _, ei := range zeroArityExecutors {
				execInstance := ei
				resultSet = execInstance()
				if resultSet.Msg != nil && resultSet.Msg.WorkingMessages != nil && len(resultSet.Msg.WorkingMessages) > 0 {
					for _, m := range resultSet.Msg.WorkingMessages {
						msgs.WorkingMessages = append(msgs.WorkingMessages, m)
					}
				}
				if resultSet.Err != nil {
					resultSet.Msg = &msgs
					return resultSet
				}
			}
			resultSet.Msg = &msgs
			return resultSet
		}
		for _, eI := range zeroArityExecutors {
			execInstance := eI
			dependentInsertPrimitive := primitive.NewHTTPRestPrimitive(
				prov,
				nil,
				nil,
				nil,
			)
			err = dependentInsertPrimitive.SetExecutor(func(pc primitive.IPrimitiveCtx) dto.ExecutorOutput {
				return execInstance()
			})
			if err != nil {
				return dto.NewErroneousExecutorOutput(err)
			}
			execPrim, err := pb.composeAsyncMonitor(handlerCtx, dependentInsertPrimitive, tbl)
			if err != nil {
				return dto.NewErroneousExecutorOutput(err)
			}
			resultSet = execPrim.Execute(pc)
			if resultSet.Err != nil {
				return resultSet
			}
		}
		return resultSet
	}
	err = insertPrimitive.SetExecutor(ex)
	if err != nil {
		return nil, err
	}
	return insertPrimitive, nil
}

func (pb *primitiveGenerator) localSelectExecutor(handlerCtx *handler.HandlerContext, node *sqlparser.Select, rowSort func(map[string]map[string]interface{}) []string) (primitive.IPrimitive, error) {
	return primitive.NewLocalPrimitive(
		func(pc primitive.IPrimitiveCtx) dto.ExecutorOutput {
			var columnOrder []string
			keys := make(map[string]map[string]interface{})
			row := make(map[string]interface{})
			for idx := range pb.PrimitiveComposer.GetValOnlyColKeys() {
				col := pb.PrimitiveComposer.GetValOnlyCol(idx)
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
			return util.PrepareResultSet(dto.NewPrepareResultSetDTO(nil, keys, columnOrder, rowSort, nil, nil))
		}), nil
}

func (pb *primitiveGenerator) insertableValsExecutor(handlerCtx *handler.HandlerContext, vals map[int]map[int]interface{}) (primitive.IPrimitive, error) {
	return primitive.NewLocalPrimitive(
		func(pc primitive.IPrimitiveCtx) dto.ExecutorOutput {
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
			return util.PrepareResultSet(dto.NewPrepareResultSetPlusRawDTO(nil, keys, columnOrder, nil, nil, nil, vals))
		}), nil
}

func (pb *primitiveGenerator) updateableValsExecutor(handlerCtx *handler.HandlerContext, vals map[*sqlparser.ColName]interface{}) (primitive.IPrimitive, error) {
	return primitive.NewLocalPrimitive(
		func(pc primitive.IPrimitiveCtx) dto.ExecutorOutput {
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
			return util.PrepareResultSet(dto.NewPrepareResultSetPlusRawDTO(nil, keys, columnOrder, nil, nil, nil, rawRows))
		}), nil
}

func (pb *primitiveGenerator) deleteExecutor(handlerCtx *handler.HandlerContext, node *sqlparser.Delete) (primitive.IPrimitive, error) {
	tbl, err := pb.PrimitiveComposer.GetTable(node)
	if err != nil {
		return nil, err
	}
	prov, err := tbl.GetProvider()
	if err != nil {
		return nil, err
	}
	m, err := tbl.GetMethod()
	if err != nil {
		return nil, err
	}
	ex := func(pc primitive.IPrimitiveCtx) dto.ExecutorOutput {
		var target map[string]interface{}
		keys := make(map[string]map[string]interface{})
		httpArmoury, err := tbl.GetHttpArmoury()
		if err != nil {
			return util.PrepareResultSet(dto.NewPrepareResultSetDTO(nil, nil, nil, nil, err, nil))
		}
		for _, req := range httpArmoury.GetRequestParams() {
			response, apiErr := httpmiddleware.HttpApiCallFromRequest(*handlerCtx, prov, m, req.Request)
			if apiErr != nil {
				return util.PrepareResultSet(dto.NewPrepareResultSetDTO(nil, nil, nil, nil, apiErr, nil))
			}
			target, err = m.DeprecatedProcessResponse(response)
			handlerCtx.LogHTTPResponseMap(target)

			log.Infoln(fmt.Sprintf("deleteExecutor() target = %v", target))
			if err != nil {
				return util.PrepareResultSet(dto.NewPrepareResultSetDTO(
					nil,
					nil,
					nil,
					nil,
					err,
					nil,
				))
			}
			log.Infoln(fmt.Sprintf("target = %v", target))
			items, ok := target[prov.GetDefaultKeyForDeleteItems()]
			if ok {
				iArr, ok := items.([]interface{})
				if ok && len(iArr) > 0 {
					for i := range iArr {
						item, ok := iArr[i].(map[string]interface{})
						if ok {
							keys[strconv.Itoa(i)] = item
						}
					}
				}
			}
		}
		msgs := dto.BackendMessages{}
		if err == nil {
			msgs.WorkingMessages = generateSuccessMessagesFromHeirarchy(tbl)
		}
		return pb.generateResultIfNeededfunc(keys, target, &msgs, err)
	}
	deletePrimitive := primitive.NewHTTPRestPrimitive(
		prov,
		ex,
		nil,
		nil,
	)
	if !pb.PrimitiveComposer.IsAwait() {
		return deletePrimitive, nil
	}
	return pb.composeAsyncMonitor(handlerCtx, deletePrimitive, tbl)
}

func generateSuccessMessagesFromHeirarchy(meta *taxonomy.ExtendedTableMetadata) []string {
	successMsgs := []string{
		"The operation completed successfully",
	}
	m, methodErr := meta.GetMethod()
	prov, err := meta.GetProvider()
	if methodErr == nil && err == nil && m != nil && prov != nil && prov.GetProviderString() == "google" {
		if m.APIMethod == "select" || m.APIMethod == "get" || m.APIMethod == "list" || m.APIMethod == "aggregatedList" {
			successMsgs = []string{
				"The operation completed successfully, consider using a SELECT statement if you are performing an operation that returns data, see https://docs.stackql.io/language-spec/select for more information",
			}
		}
	}
	return successMsgs
}

func (pb *primitiveGenerator) isShowResults() bool {
	return pb.PrimitiveComposer.GetCommentDirectives() != nil && pb.PrimitiveComposer.GetCommentDirectives().IsSet("SHOWRESULTS")
}

func (pb *primitiveGenerator) generateResultIfNeededfunc(resultMap map[string]map[string]interface{}, body map[string]interface{}, msg *dto.BackendMessages, err error) dto.ExecutorOutput {
	if pb.isShowResults() {
		return util.PrepareResultSet(dto.NewPrepareResultSetDTO(nil, resultMap, nil, nil, nil, nil))
	}
	return dto.NewExecutorOutput(nil, body, nil, msg, err)
}

func (pb *primitiveGenerator) execExecutor(handlerCtx *handler.HandlerContext, node *sqlparser.Exec) (primitivegraph.PrimitiveNode, error) {
	if pb.isShowResults() && pb.PrimitiveComposer.GetBuilder() != nil {
		err := pb.PrimitiveComposer.GetBuilder().Build()
		if err != nil {
			return primitivegraph.PrimitiveNode{}, err
		}
		return pb.PrimitiveComposer.GetBuilder().GetRoot(), nil
	}
	var target map[string]interface{}
	tbl, err := pb.PrimitiveComposer.GetTable(node)
	if err != nil {
		return primitivegraph.PrimitiveNode{}, err
	}
	prov, err := tbl.GetProvider()
	if err != nil {
		return primitivegraph.PrimitiveNode{}, err
	}
	m, err := tbl.GetMethod()
	if err != nil {
		return primitivegraph.PrimitiveNode{}, err
	}
	ex := func(pc primitive.IPrimitiveCtx) dto.ExecutorOutput {
		var err error
		var columnOrder []string
		keys := make(map[string]map[string]interface{})
		httpArmoury, err := tbl.GetHttpArmoury()
		if err != nil {
			return dto.NewErroneousExecutorOutput(err)
		}
		for i, req := range httpArmoury.GetRequestParams() {
			response, apiErr := httpmiddleware.HttpApiCallFromRequest(*handlerCtx, prov, m, req.Request)
			if apiErr != nil {
				return util.PrepareResultSet(dto.NewPrepareResultSetDTO(nil, nil, nil, nil, apiErr, nil))
			}
			target, err = m.DeprecatedProcessResponse(response)
			handlerCtx.LogHTTPResponseMap(target)
			if err != nil {
				return util.PrepareResultSet(dto.NewPrepareResultSetDTO(
					nil,
					nil,
					nil,
					nil,
					err,
					nil,
				))
			}
			log.Infoln(fmt.Sprintf("target = %v", target))
			items, ok := target[tbl.LookupSelectItemsKey()]
			if ok {
				iArr, ok := items.([]interface{})
				if ok && len(iArr) > 0 {
					for i := range iArr {
						item, ok := iArr[i].(map[string]interface{})
						if ok {
							keys[strconv.Itoa(i)] = item
						}
					}
				}
			} else {
				keys[fmt.Sprintf("%d", i)] = target
			}
			// optional data return pattern to be included in grammar subsequently
			// return util.PrepareResultSet(dto.NewPrepareResultSetDTO(nil, keys, columnOrder, nil, err, nil))
			log.Debugln(fmt.Sprintf("keys = %v", keys))
			log.Debugln(fmt.Sprintf("columnOrder = %v", columnOrder))
		}
		msgs := dto.BackendMessages{}
		if err == nil {
			msgs.WorkingMessages = generateSuccessMessagesFromHeirarchy(tbl)
		}
		return pb.generateResultIfNeededfunc(keys, target, &msgs, err)
	}
	execPrimitive := primitive.NewHTTPRestPrimitive(
		prov,
		ex,
		nil,
		nil,
	)
	graph := pb.PrimitiveComposer.GetGraph()
	if !pb.PrimitiveComposer.IsAwait() {
		return graph.CreatePrimitiveNode(execPrimitive), nil
	}
	pr, err := pb.composeAsyncMonitor(handlerCtx, execPrimitive, tbl)
	if err != nil {
		return primitivegraph.PrimitiveNode{}, err
	}
	return graph.CreatePrimitiveNode(pr), nil
}

func (pb *primitiveGenerator) composeAsyncMonitor(handlerCtx *handler.HandlerContext, precursor primitive.IPrimitive, meta *taxonomy.ExtendedTableMetadata) (primitive.IPrimitive, error) {
	prov, err := meta.GetProvider()
	if err != nil {
		return nil, err
	}
	asm, err := asyncmonitor.NewAsyncMonitor(handlerCtx, prov)
	if err != nil {
		return nil, err
	}
	// might be pointless
	_, err = handlerCtx.GetAuthContext(prov.GetProviderString())
	if err != nil {
		return nil, err
	}
	//
	pl := dto.NewBasicPrimitiveContext(
		handlerCtx.GetAuthContext,
		handlerCtx.Outfile,
		handlerCtx.OutErrFile,
	)
	primitive, err := asm.GetMonitorPrimitive(meta.HeirarchyObjects, precursor, pl, pb.PrimitiveComposer.GetCommentDirectives())
	if err != nil {
		return nil, err
	}
	return primitive, err
}
