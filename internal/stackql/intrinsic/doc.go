package intrinsic

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/stackql-labs/omnisdk/pkg/docparse/aot"
	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/any-sdk/public/formulation"
	"github.com/stackql/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"gopkg.in/yaml.v2"
)

// UnstablePrefix names the document-driven providers. The convention is
// omnisdk's own, and its addresses carry the prefixed provider name, so this is
// its constant rather than a copy of the literal.
const UnstablePrefix = aot.DefaultProviderPrefix

// IsUnstableEnabled reports whether the document-driven providers were opted
// into. They are documents read straight from disk, with none of the registry's
// curation behind them, so nothing exposes them until a caller asks.
func IsUnstableEnabled() bool {
	return previewCfg.getUnstableEnabled() || previewCfg.getOmniAll()
}

// docProvider is the bundle behind an unstable provider name, or false. Once
// every provider is routed to omnisdk, an unprefixed name is one too.
func docProvider(name string) (string, bool) {
	if !IsUnstableEnabled() {
		return "", false
	}
	trimmed := strings.TrimSpace(name)
	if !strings.HasPrefix(strings.ToLower(trimmed), UnstablePrefix) {
		if !previewCfg.getOmniAll() || trimmed == "" || strings.EqualFold(trimmed, ProviderName) {
			return "", false
		}
		return trimmed, true
	}
	bundle := trimmed[len(UnstablePrefix):]
	if bundle == "" {
		return "", false
	}
	return bundle, true
}

// bundleAliases maps the provider name stackql presents onto the directory the
// registry actually wrote. Keeping them apart matters: the name a caller typed
// is the one echoed back, and it is the only one they can address.
var bundleAliases = map[string]string{ //nolint:gochecknoglobals // fixed mapping
	"google": "googleapis.com",
}

func bundleDir(label string) string {
	if dir, ok := bundleAliases[strings.ToLower(label)]; ok {
		return dir
	}
	return label
}

// localDocRoot is where provider documents are read from, resolved exactly as
// the canonical providers resolve it, so an unstable provider reads whatever
// documents are already on disk rather than needing assets of its own.
func localDocRoot(runtimeCtx dto.RuntimeCtx) string {
	var registryCfg formulation.RegistryConfig
	if err := yaml.Unmarshal([]byte(runtimeCtx.RegistryRaw), &registryCfg); err == nil {
		if registryCfg.LocalDocRoot != "" {
			return registryCfg.LocalDocRoot
		}
		if strings.HasPrefix(registryCfg.RegistryURL, "file:") {
			return filepath.Clean(
				filepath.Join(strings.TrimPrefix(registryCfg.RegistryURL, "file:"), ".."))
		}
	}
	return runtimeCtx.ApplicationFilesRootPath
}

// docRoot is the bundle's own directory inside stackql's document root. The
// versioned directory is used rather than the root itself, because a registry
// root is addressed as "<provider>.<service>.<resource>" and a provider whose
// name carries a dot ("googleapis.com") cannot be named that way.
func docRoot(ctx queryContext, bundle string) (string, error) {
	root := filepath.Join(localDocRoot(ctx.GetRuntimeContext()), "src", bundleDir(bundle))
	matches, err := filepath.Glob(filepath.Join(root, "*", "provider.yaml"))
	if err != nil || len(matches) == 0 {
		return "", fmt.Errorf("no provider document for '%s%s' under '%s'", UnstablePrefix, bundle, root)
	}
	sort.Strings(matches)
	return filepath.Dir(matches[len(matches)-1]), nil
}

// docServices lists the services a bundle ships documents for.
func docServices(ctx queryContext, bundle string) ([]string, error) {
	dir, err := docRoot(ctx, bundle)
	if err != nil {
		return nil, err
	}
	services, _, err := omnisdk.DocCatalog(dir, bundle)
	return services, err
}

// docResourceTables presents a service's resources as relations.
func docResourceTables(ctx queryContext, bundle, service string) ([]table, error) {
	dir, err := docRoot(ctx, bundle)
	if err != nil {
		return nil, err
	}
	resources, err := omnisdk.DocResources(dir, bundle, service)
	if err != nil {
		return nil, err
	}
	sort.Strings(resources)
	out := make([]table, 0, len(resources))
	for _, resource := range resources {
		out = append(out, table{service: service, name: resource, isData: true})
	}
	return out, nil
}

// docMethods lists a resource's methods as the document declares them.
func docMethods(ctx queryContext, bundle, service, resource string) ([]relationMethod, error) {
	dir, err := docRoot(ctx, bundle)
	if err != nil {
		return nil, err
	}
	methods, err := omnisdk.DocMethods(dir, bundle, service, resource)
	if err != nil {
		return nil, err
	}
	out := make([]relationMethod, 0, len(methods))
	for _, method := range methods {
		out = append(out, relationMethod{
			name:        method.Name,
			description: method.OperationID,
			sqlVerb:     docSQLVerb(method.SQLVerb),
		})
	}
	return out, nil
}

// docSQLVerb is the verb a document maps a method to, upper-cased as SHOW
// METHODS reports it. A method no verb maps is reachable only through EXEC.
func docSQLVerb(verb string) string {
	if verb == "" {
		return "EXEC"
	}
	return strings.ToUpper(verb)
}

// docSelectFunc routes a SELECT over document-driven relations. omnisdk
// resolves the whole query against the provider documents - which method each
// relation runs, and whether a condition is a request parameter, an edge
// between relations or a row filter - and streams the rows back.
func docSelectFunc(
	ctx queryContext,
	node *sqlparser.Select,
	currentProvider string,
) (func() internaldto.ExecutorOutput, bool) {
	if previewCfg.getStagingEnabled() && needsStaging(node) {
		return stagedSelectFunc(ctx, node, currentProvider), true
	}
	dialect, err := backendDialect(ctx)
	if err != nil {
		return refuse(err), true
	}
	translated, err := newDocTranslator(currentProvider, dialect).selectQuery(node)
	if err != nil {
		return refuse(err), true
	}
	return func() internaldto.ExecutorOutput { return runDocQuery(ctx, translated) }, true
}

// docMutationFunc routes an INSERT, UPDATE or DELETE whose target is a
// document-driven relation. Without RETURNING the effects are driven to
// completion and reported as a message; with it, the returned rows stream back.
func docMutationFunc(
	ctx queryContext,
	stmt sqlparser.Statement,
	currentProvider string,
) (func() internaldto.ExecutorOutput, bool) {
	tables, isMutation := mutationTables(stmt)
	if !isMutation {
		return nil, false
	}
	if isDoc, err := fromDocProviders(tables, currentProvider); !isDoc {
		return nil, false
	} else if err != nil {
		return refuse(err), true
	}
	dialect, err := backendDialect(ctx)
	if err != nil {
		return refuse(err), true
	}
	translated, err := newDocTranslator(currentProvider, dialect).mutation(stmt)
	if err != nil {
		return refuse(err), true
	}
	return func() internaldto.ExecutorOutput { return runDocQuery(ctx, translated) }, true
}

const mutationSuccessMessage = "The operation was despatched successfully"

// openDocQuery describes each relation, resolves the query and opens its
// cursor, returning it with the alias of the relation it reports.
func openDocQuery(ctx queryContext, translated docQuery) (omnisdk.Rows, string, error) {
	registry := registryRoot(ctx)
	q := translated.getQuery()
	tables := make(map[string]omnisdk.Table, len(q.From())+1)
	for _, join := range q.From() {
		tbl, describeErr := omnisdk.DescribeTable(registry, join.Resource().Handle())
		if describeErr != nil {
			return nil, "", describeErr
		}
		tables[join.Resource().Alias()] = tbl
	}
	var relation string
	if target := q.Target(); target != nil {
		tbl, describeErr := omnisdk.DescribeMutation(
			registry, target.Resource().Handle(), target.Verb().String())
		if describeErr != nil {
			return nil, "", describeErr
		}
		tables[target.Resource().Alias()] = tbl
		relation = target.Resource().Alias()
	} else {
		relation = q.From()[0].Resource().Alias()
	}
	dialect, dialectErr := backendDialect(ctx)
	if dialectErr != nil {
		return nil, "", dialectErr
	}
	res, resolveErr := omnisdk.ResolveIn(q, tables, dialect.catalogue())
	if resolveErr != nil {
		return nil, "", resolveErr
	}
	// omnisdk takes one credential per run: the first relation's cloud - a
	// mutation's target - leaving the rest to the canonical environment
	// variables.
	args, argsErr := previewArgs(ctx, translated.getBundles()[0], res.Params())
	if argsErr != nil {
		return nil, "", argsErr
	}
	args.Tuning.Limit = translated.getLimit()
	plan, planErr := omnisdk.NewGraphSelectQuery(registry, res.Graph(), args)
	if planErr != nil {
		return nil, "", planErr
	}
	rows, openErr := plan.Open(context.Background())
	if openErr != nil {
		return nil, "", openErr
	}
	return rows, relation, nil
}

// runDocQuery runs the query and streams its rows back.
func runDocQuery(ctx queryContext, translated docQuery) internaldto.ExecutorOutput {
	rows, relation, err := openDocQuery(ctx, translated)
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	q := translated.getQuery()
	if q.Target() != nil && len(translated.getOutputs()) == 0 {
		return drainMutation(rows)
	}
	input := previewCfg
	stream := &rowStream{
		rows:          rows,
		batchSize:     input.getBatchSize(),
		flushInterval: input.getFlushInterval(),
		table:         sqldata.NewSQLTable(0, relation),
		typCfg:        ctx.GetTypingConfig(),
		outputs:       translated.getOutputs(),
	}
	primed, readErr := newPrimedStream(stream)
	if readErr != nil {
		return internaldto.NewErroneousExecutorOutput(readErr)
	}
	return internaldto.NewExecutorOutput(primed, nil, nil, nil, nil)
}

// drainMutation pulls a mutation's cursor to the end, which is what sends its
// effects, and reports the outcome.
func drainMutation(rows omnisdk.Rows) internaldto.ExecutorOutput {
	defer rows.Close()
	for rows.Next() { //nolint:revive // draining sends the effects
	}
	if err := rows.Err(); err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	return internaldto.NewExecutorOutput(nil, nil, nil,
		internaldto.NewBackendMessages([]string{mutationSuccessMessage}), nil)
}

func refuse(err error) func() internaldto.ExecutorOutput {
	return func() internaldto.ExecutorOutput {
		return internaldto.NewErroneousExecutorOutput(err)
	}
}

func showDocServices(ctx queryContext, bundle string, extended bool) internaldto.ExecutorOutput {
	services, err := docServices(ctx, bundle)
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	rows := make(map[string]map[string]interface{}, len(services))
	for i, service := range services {
		row := map[string]interface{}{"id": service, "name": service, "title": service}
		if extended {
			row["description"] = service
			row["version"] = ProviderVersion
			row["preferred"] = nil
		}
		rows[fmt.Sprintf("%06d", i)] = row
	}
	return prepare(ctx, formulation.GetServicesHeader(extended), rows, util.DefaultRowSort)
}

func showDocResources(
	ctx queryContext, bundle, service string, extended bool) internaldto.ExecutorOutput {
	tables, err := docResourceTables(ctx, bundle, service)
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	rows := make(map[string]map[string]interface{}, len(tables))
	for i, tbl := range tables {
		row := map[string]interface{}{
			"id":   fmt.Sprintf("%s%s.%s.%s", UnstablePrefix, bundle, service, tbl.name),
			"name": tbl.name,
		}
		if extended {
			row["description"] = tbl.description
		}
		rows[fmt.Sprintf("%06d", i)] = row
	}
	return prepare(ctx, formulation.GetResourcesHeader(extended), rows, util.DefaultRowSort)
}

func showDocMethods(
	ctx queryContext, bundle, service, resource string, extended bool) internaldto.ExecutorOutput {
	methods, err := docMethods(ctx, bundle, service, resource)
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	columnOrder := []string{"MethodName", "RequiredParams", "SQLVerb"}
	if extended {
		columnOrder = append(columnOrder, "description")
	}
	rows := make(map[string]map[string]interface{}, len(methods))
	for i, method := range methods {
		row := map[string]interface{}{
			"MethodName":     method.name,
			"RequiredParams": strings.Join(method.requiredParams, ", "),
			"SQLVerb":        method.sqlVerb,
		}
		if extended {
			row["description"] = method.description
		}
		rows[fmt.Sprintf("%06d", i)] = row
	}
	return prepare(ctx, columnOrder, rows, util.DefaultRowSort)
}
