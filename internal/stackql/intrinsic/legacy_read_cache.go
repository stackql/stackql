package intrinsic

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql-labs/omnisdk/pkg/query"
	"github.com/stackql-labs/omnisdk/pkg/rowcache"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
)

type legacyCacheHost interface {
	queryContext
	GetNamespaceCollection() tablenamespace.Collection
}

// Existing analytics tables and their TTL remain owned by the host. This
// adapter supplies decoded rows through the independent SDK cache contract.
// It does not start a worker or take ownership of the host's database cache.
const qualifiedResourceParts = 3

type legacyNamespaceCache struct {
	host      legacyCacheHost
	namespace tablenamespace.Configurator
}

func legacyCache(ctx queryContext) *legacyNamespaceCache {
	host, ok := ctx.(legacyCacheHost)
	if !ok || host.GetNamespaceCollection() == nil {
		return nil
	}
	namespace := host.GetNamespaceCollection().GetAnalyticsCacheTableNamespaceConfigurator()
	return &legacyNamespaceCache{host: host, namespace: namespace}
}
func (c *legacyNamespaceCache) Begin(context.Context, rowcache.Key) (rowcache.Writer, error) {
	return nil, rowcache.ErrUnsupported
}
func (c *legacyNamespaceCache) Lookup(ctx context.Context, key rowcache.Key) (rowcache.Rows, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	address := strings.TrimPrefix(key.Resource, UnstablePrefix)
	if !c.namespace.IsAllowed(address) {
		return nil, false, nil
	}
	canonical := c.namespace.GetObjectName(address)
	pieces := strings.Split(canonical, ".")
	if len(pieces) != qualifiedResourceParts {
		return nil, false, nil
	}
	table, names, err := c.metadata(canonical, key.Method)
	if err != nil {
		return nil, false, err
	}

	if table == "" {
		return nil, false, nil
	}

	encoded, err := c.requestEncoding(canonical, key)
	if err != nil {
		return nil, false, err
	}

	// Compatibility with existing analytics-cache tables: these two metadata
	// columns identify the request and its age; no transaction counters are read.
	const requestColumn = "iql_insert_encoded"
	hit := c.namespace.Match(table, encoded, "iql_last_modified", requestColumn)
	if !hit {
		return nil, false, nil
	}
	result, err := c.namespace.Read(table, encoded, requestColumn, names)
	if err != nil {
		return nil, false, err
	}
	return newLegacyCacheRows(ctx, result, names), true, nil
}

func (c *legacyNamespaceCache) requestEncoding(canonical string, key rowcache.Key) (string, error) {
	var encoded strings.Builder

	declared, err := omnisdk.DescribeTable(registryRoot(c.host), UnstablePrefix+canonical)
	if err != nil {
		return "", err
	}
	var parameters []omnisdk.ParamSignature
	for _, signature := range declared.Methods() {
		if signature.Method() == key.Method {
			parameters = signature.Params()
			break
		}
	}
	for _, location := range []string{"cookie", "header", "path", "query"} {
		var keys []string
		for _, p := range parameters {
			name := p.Name()
			if p.In() == location {
				if _, ok := key.Inputs[name]; ok {
					keys = append(keys, name)
				}
			}
		}
		sort.Strings(keys)
		for _, name := range keys {
			fmt.Fprintf(&encoded, "%%%s%%%s%%%v%%", location, name, key.Inputs[name])
		}
	}
	// The legacy encoder includes an empty region slot even for global APIs.
	// This is cache-key syntax; it supplies no scope to a provider request.
	region := ""
	if value, present := key.Inputs["region"]; present {
		region = fmt.Sprint(value)
	}
	fmt.Fprintf(&encoded, "%%region%%region%%%s%%", region)
	return encoded.String(), nil
}

func configureLegacyReadCache(ctx queryContext, q query.Unresolved, args *omnisdk.Args) {
	if ctx.GetRuntimeContext().AccessConfigFile != "" {
		return
	}
	cache := legacyCache(ctx)
	if cache == nil {
		return
	}
	for _, join := range q.From() {
		if cache.namespace.IsAllowed(strings.TrimPrefix(join.Resource().Handle(), UnstablePrefix)) {
			args.ReadCache = cache
			args.ReadCacheLookupOnly = true
			args.ReadCachePartition = ctx.GetRuntimeContext().ApplicationFilesRootPath
			return
		}
	}
}

// Cache hits stream from the host's SQL cursor; no table-sized row slice is built.
type legacyCacheRows struct {
	rows    *sql.Rows
	names   []string
	ctx     context.Context
	stop    func() bool
	current map[string]any
	err     error
}

func newLegacyCacheRows(ctx context.Context, rows *sql.Rows, names []string) rowcache.Rows {
	cursor := &legacyCacheRows{rows: rows, names: names, ctx: ctx}
	cursor.stop = context.AfterFunc(ctx, func() { _ = rows.Close() })
	return cursor
}
func (r *legacyCacheRows) Next(ctx context.Context) bool {
	if r.err = ctx.Err(); r.err != nil {
		return false
	}
	if r.err = r.ctx.Err(); r.err != nil {
		return false
	}
	if !r.rows.Next() {
		r.err = r.rows.Err()
		if r.err == nil {
			r.err = r.ctx.Err()
		}
		return false
	}
	values := make([]any, len(r.names))
	destinations := make([]any, len(values))
	for i := range values {
		destinations[i] = &values[i]
	}
	if r.err = r.rows.Scan(destinations...); r.err != nil {
		return false
	}
	r.current = make(map[string]any, len(values))
	for i, name := range r.names {
		value := values[i]
		if bytes, ok := value.([]byte); ok {
			value = string(bytes)
		}
		r.current[name] = value
	}
	return true
}
func (r *legacyCacheRows) Row() map[string]any { return r.current }
func (r *legacyCacheRows) Err() error          { return r.err }
func (r *legacyCacheRows) Close() error        { r.stop(); return r.rows.Close() }

func (c *legacyNamespaceCache) metadata(canonical, method string) (string, []string, error) {
	opts, err := documentOptions(c.host)
	if err != nil {
		return "", nil, err
	}
	signature, err := omnisdk.DescribeMethod(registryRoot(c.host), UnstablePrefix+canonical, method, opts...)
	if err != nil {
		return "", nil, err
	}
	named, ok := signature.(omnisdk.NamedResponse)
	if !ok || named.RowSchemaName() == "" {
		return "", nil, nil
	}
	names := append([]string(nil), signature.Columns()...)
	sort.Strings(names)
	return canonical + "." + named.RowSchemaName(), names, nil
}
