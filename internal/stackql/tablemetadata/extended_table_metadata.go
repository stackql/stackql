package tablemetadata

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/astindirect"
	"github.com/stackql/stackql/internal/stackql/datasource/sql_datasource"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
)

var (
	_ ExtendedTableMetadata = &standardExtendedTableMetadata{}
)

type ExtendedTableMetadata interface {
	GetAlias() string

	GetHeirarchyObjects() HeirarchyObjects
	GetInputTableName() (string, error)

	GetMethodStr() (string, error)

	GetProviderStr() (string, error)

	GetQueryUniqueID() string

	GetResourceStr() (string, error)
	GetResponseSchemaStr() (string, error)

	GetSelectItemsKey() string

	GetServiceStr() (string, error)
	GetSQLDataSource() (sql_datasource.SQLDataSource, bool)
	GetStackQLTableName() (string, error)

	GetTableName() (string, error)
	GetUniqueID() string
	IsLocallyExecutable() bool

	GetIndirect() (astindirect.Indirect, bool)
	GetView() (internaldto.RelationDTO, bool)
	GetSubquery() (internaldto.SubqueryDTO, bool)

	SetSelectItemsKey(string)
	SetSQLDataSource(sql_datasource.SQLDataSource)

	WithIndirect(astindirect.Indirect) ExtendedTableMetadata
	WithResponseSchemaStr(rss string) (ExtendedTableMetadata, error)
	IsPGInternalObject() bool
	SetIsOnClauseHoistable(bool)
	IsOnClauseHoistable() bool
	IsPhysicalTable() bool
	IsMaterializedView() bool
	Clone() ExtendedTableMetadata
	Equals(ExtendedTableMetadata) bool
}

type standardExtendedTableMetadata struct {
	colsVisited         map[string]bool
	heirarchyObjects    HeirarchyObjects
	isLocallyExecutable bool
	selectItemsKey      string
	alias               string
	inputTableName      string
	indirect            astindirect.Indirect
	sqlDataSource       sql_datasource.SQLDataSource
	isOnClauseHoistable bool
}

func (ex *standardExtendedTableMetadata) Clone() ExtendedTableMetadata {
	return &standardExtendedTableMetadata{
		colsVisited:         ex.colsVisited,
		heirarchyObjects:    ex.heirarchyObjects,
		isLocallyExecutable: ex.isLocallyExecutable,
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

func (ex *standardExtendedTableMetadata) IsPhysicalTable() bool {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetHeirarchyIDs() == nil {
		return false
	}
	return ex.heirarchyObjects.GetHeirarchyIDs().IsPhysicalTable()
}

func (ex *standardExtendedTableMetadata) IsMaterializedView() bool {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetHeirarchyIDs() == nil {
		return false
	}
	return ex.heirarchyObjects.GetHeirarchyIDs().IsMaterializedView()
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

func (ex *standardExtendedTableMetadata) GetAlias() string {
	return ex.alias
}

func (ex *standardExtendedTableMetadata) GetSubquery() (internaldto.SubqueryDTO, bool) {
	return ex.heirarchyObjects.GetSubquery()
}

func (ex *standardExtendedTableMetadata) GetView() (internaldto.RelationDTO, bool) {
	return ex.heirarchyObjects.GetView()
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

func (ex *standardExtendedTableMetadata) GetServiceStr() (string, error) {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetHeirarchyIDs().GetServiceStr() == "" {
		return "", fmt.Errorf("cannot resolve ServiceStr")
	}
	return ex.heirarchyObjects.GetHeirarchyIDs().GetServiceStr(), nil
}

func (ex *standardExtendedTableMetadata) GetResourceStr() (string, error) {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetHeirarchyIDs().GetResourceStr() == "" {
		return "", fmt.Errorf("cannot resolve ResourceStr")
	}
	return ex.heirarchyObjects.GetHeirarchyIDs().GetResourceStr(), nil
}

func (ex *standardExtendedTableMetadata) GetProviderStr() (string, error) {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetHeirarchyIDs().GetProviderStr() == "" {
		return "", fmt.Errorf("cannot resolve ProviderStr")
	}
	return ex.heirarchyObjects.GetHeirarchyIDs().GetProviderStr(), nil
}

func (ex *standardExtendedTableMetadata) GetMethodStr() (string, error) {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetHeirarchyIDs().GetMethodStr() == "" {
		return "", fmt.Errorf("cannot resolve MethodStr")
	}
	return ex.heirarchyObjects.GetHeirarchyIDs().GetMethodStr(), nil
}

func (ex *standardExtendedTableMetadata) GetResponseSchemaStr() (string, error) {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetHeirarchyIDs().GetResponseSchemaStr() == "" {
		return "", fmt.Errorf("cannot resolve ResponseSchemaStr")
	}
	return ex.heirarchyObjects.GetHeirarchyIDs().GetResponseSchemaStr(), nil
}

func (ex *standardExtendedTableMetadata) WithResponseSchemaStr(rss string) (ExtendedTableMetadata, error) {
	if ex.heirarchyObjects == nil {
		return ex, fmt.Errorf("standardExtendedTableMetadata.WithResponseSchemaStr(): cannot resolve HeirarchyObjects")
	}
	ex.heirarchyObjects.GetHeirarchyIDs().WithResponseSchemaStr(rss)
	return ex, nil
}

func (ex *standardExtendedTableMetadata) GetTableName() (string, error) {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetHeirarchyIDs().GetTableName() == "" {
		return "", fmt.Errorf("cannot resolve TableName")
	}
	return ex.heirarchyObjects.GetHeirarchyIDs().GetTableName(), nil
}

func (ex *standardExtendedTableMetadata) GetStackQLTableName() (string, error) {
	if ex.heirarchyObjects == nil || ex.heirarchyObjects.GetHeirarchyIDs().GetTableName() == "" {
		return "", fmt.Errorf("cannot resolve TableName")
	}
	return ex.heirarchyObjects.GetHeirarchyIDs().GetStackQLTableName(), nil
}

func (ex *standardExtendedTableMetadata) GetInputTableName() (string, error) {
	return ex.inputTableName, nil
}

func NewExtendedTableMetadata(heirarchyObjects HeirarchyObjects, tableName string, alias string) ExtendedTableMetadata {
	return &standardExtendedTableMetadata{
		colsVisited:      make(map[string]bool),
		heirarchyObjects: heirarchyObjects,
		alias:            alias,
		inputTableName:   tableName,
	}
}
