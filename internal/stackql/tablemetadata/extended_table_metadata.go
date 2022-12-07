package tablemetadata

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/httpbuild"
	"github.com/stackql/stackql/internal/stackql/provider"

	"github.com/stackql/go-openapistackql/openapistackql"
)

var (
	_ ExtendedTableMetadata = &standardExtendedTableMetadata{}
)

type ExtendedTableMetadata interface {
	GetAlias() string
	GetGraphQL() (*openapistackql.GraphQL, bool)
	GetHeirarchyObjects() HeirarchyObjects
	GetHttpArmoury() (httpbuild.HTTPArmoury, error)
	GetInputTableName() (string, error)
	GetMethod() (*openapistackql.OperationStore, error)
	GetMethodStr() (string, error)
	GetProvider() (provider.IProvider, error)
	GetProviderStr() (string, error)
	GetProviderObject() (*openapistackql.Provider, error)
	GetQueryUniqueId() string
	GetRequestSchema() (*openapistackql.Schema, error)
	GetRequiredParameters() map[string]openapistackql.Parameter
	GetResource() (*openapistackql.Resource, error)
	GetResourceStr() (string, error)
	GetResponseSchemaStr() (string, error)
	GetResponseSchemaAndMediaType() (*openapistackql.Schema, string, error)
	GetSelectableObjectSchema() (*openapistackql.Schema, error)
	GetSelectItemsKey() string
	GetSelectSchemaAndObjectPath() (*openapistackql.Schema, string, error)
	GetService() (*openapistackql.Service, error)
	GetServiceStr() (string, error)
	GetStackQLTableName() (string, error)
	GetTableFilter() func(openapistackql.ITable) (openapistackql.ITable, error)
	GetTableName() (string, error)
	GetUniqueId() string
	IsLocallyExecutable() bool
	IsSimple() bool
	IsView() bool
	LookupSelectItemsKey() string
	SetSelectItemsKey(string)
	SetTableFilter(f func(openapistackql.ITable) (openapistackql.ITable, error))
	WithGetHttpArmoury(f func() (httpbuild.HTTPArmoury, error)) ExtendedTableMetadata
	WithResponseSchemaStr(rss string) (ExtendedTableMetadata, error)
}

type standardExtendedTableMetadata struct {
	tableFilter         func(openapistackql.ITable) (openapistackql.ITable, error)
	colsVisited         map[string]bool
	heirarchyObjects    HeirarchyObjects
	requiredParameters  map[string]openapistackql.Parameter
	isLocallyExecutable bool
	getHttpArmoury      func() (httpbuild.HTTPArmoury, error)
	selectItemsKey      string
	alias               string
	inputTableName      string
}

func (ex *standardExtendedTableMetadata) IsLocallyExecutable() bool {
	return ex.isLocallyExecutable
}

func (ex *standardExtendedTableMetadata) GetSelectItemsKey() string {
	return ex.selectItemsKey
}

func (ex *standardExtendedTableMetadata) GetHeirarchyObjects() HeirarchyObjects {
	return ex.heirarchyObjects
}

func (ex *standardExtendedTableMetadata) SetSelectItemsKey(k string) {
	ex.selectItemsKey = k
}

func (ex *standardExtendedTableMetadata) SetTableFilter(f func(openapistackql.ITable) (openapistackql.ITable, error)) {
	ex.tableFilter = f
}

func (ex *standardExtendedTableMetadata) GetTableFilter() func(openapistackql.ITable) (openapistackql.ITable, error) {
	return ex.tableFilter
}

func (ex *standardExtendedTableMetadata) GetGraphQL() (*openapistackql.GraphQL, bool) {
	if ex.heirarchyObjects.GetMethod() != nil && ex.heirarchyObjects.GetMethod().GraphQL != nil {
		return ex.heirarchyObjects.GetMethod().GraphQL, true
	}
	return nil, false
}

func (ex *standardExtendedTableMetadata) GetRequiredParameters() map[string]openapistackql.Parameter {
	return ex.requiredParameters
}

func (ex *standardExtendedTableMetadata) GetHttpArmoury() (httpbuild.HTTPArmoury, error) {
	if ex.getHttpArmoury == nil {
		return nil, fmt.Errorf("nil getHttpAroury() function in ExtendedTableMetadata object")
	}
	return ex.getHttpArmoury()
}

func (ex *standardExtendedTableMetadata) WithGetHttpArmoury(f func() (httpbuild.HTTPArmoury, error)) ExtendedTableMetadata {
	ex.getHttpArmoury = f
	return ex
}

func (ex *standardExtendedTableMetadata) LookupSelectItemsKey() string {
	if ex.heirarchyObjects == nil {
		return defaultSelectItemsKey
	}
	return ex.heirarchyObjects.LookupSelectItemsKey()
}

func (ex *standardExtendedTableMetadata) GetAlias() string {
	return ex.alias
}

func (ex *standardExtendedTableMetadata) IsSimple() bool {
	return ex.isSimple()
}

func (ex *standardExtendedTableMetadata) IsView() bool {
	return ex.heirarchyObjects.IsView()
}

func (ex *standardExtendedTableMetadata) isSimple() bool {
	return ex.heirarchyObjects != nil && (len(ex.heirarchyObjects.GetMethodSet()) > 0 || ex.heirarchyObjects.GetMethod() != nil)
}

func (ex *standardExtendedTableMetadata) GetUniqueId() string {
	if ex.alias != "" {
		return ex.alias
	}
	return ex.heirarchyObjects.GetTableName()
}

func (ex *standardExtendedTableMetadata) GetQueryUniqueId() string {
	if ex.alias != "" {
		return ex.alias
	}
	return ex.heirarchyObjects.GetTableName()
}

func (ex *standardExtendedTableMetadata) GetProvider() (provider.IProvider, error) {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetProvider() == nil {
		return nil, fmt.Errorf("cannot resolve Provider")
	}
	return ex.heirarchyObjects.GetProvider(), nil
}

func (ex *standardExtendedTableMetadata) GetProviderObject() (*openapistackql.Provider, error) {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetProvider() == nil {
		return nil, fmt.Errorf("cannot resolve Provider")
	}
	return ex.heirarchyObjects.GetProvider().GetProvider()
}

func (ex *standardExtendedTableMetadata) GetService() (*openapistackql.Service, error) {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetServiceHdl() == nil {
		return nil, fmt.Errorf("cannot resolve ServiceHandle")
	}
	return ex.heirarchyObjects.GetServiceHdl(), nil
}

func (ex *standardExtendedTableMetadata) GetResource() (*openapistackql.Resource, error) {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetResource() == nil {
		return nil, fmt.Errorf("cannot resolve Resource")
	}
	return ex.heirarchyObjects.GetResource(), nil
}

func (ex *standardExtendedTableMetadata) GetMethod() (*openapistackql.OperationStore, error) {
	return ex.getMethod()
}

func (ex *standardExtendedTableMetadata) getMethod() (*openapistackql.OperationStore, error) {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetMethod() == nil {
		return nil, fmt.Errorf("cannot resolve Method")
	}
	return ex.heirarchyObjects.GetMethod(), nil
}

func (ex *standardExtendedTableMetadata) GetSelectSchemaAndObjectPath() (*openapistackql.Schema, string, error) {
	return ex.heirarchyObjects.GetSelectSchemaAndObjectPath()
}

func (ex *standardExtendedTableMetadata) GetResponseSchemaAndMediaType() (*openapistackql.Schema, string, error) {
	if ex.isSimple() {
		return ex.heirarchyObjects.GetResponseSchemaAndMediaType()
	}
	return nil, "", fmt.Errorf("subqueries currently not supported")
}

func (ex *standardExtendedTableMetadata) GetRequestSchema() (*openapistackql.Schema, error) {
	return ex.heirarchyObjects.GetRequestSchema()
}

func (ex *standardExtendedTableMetadata) GetServiceStr() (string, error) {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetHeirarchyIds().GetServiceStr() == "" {
		return "", fmt.Errorf("cannot resolve ServiceStr")
	}
	return ex.heirarchyObjects.GetHeirarchyIds().GetServiceStr(), nil
}

func (ex *standardExtendedTableMetadata) GetResourceStr() (string, error) {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetHeirarchyIds().GetResourceStr() == "" {
		return "", fmt.Errorf("cannot resolve ResourceStr")
	}
	return ex.heirarchyObjects.GetHeirarchyIds().GetResourceStr(), nil
}

func (ex *standardExtendedTableMetadata) GetProviderStr() (string, error) {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetHeirarchyIds().GetProviderStr() == "" {
		return "", fmt.Errorf("cannot resolve ProviderStr")
	}
	return ex.heirarchyObjects.GetHeirarchyIds().GetProviderStr(), nil
}

func (ex *standardExtendedTableMetadata) GetMethodStr() (string, error) {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetHeirarchyIds().GetMethodStr() == "" {
		return "", fmt.Errorf("cannot resolve MethodStr")
	}
	return ex.heirarchyObjects.GetHeirarchyIds().GetMethodStr(), nil
}

func (ex *standardExtendedTableMetadata) GetResponseSchemaStr() (string, error) {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetHeirarchyIds().GetResponseSchemaStr() == "" {
		return "", fmt.Errorf("cannot resolve ResponseSchemaStr")
	}
	return ex.heirarchyObjects.GetHeirarchyIds().GetResponseSchemaStr(), nil
}

func (ex *standardExtendedTableMetadata) WithResponseSchemaStr(rss string) (ExtendedTableMetadata, error) {
	if ex.heirarchyObjects == nil {
		return ex, fmt.Errorf("standardExtendedTableMetadata.WithResponseSchemaStr(): cannot resolve HeirarchyObjects")
	}
	ex.heirarchyObjects.GetHeirarchyIds().WithResponseSchemaStr(rss)
	return ex, nil
}

func (ex *standardExtendedTableMetadata) GetTableName() (string, error) {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetHeirarchyIds().GetTableName() == "" {
		return "", fmt.Errorf("cannot resolve TableName")
	}
	return ex.heirarchyObjects.GetHeirarchyIds().GetTableName(), nil
}

func (ex *standardExtendedTableMetadata) GetStackQLTableName() (string, error) {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetHeirarchyIds().GetTableName() == "" {
		return "", fmt.Errorf("cannot resolve TableName")
	}
	return ex.heirarchyObjects.GetHeirarchyIds().GetStackQLTableName(), nil
}

func (ex *standardExtendedTableMetadata) GetInputTableName() (string, error) {
	return ex.inputTableName, nil
}

func (ex *standardExtendedTableMetadata) GetSelectableObjectSchema() (*openapistackql.Schema, error) {
	return ex.heirarchyObjects.GetSelectableObjectSchema()
}

func NewExtendedTableMetadata(heirarchyObjects HeirarchyObjects, tableName string, alias string) ExtendedTableMetadata {
	return &standardExtendedTableMetadata{
		colsVisited:        make(map[string]bool),
		requiredParameters: make(map[string]openapistackql.Parameter),
		heirarchyObjects:   heirarchyObjects,
		alias:              alias,
		inputTableName:     tableName,
	}
}
