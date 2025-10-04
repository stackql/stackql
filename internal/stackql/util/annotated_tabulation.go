package util //nolint:revive // fine for now

import (
	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/stackql/internal/stackql/datasource/sql_datasource"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
)

type AnnotatedTabulation interface {
	GetAlias() string
	GetHeirarchyIdentifiers() internaldto.HeirarchyIdentifiers
	GetInputTableName() string
	GetSQLDataSource() (sql_datasource.SQLDataSource, bool)
	GetTabulation() anysdk.Tabulation
	SetSQLDataSource(sql_datasource.SQLDataSource)
	WithParameters(parameters map[string]interface{}) AnnotatedTabulation
	GetParameters() map[string]interface{}
}

type standardAnnotatedTabulation struct {
	tab            anysdk.Tabulation
	hIDs           internaldto.HeirarchyIdentifiers
	inputTableName string
	alias          string
	sqlDataSource  sql_datasource.SQLDataSource
	parameters     map[string]interface{}
}

func NewAnnotatedTabulation(
	tab anysdk.Tabulation,
	hIDs internaldto.HeirarchyIdentifiers,
	inputTableName string,
	alias string,
) AnnotatedTabulation {
	return &standardAnnotatedTabulation{
		tab:            tab,
		hIDs:           hIDs,
		inputTableName: inputTableName,
		alias:          alias,
		parameters:     make(map[string]interface{}),
	}
}

func (at *standardAnnotatedTabulation) WithParameters(parameters map[string]interface{}) AnnotatedTabulation {
	at.parameters = parameters
	return at
}

func (at *standardAnnotatedTabulation) GetParameters() map[string]interface{} {
	if at.parameters == nil {
		return make(map[string]interface{})
	}
	return at.parameters
}

func (at *standardAnnotatedTabulation) GetTabulation() anysdk.Tabulation {
	return at.tab
}

func (at *standardAnnotatedTabulation) GetAlias() string {
	return at.alias
}

func (at *standardAnnotatedTabulation) GetInputTableName() string {
	return at.inputTableName
}

func (at *standardAnnotatedTabulation) GetHeirarchyIdentifiers() internaldto.HeirarchyIdentifiers {
	return at.hIDs
}

func (at *standardAnnotatedTabulation) SetSQLDataSource(sqlDataSource sql_datasource.SQLDataSource) {
	at.sqlDataSource = sqlDataSource
}

func (at *standardAnnotatedTabulation) GetSQLDataSource() (sql_datasource.SQLDataSource, bool) {
	return at.sqlDataSource, at.sqlDataSource != nil
}
