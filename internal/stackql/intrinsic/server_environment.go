package intrinsic

import (
	"fmt"
	"maps"
	"os"
	"slices"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql-labs/omnisdk/pkg/query"
)

// Environment resolution is host policy, limited to explicit document annotations.
// SQL parameters take precedence. The SDK receives ordinary explicit bindings.
func bindServerEnvironment(q query.Unresolved, tables map[string]omnisdk.Table) (query.Unresolved, error) {
	where := slices.Clone(q.Where())
	for _, alias := range slices.Sorted(maps.Keys(tables)) {
		declared := map[string]string{}
		for _, method := range tables[alias].Methods() {
			if metadata, ok := method.(omnisdk.ServerEnvironmentBindings); ok {
				for name, environment := range metadata.ServerEnvironmentBindings() {
					declared[name] = environment
				}
			}
		}
		for _, name := range slices.Sorted(maps.Keys(declared)) {
			if explicitServerBinding(q, alias, name) {
				continue
			}
			value, exists := os.LookupEnv(declared[name])
			if !exists || value == "" {
				return nil, fmt.Errorf("cannot find any viable servers: %s requires %s", alias, name)
			}
			where = append(where, query.NewEq(query.NewColumn(alias, name), query.NewLiteral(value)))
		}
	}
	bound, err := queryWithWhere(q, where)
	if err != nil {
		return nil, err
	}
	return bindServerTransport(bound, tables)
}

func explicitServerBinding(q query.Unresolved, alias, name string) bool {
	matches := func(e query.Expr) bool {
		col, ok := e.(query.Column)
		return ok && col.Name() == name && (col.Qualifier() == "" || col.Qualifier() == alias)
	}
	predicates := slices.Clone(q.Where())
	for _, join := range q.From() {
		predicates = append(predicates, join.On()...)
	}
	for _, predicate := range predicates {
		switch p := predicate.(type) {
		case query.Compare:
			if p.Op() == query.Eq && (matches(p.Left()) || matches(p.Right())) {
				return true
			}
		case query.In:
			if matches(p.Expr()) {
				return true
			}
		}
	}
	target := q.Target()
	if target == nil || target.Resource().Alias() != alias {
		return false
	}
	return slices.ContainsFunc(target.Set(), func(assignment query.Assignment) bool { return assignment.Column() == name })
}

func unresolvedServerParams(method omnisdk.DocMethod) []string {
	required := make([]string, 0, len(method.RequiredParams))
	for _, name := range method.RequiredParams {
		if method.TransportDefaults[name] != "" {
			continue
		}
		if environment := method.ServerEnvironment[name]; environment != "" {
			if value, exists := os.LookupEnv(environment); exists && value != "" {
				continue
			}
		}
		required = append(required, name)
	}
	return required
}
