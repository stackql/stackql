package taxonomy

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/httpbuild"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/provider"

	"github.com/stackql/go-openapistackql/openapistackql"

	"strings"

	"vitess.io/vitess/go/vt/sqlparser"
)

const (
	defaultSelectItemsKEy = "items"
)

func (ex ExtendedTableMetadata) LookupSelectItemsKey() string {
	if ex.HeirarchyObjects == nil {
		return defaultSelectItemsKEy
	}
	return ex.HeirarchyObjects.LookupSelectItemsKey()
}

func (ho *HeirarchyObjects) LookupSelectItemsKey() string {
	prov := ho.Provider
	svcHdl := ho.ServiceHdl
	rsc := ho.Resource
	method := ho.Method
	if method == nil {
		return defaultSelectItemsKEy
	}
	if sk := method.GetSelectItemsKey(); sk != "" {
		return sk
	}
	responseSchema, err := method.GetResponseBodySchema()
	if responseSchema == nil || err != nil {
		return ""
	}
	switch responseSchema.Type {
	case "string", "integer":
		return openapistackql.AnonymousColumnName
	}
	if prov.GetProviderString() == "google" {
		sn := svcHdl.GetName()
		if sn == "bigquery" && svcHdl.Info.Version == "v2" {
			if rsc.ID != "" {
			}
			if method.GetName() != "" {
			}
			if responseSchema.GetName() == "GetQueryResultsResponse" {
				return "rows"
			}
		}
		if sn == "container" && svcHdl.Info.Version == "v1" {
			if rsc.ID != "" {
			}
			if method.GetName() != "" {
			}
			if responseSchema.GetName() == "ListUsableSubnetworksResponse" {
				return "subnetworks"
			}
		}
		if sn == "cloudresourcemanager" && svcHdl.Info.Version == "v3" {
			if responseSchema.GetName() == "ListProjectsResponse" {
				return "projects"
			}
		}
		if responseSchema.GetName() == "Policy" {
			return "bindings"
		}
	}
	return "items"
}

type TblMap map[sqlparser.SQLNode]ExtendedTableMetadata

func (tm TblMap) GetTable(node sqlparser.SQLNode) (ExtendedTableMetadata, error) {
	tbl, ok := tm[node]
	if !ok {
		return ExtendedTableMetadata{}, fmt.Errorf("could not locate table for AST node: %v", node)
	}
	return tbl, nil
}

func (tm TblMap) SetTable(node sqlparser.SQLNode, table ExtendedTableMetadata) {
	tm[node] = table
}

type ExtendedTableMetadata struct {
	TableFilter         func(openapistackql.ITable) (openapistackql.ITable, error)
	ColsVisited         map[string]bool
	HeirarchyObjects    *HeirarchyObjects
	RequiredParameters  map[string]openapistackql.Parameter
	IsLocallyExecutable bool
	HttpArmoury         *httpbuild.HTTPArmoury
	SelectItemsKey      string
	Alias               string
}

func (ex ExtendedTableMetadata) GetAlias() string {
	return ex.Alias
}

func (ex ExtendedTableMetadata) GetUniqueId() string {
	if ex.Alias != "" {
		return ex.Alias
	}
	return ex.HeirarchyObjects.GetTableName()
}

func (ex ExtendedTableMetadata) GetProvider() (provider.IProvider, error) {
	if ex.HeirarchyObjects == nil || ex.HeirarchyObjects.Provider == nil {
		return nil, fmt.Errorf("cannot resolve Provider")
	}
	return ex.HeirarchyObjects.Provider, nil
}

func (ex ExtendedTableMetadata) GetProviderObject() (*openapistackql.Provider, error) {
	if ex.HeirarchyObjects == nil || ex.HeirarchyObjects.Provider == nil {
		return nil, fmt.Errorf("cannot resolve Provider")
	}
	return ex.HeirarchyObjects.Provider.GetProvider()
}

func (ex ExtendedTableMetadata) GetService() (*openapistackql.Service, error) {
	if ex.HeirarchyObjects == nil || ex.HeirarchyObjects.ServiceHdl == nil {
		return nil, fmt.Errorf("cannot resolve ServiceHandle")
	}
	return ex.HeirarchyObjects.ServiceHdl, nil
}

func (ex ExtendedTableMetadata) GetResource() (*openapistackql.Resource, error) {
	if ex.HeirarchyObjects == nil || ex.HeirarchyObjects.Resource == nil {
		return nil, fmt.Errorf("cannot resolve Resource")
	}
	return ex.HeirarchyObjects.Resource, nil
}

func (ex ExtendedTableMetadata) GetMethod() (*openapistackql.OperationStore, error) {
	return ex.getMethod()
}

func (ex ExtendedTableMetadata) getMethod() (*openapistackql.OperationStore, error) {
	if ex.HeirarchyObjects == nil || ex.HeirarchyObjects.Method == nil {
		return nil, fmt.Errorf("cannot resolve Method")
	}
	return ex.HeirarchyObjects.Method, nil
}

func (ex ExtendedTableMetadata) GetResponseSchema() (*openapistackql.Schema, error) {
	return ex.HeirarchyObjects.GetResponseSchema()
}

func (ho *HeirarchyObjects) GetResponseSchema() (*openapistackql.Schema, error) {
	m := ho.Method
	if m == nil {
		return nil, fmt.Errorf("method is nil")
	}
	return m.GetResponseBodySchema()
}

func (ex ExtendedTableMetadata) GetRequestSchema() (*openapistackql.Schema, error) {
	return ex.HeirarchyObjects.GetRequestSchema()
}

func (ho *HeirarchyObjects) GetRequestSchema() (*openapistackql.Schema, error) {
	m := ho.Method
	if m == nil {
		return nil, fmt.Errorf("method is nil")
	}
	return ho.GetRequestSchema()
}

func (ex ExtendedTableMetadata) GetServiceStr() (string, error) {
	if ex.HeirarchyObjects == nil || ex.HeirarchyObjects.HeirarchyIds.ServiceStr == "" {
		return "", fmt.Errorf("cannot resolve ServiceStr")
	}
	return ex.HeirarchyObjects.HeirarchyIds.ServiceStr, nil
}

func (ex ExtendedTableMetadata) GetResourceStr() (string, error) {
	if ex.HeirarchyObjects == nil || ex.HeirarchyObjects.HeirarchyIds.ResourceStr == "" {
		return "", fmt.Errorf("cannot resolve ResourceStr")
	}
	return ex.HeirarchyObjects.HeirarchyIds.ResourceStr, nil
}

func (ex ExtendedTableMetadata) GetProviderStr() (string, error) {
	if ex.HeirarchyObjects == nil || ex.HeirarchyObjects.HeirarchyIds.ProviderStr == "" {
		return "", fmt.Errorf("cannot resolve ProviderStr")
	}
	return ex.HeirarchyObjects.HeirarchyIds.ProviderStr, nil
}

func (ex ExtendedTableMetadata) GetMethodStr() (string, error) {
	if ex.HeirarchyObjects == nil || ex.HeirarchyObjects.HeirarchyIds.MethodStr == "" {
		return "", fmt.Errorf("cannot resolve MethodStr")
	}
	return ex.HeirarchyObjects.HeirarchyIds.MethodStr, nil
}

func (ex ExtendedTableMetadata) GetHTTPArmoury() (*httpbuild.HTTPArmoury, error) {
	return ex.HttpArmoury, nil
}

func (ex ExtendedTableMetadata) GetTableName() (string, error) {
	if ex.HeirarchyObjects == nil || ex.HeirarchyObjects.HeirarchyIds.GetTableName() == "" {
		return "", fmt.Errorf("cannot resolve TableName")
	}
	return ex.HeirarchyObjects.HeirarchyIds.GetTableName(), nil
}

func (ex ExtendedTableMetadata) GetSelectableObjectSchema() (*openapistackql.Schema, error) {
	return ex.HeirarchyObjects.GetSelectableObjectSchema()
}

func NewExtendedTableMetadata(heirarchyObjects *HeirarchyObjects, alias string) ExtendedTableMetadata {
	return ExtendedTableMetadata{
		ColsVisited:        make(map[string]bool),
		RequiredParameters: make(map[string]openapistackql.Parameter),
		HeirarchyObjects:   heirarchyObjects,
		Alias:              alias,
	}
}

type HeirarchyObjects struct {
	HeirarchyIds dto.HeirarchyIdentifiers
	Provider     provider.IProvider
	ServiceHdl   *openapistackql.Service
	Resource     *openapistackql.Resource
	MethodSet    openapistackql.MethodSet
	Method       *openapistackql.OperationStore
}

func (ho *HeirarchyObjects) GetTableName() string {
	return ho.HeirarchyIds.GetTableName()
}

func (ho *HeirarchyObjects) GetObjectSchema() (*openapistackql.Schema, error) {
	return ho.getObjectSchema()
}

func (ho *HeirarchyObjects) getObjectSchema() (*openapistackql.Schema, error) {
	return ho.Method.GetResponseBodySchema()
}

func (ho *HeirarchyObjects) GetSelectableObjectSchema() (*openapistackql.Schema, error) {
	responseObj, err := ho.getObjectSchema()
	if err != nil {
		return nil, err
	}
	itemsKey := ho.Method.Response.ObjectKey
	if itemsKey == "" {
		itemsKey = ho.LookupSelectItemsKey()
	}
	itemObjS, _, err := responseObj.GetSelectSchema(itemsKey)
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

func GetHeirarchyFromStatement(handlerCtx *handler.HandlerContext, node sqlparser.SQLNode, parameters map[string]interface{}) (*HeirarchyObjects, error) {
	var hIds *dto.HeirarchyIdentifiers
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
		case "METHODS":
			methodRequired = false
		default:
			return nil, fmt.Errorf("cannot resolve taxonomy for SHOW statement of type = '%s'", strings.ToUpper(n.Type))
		}
	case *sqlparser.Insert:
		methodAction = "insert"
	case *sqlparser.Delete:
		methodAction = "delete"
	default:
		return nil, fmt.Errorf("cannot resolve taxonomy")
	}
	retVal := HeirarchyObjects{
		HeirarchyIds: *hIds,
	}
	prov, err := handlerCtx.GetProvider(hIds.ProviderStr)
	retVal.Provider = prov
	if err != nil {
		return nil, err
	}
	svcHdl, err := prov.GetServiceShard(hIds.ServiceStr, hIds.ResourceStr, handlerCtx.RuntimeContext)
	if err != nil {
		return nil, err
	}
	retVal.ServiceHdl = svcHdl
	rsc, err := prov.GetResource(hIds.ServiceStr, hIds.ResourceStr, handlerCtx.RuntimeContext)
	if err != nil {
		return nil, err
	}
	retVal.Resource = rsc
	method, methodErr := rsc.FindMethod(hIds.MethodStr) // rsc.Methods[hIds.MethodStr]
	if methodErr != nil && methodRequired {
		switch node.(type) {
		case *sqlparser.DescribeTable:
			m, mStr, err := prov.InferDescribeMethod(rsc)
			if err != nil {
				return nil, err
			}
			retVal.Method = m
			retVal.HeirarchyIds.MethodStr = mStr
			return &retVal, nil
		}
		if methodAction == "" {
			methodAction = "select"
		}
		meth, methStr, err := prov.GetMethodForAction(retVal.HeirarchyIds.ServiceStr, retVal.HeirarchyIds.ResourceStr, methodAction, parameters, handlerCtx.RuntimeContext)
		if err != nil {
			return nil, fmt.Errorf("could not find method in taxonomy: %s", err.Error())
		}
		method = meth
		retVal.HeirarchyIds.MethodStr = methStr
	}
	if methodRequired {
		retVal.Method = method
	}
	return &retVal, nil
}
