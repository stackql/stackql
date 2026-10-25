package tablemetadata

import (
	"github.com/stackql/stackql/internal/stackql/datasource/sql_datasource"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
)

var (
	_ HeirarchyObjects = &standardHeirarchyObjects{}
)

type HeirarchyObjects interface {
	GetHeirarchyIDs() internaldto.HeirarchyIdentifiers

	GetSQLDataSource() (sql_datasource.SQLDataSource, bool)
	GetTableName() string
	GetSubquery() (internaldto.SubqueryDTO, bool)
	GetView() (internaldto.RelationDTO, bool)

	SetSQLDataSource(sql_datasource.SQLDataSource)
	// De facto inheritance

	IsPGInternalObject() bool
	SetIndirect(internaldto.RelationDTO)
	GetIndirect() (internaldto.RelationDTO, bool)
	IsAwait() bool
}

func NewHeirarchyObjects(hIDs internaldto.HeirarchyIdentifiers, isAwait bool) HeirarchyObjects {
	return &standardHeirarchyObjects{
		heirarchyIDs: hIDs,
		isAwait:      isAwait,
	}
}

type standardHeirarchyObjects struct {
	heirarchyIDs  internaldto.HeirarchyIdentifiers
	sqlDataSource sql_datasource.SQLDataSource
	indirect      internaldto.RelationDTO
	isAwait       bool
}

func (ho *standardHeirarchyObjects) IsAwait() bool {
	return ho.isAwait
}

func (ho *standardHeirarchyObjects) GetIndirect() (internaldto.RelationDTO, bool) {
	return ho.indirect, ho.indirect != nil
}

func (ho *standardHeirarchyObjects) SetIndirect(indirect internaldto.RelationDTO) {
	ho.indirect = indirect
}

func (ho *standardHeirarchyObjects) IsPGInternalObject() bool {
	return ho.heirarchyIDs.IsPgInternalObject()
}

func (ho *standardHeirarchyObjects) GetSQLDataSource() (sql_datasource.SQLDataSource, bool) {
	return ho.sqlDataSource, ho.sqlDataSource != nil
}

func (ho *standardHeirarchyObjects) SetSQLDataSource(sqlDataSource sql_datasource.SQLDataSource) {
	ho.sqlDataSource = sqlDataSource
}

func (ho *standardHeirarchyObjects) GetView() (internaldto.RelationDTO, bool) {
	return ho.heirarchyIDs.GetView()
}

func (ho *standardHeirarchyObjects) GetSubquery() (internaldto.SubqueryDTO, bool) {
	return ho.heirarchyIDs.GetSubquery()
}

func (ho *standardHeirarchyObjects) GetHeirarchyIDs() internaldto.HeirarchyIdentifiers {
	return ho.heirarchyIDs
}

func (ho *standardHeirarchyObjects) GetTableName() string {
	return ho.heirarchyIDs.GetTableName()
}
