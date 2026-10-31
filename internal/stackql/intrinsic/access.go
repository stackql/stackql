package intrinsic

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql-labs/omnisdk/public/authorization"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/accessmanagement"
)

func executionContext(ctx queryContext) context.Context {
	if host, ok := ctx.(interface{ GetExecutionContext() context.Context }); ok {
		return host.GetExecutionContext()
	}
	return context.Background()
}
func configureCaller(ctx queryContext, args *omnisdk.Args) error {
	args.RequireCaller = ctx.GetRuntimeContext().AccessConfigFile != ""
	if host, ok := ctx.(interface{ GetCaller() authorization.Session }); ok {
		args.Caller = host.GetCaller()
	}
	if args.RequireCaller && args.Caller == nil {
		return errors.New("authenticated caller is required")
	}
	return nil
}
func checkHost(ctx queryContext, resource, action string) error {
	if ctx.GetRuntimeContext().AccessConfigFile == "" {
		return nil
	}
	return authorization.Check(executionContext(ctx), resource, action)
}
func checkSQLNodes(ctx queryContext, node sqlparser.SQLNode) error {
	if ctx.GetRuntimeContext().AccessConfigFile == "" {
		return nil
	}
	return accessmanagement.CheckNodes(executionContext(ctx), node)
}

// identitySelect exposes only the established directory snapshot, never JWT claims.
func identitySelect(ctx queryContext, node *sqlparser.Select) (func() internaldto.ExecutorOutput, bool) {
	if !backendOnlySelect(node) {
		return nil, false
	}
	host, hasCaller := ctx.(interface{ GetCaller() authorization.Session })
	if !hasCaller || host.GetCaller() == nil {
		return nil, false
	}
	row := map[string]interface{}{}
	var columns []string
	for _, expression := range node.SelectExprs {
		expr, ok := expression.(*sqlparser.AliasedExpr)
		if !ok {
			return nil, false
		}
		function, ok := expr.Expr.(*sqlparser.FuncExpr)
		if !ok || len(function.Exprs) != 0 || !function.Qualifier.IsEmpty() {
			return nil, false
		}
		name := strings.ToLower(function.Name.GetRawVal())
		var value string
		switch name {
		case "whoami":
			value = host.GetCaller().Principal().ID()
		case "groups":
			data, _ := json.Marshal(host.GetCaller().Principal().Groups())
			value = string(data)
		default:
			return nil, false
		}
		output := expr.As.GetRawVal()
		if output == "" {
			output = name
		}
		columns = append(columns, output)
		row[output] = value
	}
	if len(columns) == 0 {
		return nil, false
	}
	return func() internaldto.ExecutorOutput {
		return prepare(ctx, columns, map[string]map[string]interface{}{"0": row}, util.DefaultRowSort)
	}, true
}
