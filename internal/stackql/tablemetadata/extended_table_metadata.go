package tablemetadata

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/stackql/internal/stackql/astindirect"
	"github.com/stackql/stackql/internal/stackql/datasource/sql_datasource"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/provider"
)

var (
	_ ExtendedTableMetadata = &standardExtendedTableMetadata{}
)

type ExtendedTableMetadata interface {
	GetAlias() string
	GetGraphQL() (anysdk.GraphQL, bool)
	GetHeirarchyObjects() HeirarchyObjects
	GetHTTPArmoury() (anysdk.HTTPArmoury, error)
	GetInputTableName() (string, error)
	GetMethod() (anysdk.OperationStore, error)
	GetMethodStr() (string, error)
	GetProvider() (provider.IProvider, error)
	GetProviderStr() (string, error)
	GetProviderObject() (anysdk.Provider, error)
	GetQueryUniqueID() string
	GetRequestSchema() (anysdk.Schema, error)
	GetOptionalParameters() map[string]anysdk.Addressable
	GetRequiredParameters() map[string]anysdk.Addressable
	GetResource() (anysdk.Resource, error)
	GetResourceStr() (string, error)
	GetResponseSchemaStr() (string, error)
	GetResponseSchemaAndMediaType() (anysdk.Schema, string, error)
	GetSelectableObjectSchema() (anysdk.Schema, error)
	GetSelectItemsKey() string
	GetSelectSchemaAndObjectPath() (anysdk.Schema, string, error)
	GetService() (anysdk.Service, error)
	GetServiceStr() (string, error)
	GetSQLDataSource() (sql_datasource.SQLDataSource, bool)
	GetStackQLTableName() (string, error)
	GetTableFilter() func(anysdk.ITable) (anysdk.ITable, error)
	GetTableName() (string, error)
	GetUniqueID() string
	IsLocallyExecutable() bool
	IsSimple() bool
	GetIndirect() (astindirect.Indirect, bool)
	GetView() (internaldto.RelationDTO, bool)
	GetSubquery() (internaldto.SubqueryDTO, bool)
	LookupSelectItemsKey() string
	SetSelectItemsKey(string)
	SetSQLDataSource(sql_datasource.SQLDataSource)
	SetTableFilter(f func(anysdk.ITable) (anysdk.ITable, error))
	WithGetHTTPArmoury(f func() (anysdk.HTTPArmoury, error)) ExtendedTableMetadata
	WithIndirect(astindirect.Indirect) ExtendedTableMetadata
	WithResponseSchemaStr(rss string) (ExtendedTableMetadata, error)
	IsPGInternalObject() bool
	SetIsOnClauseHoistable(bool)
	IsOnClauseHoistable() bool
	IsPhysicalTable() bool
	IsMaterializedView() bool
	GetServerVariables() (map[string]*openapi3.ServerVariable, bool)
	Clone() ExtendedTableMetadata
	Equals(ExtendedTableMetadata) bool
}

type standardExtendedTableMetadata struct {
	tableFilter         func(anysdk.ITable) (anysdk.ITable, error)
	colsVisited         map[string]bool
	heirarchyObjects    HeirarchyObjects
	isLocallyExecutable bool
	getHTTPArmoury      func() (anysdk.HTTPArmoury, error)
	selectItemsKey      string
	alias               string
	inputTableName      string
	indirect            astindirect.Indirect
	sqlDataSource       sql_datasource.SQLDataSource
	isOnClauseHoistable bool
}

func (ex *standardExtendedTableMetadata) Clone() ExtendedTableMetadata {
	return &standardExtendedTableMetadata{
		tableFilter:         ex.tableFilter,
		colsVisited:         ex.colsVisited,
		heirarchyObjects:    ex.heirarchyObjects,
		isLocallyExecutable: ex.isLocallyExecutable,
		getHTTPArmoury:      ex.getHTTPArmoury,
		selectItemsKey:      ex.selectItemsKey,
		alias:               ex.alias,
		inputTableName:      ex.inputTableName,
		indirect:            ex.indirect,
		sqlDataSource:       ex.sqlDataSource,
		isOnClauseHoistable: ex.isOnClauseHoistable,
	}
}

func (ex *standardExtendedTableMetadata) Equals(other ExtendedTableMetadata) bool {
	otherStandard, isStandard := other.(*standardExtendedTableMetadata)
	if !isStandard {
		return false
	}
	if ex.heirarchyObjects != otherStandard.heirarchyObjects {
		return false
	}
	if ex.isLocallyExecutable != otherStandard.isLocallyExecutable {
		return false
	}
	if ex.selectItemsKey != otherStandard.selectItemsKey {
		return false
	}
	if ex.alias != otherStandard.alias {
		return false
	}
	if ex.inputTableName != otherStandard.inputTableName {
		return false
	}
	if ex.indirect != otherStandard.indirect {
		return false
	}
	if ex.sqlDataSource != otherStandard.sqlDataSource {
		return false
	}
	if ex.isOnClauseHoistable != otherStandard.isOnClauseHoistable {
		return false
	}
	return true
}

func (ex *standardExtendedTableMetadata) GetServerVariables() (map[string]*openapi3.ServerVariable, bool) {
	m, err := ex.getMethod()
	if err != nil {
		return nil, false
	}
	rv := make(map[string]*openapi3.ServerVariable)
	availableServers, availableServersDoExist := m.GetServers()
	if availableServersDoExist {
		for _, s := range availableServers {
			for k, v := range s.Variables {
				rv[k] = v
			}
		}
	}
	return rv, true
}

func (ex *standardExtendedTableMetadata) IsPhysicalTable() bool {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetHeirarchyIds() == nil {
		return false
	}
	return ex.heirarchyObjects.GetHeirarchyIds().IsPhysicalTable()
}

func (ex *standardExtendedTableMetadata) IsMaterializedView() bool {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetHeirarchyIds() == nil {
		return false
	}
	return ex.heirarchyObjects.GetHeirarchyIds().IsMaterializedView()
}

func (ex *standardExtendedTableMetadata) SetIsOnClauseHoistable(isOnClauseHoistable bool) {
	ex.isOnClauseHoistable = isOnClauseHoistable
}

func (ex *standardExtendedTableMetadata) IsOnClauseHoistable() bool {
	return ex.isOnClauseHoistable
}

func (ex *standardExtendedTableMetadata) IsPGInternalObject() bool {
	return ex.heirarchyObjects.IsPGInternalObject()
}

func (ex *standardExtendedTableMetadata) IsLocallyExecutable() bool {
	return ex.isLocallyExecutable
}

func (ex *standardExtendedTableMetadata) WithIndirect(indirect astindirect.Indirect) ExtendedTableMetadata {
	ex.indirect = indirect
	return ex
}

func (ex *standardExtendedTableMetadata) GetSQLDataSource() (sql_datasource.SQLDataSource, bool) {
	return ex.heirarchyObjects.GetSQLDataSource()
}

func (ex *standardExtendedTableMetadata) SetSQLDataSource(sqlDataSource sql_datasource.SQLDataSource) {
	ex.sqlDataSource = sqlDataSource
}

func (ex *standardExtendedTableMetadata) GetIndirect() (astindirect.Indirect, bool) {
	return ex.indirect, ex.indirect != nil
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

func (ex *standardExtendedTableMetadata) SetTableFilter(f func(anysdk.ITable) (anysdk.ITable, error)) {
	ex.tableFilter = f
}

func (ex *standardExtendedTableMetadata) GetTableFilter() func(anysdk.ITable) (anysdk.ITable, error) {
	return ex.tableFilter
}

func (ex *standardExtendedTableMetadata) GetGraphQL() (anysdk.GraphQL, bool) {
	if ex.heirarchyObjects.GetMethod() != nil && ex.heirarchyObjects.GetMethod().GetGraphQL() != nil {
		return ex.heirarchyObjects.GetMethod().GetGraphQL(), true
	}
	return nil, false
}

func (ex *standardExtendedTableMetadata) GetRequiredParameters() map[string]anysdk.Addressable {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetMethod() == nil {
		return nil
	}
	return ex.heirarchyObjects.GetMethod().GetRequiredParameters()
}

func (ex *standardExtendedTableMetadata) GetOptionalParameters() map[string]anysdk.Addressable {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetMethod() == nil {
		return nil
	}
	return ex.heirarchyObjects.GetMethod().GetOptionalParameters()
}

func (ex *standardExtendedTableMetadata) GetHTTPArmoury() (anysdk.HTTPArmoury, error) {
	if ex.getHTTPArmoury == nil {
		return nil, fmt.Errorf("nil getHttpAroury() function in ExtendedTableMetadata object")
	}
	return ex.getHTTPArmoury()
}

func (ex *standardExtendedTableMetadata) WithGetHTTPArmoury(
	f func() (anysdk.HTTPArmoury, error),
) ExtendedTableMetadata {
	ex.getHTTPArmoury = f
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

func (ex *standardExtendedTableMetadata) GetSubquery() (internaldto.SubqueryDTO, bool) {
	return ex.heirarchyObjects.GetSubquery()
}

func (ex *standardExtendedTableMetadata) GetView() (internaldto.RelationDTO, bool) {
	return ex.heirarchyObjects.GetView()
}

func (ex *standardExtendedTableMetadata) isSimple() bool {
	//nolint:lll // complex boolean
	return ex.heirarchyObjects != nil && (len(ex.heirarchyObjects.GetMethodSet()) > 0 || ex.heirarchyObjects.GetMethod() != nil)
}

func (ex *standardExtendedTableMetadata) GetUniqueID() string {
	if ex.alias != "" {
		return ex.alias
	}
	return ex.heirarchyObjects.GetTableName()
}

func (ex *standardExtendedTableMetadata) GetQueryUniqueID() string {
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

func (ex *standardExtendedTableMetadata) GetProviderObject() (anysdk.Provider, error) {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetProvider() == nil {
		return nil, fmt.Errorf("cannot resolve Provider object")
	}
	return ex.heirarchyObjects.GetProvider().GetProvider()
}

func (ex *standardExtendedTableMetadata) GetService() (anysdk.Service, error) {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetServiceHdl() == nil {
		return nil, fmt.Errorf("cannot resolve ServiceHandle")
	}
	return ex.heirarchyObjects.GetServiceHdl(), nil
}

func (ex *standardExtendedTableMetadata) GetResource() (anysdk.Resource, error) {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetResource() == nil {
		return nil, fmt.Errorf("cannot resolve Resource")
	}
	return ex.heirarchyObjects.GetResource(), nil
}

func (ex *standardExtendedTableMetadata) GetMethod() (anysdk.OperationStore, error) {
	return ex.getMethod()
}

func (ex *standardExtendedTableMetadata) getMethod() (anysdk.OperationStore, error) {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetMethod() == nil {
		return nil, fmt.Errorf("cannot resolve Method")
	}
	return ex.heirarchyObjects.GetMethod(), nil
}

func (ex *standardExtendedTableMetadata) GetSelectSchemaAndObjectPath() (anysdk.Schema, string, error) {
	return ex.heirarchyObjects.GetSelectSchemaAndObjectPath()
}

func (ex *standardExtendedTableMetadata) GetResponseSchemaAndMediaType() (anysdk.Schema, string, error) {
	if ex.isSimple() {
		return ex.heirarchyObjects.GetResponseSchemaAndMediaType()
	}
	return nil, "", fmt.Errorf("error extracting response schema and media type: views not yet supported")
}

func (ex *standardExtendedTableMetadata) GetRequestSchema() (anysdk.Schema, error) {
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

func (ex *standardExtendedTableMetadata) GetSelectableObjectSchema() (anysdk.Schema, error) {
	return ex.heirarchyObjects.GetSelectableObjectSchema()
}

func NewExtendedTableMetadata(heirarchyObjects HeirarchyObjects, tableName string, alias string) ExtendedTableMetadata {
	return &standardExtendedTableMetadata{
		colsVisited:      make(map[string]bool),
		heirarchyObjects: heirarchyObjects,
		alias:            alias,
		inputTableName:   tableName,
	}
}
