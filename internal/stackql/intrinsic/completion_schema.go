package intrinsic

import (
	"maps"
	"slices"

	"github.com/stackql-labs/omnisdk/pkg/docparse/stackqldoc"
	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql-labs/omnisdk/pkg/query"
)

type completionSchemaTable struct {
	omnisdk.Table
	columns []string
	types   map[string]omnisdk.ColumnType
}

func withCompletionSchema(target, resource omnisdk.Table) omnisdk.Table {
	columns := map[string]bool{}
	types := map[string]omnisdk.ColumnType{}
	for _, method := range resource.Methods() {
		for _, name := range method.Columns() {
			columns[name] = true
		}
		maps.Copy(types, method.ColumnTypes())
	}
	return &completionSchemaTable{Table: target, columns: slices.Sorted(maps.Keys(columns)), types: types}
}
func (t *completionSchemaTable) Methods() []omnisdk.MethodSignature {
	methods := t.Table.Methods()
	out := make([]omnisdk.MethodSignature, len(methods))
	for i, method := range methods {
		out[i] = newCompletedMethod(method, t.columns, t.types)
	}
	return out
}

type completedMethod struct {
	omnisdk.MethodSignature
	columns []string
	types   map[string]omnisdk.ColumnType
}

func newCompletedMethod(
	base omnisdk.MethodSignature, columns []string, types map[string]omnisdk.ColumnType,
) omnisdk.MethodSignature {
	return &completedMethod{MethodSignature: base, columns: slices.Clone(columns), types: maps.Clone(types)}
}
func (m *completedMethod) Columns() []string                          { return slices.Clone(m.columns) }
func (m *completedMethod) ColumnTypes() map[string]omnisdk.ColumnType { return maps.Clone(m.types) }
func (m *completedMethod) ServerEnvironmentBindings() map[string]string {
	if metadata, ok := m.MethodSignature.(omnisdk.ServerEnvironmentBindings); ok {
		return metadata.ServerEnvironmentBindings()
	}
	return nil
}

func describeMutationTable(
	registry string, target query.Target, await bool, opts ...stackqldoc.Option) (omnisdk.Table, error) {
	table, err := omnisdk.DescribeMutation(registry, target.Resource().Handle(), target.Verb().String(), opts...)
	if err != nil || !await {
		return table, err
	}
	resource, err := omnisdk.DescribeTable(registry, target.Resource().Handle())
	if err != nil {
		return nil, err
	}
	return withCompletionSchema(table, resource), nil
}

func (m *completedMethod) ServerTransportDefaults() map[string]string {
	if metadata, ok := m.MethodSignature.(omnisdk.ServerTransportDefaults); ok {
		return metadata.ServerTransportDefaults()
	}
	return nil
}
