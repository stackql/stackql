package taxonomy

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/httpbuild"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/provider"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/go-openapistackql/openapistackql"

	"strings"

	"vitess.io/vitess/go/vt/sqlparser"
)

const (
	defaultSelectItemsKEy = "items"
)

type AnnotationCtx struct {
	Schema     *openapistackql.Schema
	HIDs       *dto.HeirarchyIdentifiers
	TableMeta  *ExtendedTableMetadata
	Parameters map[string]interface{}
}

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
	responseSchema, _, err := method.GetResponseBodySchemaAndMediaType()
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
	return defaultSelectItemsKEy
}

type TblMap map[sqlparser.SQLNode]*ExtendedTableMetadata

type AnnotationCtxMap map[sqlparser.SQLNode]AnnotationCtx

type AnnotatedTabulationMap map[sqlparser.SQLNode]util.AnnotatedTabulation

func (tm TblMap) GetTable(node sqlparser.SQLNode) (*ExtendedTableMetadata, error) {
	tbl, ok := tm[node]
	if !ok {
		return nil, fmt.Errorf("could not locate table for AST node: %v", node)
	}
	return tbl, nil
}

func (tm TblMap) getUniqueCount() int {
	found := make(map[*ExtendedTableMetadata]struct{})
	for _, v := range tm {
		if _, ok := found[v]; !ok {
			found[v] = struct{}{}
		}
	}
	return len(found)
}

func (tm TblMap) getFirst() (*ExtendedTableMetadata, bool) {
	for _, v := range tm {
		return v, true
	}
	return nil, false
}

func (tm TblMap) GetTableLoose(node sqlparser.SQLNode) (*ExtendedTableMetadata, error) {
	tbl, ok := tm[node]
	if ok {
		return tbl, nil
	}
	searchAlias := ""
	switch node := node.(type) {
	case *sqlparser.AliasedExpr:
		switch expr := node.Expr.(type) {
		case *sqlparser.ColName:
			searchAlias = expr.Qualifier.GetRawVal()
		}
	}
	if searchAlias != "" {
		for k, v := range tm {
			switch k := k.(type) {
			case *sqlparser.AliasedTableExpr:
				alias := k.As.GetRawVal()
				if searchAlias == alias {
					return v, nil
				}
			}
		}
	}
	if searchAlias == "" && tm.getUniqueCount() == 1 {
		if first, ok := tm.getFirst(); ok {
			return first, nil
		}
	}
	return nil, fmt.Errorf("could not locate table for AST node: %v", node)
}

func (tm TblMap) SetTable(node sqlparser.SQLNode, table *ExtendedTableMetadata) {
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

func (ex ExtendedTableMetadata) IsSimple() bool {
	return ex.isSimple()
}

func (ex ExtendedTableMetadata) isSimple() bool {
	return ex.HeirarchyObjects != nil && (len(ex.HeirarchyObjects.MethodSet) > 0 || ex.HeirarchyObjects.Method != nil)
}

func (ex ExtendedTableMetadata) GetUniqueId() string {
	if ex.Alias != "" {
		return ex.Alias
	}
	return ex.HeirarchyObjects.GetTableName()
}

func (ex ExtendedTableMetadata) GetQueryUniqueId() string {
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

func (ex ExtendedTableMetadata) GetSelectSchemaAndObjectPath() (*openapistackql.Schema, string, error) {
	return ex.HeirarchyObjects.GetSelectSchemaAndObjectPath()
}

func (ex ExtendedTableMetadata) GetResponseSchemaAndMediaType() (*openapistackql.Schema, string, error) {
	if ex.isSimple() {
		return ex.HeirarchyObjects.GetResponseSchemaAndMediaType()
	}
	return nil, "", fmt.Errorf("subqueries currently not supported")
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

func NewExtendedTableMetadata(heirarchyObjects *HeirarchyObjects, alias string) *ExtendedTableMetadata {
	return &ExtendedTableMetadata{
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

func GetHeirarchyFromStatement(handlerCtx *handler.HandlerContext, node sqlparser.SQLNode, parameters map[string]interface{}) (*HeirarchyObjects, map[string]interface{}, error) {
	var hIds *dto.HeirarchyIdentifiers
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
		meth, methStr, remainingParams, err = prov.GetMethodForAction(retVal.HeirarchyIds.ServiceStr, retVal.HeirarchyIds.ResourceStr, methodAction, remainingParams, handlerCtx.RuntimeContext)
		if err != nil {
			return nil, remainingParams, fmt.Errorf("could not find method in taxonomy: %s", err.Error())
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
