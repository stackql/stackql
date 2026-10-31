package intrinsic

import (
	"fmt"
	"strings"

	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	dto "github.com/stackql/stackql/internal/stackql/runtimeconfig"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/typing"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

const (
	ProviderName    = "stackql_preview"
	ProviderVersion = "internal"
)

const (
	auditService     = "audit"
	auditTitle       = "omnisdk audited resources"
	selectMethodName = "select"
	columnType       = "text"
)

type column struct {
	name        string
	sourceName  string
	description string
	dataType    string
}

type table struct {
	service     string
	name        string
	isData      bool
	description string
	columns     []column
}

func allTables() ([]table, error) {
	return dataTables()
}

type queryContext interface {
	GetCurrentProvider() string
	SetCurrentProvider(string)
	GetTypingConfig() typing.Config
	GetAuthContext(providerName string) (*dto.AuthCtx, error)
	GetRuntimeContext() dto.RuntimeCtx
	GetSQLEngine() sqlengine.SQLEngine
	GetASTFormatter() sqlparser.NodeFormatter
}

func GeneratePrimitiveFunc(
	ctx queryContext,
	stmt sqlparser.SQLNode,
) (func() internaldto.ExecutorOutput, bool) {
	current := ctx.GetCurrentProvider()
	switch node := stmt.(type) {
	case *sqlparser.Auth:
		return authFunc(ctx, node), true
	case *sqlparser.AuthRevoke:
		return authRevokeFunc(ctx, node), true
	case *sqlparser.Sleep:
		return sleepFunc(ctx, node), true
	case *sqlparser.Use:
		return useFunc(ctx, node)
	case *sqlparser.Show:
		return showFunc(ctx, node, current)
	case *sqlparser.DescribeTable:
		return describeTableFunc(ctx, node, current)
	case *sqlparser.DescribeMethod:
		return describeMethodFunc(ctx, node, current)
	}
	return nil, false
}

// GenerateStreamFunc plans a statement omnisdk runs end to end: a SELECT over
// its relations, or a mutation whose target is a document-driven relation.
func GenerateStreamFunc(
	ctx queryContext,
	stmt sqlparser.Statement,
) (func() internaldto.ExecutorOutput, bool) {
	switch node := stmt.(type) {
	case *sqlparser.Union:
		return statementReadFunc(ctx, node), true
	case *sqlparser.ParenSelect:
		return statementReadFunc(ctx, node), true
	}
	if node, isExec := stmt.(*sqlparser.Exec); isExec {
		return execFunc(ctx, node), true
	}
	if execute, handled := materializedFunc(ctx, stmt); handled {
		return execute, true
	}
	if node, isSelect := stmt.(*sqlparser.Select); isSelect {
		if action, handled := identitySelect(ctx, node); handled {
			return action, true
		}
		action, handled := selectFunc(ctx, node, ctx.GetCurrentProvider())
		if !handled && ctx.GetRuntimeContext().AccessConfigFile != "" && backendOnlySelect(node) {
			return statementReadFunc(ctx, node), true
		}
		return action, handled
	}
	if action, handled := storedInsertSelectFunc(ctx, stmt); handled {
		return action, true
	}
	return docMutationFunc(ctx, stmt, ctx.GetCurrentProvider())
}

func IsProvider(name string) bool {
	if strings.EqualFold(strings.TrimSpace(name), ProviderName) {
		return true
	}
	_, isDoc := docProvider(name)
	return isDoc
}

func resolveProvider(providerName string, currentProvider string) string {
	if strings.TrimSpace(providerName) != "" {
		return providerName
	}
	return currentProvider
}

func isService(providerName, serviceStr, currentProvider string) bool {
	if !IsProvider(resolveProvider(providerName, currentProvider)) {
		return false
	}
	return strings.EqualFold(serviceStr, auditService)
}

func lookupTable(providerName, serviceStr, resourceStr, currentProvider string) (table, bool) {
	if !isService(providerName, serviceStr, currentProvider) {
		return table{}, false
	}
	registry, err := allTables()
	if err != nil {
		return table{}, false
	}
	for _, tbl := range registry {
		if strings.EqualFold(serviceStr, tbl.service) && strings.EqualFold(resourceStr, tbl.name) {
			return tbl, true
		}
	}
	return table{}, false
}

func isExtended(extended string) bool {
	return strings.EqualFold(strings.TrimSpace(extended), "extended")
}

func useFunc(ctx queryContext, node *sqlparser.Use) (func() internaldto.ExecutorOutput, bool) {
	if !IsProvider(node.DBName.GetRawVal()) {
		return nil, false
	}
	return func() internaldto.ExecutorOutput {
		ctx.SetCurrentProvider(node.DBName.GetRawVal())
		return internaldto.NewExecutorOutput(nil, nil, nil, nil, nil)
	}, true
}

//nolint:gocritic // a switch on node.Type reads better than the alternatives
func showFunc(
	ctx queryContext,
	node *sqlparser.Show,
	currentProvider string,
) (func() internaldto.ExecutorOutput, bool) {
	extended := isExtended(node.Extended)
	if fn, isPreview := showPreviewFunc(ctx, node, currentProvider, extended); isPreview {
		return fn, true
	}
	switch strings.ToUpper(strings.TrimSpace(node.Type)) {
	case "INSERT":
		if bundle, isDoc := docProvider(resolveProvider(node.OnTable.QualifierSecond.GetRawVal(), currentProvider)); isDoc {
			return func() internaldto.ExecutorOutput { return showDocInsert(ctx, node, bundle) }, true
		}
	case "SERVICES":
		provider := resolveProvider(node.OnTable.Name.GetRawVal(), currentProvider)
		if bundle, isDoc := docProvider(provider); isDoc {
			return func() internaldto.ExecutorOutput { return showDocServices(ctx, bundle, extended, node) }, true
		}
		if !IsProvider(provider) {
			return nil, false
		}
		return func() internaldto.ExecutorOutput { return showServices(ctx, extended) }, true
	case "RESOURCES":
		serviceStr := node.OnTable.Name.GetRawVal()
		if bundle, isDoc := docProvider(resolveProvider(node.OnTable.Qualifier.GetRawVal(), currentProvider)); isDoc {
			return func() internaldto.ExecutorOutput {
				return showDocResources(ctx, bundle, serviceStr, extended, node)
			}, true
		}
		if !isService(node.OnTable.Qualifier.GetRawVal(), serviceStr, currentProvider) {
			return nil, false
		}
		return func() internaldto.ExecutorOutput { return showResources(ctx, serviceStr, extended) }, true
	case "METHODS":
		if bundle, isDoc := docProvider(
			resolveProvider(node.OnTable.QualifierSecond.GetRawVal(), currentProvider)); isDoc {
			service := node.OnTable.Qualifier.GetRawVal()
			resource := node.OnTable.Name.GetRawVal()
			return func() internaldto.ExecutorOutput {
				isUnstable := strings.HasPrefix(
					resolveProvider(node.OnTable.QualifierSecond.GetRawVal(), currentProvider),
					UnstablePrefix,
				)
				return showDocMethods(ctx, bundle, service, resource, extended, isUnstable)
			}, true
		}
		tbl, ok := lookupTable(
			node.OnTable.QualifierSecond.GetRawVal(),
			node.OnTable.Qualifier.GetRawVal(),
			node.OnTable.Name.GetRawVal(),
			currentProvider,
		)
		if !ok {
			return nil, false
		}
		return func() internaldto.ExecutorOutput { return showMethods(ctx, tbl, extended) }, true
	}
	return nil, false
}

func describeTableFunc(
	ctx queryContext,
	node *sqlparser.DescribeTable,
	currentProvider string,
) (func() internaldto.ExecutorOutput, bool) {
	if fn, isPreview := describePreviewTableFunc(ctx, node, currentProvider); isPreview {
		return fn, true
	}
	if fn, stored := describeStoredFunc(ctx, node, currentProvider); stored {
		return fn, true
	}
	if bundle, ok := docProvider(resolveProvider(node.Table.QualifierSecond.GetRawVal(), currentProvider)); ok {
		return func() internaldto.ExecutorOutput {
			return describeDocument(ctx, bundle, node.Table.Qualifier.GetRawVal(),
				node.Table.Name.GetRawVal(), "", isExtended(node.Extended))
		}, true
	}
	tbl, ok := lookupTable(
		node.Table.QualifierSecond.GetRawVal(),
		node.Table.Qualifier.GetRawVal(),
		node.Table.Name.GetRawVal(),
		currentProvider,
	)
	if !ok {
		return nil, false
	}
	extended := isExtended(node.Extended)
	return func() internaldto.ExecutorOutput { return describeTable(ctx, tbl, extended) }, true
}

func describeMethodFunc(
	ctx queryContext,
	node *sqlparser.DescribeMethod,
	currentProvider string,
) (func() internaldto.ExecutorOutput, bool) {
	if bundle, ok := docProvider(resolveProvider(node.Provider.GetRawVal(), currentProvider)); ok {
		return func() internaldto.ExecutorOutput {
			return describeDocument(ctx, bundle, node.Service.GetRawVal(),
				node.Resource.GetRawVal(), node.Method.GetRawVal(), isExtended(node.Extended))
		}, true
	}
	tbl, ok := lookupTable(
		node.Provider.GetRawVal(),
		node.Service.GetRawVal(),
		node.Resource.GetRawVal(),
		currentProvider,
	)
	if !ok {
		return nil, false
	}
	methodName := node.Method.GetRawVal()
	columns, ok := tbl.methodColumns(methodName)
	if !ok {
		return func() internaldto.ExecutorOutput {
			return internaldto.NewErroneousExecutorOutput(
				fmt.Errorf(
					"relation '%s.%s.%s' has no method '%s'; run SHOW METHODS to list them",
					ProviderName, tbl.service, tbl.name, methodName,
				),
			)
		}, true
	}
	extended := isExtended(node.Extended)
	return func() internaldto.ExecutorOutput { return describeMethod(ctx, columns, extended) }, true
}

func prepare(
	ctx queryContext,
	columnOrder []string,
	rows map[string]map[string]interface{},
	rowSort func(map[string]map[string]interface{}) []string,
) internaldto.ExecutorOutput {
	return util.PrepareResultSet(
		internaldto.NewPrepareResultSetDTO(
			nil, rows, columnOrder, rowSort, nil, nil, ctx.GetTypingConfig(),
		),
	)
}

func showServices(ctx queryContext, extended bool) internaldto.ExecutorOutput {
	row := map[string]interface{}{
		"id":    fmt.Sprintf("%s:%s", auditService, ProviderVersion),
		"name":  auditService,
		"title": auditTitle,
	}
	if extended {
		row["description"] = auditTitle
		row["version"] = ProviderVersion
		row["preferred"] = nil
	}
	rows := map[string]map[string]interface{}{"000001": row}
	appendPreviewServices(rows, extended)
	return prepare(ctx, util.GetServicesHeader(extended), rows, util.DefaultRowSort)
}

func showResources(ctx queryContext, serviceStr string, extended bool) internaldto.ExecutorOutput {
	all, err := allTables()
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	var registry []table
	for _, tbl := range all {
		if strings.EqualFold(serviceStr, tbl.service) {
			registry = append(registry, tbl)
		}
	}
	rows := make(map[string]map[string]interface{}, len(registry))
	for i, tbl := range registry {
		row := map[string]interface{}{
			"id":   fmt.Sprintf("%s.%s.%s", ProviderName, tbl.service, tbl.name),
			"name": tbl.name,
		}
		if extended {
			row["description"] = tbl.description
		}
		rows[fmt.Sprintf("%06d", i)] = row
	}
	return prepare(ctx, util.GetResourcesHeader(extended), rows, util.DefaultRowSort)
}

func showMethods(ctx queryContext, tbl table, extended bool) internaldto.ExecutorOutput {
	columnOrder := []string{"MethodName", "RequiredParams", "SQLVerb"}
	if extended {
		columnOrder = append(columnOrder, "description")
	}
	rows := make(map[string]map[string]interface{})
	for i, meth := range tbl.methods() {
		row := map[string]interface{}{
			"MethodName":     meth.name,
			"RequiredParams": strings.Join(meth.requiredParams, ", "),
			"SQLVerb":        strings.ToUpper(selectMethodName),
		}
		if extended {
			row["description"] = meth.description
		}
		rows[fmt.Sprintf("%06d", i)] = row
	}
	return prepare(ctx, columnOrder, rows, util.DefaultRowSort)
}

func describeTable(ctx queryContext, tbl table, extended bool) internaldto.ExecutorOutput {
	return prepare(
		ctx,
		util.GetDescribeHeader(extended),
		columnRows(tbl.columns, extended, nil),
		util.DescribeRowSort,
	)
}

func describeMethod(ctx queryContext, cols []column, extended bool) internaldto.ExecutorOutput {
	columnOrder := []string{"name", "type", "param_type", "shape"}
	if extended {
		columnOrder = append(columnOrder, "description")
	}
	rows := columnRows(cols, extended, func(row map[string]interface{}) {
		row["param_type"] = "response"
		row["shape"] = row["type"]
	})
	return prepare(ctx, columnOrder, rows, util.DescribeRowSort)
}

func columnRows(
	cols []column,
	extended bool,
	decorate func(map[string]interface{}),
) map[string]map[string]interface{} {
	rows := make(map[string]map[string]interface{}, len(cols))
	for i, col := range cols {
		row := map[string]interface{}{
			"name": col.name,
			"type": col.reportedType(),
		}
		if extended {
			row["description"] = col.description
		}
		if decorate != nil {
			decorate(row)
		}
		rows[fmt.Sprintf("%06d", i)] = row
	}
	return rows
}

// ValidateStatement builds an Omni plan without opening its cursor or dispatching requests.
func ValidateStatement(ctx queryContext, stmt sqlparser.Statement) (bool, error) {
	dialect, err := backendDialect(ctx)
	if err != nil {
		return true, err
	}
	translator := newDocTranslator(ctx.GetCurrentProvider(), dialect)
	var translated docQuery
	switch node := stmt.(type) {
	case *sqlparser.Select:
		handled, sourceErr := fromDocProviders(node.From, ctx.GetCurrentProvider())
		if !handled {
			return false, sourceErr
		}
		if sourceErr != nil {
			return true, sourceErr
		}
		_, _, err = planStatementRead(ctx, node, ctx.GetCurrentProvider(), 0, nil)
		return true, err
	case *sqlparser.Insert, *sqlparser.Update, *sqlparser.Delete:
		translated, err = translator.mutation(stmt)
	case *sqlparser.Exec:
		_, _, err = execPlan(ctx, node)
		return true, err
	default:
		return false, nil
	}
	if err != nil {
		return true, err
	}
	_, _, err = planDocQuery(ctx, translated)
	return true, err
}
