package intrinsic

import (
	"maps"
	"os"
	"slices"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql-labs/omnisdk/pkg/query"
)

func bindServerEnvironment(q query.Unresolved, tables map[string]omnisdk.Table) (query.Unresolved, error) {
	return omnisdk.BindServerInputs(q, tables, os.LookupEnv)
}

func unresolvedServerParams(method omnisdk.DocMethod, defaults map[string]string) ([]string, error) {
	return omnisdk.NewDocMethodInputResolver(method, defaults, os.LookupEnv).Required(method.RequiredParams, nil)
}

func sessionParameterDefaults(ctx queryContext, address string) map[string]string {
	if host, ok := ctx.(interface {
		ProviderBindingParams(string) map[string]string
	}); ok {
		return host.ProviderBindingParams(address)
	}
	return nil
}

func bindExecServerInputs(
	ctx queryContext, address string, signature omnisdk.MethodSignature, params map[string]string,
) error {
	resolver := omnisdk.NewMethodInputResolver(
		[]omnisdk.MethodSignature{signature}, sessionParameterDefaults(ctx, address), os.LookupEnv,
	)
	bindings, err := resolver.Resolve(func(name string) bool {
		_, exists := params[name]
		return exists
	})
	if err != nil {
		return err
	}
	maps.Copy(params, bindings)
	return nil
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

// bindReadPageSize retains the CLI's request size independently of a SQL LIMIT.
// Only declared query parameters are supplied, and explicit SQL bindings win.
func bindReadPageSize(q query.Unresolved, tables map[string]omnisdk.Table, size int) (query.Unresolved, error) {
	if size <= 0 {
		return q, nil
	}
	where := slices.Clone(q.Where())
	for _, alias := range slices.Sorted(maps.Keys(tables)) {
		if q.Target() != nil && q.Target().Resource().Alias() == alias {
			continue
		}
		for _, method := range tables[alias].Methods() {
			for _, parameter := range method.Params() {
				if parameter.In() == "query" && parameter.Name() == "maxResults" && !explicitServerBinding(q, alias, "maxResults") {
					where = append(where, query.NewEq(query.NewColumn(alias, "maxResults"), query.NewLiteral(size)))
					break
				}
			}
			break
		}
	}
	return queryWithWhere(q, where)
}

func queryWithWhere(q query.Unresolved, where []query.Predicate) (query.Unresolved, error) {
	if target := q.Target(); target != nil {
		return query.NewMutation(target, q.From(), where, q.Select())
	}
	return query.New(q.From(), where, q.Select())
}
