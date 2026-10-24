package intrinsic

import (
	"fmt"
	"maps"
	"slices"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql-labs/omnisdk/pkg/query"
)

// Transport defaults are explicit document metadata for the URL scheme only.
// Neither the SDK nor this host substitutes default hosts, regions or accounts.
func bindServerTransport(q query.Unresolved, tables map[string]omnisdk.Table) (query.Unresolved, error) {
	where := slices.Clone(q.Where())
	for _, alias := range slices.Sorted(maps.Keys(tables)) {
		defaults := map[string]string{}
		for _, method := range tables[alias].Methods() {
			metadata, ok := method.(omnisdk.ServerTransportDefaults)
			if !ok {
				continue
			}
			if err := mergeTransportDefaults(defaults, metadata.ServerTransportDefaults(), q, alias); err != nil {
				return nil, err
			}
		}
		for _, name := range slices.Sorted(maps.Keys(defaults)) {
			where = append(where, query.NewEq(query.NewColumn(alias, name), query.NewLiteral(defaults[name])))
		}
	}
	return queryWithWhere(q, where)
}

func mergeTransportDefaults(target, declared map[string]string, q query.Unresolved, alias string) error {
	for name, value := range declared {
		if explicitServerBinding(q, alias, name) {
			continue
		}
		if prior, exists := target[name]; exists && prior != value {
			return fmt.Errorf("conflicting transport defaults for %s.%s; supply an explicit binding", alias, name)
		}
		target[name] = value
	}
	return nil
}

func queryWithWhere(q query.Unresolved, where []query.Predicate) (query.Unresolved, error) {
	if target := q.Target(); target != nil {
		return query.NewMutation(target, q.From(), where, q.Select())
	}
	return query.New(q.From(), where, q.Select())
}
