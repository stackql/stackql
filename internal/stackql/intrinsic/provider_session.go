package intrinsic

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/stackql-labs/omnisdk/pkg/docparse/stackqldoc"
	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql-labs/omnisdk/pkg/query"
)

func providerSessionArgs(ctx queryContext) (omnisdk.Args, error) {
	if host, ok := ctx.(interface {
		ProviderSessionArgs(context.Context) (omnisdk.Args, error)
	}); ok {
		return host.ProviderSessionArgs(executionContext(ctx))
	}
	return omnisdk.Args{}, nil
}

func providerAddress(ctx queryContext, address string) string {
	if host, ok := ctx.(interface{ ResolveProviderAddress(string) string }); ok {
		return host.ResolveProviderAddress(address)
	}
	return address
}

type aliasedProviderTable struct {
	omnisdk.Table
	address string
}

func (table aliasedProviderTable) Address() string { return table.address }
func (table aliasedProviderTable) ServerEnvironmentBindings() map[string]string {
	if metadata, ok := table.Table.(omnisdk.ServerEnvironmentBindings); ok {
		return metadata.ServerEnvironmentBindings()
	}
	return nil
}
func (table aliasedProviderTable) ServerTransportDefaults() map[string]string {
	if metadata, ok := table.Table.(omnisdk.ServerTransportDefaults); ok {
		return metadata.ServerTransportDefaults()
	}
	return nil
}

//nolint:lll // Keep the provider API signature together.
func describeProviderTable(ctx queryContext, registry, address string, opts ...stackqldoc.Option) (omnisdk.Table, error) {
	canonical := providerAddress(ctx, address)
	table, err := omnisdk.DescribeTable(registry, canonical, opts...)
	if err != nil {
		return nil, err
	}
	if canonical == address {
		return table, nil
	}
	return aliasedProviderTable{Table: table, address: address}, nil
}

//nolint:lll // Keep the provider API signature together.
func bindProviderDefaults(ctx queryContext, q query.Unresolved, tables map[string]omnisdk.Table) (query.Unresolved, error) {
	host, ok := ctx.(interface {
		ProviderBindingParams(string) map[string]string
	})
	if !ok {
		return q, nil
	}
	where := slices.Clone(q.Where())
	for _, alias := range slices.Sorted(maps.Keys(tables)) {
		table := tables[alias]
		defaults := host.ProviderBindingParams(table.Address())
		declared := map[string]bool{}
		for _, method := range table.Methods() {
			for _, parameter := range method.Params() {
				declared[parameter.Name()] = true
			}
		}
		for _, name := range slices.Sorted(maps.Keys(defaults)) {
			if !declared[name] || explicitServerBinding(q, alias, name) {
				continue
			}
			where = append(where, query.NewEq(query.NewColumn(alias, name), query.NewLiteral(defaults[name])))
		}
	}
	return queryWithWhere(q, where)
}

func providerBundle(ctx queryContext, bundle string) string {
	return strings.TrimPrefix(providerAddress(ctx, bundle), UnstablePrefix)
}
