package taxonomy

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/provider"

	"github.com/stackql/go-openapistackql/openapistackql"

	"strings"

	"vitess.io/vitess/go/vt/sqlparser"
)

type HeirarchyObjects struct {
	dto.Heirarchy
	HeirarchyIds dto.HeirarchyIdentifiers
	Provider     provider.IProvider
}

func (ho *HeirarchyObjects) LookupSelectItemsKey() string {
	method := ho.Method
	if method == nil {
		return defaultSelectItemsKey
	}
	if sk := method.GetSelectItemsKey(); sk != "" {
		return sk
	}
	responseSchema, _, err := method.GetResponseBodySchemaAndMediaType()
	if responseSchema == nil || err != nil {
		return ""
	}
	switch responseSchema.Type {
	case "string", "integer":
		return openapistackql.AnonymousColumnName
	}
	return defaultSelectItemsKey
}

func (ho *HeirarchyObjects) GetResponseSchemaAndMediaType() (*openapistackql.Schema, string, error) {
	m := ho.Method
	if m == nil {
		return nil, "", fmt.Errorf("method is nil")
	}
	return m.GetResponseBodySchemaAndMediaType()
}

func (ho *HeirarchyObjects) GetSelectSchemaAndObjectPath() (*openapistackql.Schema, string, error) {
	m := ho.Method
	if m == nil {
		return nil, "", fmt.Errorf("method is nil")
	}
	return m.GetSelectSchemaAndObjectPath()
}

func (ho *HeirarchyObjects) GetRequestSchema() (*openapistackql.Schema, error) {
	m := ho.Method
	if m == nil {
		return nil, fmt.Errorf("method is nil")
	}
	return ho.GetRequestSchema()
}

func (ho *HeirarchyObjects) GetTableName() string {
	return ho.HeirarchyIds.GetTableName()
}

func (ho *HeirarchyObjects) GetObjectSchema() (*openapistackql.Schema, error) {
	return ho.getObjectSchema()
}

func (ho *HeirarchyObjects) getObjectSchema() (*openapistackql.Schema, error) {
	rv, _, err := ho.Method.GetResponseBodySchemaAndMediaType()
	return rv, err
}

func (ho *HeirarchyObjects) GetSelectableObjectSchema() (*openapistackql.Schema, error) {
	unsuitableSchemaMsg := "schema unsuitable for select query"
	itemObjS, _, err := ho.Method.GetSelectSchemaAndObjectPath()
	// rscStr, _ := tbl.GetResourceStr()
	if err != nil {
		return nil, fmt.Errorf(unsuitableSchemaMsg)
	}
	if itemObjS == nil || err != nil {
		return nil, fmt.Errorf("could not locate dml object for response type '%v'", ho.Method.Response.ObjectKey)
	}
	return itemObjS, nil
}

func GetHeirarchyIDs(handlerCtx *handler.HandlerContext, node sqlparser.SQLNode) (*dto.HeirarchyIdentifiers, error) {
	return getHids(handlerCtx, node)
}

func getHids(handlerCtx *handler.HandlerContext, node sqlparser.SQLNode) (*dto.HeirarchyIdentifiers, error) {
	var hIds *dto.HeirarchyIdentifiers
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
	case *sqlparser.Delete:
		currentSvcRsc, err := parserutil.ExtractSingleTableFromTableExprs(n.TableExprs)
		if err != nil {
			return nil, err
		}
		hIds = dto.ResolveResourceTerminalHeirarchyIdentifiers(*currentSvcRsc)
	default:
		return nil, fmt.Errorf("cannot resolve taxonomy")
	}
	if hIds.ProviderStr == "" {
		if handlerCtx.CurrentProvider == "" {
			return nil, fmt.Errorf("No provider selected, please set a provider using the USE command, or specify a three part object identifier in your IQL query.")
		}
		hIds.ProviderStr = handlerCtx.CurrentProvider
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

// Hierarchy inference function.
// Returns:
//   - Hierarchy
//   - Supplied parameters that are **not** consumed in Hierarchy inference
//   - Error if applicable.
func GetHeirarchyFromStatement(handlerCtx *handler.HandlerContext, node sqlparser.SQLNode, parameters map[string]interface{}) (*HeirarchyObjects, map[string]interface{}, error) {
	var hIds *dto.HeirarchyIdentifiers
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
	default:
		return nil, remainingParams, fmt.Errorf("cannot resolve taxonomy")
	}
	retVal := HeirarchyObjects{
		HeirarchyIds: *hIds,
	}
	prov, err := handlerCtx.GetProvider(hIds.ProviderStr)
	retVal.Provider = prov
	if err != nil {
		return nil, remainingParams, err
	}
	svcHdl, err := prov.GetServiceShard(hIds.ServiceStr, hIds.ResourceStr, handlerCtx.RuntimeContext)
	if err != nil {
		return nil, remainingParams, err
	}
	retVal.ServiceHdl = svcHdl
	rsc, err := prov.GetResource(hIds.ServiceStr, hIds.ResourceStr, handlerCtx.RuntimeContext)
	if err != nil {
		return nil, remainingParams, err
	}
	retVal.Resource = rsc
	var method *openapistackql.OperationStore
	switch node.(type) {
	case *sqlparser.Exec, *sqlparser.ExecSubquery:
		method, err = rsc.FindMethod(hIds.MethodStr)
		if err != nil {
			return nil, remainingParams, err
		}
		retVal.Method = method
		return &retVal, remainingParams, nil
	}
	if methodRequired {
		switch node.(type) {
		case *sqlparser.DescribeTable:
			m, mStr, err := prov.InferDescribeMethod(rsc)
			if err != nil {
				return nil, remainingParams, err
			}
			retVal.Method = m
			retVal.HeirarchyIds.MethodStr = mStr
			return &retVal, remainingParams, nil
		}
		if methodAction == "" {
			methodAction = "select"
		}
		var meth *openapistackql.OperationStore
		var methStr string
		if getFirstAvailableMethod {
			meth, methStr, err = prov.GetFirstMethodForAction(retVal.HeirarchyIds.ServiceStr, retVal.HeirarchyIds.ResourceStr, methodAction, handlerCtx.RuntimeContext)
		} else {
			meth, methStr, remainingParams, err = prov.GetMethodForAction(retVal.HeirarchyIds.ServiceStr, retVal.HeirarchyIds.ResourceStr, methodAction, remainingParams, handlerCtx.RuntimeContext)
			if err != nil {
				return nil, remainingParams, fmt.Errorf("Cannot find matching operation, possible causes include missing required parameters or an unsupported method for the resource, to find required parameters for supported methods run SHOW METHODS IN %s: %s", retVal.HeirarchyIds.GetTableName(), err.Error())
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
		retVal.HeirarchyIds.MethodStr = methStr
	}
	if methodRequired {
		retVal.Method = method
	}
	return &retVal, remainingParams, nil
}
