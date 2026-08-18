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
	return previewCfg.getUnstableEnabled()
}

// docProvider is the bundle behind an unstable provider name, or false.
func docProvider(name string) (string, bool) {
	if !IsUnstableEnabled() {
		return "", false
	}
	trimmed := strings.TrimSpace(name)
	if !strings.HasPrefix(strings.ToLower(trimmed), UnstablePrefix) {
		return "", false
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
		out = append(out, relationMethod{name: method.Name, description: method.OperationID})
	}
	return out, nil
}

// docSelectFunc routes a SELECT over a document-driven relation. The address is
// the bundle's own "<provider>.<service>.<resource>", and the plan it yields
// streams exactly as a hand-authored one does.
func docSelectFunc(
	ctx queryContext,
	node *sqlparser.Select,
	bundle, service, resource string,
) (func() internaldto.ExecutorOutput, bool) {
	if unsupported := unsupportedClauses(node); len(unsupported) > 0 {
		return refuse(fmt.Errorf(
			"relation '%s%s.%s.%s' streams its rows, so %s cannot be applied; remove %s from the query",
			UnstablePrefix, bundle, service, resource,
			strings.Join(unsupported, ", "), pluralClause(len(unsupported)))), true
	}
	params, badPredicates := equalityPredicates(node.Where)
	if len(badPredicates) > 0 {
		return refuse(fmt.Errorf(
			"relation '%s%s.%s.%s' streams its rows, so only equality predicates are applied; "+
				"%s cannot be honoured",
			UnstablePrefix, bundle, service, resource, strings.Join(badPredicates, ", "))), true
	}
	address := fmt.Sprintf("%s%s.%s.%s", UnstablePrefix, bundle, service, resource)
	return func() internaldto.ExecutorOutput {
		input := previewCfg
		dir, dirErr := docRoot(ctx, bundle)
		if dirErr != nil {
			return internaldto.NewErroneousExecutorOutput(dirErr)
		}
		plan, err := omnisdk.NewFromCatalog(dir, address, omnisdk.Args{
			Params:                params,
			Auth:                  omnisdkAuth(providerAuthContext(ctx, bundle)),
			Endpoint:              input.getEndpoint(),
			InsecureSkipTLSVerify: input.getInsecureSkipTLSVerify(),
		})
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		rows, openErr := plan.Open(context.Background())
		if openErr != nil {
			return internaldto.NewErroneousExecutorOutput(openErr)
		}
		// A document declares no egress schema, so the columns are those the
		// first row carries; projection is applied over them.
		stream := &rowStream{
			rows:          rows,
			batchSize:     input.getBatchSize(),
			flushInterval: input.getFlushInterval(),
			table:         sqldata.NewSQLTable(0, resource),
			typCfg:        ctx.GetTypingConfig(),
			projection:    node.SelectExprs,
		}
		primed, readErr := newPrimedStream(stream)
		if readErr != nil {
			return internaldto.NewErroneousExecutorOutput(readErr)
		}
		return internaldto.NewExecutorOutput(primed, nil, nil, nil, nil)
	}, true
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
			"SQLVerb":        strings.ToUpper(selectMethodName),
		}
		if extended {
			row["description"] = method.description
		}
		rows[fmt.Sprintf("%06d", i)] = row
	}
	return prepare(ctx, columnOrder, rows, util.DefaultRowSort)
}
