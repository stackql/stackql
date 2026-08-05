package intrinsic

import (
	"fmt"
	"strings"

	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/any-sdk/public/formulation"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
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
	description string
	dataType    string
}

type table struct {
	service     string
	name        string
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
}

func GeneratePrimitiveFunc(
	ctx queryContext,
	stmt sqlparser.SQLNode,
) (func() internaldto.ExecutorOutput, bool) {
	current := ctx.GetCurrentProvider()
	switch node := stmt.(type) {
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

func GenerateStreamFunc(
	ctx queryContext,
	node *sqlparser.Select,
) (func() internaldto.ExecutorOutput, bool) {
	return selectFunc(ctx, node, ctx.GetCurrentProvider())
}

func IsProvider(name string) bool {
	return strings.EqualFold(strings.TrimSpace(name), ProviderName)
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
		ctx.SetCurrentProvider(ProviderName)
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
	switch strings.ToUpper(strings.TrimSpace(node.Type)) {
	case "SERVICES":
		if !IsProvider(resolveProvider(node.OnTable.Name.GetRawVal(), currentProvider)) {
			return nil, false
		}
		return func() internaldto.ExecutorOutput { return showServices(ctx, extended) }, true
	case "RESOURCES":
		serviceStr := node.OnTable.Name.GetRawVal()
		if !isService(node.OnTable.Qualifier.GetRawVal(), serviceStr, currentProvider) {
			return nil, false
		}
		return func() internaldto.ExecutorOutput { return showResources(ctx, serviceStr, extended) }, true
	case "METHODS":
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
	return prepare(ctx, formulation.GetServicesHeader(extended),
		map[string]map[string]interface{}{"000001": row}, util.DefaultRowSort)
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
	return prepare(ctx, formulation.GetResourcesHeader(extended), rows, util.DefaultRowSort)
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
		formulation.GetDescribeHeader(extended),
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
