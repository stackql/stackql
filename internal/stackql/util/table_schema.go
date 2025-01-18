package util

import (
	"strings"

	"github.com/stackql/any-sdk/anysdk"
)

type TableSchemaAnalyzer interface {
	GetColumns() ([]Column, error)
	GetColumnDescriptors(AnnotatedTabulation) ([]anysdk.ColumnDescriptor, error)
}

type simpleTableSchemaAnalyzer struct {
	s anysdk.Schema
	m anysdk.OperationStore
}

func NewTableSchemaAnalyzer(s anysdk.Schema, m anysdk.OperationStore) TableSchemaAnalyzer {
	return &simpleTableSchemaAnalyzer{
		s: s,
		m: m,
	}
}

func TrimSelectItemsKey(selectItemsKey string) string {
	splitSet := strings.Split(selectItemsKey, "/")
	if len(splitSet) == 0 {
		return ""
	}
	return splitSet[len(splitSet)-1]
}

func (ta *simpleTableSchemaAnalyzer) GetColumns() ([]Column, error) {
	var rv []Column
	tableColumns := ta.s.Tabulate(false, "").GetColumns()
	existingColumns := make(map[string]struct{})
	for _, col := range tableColumns {
		existingColumns[col.GetName()] = struct{}{}
		rv = append(rv, newSimpleColumn(col.GetName(), col.GetSchema()))
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
		existingColumns[col.GetName()] = struct{}{}
		rv = append(rv, newSimpleColumn(k, schema))
	}
	servers, serversDoExist := ta.m.GetServers()
	if serversDoExist {
		for _, srv := range servers {
			for k := range srv.Variables {
				if _, ok := existingColumns[k]; ok {
					continue
				}
				existingColumns[k] = struct{}{}
				rv = append(rv, newSimpleStringColumn(k, ta.m))
			}
		}
	}
	return rv, nil
}

func (ta *simpleTableSchemaAnalyzer) generateServerVarColumnDescriptor(
	k string, m anysdk.OperationStore) anysdk.ColumnDescriptor {
	schema := anysdk.NewStringSchema(
		m.GetService(),
		"",
		"",
	)
	colDesc := anysdk.NewColumnDescriptor(
		"",
		k,
		"",
		"",
		nil,
		schema,
		nil,
	)
	return colDesc
}

func (ta *simpleTableSchemaAnalyzer) GetColumnDescriptors(
	tabAnnotated AnnotatedTabulation,
) ([]anysdk.ColumnDescriptor, error) {
	existingColumns := make(map[string]struct{})
	var rv []anysdk.ColumnDescriptor
	for _, col := range tabAnnotated.GetTabulation().GetColumns() {
		colName := col.GetName()
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
		existingColumns[k] = struct{}{}
		schema, _ := col.GetSchema()
		colDesc := anysdk.NewColumnDescriptor(
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
	servers, serversDoExist := ta.m.GetServers()
	if serversDoExist {
		for _, srv := range servers {
			for k := range srv.Variables {
				if _, ok := existingColumns[k]; ok {
					continue
				}
				existingColumns[k] = struct{}{}
				colDesc := ta.generateServerVarColumnDescriptor(k, ta.m)
				rv = append(rv, colDesc)
			}
		}
	}
	return rv, nil
}
