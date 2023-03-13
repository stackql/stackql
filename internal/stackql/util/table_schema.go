package util

import "github.com/stackql/go-openapistackql/openapistackql"

type TableSchemaAnalyzer interface {
	GetColumns() ([]Column, error)
	GetColumnDescriptors(AnnotatedTabulation) ([]openapistackql.ColumnDescriptor, error)
}

type simpleTableSchemaAnalyzer struct {
	s openapistackql.Schema
	m openapistackql.OperationStore
}

func NewTableSchemaAnalyzer(s openapistackql.Schema, m openapistackql.OperationStore) TableSchemaAnalyzer {
	return &simpleTableSchemaAnalyzer{
		s: s,
		m: m,
	}
}

func (ta *simpleTableSchemaAnalyzer) GetColumns() ([]Column, error) {
	var rv []Column
	tableColumns := ta.s.Tabulate(false).GetColumns()
	existingColumns := make(map[string]struct{})
	for _, col := range tableColumns {
		existingColumns[col.Name] = struct{}{}
		rv = append(rv, newSimpleColumn(col.Name, col.Schema))
	}
	unionedRequiredParams, err := ta.m.GetUnionRequiredParameters()
	if err != nil {
		return nil, err
	}
	for k, col := range unionedRequiredParams {
		if _, ok := existingColumns[k]; ok {
			continue
		}
		schema, _ := col.GetSchema()
		rv = append(rv, newSimpleColumn(k, schema))
	}
	return rv, nil
}

func (ta *simpleTableSchemaAnalyzer) GetColumnDescriptors(tabAnnotated AnnotatedTabulation) ([]openapistackql.ColumnDescriptor, error) {
	existingColumns := make(map[string]struct{})
	var rv []openapistackql.ColumnDescriptor
	for _, col := range tabAnnotated.GetTabulation().GetColumns() {
		colName := col.Name
		existingColumns[colName] = struct{}{}
		rv = append(rv, col)
	}
	unionedRequiredParams, err := ta.m.GetUnionRequiredParameters()
	if err != nil {
		return nil, err
	}
	for k, col := range unionedRequiredParams {
		if _, ok := existingColumns[k]; ok {
			continue
		}
		schema, _ := col.GetSchema()
		colDesc := openapistackql.NewColumnDescriptor(
			"",
			k,
			"",
			"",
			nil,
			schema,
			nil,
		)
		rv = append(rv, colDesc)
	}
	return rv, nil
}
