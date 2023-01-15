package util

import (
	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql/internal/stackql/datasource/sql_datasource"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
)

type AnnotatedTabulation interface {
	GetAlias() string
	GetHeirarchyIdentifiers() internaldto.HeirarchyIdentifiers
	GetInputTableName() string
	GetSQLDataSource() (sql_datasource.SQLDataSource, bool)
	GetTabulation() *openapistackql.Tabulation
	SetSQLDataSource(sql_datasource.SQLDataSource)
}

type standardAnnotatedTabulation struct {
	tab            *openapistackql.Tabulation
	hIds           internaldto.HeirarchyIdentifiers
	inputTableName string
	alias          string
	sqlDataSource  sql_datasource.SQLDataSource
}

func NewAnnotatedTabulation(tab *openapistackql.Tabulation, hIds internaldto.HeirarchyIdentifiers, inputTableName string, alias string) AnnotatedTabulation {
	return &standardAnnotatedTabulation{
		tab:            tab,
		hIds:           hIds,
		inputTableName: inputTableName,
		alias:          alias,
	}
}

func (at *standardAnnotatedTabulation) GetTabulation() *openapistackql.Tabulation {
	return at.tab
}

func (at *standardAnnotatedTabulation) GetAlias() string {
	return at.alias
}

func (at *standardAnnotatedTabulation) GetInputTableName() string {
	return at.inputTableName
}

func (at *standardAnnotatedTabulation) GetHeirarchyIdentifiers() internaldto.HeirarchyIdentifiers {
	return at.hIds
}

func (at *standardAnnotatedTabulation) SetSQLDataSource(sqlDataSource sql_datasource.SQLDataSource) {
	at.sqlDataSource = sqlDataSource
}

func (at *standardAnnotatedTabulation) GetSQLDataSource() (sql_datasource.SQLDataSource, bool) {
	return at.sqlDataSource, at.sqlDataSource != nil
}
