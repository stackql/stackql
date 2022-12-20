package taxonomy

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/astformat"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"

	"github.com/stackql/go-openapistackql/openapistackql"

	"strings"

	"vitess.io/vitess/go/vt/sqlparser"
)

func GetHeirarchyIDsFromParserNode(handlerCtx handler.HandlerContext, node sqlparser.SQLNode) (internaldto.HeirarchyIdentifiers, error) {
	return getHids(handlerCtx, node)
}

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
	default:
		return nil, fmt.Errorf("cannot resolve taxonomy")
	}
	viewDTO, isView := handlerCtx.GetSQLDialect().GetViewByName(hIds.GetTableName())
	if isView {
		hIds = hIds.WithView(viewDTO)
	}
	if !(isView) && hIds.GetProviderStr() == "" {
		if handlerCtx.GetCurrentProvider() == "" {
			return nil, fmt.Errorf("No provider selected, please set a provider using the USE command, or specify a three part object identifier in your IQL query")
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
func GetHeirarchyFromStatement(handlerCtx handler.HandlerContext, node sqlparser.SQLNode, parameters parserutil.ColumnKeyedDatastore) (tablemetadata.HeirarchyObjects, error) {
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
	viewDTO, isView := retVal.GetView()
	if isView {
		logging.GetLogger().Debugf("viewDTO = %v\n", viewDTO)
		return retVal, nil
	}
	prov, err := handlerCtx.GetProvider(hIds.GetProviderStr())
	retVal.SetProvider(prov)
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
	var method *openapistackql.OperationStore
	switch node.(type) {
	case *sqlparser.Exec, *sqlparser.ExecSubquery:
		method, err = rsc.FindMethod(hIds.GetMethodStr())
		if err != nil {
			return nil, err
		}
		retVal.SetMethod(method)
		return retVal, nil
	}
	if methodRequired {
		switch node.(type) {
		case *sqlparser.DescribeTable:
			m, mStr, err := prov.InferDescribeMethod(rsc)
			if err != nil {
				return nil, err
			}
			retVal.SetMethod(m)
			retVal.SetMethodStr(mStr)
			return retVal, nil
		}
		if methodAction == "" {
			methodAction = "select"
		}
		var meth *openapistackql.OperationStore
		var methStr string
		if getFirstAvailableMethod {
			meth, methStr, err = prov.GetFirstMethodForAction(retVal.GetHeirarchyIds().GetServiceStr(), retVal.GetHeirarchyIds().GetResourceStr(), methodAction, handlerCtx.GetRuntimeContext())
		} else {
			meth, methStr, err = prov.GetMethodForAction(retVal.GetHeirarchyIds().GetServiceStr(), retVal.GetHeirarchyIds().GetResourceStr(), methodAction, parameters, handlerCtx.GetRuntimeContext())
			if err != nil {
				return nil, fmt.Errorf("Cannot find matching operation, possible causes include missing required parameters or an unsupported method for the resource, to find required parameters for supported methods run SHOW METHODS IN %s: %s", retVal.GetHeirarchyIds().GetTableName(), err.Error())
			}
		}
		for _, srv := range svcHdl.Servers {
			for k := range srv.Variables {
				logging.GetLogger().Debugf("server paramter = '%s'\n", k)
				// parameters.DeleteByString(k)
			}
		}
		// for _, srv := range svcHdl.Servers {
		// 	for k, _ := range srv.Variables {
		// 		_, ok := remainingParams[k]
		// 		if ok {
		// 			delete(remainingParams, k)
		// 		}
		// 	}
		// }
		method = meth
		retVal.SetMethodStr(methStr)
	}
	if methodRequired {
		retVal.SetMethod(method)
	}
	return retVal, nil
}
