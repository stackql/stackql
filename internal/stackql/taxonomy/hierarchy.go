package taxonomy

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/astformat"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"

	"github.com/stackql/go-openapistackql/openapistackql"

	"strings"

	"vitess.io/vitess/go/vt/sqlparser"
)

func GetHeirarchyIDs(handlerCtx *handler.HandlerContext, node sqlparser.SQLNode) (dto.HeirarchyIdentifiers, error) {
	return getHids(handlerCtx, node)
}

func getHids(handlerCtx *handler.HandlerContext, node sqlparser.SQLNode) (dto.HeirarchyIdentifiers, error) {
	var hIds dto.HeirarchyIdentifiers
	switch n := node.(type) {
	case *sqlparser.Exec:
		hIds = dto.ResolveMethodTerminalHeirarchyIdentifiers(n.MethodName)
	case *sqlparser.ExecSubquery:
		hIds = dto.ResolveMethodTerminalHeirarchyIdentifiers(n.Exec.MethodName)
	case *sqlparser.Select:
		currentSvcRsc, err := parserutil.TableFromSelectNode(n)
		if err != nil {
			return nil, err
		}
		hIds = dto.ResolveResourceTerminalHeirarchyIdentifiers(currentSvcRsc)
	case sqlparser.TableName:
		hIds = dto.ResolveResourceTerminalHeirarchyIdentifiers(n)
	case *sqlparser.AliasedTableExpr:
		return getHids(handlerCtx, n.Expr)
	case *sqlparser.DescribeTable:
		return getHids(handlerCtx, n.Table)
	case *sqlparser.Show:
		switch strings.ToUpper(n.Type) {
		case "INSERT":
			hIds = dto.ResolveResourceTerminalHeirarchyIdentifiers(n.OnTable)
		case "METHODS":
			hIds = dto.ResolveResourceTerminalHeirarchyIdentifiers(n.OnTable)
		default:
			return nil, fmt.Errorf("cannot resolve taxonomy for SHOW statement of type = '%s'", strings.ToUpper(n.Type))
		}
	case *sqlparser.Insert:
		hIds = dto.ResolveResourceTerminalHeirarchyIdentifiers(n.Table)
	case *sqlparser.Update:
		currentSvcRsc, err := parserutil.ExtractSingleTableFromTableExprs(n.TableExprs)
		if err != nil {
			return nil, err
		}
		hIds = dto.ResolveResourceTerminalHeirarchyIdentifiers(*currentSvcRsc)
	case *sqlparser.Delete:
		currentSvcRsc, err := parserutil.ExtractSingleTableFromTableExprs(n.TableExprs)
		if err != nil {
			return nil, err
		}
		hIds = dto.ResolveResourceTerminalHeirarchyIdentifiers(*currentSvcRsc)
	default:
		return nil, fmt.Errorf("cannot resolve taxonomy")
	}
	if hIds.GetProviderStr() == "" {
		if handlerCtx.CurrentProvider == "" {
			return nil, fmt.Errorf("No provider selected, please set a provider using the USE command, or specify a three part object identifier in your IQL query.")
		}
		hIds.WithProviderStr(handlerCtx.CurrentProvider)
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
	default:
		return astformat.String(n, formatter)
	}
}

// Hierarchy inference function.
// Returns:
//   - Hierarchy
//   - Supplied parameters that are **not** consumed in Hierarchy inference
//   - Error if applicable.
func GetHeirarchyFromStatement(handlerCtx *handler.HandlerContext, node sqlparser.SQLNode, parameters map[string]interface{}) (tablemetadata.HeirarchyObjects, map[string]interface{}, error) {
	var hIds dto.HeirarchyIdentifiers
	getFirstAvailableMethod := false
	remainingParams := make(map[string]interface{})
	for k, v := range parameters {
		remainingParams[k] = v
	}
	hIds, err := getHids(handlerCtx, node)
	if err != nil {
		return nil, remainingParams, err
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
		return GetHeirarchyFromStatement(handlerCtx, n.Expr, remainingParams)
	case *sqlparser.Show:
		switch strings.ToUpper(n.Type) {
		case "INSERT":
			methodAction = "insert"
			getFirstAvailableMethod = true
		case "METHODS":
			methodRequired = false
		default:
			return nil, remainingParams, fmt.Errorf("cannot resolve taxonomy for SHOW statement of type = '%s'", strings.ToUpper(n.Type))
		}
	case *sqlparser.Insert:
		methodAction = "insert"
	case *sqlparser.Delete:
		methodAction = "delete"
	case *sqlparser.Update:
		methodAction = "update"
	default:
		return nil, remainingParams, fmt.Errorf("cannot resolve taxonomy")
	}
	retVal := tablemetadata.NewHeirarchyObjects(hIds)
	prov, err := handlerCtx.GetProvider(hIds.GetProviderStr())
	retVal.SetProvider(prov)
	if err != nil {
		return nil, remainingParams, err
	}
	svcHdl, err := prov.GetServiceShard(hIds.GetServiceStr(), hIds.GetResourceStr(), handlerCtx.RuntimeContext)
	if err != nil {
		return nil, remainingParams, err
	}
	retVal.SetServiceHdl(svcHdl)
	rsc, err := prov.GetResource(hIds.GetServiceStr(), hIds.GetResourceStr(), handlerCtx.RuntimeContext)
	if err != nil {
		return nil, remainingParams, err
	}
	retVal.SetResource(rsc)
	var method *openapistackql.OperationStore
	switch node.(type) {
	case *sqlparser.Exec, *sqlparser.ExecSubquery:
		method, err = rsc.FindMethod(hIds.GetMethodStr())
		if err != nil {
			return nil, remainingParams, err
		}
		retVal.SetMethod(method)
		return retVal, remainingParams, nil
	}
	if methodRequired {
		switch node.(type) {
		case *sqlparser.DescribeTable:
			m, mStr, err := prov.InferDescribeMethod(rsc)
			if err != nil {
				return nil, remainingParams, err
			}
			retVal.SetMethod(m)
			retVal.SetMethodStr(mStr)
			return retVal, remainingParams, nil
		}
		if methodAction == "" {
			methodAction = "select"
		}
		var meth *openapistackql.OperationStore
		var methStr string
		if getFirstAvailableMethod {
			meth, methStr, err = prov.GetFirstMethodForAction(retVal.GetHeirarchyIds().GetServiceStr(), retVal.GetHeirarchyIds().GetResourceStr(), methodAction, handlerCtx.RuntimeContext)
		} else {
			meth, methStr, remainingParams, err = prov.GetMethodForAction(retVal.GetHeirarchyIds().GetServiceStr(), retVal.GetHeirarchyIds().GetResourceStr(), methodAction, remainingParams, handlerCtx.RuntimeContext)
			if err != nil {
				return nil, remainingParams, fmt.Errorf("Cannot find matching operation, possible causes include missing required parameters or an unsupported method for the resource, to find required parameters for supported methods run SHOW METHODS IN %s: %s", retVal.GetHeirarchyIds().GetTableName(), err.Error())
			}
		}
		for _, srv := range svcHdl.Servers {
			for k, _ := range srv.Variables {
				_, ok := remainingParams[k]
				if ok {
					delete(remainingParams, k)
				}
			}
		}
		method = meth
		retVal.SetMethodStr(methStr)
	}
	if methodRequired {
		retVal.SetMethod(method)
	}
	return retVal, remainingParams, nil
}
