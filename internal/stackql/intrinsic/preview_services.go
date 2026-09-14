package intrinsic

// Two further services sit alongside "audit" under stackql_preview:
// "dynamic_graph" for a query wired across several exchanges, and "iac" for an
// idempotent converge run. Both are deliberately thin. A predicate carries the
// specification, omnisdk does the work, and the rows it yields stream through
// the same cursor a single-method select already uses.
//
// They are gated behind the same opt-in as the document-driven providers,
// because that is what they read: documents straight from disk, with none of
// the registry's curation behind them.

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/any-sdk/public/formulation"
	"github.com/stackql/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

const (
	dynamicGraphService = "dynamic_graph"
	iacService          = "iac"
)

const (
	graphRelation    = "query"
	convergeRelation = "run"
)

const (
	dynamicGraphTitle = "queries wired across several document-declared exchanges"
	iacTitle          = "idempotent converge runs, and the blueprints that render one"
)

const (
	specPredicate       = "spec"
	collectionPredicate = "collection"
	blueprintPredicate  = "blueprint"
	resourcesPredicate  = "resources"
	statePredicate      = "state"
	runIDPredicate      = "run_id"
)

// previewService canonicalises one of the extended stackql_preview services, or
// reports false. They read documents from disk, so they appear only once that
// was opted into, exactly as the unstable providers do.
func previewService(providerName, serviceStr, currentProvider string) (string, bool) {
	if !IsUnstableEnabled() {
		return "", false
	}
	resolved := strings.TrimSpace(resolveProvider(providerName, currentProvider))
	if !strings.EqualFold(resolved, ProviderName) {
		return "", false
	}
	switch trimmed := strings.TrimSpace(serviceStr); {
	case strings.EqualFold(trimmed, dynamicGraphService):
		return dynamicGraphService, true
	case strings.EqualFold(trimmed, iacService):
		return iacService, true
	default:
		return "", false
	}
}

// appendPreviewServices adds the extended services to a SHOW SERVICES listing.
// It is a no-op without the opt-in, which keeps the catalogue exactly as it was
// for anyone who has not asked for them.
func appendPreviewServices(rows map[string]map[string]interface{}, extended bool) {
	if !IsUnstableEnabled() {
		return
	}
	for i, svc := range []struct{ name, title string }{
		{dynamicGraphService, dynamicGraphTitle},
		{iacService, iacTitle},
	} {
		row := map[string]interface{}{
			"id":    fmt.Sprintf("%s:%s", svc.name, ProviderVersion),
			"name":  svc.name,
			"title": svc.title,
		}
		if extended {
			row["description"] = svc.title
			row["version"] = ProviderVersion
			row["preferred"] = nil
		}
		rows[fmt.Sprintf("%06d", i+2)] = row //nolint:mnd // audit already holds 000001
	}
}

// registryRoot is the provider-document root omnisdk resolves addresses
// against. It is the directory holding "<provider>/<version>/provider.yaml",
// which is where stackql's own document root keeps them.
func registryRoot(ctx queryContext) string {
	return filepath.Join(localDocRoot(ctx.GetRuntimeContext()), "src")
}

// popPredicate takes a predicate out of the parameter map. What remains after
// every control predicate is popped is the run's scope, which rides through to
// omnisdk untouched.
func popPredicate(params map[string]string, key string) string {
	value := params[key]
	delete(params, key)
	return value
}

// streamPlan opens a plan and hands its cursor to the caller. Neither service
// declares an egress schema, so the columns are the ones the first row carries
// and the projection is applied over them.
func streamPlan(
	ctx queryContext,
	plan omnisdk.Plan,
	relation string,
	exprs sqlparser.SelectExprs,
) internaldto.ExecutorOutput {
	rows, openErr := plan.Open(context.Background())
	if openErr != nil {
		return internaldto.NewErroneousExecutorOutput(openErr)
	}
	input := previewCfg
	stream := &rowStream{
		rows:          rows,
		batchSize:     input.getBatchSize(),
		flushInterval: input.getFlushInterval(),
		table:         sqldata.NewSQLTable(0, relation),
		typCfg:        ctx.GetTypingConfig(),
		projection:    exprs,
	}
	primed, readErr := newPrimedStream(stream)
	if readErr != nil {
		return internaldto.NewErroneousExecutorOutput(readErr)
	}
	return internaldto.NewExecutorOutput(primed, nil, nil, nil, nil)
}

// previewArgs assembles the SDK arguments both services need: the scope left
// over after the control predicates, the credential for the cloud in play, and
// the backend tuning.
func previewArgs(ctx queryContext, cloud string, params map[string]string) omnisdk.Args {
	input := previewCfg
	return omnisdk.Args{
		Params:                params,
		Auth:                  omnisdkAuth(providerAuthContext(ctx, cloud)),
		Endpoint:              input.getEndpoint(),
		InsecureSkipTLSVerify: input.getInsecureSkipTLSVerify(),
	}
}

// previewSelectFunc routes a SELECT over an extended preview relation.
func previewSelectFunc(
	ctx queryContext,
	node *sqlparser.Select,
	service, resource string,
) (func() internaldto.ExecutorOutput, bool) {
	if service == dynamicGraphService {
		return dynamicSelectFunc(ctx, node, resource)
	}
	return iacSelectFunc(ctx, node, resource)
}

// previewPredicates reads a select's WHERE clause as the flat equality map both
// services take, refusing anything the streaming path cannot honour.
func previewPredicates(
	node *sqlparser.Select,
	relation string,
) (map[string]string, func() internaldto.ExecutorOutput) {
	if unsupported := unsupportedClauses(node); len(unsupported) > 0 {
		return nil, refuse(fmt.Errorf(
			"relation '%s' streams its rows, so %s cannot be applied; remove %s from the query",
			relation, strings.Join(unsupported, ", "), pluralClause(len(unsupported))))
	}
	params, bad := equalityPredicates(node.Where)
	if len(bad) > 0 {
		return nil, refuse(fmt.Errorf(
			"relation '%s' streams its rows, so only equality predicates are applied; "+
				"%s cannot be honoured",
			relation, strings.Join(bad, ", ")))
	}
	return params, nil
}

// relationAddress names an extended preview relation the way a caller wrote it.
func relationAddress(service, resource string) string {
	return fmt.Sprintf("%s.%s.%s", ProviderName, service, resource)
}

// showPreviewFunc answers SHOW for the extended preview services.
func showPreviewFunc(
	ctx queryContext,
	node *sqlparser.Show,
	currentProvider string,
	extended bool,
) (func() internaldto.ExecutorOutput, bool) {
	switch strings.ToUpper(strings.TrimSpace(node.Type)) {
	case "RESOURCES":
		service, isPreview := previewService(
			node.OnTable.Qualifier.GetRawVal(), node.OnTable.Name.GetRawVal(), currentProvider)
		if !isPreview {
			return nil, false
		}
		return func() internaldto.ExecutorOutput {
			return showPreviewResources(ctx, service, extended)
		}, true
	case "METHODS":
		// Every extended relation is select-only, so the method list does not
		// vary by resource.
		if _, isPreview := previewService(
			node.OnTable.QualifierSecond.GetRawVal(),
			node.OnTable.Qualifier.GetRawVal(), currentProvider); !isPreview {
			return nil, false
		}
		return func() internaldto.ExecutorOutput { return showPreviewMethods(ctx, extended) }, true
	}
	return nil, false
}

func showPreviewResources(
	ctx queryContext, service string, extended bool) internaldto.ExecutorOutput {
	tables := previewTables(service)
	rows := make(map[string]map[string]interface{}, len(tables))
	for i, tbl := range tables {
		row := map[string]interface{}{
			"id":   relationAddress(service, tbl.name),
			"name": tbl.name,
		}
		if extended {
			row["description"] = tbl.description
		}
		rows[fmt.Sprintf("%06d", i)] = row
	}
	return prepare(ctx, formulation.GetResourcesHeader(extended), rows, util.DefaultRowSort)
}

func showPreviewMethods(ctx queryContext, extended bool) internaldto.ExecutorOutput {
	columnOrder := []string{"MethodName", "RequiredParams", "SQLVerb"}
	if extended {
		columnOrder = append(columnOrder, "description")
	}
	row := map[string]interface{}{
		"MethodName":     selectMethodName,
		"RequiredParams": "",
		"SQLVerb":        strings.ToUpper(selectMethodName),
	}
	if extended {
		row["description"] = "select-only intrinsic method"
	}
	return prepare(ctx, columnOrder,
		map[string]map[string]interface{}{"000001": row}, util.DefaultRowSort)
}

// previewTables presents an extended service's relations. The iac service holds
// the converge relation alongside the blueprints, because a blueprint is a way
// of building a run's resources rather than a thing of its own.
func previewTables(service string) []table {
	if service == dynamicGraphService {
		return []table{{
			service:     dynamicGraphService,
			name:        graphRelation,
			isData:      true,
			description: "a graph query; the '" + specPredicate + "' predicate carries its wiring",
		}}
	}
	out := []table{{
		service:     iacService,
		name:        convergeRelation,
		isData:      true,
		description: "an idempotent converge run over a named collection",
	}}
	return append(out, blueprintTables()...)
}

// describePreviewTableFunc answers DESCRIBE for the extended preview services.
// Only a blueprint has columns worth describing: the two verb relations take a
// specification rather than a column list, and yield whatever the run reports.
func describePreviewTableFunc(
	ctx queryContext,
	node *sqlparser.DescribeTable,
	currentProvider string,
) (func() internaldto.ExecutorOutput, bool) {
	service, isPreview := previewService(
		node.Table.QualifierSecond.GetRawVal(),
		node.Table.Qualifier.GetRawVal(), currentProvider)
	if !isPreview {
		return nil, false
	}
	resource := node.Table.Name.GetRawVal()
	extended := isExtended(node.Extended)
	if service == iacService && !strings.EqualFold(resource, convergeRelation) {
		blueprint, ok := blueprintFor(resource)
		if !ok {
			return refuse(fmt.Errorf(
				"'%s.%s' has no blueprint '%s'; run SHOW RESOURCES IN %s.%s to list them",
				ProviderName, iacService, resource, ProviderName, iacService)), true
		}
		return func() internaldto.ExecutorOutput {
			return describeTable(ctx, table{columns: blueprintColumns(blueprint)}, extended)
		}, true
	}
	return refuse(fmt.Errorf(
		"relation '%s' takes a specification rather than columns; "+
			"run SHOW RESOURCES IN %s.%s for what it accepts",
		relationAddress(service, resource), ProviderName, service)), true
}
