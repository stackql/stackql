package util

import "github.com/stackql/go-openapistackql/openapistackql"

type TableSchemaAnalyzer interface {
	GetColumns() []Column
	GetColumnDescriptors(AnnotatedTabulation) []openapistackql.ColumnDescriptor
}

type simpleTableSchemaAnalyzer struct {
	s *openapistackql.Schema
	m *openapistackql.OperationStore
}

func NewTableSchemaAnalyzer(s *openapistackql.Schema, m *openapistackql.OperationStore) TableSchemaAnalyzer {
	return &simpleTableSchemaAnalyzer{
		s: s,
		m: m,
	}
}

func (ta *simpleTableSchemaAnalyzer) GetColumns() []Column {
	var rv []Column
	tableColumns := ta.s.Tabulate(false).GetColumns()
	existingColumns := make(map[string]struct{})
	for _, col := range tableColumns {
		existingColumns[col.Name] = struct{}{}
		rv = append(rv, newSimpleColumn(col.Name, col.Schema))
	}
	for k, col := range ta.m.GetRequiredParameters() {
		if _, ok := existingColumns[k]; ok {
			continue
		}
		schema, _ := col.GetSchema()
		rv = append(rv, newSimpleColumn(k, schema))
	}
	return rv
}

func (ta *simpleTableSchemaAnalyzer) GetColumnDescriptors(tabAnnotated AnnotatedTabulation) []openapistackql.ColumnDescriptor {
	existingColumns := make(map[string]struct{})
	var rv []openapistackql.ColumnDescriptor
	for _, col := range tabAnnotated.GetTabulation().GetColumns() {
		colName := col.Name
		existingColumns[colName] = struct{}{}
		rv = append(rv, col)
	}
	for k, col := range ta.m.GetRequiredParameters() {
		if _, ok := existingColumns[k]; ok {
			continue
		}
		schema, _ := col.GetSchema()
		colDesc := openapistackql.NewColumnDescriptor(
			"",
			k,
			"",
			schema,
			nil,
		)
		rv = append(rv, colDesc)
	}
	return rv
}
