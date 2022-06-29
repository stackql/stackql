package taxonomy

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/httpbuild"
	"github.com/stackql/stackql/internal/stackql/provider"

	"github.com/stackql/go-openapistackql/openapistackql"
)

type ExtendedTableMetadata struct {
	TableFilter         func(openapistackql.ITable) (openapistackql.ITable, error)
	ColsVisited         map[string]bool
	HeirarchyObjects    *HeirarchyObjects
	RequiredParameters  map[string]openapistackql.Parameter
	IsLocallyExecutable bool
	GetHttpArmoury      func() (httpbuild.HTTPArmoury, error)
	SelectItemsKey      string
	Alias               string
}

func (ex ExtendedTableMetadata) LookupSelectItemsKey() string {
	if ex.HeirarchyObjects == nil {
		return defaultSelectItemsKey
	}
	return ex.HeirarchyObjects.LookupSelectItemsKey()
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

func (ex ExtendedTableMetadata) GetRequestSchema() (*openapistackql.Schema, error) {
	return ex.HeirarchyObjects.GetRequestSchema()
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
