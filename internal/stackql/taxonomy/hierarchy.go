package taxonomy

import (
	"fmt"

	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/any-sdk/pkg/logging"
	"github.com/stackql/stackql/internal/stackql/astformat"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"github.com/stackql/stackql/pkg/name_mangle"

	"strings"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

func GetHeirarchyIDsFromParserNode(
	handlerCtx handler.HandlerContext,
	node sqlparser.SQLNode,
) (internaldto.HeirarchyIdentifiers, error) {
	return GetHIDs(
		handlerCtx, node, parserutil.NewParameterMap(), false)
}

//nolint:funlen,gocognit // lots of moving parts
func GetHIDs(
	handlerCtx handler.HandlerContext,
	node sqlparser.SQLNode,
	params parserutil.ColumnKeyedDatastore,
	viewPermissive bool) (internaldto.HeirarchyIdentifiers, error) {
	var hIDs internaldto.HeirarchyIdentifiers
	switch n := node.(type) {
	case *sqlparser.Exec:
		hIDs = internaldto.ResolveMethodTerminalHeirarchyIdentifiers(n.MethodName)
	case *sqlparser.ExecSubquery:
		hIDs = internaldto.ResolveMethodTerminalHeirarchyIdentifiers(n.Exec.MethodName)
	case *sqlparser.Select:
		currentSvcRsc, err := parserutil.TableFromSelectNode(n)
		if err != nil {
			return nil, err
		}
		hIDs = internaldto.ResolveResourceTerminalHeirarchyIdentifiers(currentSvcRsc)
	case sqlparser.TableName:
		hIDs = internaldto.ResolveResourceTerminalHeirarchyIdentifiers(n)
	case *sqlparser.AliasedTableExpr:
		switch t := n.Expr.(type) { //nolint:gocritic // this is expressive enough
		case *sqlparser.Subquery:
			sq := internaldto.NewSubqueryDTO(n, t)
			return internaldto.ObtainSubqueryHeirarchyIdentifiers(sq), nil
		}
		return GetHIDs(handlerCtx, n.Expr, params, viewPermissive)
	case *sqlparser.DescribeTable:
		return GetHIDs(handlerCtx, n.Table, params, viewPermissive)
	case *sqlparser.Show:
		switch strings.ToUpper(n.Type) {
		case "INSERT":
			hIDs = internaldto.ResolveResourceTerminalHeirarchyIdentifiers(n.OnTable)
		case "METHODS":
			hIDs = internaldto.ResolveResourceTerminalHeirarchyIdentifiers(n.OnTable)
		default:
			return nil, fmt.Errorf("cannot resolve taxonomy for SHOW statement of type = '%s'", strings.ToUpper(n.Type))
		}
	case *sqlparser.Insert:
		hIDs = internaldto.ResolveResourceTerminalHeirarchyIdentifiers(n.Table)
	case *sqlparser.Update:
		currentSvcRsc, err := parserutil.ExtractSingleTableFromTableExprs(n.TableExprs)
		if err != nil {
			return nil, err
		}
		hIDs = internaldto.ResolveResourceTerminalHeirarchyIdentifiers(*currentSvcRsc)
	case *sqlparser.Delete:
		currentSvcRsc, err := parserutil.ExtractSingleTableFromTableExprs(n.TableExprs)
		if err != nil {
			return nil, err
		}
		hIDs = internaldto.ResolveResourceTerminalHeirarchyIdentifiers(*currentSvcRsc)
	// case *sqlparser.Subquery:
	// suq := internaldto
	// hIDs = internaldto.ObtainSubqueryHeirarchyIdentifiers()
	default:
		return nil, fmt.Errorf("cannot resolve taxonomy")
	}
	viewDTO, isView := handlerCtx.GetSQLSystem().GetViewByNameAndParameters(
		hIDs.GetTableName(), params.GetStringified())
	if viewPermissive && !isView {
		viewDTO, isView = handlerCtx.GetSQLSystem().GetViewByName(hIDs.GetTableName())
	}
	if isView {
		hIDs = hIDs.WithView(viewDTO)
	}
	materializedViewDTO, isMaterializedView := handlerCtx.GetSQLSystem().GetMaterializedViewByName(hIDs.GetTableName())
	if isMaterializedView {
		hIDs = hIDs.WithView(materializedViewDTO)
		hIDs.SetIsMaterializedView(true)
	}
	// TODO: pass in current counters
	physicalTableDTO, isPhysicalTable := handlerCtx.GetSQLSystem().GetPhysicalTableByName(hIDs.GetTableName())
	if isPhysicalTable {
		hIDs.SetIsPhysicalTable(true)
		hIDs = hIDs.WithView(physicalTableDTO)
	}
	isInternallyRoutable := handlerCtx.GetPGInternalRouter().ExprIsRoutable(node)
	if isInternallyRoutable {
		hIDs.SetContainsNativeDBMSTable(true)
		return hIDs, nil
	}
	if !(isView || isMaterializedView || isPhysicalTable) && hIDs.GetProviderStr() == "" {
		if handlerCtx.GetCurrentProvider() == "" {
			return nil, fmt.Errorf("could not locate table '%s'", hIDs.GetTableName())
		}
		hIDs.WithProviderStr(handlerCtx.GetCurrentProvider())
	}
	return hIDs, nil
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
//nolint:funlen,gocognit,gocyclo,cyclop,goconst // lots of moving parts
func GetHeirarchyFromStatement(
	handlerCtx handler.HandlerContext,
	node sqlparser.SQLNode,
	parameters parserutil.ColumnKeyedDatastore,
	viewPermissive bool,
) (tablemetadata.HeirarchyObjects, error) {
	var hIDs internaldto.HeirarchyIdentifiers
	getFirstAvailableMethod := false
	if parameters == nil {
		parameters = parserutil.NewParameterMap()
	}
	hIDs, err := GetHIDs(handlerCtx, node, parameters, viewPermissive)
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
			retVal := tablemetadata.NewHeirarchyObjects(hIDs)
			return retVal, nil
		}
		return GetHeirarchyFromStatement(handlerCtx, n.Expr, parameters, viewPermissive)
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
		methodAction = strings.ToLower(n.Action)
	default:
		return nil, fmt.Errorf("cannot resolve taxonomy")
	}
	retVal := tablemetadata.NewHeirarchyObjects(hIDs)
	sqlDataSource, isSQLDataSource := handlerCtx.GetSQLDataSource(hIDs.GetProviderStr())
	if isSQLDataSource {
		retVal.SetSQLDataSource(sqlDataSource)
		return retVal, nil
	}
	// TODO: accomodate complex PG internal queries
	isPgInternal := hIDs.IsPgInternalObject()
	if isPgInternal {
		return retVal, nil
	}
	prov, err := handlerCtx.GetProvider(hIDs.GetProviderStr())
	retVal.SetProvider(prov)
	viewDTO, viewExists := retVal.GetView()
	var meth anysdk.OperationStore
	var methStr string
	var methodErr error
	if methodAction == "" {
		methodAction = "select"
	}
	if viewExists {
		retVal.SetIndirect(viewDTO)
		logging.GetLogger().Debugf("viewDTO = %v\n", viewDTO)
		// return retVal, nil //nolint:nilerr // acceptable
	}
	if err != nil {
		return returnViewOnErrorIfPresent(retVal, err, viewExists)
	}
	svcHdl, err := prov.GetServiceShard(hIDs.GetServiceStr(), hIDs.GetResourceStr(), handlerCtx.GetRuntimeContext())
	if err != nil {
		return returnViewOnErrorIfPresent(retVal, err, viewExists)
	}
	retVal.SetServiceHdl(svcHdl)
	rsc, err := prov.GetResource(hIDs.GetServiceStr(), hIDs.GetResourceStr(), handlerCtx.GetRuntimeContext())
	if err != nil {
		return returnViewOnErrorIfPresent(retVal, err, viewExists)
	}
	retVal.SetResource(rsc)
	viewNameMangler := name_mangle.NewViewNameMangler()
	//nolint:nestif // not overly complex
	if viewCollection, ok := rsc.GetViewsForSqlDialect(
		handlerCtx.GetSQLSystem().GetName()); ok && methodAction == "select" && !viewExists {
		for i, view := range viewCollection {
			viewNameNaive := view.GetNameNaive()
			viewName := viewNameMangler.MangleName(viewNameNaive, i)
			// TODO: mutex required or some other strategy
			viewDTO, viewExists = handlerCtx.GetSQLSystem().GetViewByNameAndParameters(
				viewName, parameters.GetStringified())
			if !viewExists {
				// TODO: resolve any possible data race
				err = handlerCtx.GetSQLSystem().CreateView(viewName, view.GetDDL(), true, nil)
				if err != nil {
					return nil, err
				}
				params := parameters.GetStringified()
				viewDTO, viewExists = handlerCtx.GetSQLSystem().GetViewByNameAndParameters(
					hIDs.GetTableName(), params)
				if viewPermissive {
					viewDTO, viewExists = handlerCtx.GetSQLSystem().GetViewByName(hIDs.GetTableName())
				}
				if viewExists {
					retVal.SetIndirect(viewDTO)
				}
				return retVal, nil
			}
			retVal.SetIndirect(viewDTO)
			return retVal, nil //nolint:staticcheck // TODO: fix this
		}
	}
	var method anysdk.OperationStore
	switch node.(type) {
	case *sqlparser.Exec, *sqlparser.ExecSubquery:
		method, err = rsc.FindMethod(hIDs.GetMethodStr())
		if err != nil {
			return returnViewOnErrorIfPresent(retVal, err, viewExists)
		}
		retVal.SetMethod(method)
		return retVal, nil
	}
	//nolint:nestif,ineffassign // acceptable for now
	if methodRequired {
		switch node.(type) { //nolint:gocritic // this is expressive enough
		case *sqlparser.DescribeTable:
			if viewExists {
				return retVal, nil
			}
			m, mStr, mErr := prov.InferDescribeMethod(rsc)
			if mErr != nil {
				return nil, mErr
			}
			retVal.SetMethod(m)
			retVal.SetMethodStr(mStr)
			return retVal, nil
		}
		if getFirstAvailableMethod {
			meth, methStr, methodErr = prov.GetFirstMethodForAction( //nolint:staticcheck,wastedassign // acceptable
				retVal.GetHeirarchyIDs().GetServiceStr(),
				retVal.GetHeirarchyIDs().GetResourceStr(),
				methodAction,
				handlerCtx.GetRuntimeContext())
		} else {
			meth, methStr, methodErr = prov.GetMethodForAction(
				retVal.GetHeirarchyIDs().GetServiceStr(),
				retVal.GetHeirarchyIDs().GetResourceStr(),
				methodAction,
				parameters,
				handlerCtx.GetRuntimeContext())
			if methodErr != nil {
				return returnViewOnErrorIfPresent(retVal, fmt.Errorf(
					"cannot find matching operation, possible causes include missing required parameters or an unsupported method for the resource, to find required parameters for supported methods run SHOW METHODS IN %s: %w", //nolint:lll // long string
					retVal.GetHeirarchyIDs().GetTableName(), methodErr),
					viewExists)
			}
		}
		availableServers, availableServersDoExist := svcHdl.GetServers()
		if meth != nil {
			availableServers, availableServersDoExist = meth.GetServers()
		}
		if availableServersDoExist {
			for _, srv := range availableServers {
				for k := range srv.Variables {
					logging.GetLogger().Debugf("server parameter = '%s'\n", k)
				}
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

// TODO: remove this rubbish
func returnViewOnErrorIfPresent(
	input tablemetadata.HeirarchyObjects, err error, hasView bool) (tablemetadata.HeirarchyObjects, error) {
	if hasView {
		return input, nil
	}
	if err != nil {
		return nil, err
	}
	return input, nil
}
