package intrinsic

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/stackql-labs/omnisdk/pkg/docparse/aot"
	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql-labs/omnisdk/pkg/query"
	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/any-sdk/public/formulation"
	"github.com/stackql/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql/internal/stackql/acid/binlog"
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

// IsExtendedCatalogEnabled keeps the compatibility catalogue stable when
// Omni replaces the execution backend. Explicit unstable mode adds SDK services.
func IsExtendedCatalogEnabled() bool {
	return previewCfg.getUnstableEnabled() && !previewCfg.getOmniAll()
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
	runtime := ctx.GetRuntimeContext()
	prefix := "src"
	var registryCfg formulation.RegistryConfig
	if err := yaml.Unmarshal([]byte(runtime.RegistryRaw), &registryCfg); err != nil {
		return "", err
	}
	if registryCfg.SrcPrefix != nil {
		prefix = *registryCfg.SrcPrefix
	}
	root := filepath.Join(localDocRoot(runtime), prefix, bundleDir(bundle))
	matches, err := filepath.Glob(filepath.Join(root, "*", "provider.yaml"))
	if err != nil || len(matches) == 0 {
		return "", fmt.Errorf("no provider document for '%s%s' under '%s'", UnstablePrefix, bundle, root)
	}
	sort.Strings(matches)
	return filepath.Dir(matches[len(matches)-1]), nil
}

// docResourceTables presents a service's resources as relations.
func docResourceTables(ctx queryContext, bundle, service string) ([]table, error) {
	dir, err := docRoot(ctx, bundle)
	if err != nil {
		return nil, err
	}
	opts, err := documentOptions(ctx)
	if err != nil {
		return nil, err
	}
	resources, err := omnisdk.DocResources(dir, bundle, service, opts...)
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
	opts, err := documentOptions(ctx)
	if err != nil {
		return nil, err
	}
	methods, err := omnisdk.DocMethods(dir, bundle, service, resource, opts...)
	if err != nil {
		return nil, err
	}
	out := make([]relationMethod, 0, len(methods))
	for _, method := range methods {
		out = append(out, relationMethod{
			requiredParams: unresolvedServerParams(method),
			name:           method.Name,
			description:    method.OperationID,
			sqlVerb:        docSQLVerb(method.SQLVerb),
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
	initial ...map[string]omnisdk.Relation,
) (func() internaldto.ExecutorOutput, bool) {
	expanded, expandErr := expandDocumentViews(ctx, node, currentProvider)
	if expandErr != nil {
		return refuse(expandErr), true
	}
	node, relations, relationErr := resolveDerivedRelations(ctx, expanded, currentProvider, 0, initial...)
	if relationErr != nil {
		return refuse(relationErr), true
	}
	if previewCfg.getStagingEnabled() && needsStaging(node) {
		return stagedSelectWithRelations(ctx, node, currentProvider, relations), true
	}
	dialect, err := backendDialect(ctx)
	if err != nil {
		return refuse(err), true
	}
	translator := newDocTranslator(currentProvider, dialect)
	if err = expandDerivedStars(ctx, node, translator, relations); err != nil {
		return refuse(err), true
	}
	translated, err := translator.selectQuery(node)
	if err != nil {
		return refuse(err), true
	}
	translated = withRelations(translated, relations)
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
	stmt, relations, err := prepareMutationRelations(ctx, stmt, currentProvider)
	if err != nil {
		return refuse(err), true
	}
	translated, err := newDocTranslator(currentProvider, dialect).mutation(stmt)
	if err != nil {
		return refuse(err), true
	}
	translated = withRelations(translated, relations)
	if mutationAwait(stmt) {
		translated = awaitedDocQuery{docQuery: translated}
	}
	return func() internaldto.ExecutorOutput { return runDocQuery(ctx, translated) }, true
}

const mutationSuccessMessage = "The operation was despatched successfully"

// openDocQuery describes each relation, resolves the query and opens its
// cursor, returning it with the alias of the relation it reports.
func openDocQuery(ctx queryContext, translated docQuery) (omnisdk.Rows, string, error) {
	return openDocQueryIn(ctx, translated, context.Background())
}

//nolint:gocognit,funlen // The cognitive complexity of this function is high but acceptable for now
func openDocQueryIn(ctx queryContext, translated docQuery, runCtx context.Context) (omnisdk.Rows, string, error) {
	opts, err := documentOptions(ctx)
	if err != nil {
		return nil, "", err
	}
	registry := registryRoot(ctx)
	q := translated.getQuery()
	relations := queryRelations(translated)
	tables := make(map[string]omnisdk.Table, len(q.From())+1)
	for _, join := range q.From() {
		if _, function := join.Resource().(query.TableFunction); function {
			continue
		}
		var tbl omnisdk.Table
		var describeErr error
		if relation, ok := relations[join.Resource().Handle()]; ok {
			tbl = relation
		} else {
			tbl, describeErr = omnisdk.DescribeTable(registry, join.Resource().Handle(), opts...)
		}
		if describeErr != nil {
			return nil, "", describeErr
		}
		tables[join.Resource().Alias()] = tbl
	}
	var relation string
	if target := q.Target(); target != nil {
		tbl, describeErr := describeMutationTable(registry, target, queryAwaitsMutation(translated), opts...)
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
	q, environmentErr := bindServerEnvironment(q, tables)
	if environmentErr != nil {
		return nil, "", environmentErr
	}
	res, resolveErr := omnisdk.ResolveIn(q, tables, dialect.catalogue())
	if resolveErr != nil {
		return nil, "", resolveErr
	}
	// Preserve the host's configured credential for every provider, including
	// providers following an opaque derived relation in the FROM order.
	args, argsErr := previewArgs(ctx, translated.getBundles()[0], res.Params())
	if argsErr != nil {
		return nil, "", argsErr
	}
	args.AuthByProvider, argsErr = queryProviderAuth(ctx, translated.getBundles())
	if argsErr != nil {
		return nil, "", argsErr
	}
	if completionErr := configureMutationCompletion(ctx, translated, &args); completionErr != nil {
		return nil, "", completionErr
	}
	configureMutationObserver(ctx, q.Target() != nil, &args)
	args.Relations = relations
	args.Tuning.Limit = translated.getLimit()
	configureReadHints(translated, &args)
	configureLegacyReadCache(ctx, q, &args)
	plan, planErr := omnisdk.NewGraphQuery(registry, res.Graph(), args)
	if planErr != nil {
		return nil, "", planErr
	}
	rows, openErr := plan.Open(runCtx)
	if openErr != nil {
		return nil, "", openErr
	}
	return rows, relation, nil
}

// queryProviderAuth preserves configured credentials independently for every provider.
func queryProviderAuth(ctx queryContext, bundles []string) (map[string]*omnisdk.Auth, error) {
	credentials := map[string]*omnisdk.Auth{}
	for _, bundle := range bundles {
		auth := providerAuthContext(ctx, bundle)
		if auth != nil && (auth.KeyFilePath != "" || auth.KeyFilePathEnvVar != "" || auth.KeyEnvVar != "") {
			if _, err := auth.GetCredentialsBytes(); err != nil {
				return nil, fmt.Errorf("service account credentials error: %w", err)
			}
		}
		for provider, credential := range providerAuthByProvider(ctx, bundle) {
			credentials[provider] = credential
		}
	}
	return credentials, nil
}

// runDocQuery runs the query and streams its rows back.
func runDocQuery(ctx queryContext, translated docQuery) internaldto.ExecutorOutput {
	rows, relation, err := openDocQuery(ctx, translated)
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	q := translated.getQuery()
	if q.Target() != nil && len(translated.getOutputs()) == 0 {
		result := drainMutation(rows)
		if mutationErr := result.GetError(); mutationErr != nil {
			return internaldto.NewErroneousExecutorOutput(mutationHTTPFailure(ctx, q.Target().Verb().String(), mutationErr))
		}
		address := strings.TrimPrefix(q.Target().Resource().Handle(), UnstablePrefix)
		message := "Undo the " + q.Target().Verb().String() + " on " + address
		result.SetUndoLog(binlog.NewSimpleLogEntry(nil, []string{message}))
		if queryAwaitsMutation(translated) && result.GetError() == nil {
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, nil)
		}
		if result.GetError() == nil && inlineMutation(ctx, q.Target().Resource().Handle(), q.Target().Verb().String()) {
			return internaldto.NewExecutorOutput(nil, nil, nil, internaldto.NewBackendMessages([]string{"OK"}), nil)
		}
		return result
	}
	input := previewCfg
	stream := &rowStream{
		suppressFailedHeaders: true,
		rows:                  rows,
		batchSize:             input.getBatchSize(),
		flushInterval:         input.getFlushInterval(),
		table:                 sqldata.NewSQLTable(0, relation),
		typCfg:                ctx.GetTypingConfig(),
		outputs:               translated.getOutputs(),
		emptyColumns:          func() []column { return declaredEmptyColumns(ctx, translated) },
	}
	primed, readErr := newPrimedStream(stream)
	if readErr != nil {
		return internaldto.NewErroneousExecutorOutput(readErr)
	}
	return internaldto.NewExecutorOutput(primed, nil, nil, nil, nil)
}

// Empty star results still have the schema declared by their read relations.
// Actual rows continue to determine nonempty star results, including ragged fields.
func declaredEmptyColumns(ctx queryContext, translated docQuery) []column {
	q := translated.getQuery()
	resources := q.From()
	if target := q.Target(); target != nil {
		resources = []query.Join{query.NewJoin(target.Resource(), query.Base)}
	}
	fields := omnisdk.Row{}
	for _, join := range resources {
		if _, function := join.Resource().(query.TableFunction); function {
			continue
		}
		names, err := declaredRelationColumns(ctx, join.Resource().Handle(), queryRelations(translated))
		if err != nil {
			continue
		}
		for _, name := range names {
			fields[name] = nil
		}
	}
	if len(fields) == 0 {
		return nil
	}
	return outputColumns(translated.getOutputs(), []omnisdk.Row{fields})
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

func showDocServices(
	ctx queryContext,
	bundle string,
	extended bool,
	filters ...*sqlparser.Show,
) internaldto.ExecutorOutput {
	dir, err := docRoot(ctx, bundle)
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	opts, err := documentOptions(ctx)
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	services, err := omnisdk.DocServices(dir, bundle, opts...)
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	rows := make(map[string]map[string]interface{}, len(services))
	for i, service := range services {
		row := map[string]interface{}{"id": service.ID, "name": service.Name, "title": service.Title}
		if extended {
			row["description"] = service.Description
			row["version"] = service.Version
			row["preferred"] = service.Preferred
		}
		rows[fmt.Sprintf("%06d", i)] = row
	}
	return prepareDocMetadata(ctx, formulation.GetServicesHeader(extended), rows, filters)
}

func showDocResources(
	ctx queryContext, bundle, service string, extended bool, filters ...*sqlparser.Show) internaldto.ExecutorOutput {
	tables, err := docResourceTables(ctx, bundle, service)
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	names, nameErr := registryResourceNames(ctx, bundle, service, extended)
	if nameErr != nil {
		return internaldto.NewErroneousExecutorOutput(nameErr)
	}
	rows := make(map[string]map[string]interface{}, len(tables))
	for i, tbl := range tables {
		displayName := tbl.name
		if declared, present := names[tbl.name]; present {
			displayName = declared
		}
		row := map[string]interface{}{
			"id":   fmt.Sprintf("%s.%s.%s", metadataProviderName(bundle, filters), service, tbl.name),
			"name": displayName,
		}
		if extended {
			row["description"] = tbl.description
		}
		rows[fmt.Sprintf("%06d", i)] = row
	}
	return prepareDocMetadata(ctx, formulation.GetResourcesHeader(extended), rows, filters)
}

func showDocMethods(
	ctx queryContext, bundle, service, resource string, extended bool, alphabetical ...bool) internaldto.ExecutorOutput {
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
		//nolint:mnd // These values define stable output ordering.
		rank := map[string]int{"SELECT": 0, "INSERT": 1, "DELETE": 2, "UPDATE": 3, "REPLACE": 4, "EXEC": 5}[method.sqlVerb]
		if len(alphabetical) > 0 && alphabetical[0] {
			rank = 0
		}
		rows[fmt.Sprintf("%02d_%s_%06d", rank, method.name, i)] = row
	}
	return prepare(ctx, columnOrder, rows, util.DefaultRowSort)
}

func inlineMutation(ctx queryContext, address, verb string) bool {
	table, err := omnisdk.DescribeMutation(registryRoot(ctx), address, verb)
	if err != nil {
		return false
	}
	for _, method := range table.Methods() {
		if local, ok := method.(omnisdk.InlineCommand); ok && len(local.InlineCommand()) > 0 {
			return true
		}
	}
	return false
}

func configureReadHints(translated docQuery, args *omnisdk.Args) {
	if hinted, ok := translated.(interface {
		getReadHints() map[string]omnisdk.ReadHints
	}); ok {
		args.ReadHintsByAlias = hinted.getReadHints()
	}
}
