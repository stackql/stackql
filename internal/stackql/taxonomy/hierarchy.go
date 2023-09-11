package taxonomy

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/astformat"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"

	"github.com/stackql/go-openapistackql/openapistackql"

	"strings"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

func GetHeirarchyIDsFromParserNode(
	handlerCtx handler.HandlerContext,
	node sqlparser.SQLNode,
) (internaldto.HeirarchyIdentifiers, error) {
	return getHids(handlerCtx, node)
}

//nolint:funlen,gocognit // lots of moving parts
func getHids(handlerCtx handler.HandlerContext, node sqlparser.SQLNode) (internaldto.HeirarchyIdentifiers, error) {
	var hIds internaldto.HeirarchyIdentifiers
	switch n := node.(type) {
	case *sqlparser.Exec:
		hIds = internaldto.ResolveMethodTerminalHeirarchyIdentifiers(n.MethodName)
	case *sqlparser.ExecSubquery:
		hIds = internaldto.ResolveMethodTerminalHeirarchyIdentifiers(n.Exec.MethodName)
	case *sqlparser.Select:
		currentSvcRsc, err := parserutil.TableFromSelectNode(n)
		if err != nil {
			return nil, err
		}
		hIds = internaldto.ResolveResourceTerminalHeirarchyIdentifiers(currentSvcRsc)
	case sqlparser.TableName:
		hIds = internaldto.ResolveResourceTerminalHeirarchyIdentifiers(n)
	case *sqlparser.AliasedTableExpr:
		switch t := n.Expr.(type) { //nolint:gocritic // this is expressive enough
		case *sqlparser.Subquery:
			sq := internaldto.NewSubqueryDTO(n, t)
			return internaldto.ObtainSubqueryHeirarchyIdentifiers(sq), nil
		}
		return getHids(handlerCtx, n.Expr)
	case *sqlparser.DescribeTable:
		return getHids(handlerCtx, n.Table)
	case *sqlparser.Show:
		switch strings.ToUpper(n.Type) {
		case "INSERT":
			hIds = internaldto.ResolveResourceTerminalHeirarchyIdentifiers(n.OnTable)
		case "METHODS":
			hIds = internaldto.ResolveResourceTerminalHeirarchyIdentifiers(n.OnTable)
		default:
			return nil, fmt.Errorf("cannot resolve taxonomy for SHOW statement of type = '%s'", strings.ToUpper(n.Type))
		}
	case *sqlparser.Insert:
		hIds = internaldto.ResolveResourceTerminalHeirarchyIdentifiers(n.Table)
	case *sqlparser.Update:
		currentSvcRsc, err := parserutil.ExtractSingleTableFromTableExprs(n.TableExprs)
		if err != nil {
			return nil, err
		}
		hIds = internaldto.ResolveResourceTerminalHeirarchyIdentifiers(*currentSvcRsc)
	case *sqlparser.Delete:
		currentSvcRsc, err := parserutil.ExtractSingleTableFromTableExprs(n.TableExprs)
		if err != nil {
			return nil, err
		}
		hIds = internaldto.ResolveResourceTerminalHeirarchyIdentifiers(*currentSvcRsc)
	// case *sqlparser.Subquery:
	// suq := internaldto
	// hIds = internaldto.ObtainSubqueryHeirarchyIdentifiers()
	default:
		return nil, fmt.Errorf("cannot resolve taxonomy")
	}
	viewDTO, isView := handlerCtx.GetSQLSystem().GetViewByName(hIds.GetTableName())
	if isView {
		hIds = hIds.WithView(viewDTO)
	}
	materializedViewDTO, isMaterializedView := handlerCtx.GetSQLSystem().GetMaterializedViewByName(hIds.GetTableName())
	if isMaterializedView {
		hIds = hIds.WithView(materializedViewDTO)
	}
	// TODO: pass in current counters
	physicalTableDTO, isPhysicalTable := handlerCtx.GetSQLSystem().GetTableByName(hIds.GetTableName(), nil)
	if isPhysicalTable {
		hIds = hIds.WithView(physicalTableDTO)
	}
	isInternallyRoutable := handlerCtx.GetPGInternalRouter().ExprIsRoutable(node)
	if isInternallyRoutable {
		hIds.SetContainsNativeDBMSTable(true)
		return hIds, nil
	}
	if !(isView || isMaterializedView || isPhysicalTable) && hIds.GetProviderStr() == "" {
		if handlerCtx.GetCurrentProvider() == "" {
			return nil, fmt.Errorf("could not locate table '%s'", hIds.GetTableName())
		}
		hIds.WithProviderStr(handlerCtx.GetCurrentProvider())
	}
	return hIds, nil
}

func GetAliasFromStatement(node sqlparser.SQLNode) string {
	switch n := node.(type) {
	case *sqlparser.AliasedTableExpr:
		return n.As.GetRawVal()
	default:
		return ""
	}
}

func GetTableNameFromStatement(node sqlparser.SQLNode, formatter sqlparser.NodeFormatter) string {
	switch n := node.(type) {
	case *sqlparser.AliasedTableExpr:
		switch et := n.Expr.(type) {
		case sqlparser.TableName:
			return et.GetRawVal()
		default:
			return astformat.String(n.Expr, formatter)
		}
	case *sqlparser.Exec:
		return n.MethodName.GetRawVal()
	default:
		return astformat.String(n, formatter)
	}
}

// Hierarchy inference function.
// Returns:
//   - Hierarchy
//   - Supplied parameters that are **not** consumed in Hierarchy inference
//   - Error if applicable.
//
//nolint:funlen,gocognit,gocyclo,cyclop // lots of moving parts
func GetHeirarchyFromStatement(
	handlerCtx handler.HandlerContext,
	node sqlparser.SQLNode,
	parameters parserutil.ColumnKeyedDatastore,
) (tablemetadata.HeirarchyObjects, error) {
	var hIds internaldto.HeirarchyIdentifiers
	getFirstAvailableMethod := false
	hIds, err := getHids(handlerCtx, node)
	if err != nil {
		return nil, err
	}
	methodRequired := true
	var methodAction string
	switch n := node.(type) {
	case *sqlparser.Exec, *sqlparser.ExecSubquery:
	case *sqlparser.Select:
		methodAction = "select"
	case *sqlparser.DescribeTable:
	case sqlparser.TableName:
	case *sqlparser.AliasedTableExpr:
		switch n.Expr.(type) { //nolint:gocritic // this is expressive enough
		case *sqlparser.Subquery:
			retVal := tablemetadata.NewHeirarchyObjects(hIds)
			return retVal, nil
		}
		return GetHeirarchyFromStatement(handlerCtx, n.Expr, parameters)
	case *sqlparser.Show:
		switch strings.ToUpper(n.Type) {
		case "INSERT":
			methodAction = "insert"
			getFirstAvailableMethod = true
		case "METHODS":
			methodRequired = false
		default:
			return nil, fmt.Errorf("cannot resolve taxonomy for SHOW statement of type = '%s'", strings.ToUpper(n.Type))
		}
	case *sqlparser.Insert:
		methodAction = "insert"
	case *sqlparser.Delete:
		methodAction = "delete"
	case *sqlparser.Update:
		methodAction = "update"
	default:
		return nil, fmt.Errorf("cannot resolve taxonomy")
	}
	retVal := tablemetadata.NewHeirarchyObjects(hIds)
	sqlDataSource, isSQLDataSource := handlerCtx.GetSQLDataSource(hIds.GetProviderStr())
	if isSQLDataSource {
		retVal.SetSQLDataSource(sqlDataSource)
		return retVal, nil
	}
	// TODO: accomodate complex PG internal queries
	isPgInternal := hIds.IsPgInternalObject()
	if isPgInternal {
		return retVal, nil
	}
	prov, err := handlerCtx.GetProvider(hIds.GetProviderStr())
	retVal.SetProvider(prov)
	viewDTO, isView := retVal.GetView()
	if isView {
		logging.GetLogger().Debugf("viewDTO = %v\n", viewDTO)
		return retVal, nil
	}
	if err != nil {
		return nil, err
	}
	svcHdl, err := prov.GetServiceShard(hIds.GetServiceStr(), hIds.GetResourceStr(), handlerCtx.GetRuntimeContext())
	if err != nil {
		return nil, err
	}
	retVal.SetServiceHdl(svcHdl)
	rsc, err := prov.GetResource(hIds.GetServiceStr(), hIds.GetResourceStr(), handlerCtx.GetRuntimeContext())
	if err != nil {
		return nil, err
	}
	retVal.SetResource(rsc)
	//nolint:nestif // not overly complex
	if viewBodyDDL, ok := rsc.GetViewBodyDDLForSQLDialect(
		handlerCtx.GetSQLSystem().GetName()); ok {
		viewName := hIds.GetStackQLTableName()
		// TODO: mutex required or some other strategy
		viewDTO, viewExists := handlerCtx.GetSQLSystem().GetViewByName(viewName) //nolint:govet // acceptable shadow
		if !viewExists {
			// TODO: resolve any possible data race
			err = handlerCtx.GetSQLSystem().CreateView(viewName, viewBodyDDL, true)
			if err != nil {
				return nil, err
			}
			viewDTO, isView := handlerCtx.GetSQLSystem().GetViewByName(hIds.GetTableName()) //nolint:govet // acceptable shadow
			if isView {
				hIds = hIds.WithView(viewDTO) //nolint:staticcheck,wastedassign // TODO: fix this
			}
			return retVal, nil
		}
		hIds = hIds.WithView(viewDTO) //nolint:staticcheck,wastedassign // TODO: fix this
		return retVal, nil
	}
	var method openapistackql.OperationStore
	switch node.(type) {
	case *sqlparser.Exec, *sqlparser.ExecSubquery:
		method, err = rsc.FindMethod(hIds.GetMethodStr())
		if err != nil {
			return nil, err
		}
		retVal.SetMethod(method)
		return retVal, nil
	}
	if methodRequired { //nolint:nestif // not overly complex
		switch node.(type) { //nolint:gocritic // this is expressive enough
		case *sqlparser.DescribeTable:
			m, mStr, mErr := prov.InferDescribeMethod(rsc)
			if mErr != nil {
				return nil, mErr
			}
			retVal.SetMethod(m)
			retVal.SetMethodStr(mStr)
			return retVal, nil
		}
		if methodAction == "" {
			methodAction = "select"
		}
		var meth openapistackql.OperationStore
		var methStr string
		if getFirstAvailableMethod {
			meth, methStr, err = prov.GetFirstMethodForAction( //nolint:staticcheck,ineffassign,wastedassign // TODO: fix this
				retVal.GetHeirarchyIds().GetServiceStr(),
				retVal.GetHeirarchyIds().GetResourceStr(),
				methodAction,
				handlerCtx.GetRuntimeContext())
		} else {
			meth, methStr, err = prov.GetMethodForAction(
				retVal.GetHeirarchyIds().GetServiceStr(),
				retVal.GetHeirarchyIds().GetResourceStr(),
				methodAction,
				parameters,
				handlerCtx.GetRuntimeContext())
			if err != nil {
				return nil, fmt.Errorf(
					"cannot find matching operation, possible causes include missing required parameters or an unsupported method for the resource, to find required parameters for supported methods run SHOW METHODS IN %s: %w", //nolint:lll // long string
					retVal.GetHeirarchyIds().GetTableName(), err)
			}
		}
		for _, srv := range svcHdl.GetServers() {
			for k := range srv.Variables {
				logging.GetLogger().Debugf("server parameter = '%s'\n", k)
			}
		}
		method = meth
		retVal.SetMethodStr(methStr)
	}
	if methodRequired {
		retVal.SetMethod(method)
	}
	return retVal, nil
}
