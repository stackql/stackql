package intrinsic

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql-labs/omnisdk/pkg/query"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
)

// Legacy EXEC names a method explicitly; the SDK still validates and executes it.
func execPlan(ctx queryContext, node *sqlparser.Exec) (omnisdk.Plan, omnisdk.MethodSignature, error) {
	name := node.MethodName
	provider := resolveProvider(name.QualifierThird.GetRawVal(), ctx.GetCurrentProvider())
	bundle, ok := docProvider(provider)
	if !ok {
		return nil, nil, fmt.Errorf("EXEC requires a document provider")
	}
	address := UnstablePrefix + bundle + "." + name.QualifierSecond.GetRawVal() + "." + name.Qualifier.GetRawVal()
	method := name.Name.GetRawVal()
	opts, err := documentOptions(ctx)
	if err != nil {
		return nil, nil, err
	}
	signature, err := omnisdk.DescribeMethod(registryRoot(ctx), address, method, opts...)
	if err != nil {
		return nil, nil, err
	}
	params := map[string]string{}
	for _, variable := range node.ExecVarDefs {
		value, valueErr := execLiteral(variable.Val)
		if valueErr != nil {
			return nil, nil, valueErr
		}
		name := execInputName(signature, variable.ColIdent.GetRawVal())
		params[name] = fmt.Sprint(textValue(value))
	}
	body, err := execPayload(node.OptExecPayload, signature)
	if err != nil {
		return nil, nil, err
	}
	args, err := previewArgs(ctx, bundle, nil)
	if err != nil {
		return nil, nil, err
	}
	if node.Await || sqlparser.ExtractCommentDirectives(node.Comments).IsSet("AWAIT") {
		args.MutationCompletion, err = newOperationCompletion(ctx, address, false)
		if err != nil {
			return nil, nil, err
		}
	}
	graph, err := omnisdk.NewGraph([]omnisdk.Node{omnisdk.NewExecNode("exec", address, method, params, body)}, nil)
	if err != nil {
		return nil, nil, err
	}
	plan, err := omnisdk.NewGraphQuery(registryRoot(ctx), graph, args)
	return plan, signature, err
}

func execPayload(payload *sqlparser.ExecVarDef, signature omnisdk.MethodSignature) (map[string]any, error) {
	if payload == nil {
		return nil, nil //nolint:nilnil // No payload was supplied.
	}
	if !signature.TakesBody() {
		return nil, fmt.Errorf("method %s has no request body schema", signature.Method())
	}
	if !strings.EqualFold(payload.ColIdent.GetRawVal(), "json") {
		return nil, fmt.Errorf("EXEC supports only a JSON payload")
	}
	value, err := execLiteral(payload.Val)
	if err != nil {
		return nil, err
	}
	var body map[string]any
	if err = json.Unmarshal([]byte(fmt.Sprint(value)), &body); err != nil {
		return nil, err
	}
	return body, nil
}

func execLiteral(expr sqlparser.Expr) (any, error) {
	value, err := expression(expr)
	if err != nil {
		return nil, err
	}
	literal, ok := value.(query.Literal)
	if !ok {
		return nil, fmt.Errorf("EXEC arguments must be literal values")
	}
	return literal.Value(), nil
}

func execFunc(ctx queryContext, node *sqlparser.Exec) func() internaldto.ExecutorOutput {
	return func() internaldto.ExecutorOutput {
		plan, _, err := execPlan(ctx, node)
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		rows, err := plan.Open(executionContext(ctx))
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		result := drainMutation(rows)
		if result.GetError() == nil && (node.Await || sqlparser.ExtractCommentDirectives(node.Comments).IsSet("AWAIT")) {
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, nil)
		}
		return result
	}
}

// An EXEC used as a read source is a statement-local SDK relation.
func prepareExecRelations(
	ctx queryContext, original *sqlparser.Select,
) (*sqlparser.Select, map[string]omnisdk.Relation, error) {
	clone := *original
	relations := map[string]omnisdk.Relation{}
	var replace func(sqlparser.TableExpr) (sqlparser.TableExpr, error)
	replace = func(expr sqlparser.TableExpr) (sqlparser.TableExpr, error) {
		switch node := expr.(type) {
		case *sqlparser.ExecSubquery:
			return replace(&sqlparser.AliasedTableExpr{Expr: node})
		case *sqlparser.JoinTableExpr:
			cloned := *node
			var err error
			cloned.LeftExpr, err = replace(node.LeftExpr)
			if err != nil {
				return nil, err
			}
			cloned.RightExpr, err = replace(node.RightExpr)
			return &cloned, err
		case *sqlparser.AliasedTableExpr:
			return execRelation(ctx, node, relations)
		}
		return expr, nil
	}
	clone.From = make(sqlparser.TableExprs, len(original.From))
	for i, table := range original.From {
		var err error
		clone.From[i], err = replace(table)
		if err != nil {
			return nil, nil, err
		}
	}
	return &clone, relations, nil
}

func execRelation(
	ctx queryContext, node *sqlparser.AliasedTableExpr, relations map[string]omnisdk.Relation,
) (sqlparser.TableExpr, error) {
	execution, ok := node.Expr.(*sqlparser.ExecSubquery)
	if !ok {
		return node, nil
	}
	plan, signature, err := execPlan(ctx, execution.Exec)
	if err != nil {
		return nil, err
	}
	local := fmt.Sprintf("exec%d", len(relations))
	handle := UnstablePrefix + "omni_relations.sources." + local
	columns := signature.ColumnTypes()
	if columns == nil {
		columns = map[string]omnisdk.ColumnType{}
	}
	for _, name := range signature.Columns() {
		if _, present := columns[name]; !present {
			columns[name] = omnisdk.ColumnType{}
		}
	}
	relation, err := omnisdk.NewRelation(handle, nil, columns,
		func(runCtx context.Context, _ map[string]any) (omnisdk.Rows, error) {
			return plan.Open(runCtx)
		})
	if err != nil {
		return nil, err
	}
	relations[handle] = relation
	cloned := *node
	cloned.Expr = sqlparser.TableName{
		QualifierSecond: sqlparser.NewTableIdent(UnstablePrefix + "omni_relations"),
		Qualifier:       sqlparser.NewTableIdent("sources"),
		Name:            sqlparser.NewTableIdent(local),
	}
	return &cloned, nil
}

func execInputName(signature omnisdk.MethodSignature, name string) string {
	if metadata, ok := signature.(omnisdk.RequestMetadata); ok {
		for _, parameter := range signature.Params() {
			if metadata.InputName(parameter.Name()) == name {
				return parameter.Name()
			}
		}
	}
	return name
}
