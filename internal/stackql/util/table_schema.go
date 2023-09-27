package util

import (
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stackql/go-openapistackql/openapistackql"
)

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

//nolint:gocognit,nestif // tactical
func (ta *simpleTableSchemaAnalyzer) GetColumns() ([]Column, error) {
	var rv []Column
	tableColumns := ta.s.Tabulate(false).GetColumns()
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
	servers := ta.m.GetServers()
	if servers != nil && len(*servers) > 0 {
		for _, srv := range *servers {
			for k := range srv.Variables {
				if _, ok := existingColumns[k]; ok {
					continue
				}
				existingColumns[k] = struct{}{}
				rv = append(rv, newSimpleStringColumn(k, ta.m))
			}
		}
	} else {
		svc := ta.m.GetService()
		if svc != nil {
			svcServers := svc.GetServers()
			if len(svcServers) > 0 {
				for _, srv := range svcServers {
					for k := range srv.Variables {
						if _, ok := existingColumns[k]; ok {
							continue
						}
						existingColumns[k] = struct{}{}
						rv = append(rv, newSimpleStringColumn(k, ta.m))
					}
				}
			}
		}
	}
	return rv, nil
}

func (ta *simpleTableSchemaAnalyzer) generateServerVarColumnDescriptor(
	k string, m openapistackql.OperationStore) openapistackql.ColumnDescriptor {
	sc := openapi3.NewSchema()
	sc.Type = "string"
	schema := openapistackql.NewSchema(
		sc,
		m.GetService(),
		"",
		"",
	)
	colDesc := openapistackql.NewColumnDescriptor(
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

//nolint:gocognit,nestif // tactical
func (ta *simpleTableSchemaAnalyzer) GetColumnDescriptors(
	tabAnnotated AnnotatedTabulation,
) ([]openapistackql.ColumnDescriptor, error) {
	existingColumns := make(map[string]struct{})
	var rv []openapistackql.ColumnDescriptor
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
	servers := ta.m.GetServers()
	if servers != nil && len(*servers) > 0 {
		for _, srv := range *servers {
			for k := range srv.Variables {
				if _, ok := existingColumns[k]; ok {
					continue
				}
				existingColumns[k] = struct{}{}
				colDesc := ta.generateServerVarColumnDescriptor(k, ta.m)
				rv = append(rv, colDesc)
			}
		}
	} else {
		svc := ta.m.GetService()
		if svc != nil {
			svcServers := svc.GetServers()
			if len(svcServers) > 0 {
				for _, srv := range svcServers {
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
		}
	}
	return rv, nil
}
