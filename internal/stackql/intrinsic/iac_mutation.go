package intrinsic

import (
	"errors"
	"fmt"
	"strings"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
)

func iacMutationFunc(ctx queryContext, stmt sqlparser.Statement) (func() internaldto.ExecutorOutput, bool) {
	action, handled, err := planIaCMutation(ctx, stmt)
	if err != nil {
		return refuse(iacOutcomeError{cause: err}), true
	}
	return action, handled
}

func planIaCMutation(ctx queryContext, stmt sqlparser.Statement) (func() internaldto.ExecutorOutput, bool, error) {
	var target sqlparser.TableName
	switch node := stmt.(type) {
	case *sqlparser.Insert:
		target = node.Table
	case *sqlparser.Update:
		target = iacMutationTable(node.TableExprs)
	case *sqlparser.Delete:
		target = iacMutationTable(node.TableExprs)
	default:
		return nil, false, nil
	}
	service, ok := previewService(target.QualifierSecond.GetRawVal(),
		target.Qualifier.GetRawVal(), ctx.GetCurrentProvider())
	if !ok || service != iacService || !target.QualifierThird.IsEmpty() {
		return nil, false, nil
	}
	resource := strings.ToLower(target.Name.GetRawVal())
	operation := resource
	if _, deleting := stmt.(*sqlparser.Delete); deleting && resource == convergeRelation {
		operation = destroyRelation
	}
	if err := checkHost(ctx, "host/iac/"+operation, "execute"); err != nil {
		return nil, true, err
	}
	action, err := prepareIaCMutation(ctx, stmt, resource)
	if err != nil {
		return nil, true, err
	}
	return func() internaldto.ExecutorOutput {
		output := action()
		if runErr := output.GetError(); runErr != nil {
			return internaldto.NewErroneousExecutorOutput(iacOutcomeError{cause: runErr})
		}
		return output
	}, true, nil
}

func iacMutationTable(exprs sqlparser.TableExprs) sqlparser.TableName {
	if len(exprs) == 1 {
		if aliased, ok := exprs[0].(*sqlparser.AliasedTableExpr); ok {
			target, _ := aliased.Expr.(sqlparser.TableName)
			return target
		}
	}
	return sqlparser.TableName{}
}

//nolint:cyclop // Each SQL verb has its own supported clauses and input form.
func prepareIaCMutation(
	ctx queryContext, stmt sqlparser.Statement, resource string,
) (func() internaldto.ExecutorOutput, error) {
	switch node := stmt.(type) {
	case *sqlparser.Insert:
		if resource == convergeRelation {
			params, err := iacInsertParams(node)
			if err != nil {
				return nil, err
			}
			return convergeFunc(ctx, params, node.SelectExprs)
		}
	case *sqlparser.Delete:
		if resource == convergeRelation && len(node.Targets) == 0 && len(node.Partitions) == 0 &&
			len(node.OrderBy) == 0 && node.Limit == nil &&
			!sqlparser.ExtractCommentDirectives(node.Comments).IsSet("AWAIT") {
			params, err := iacWhereParams(node.Where)
			if err != nil {
				return nil, err
			}
			return destroyFunc(ctx, params, node.SelectExprs)
		}
	case *sqlparser.Update:
		if resource == recoverRelation && node.Action == sqlparser.UpdateStr && node.Ignore == "" &&
			len(updateSources(node)) == 0 && len(node.OrderBy) == 0 && node.Limit == nil {
			params, err := iacRecoveryParams(node)
			if err != nil {
				return nil, err
			}
			return recoverFunc(ctx, params, node.SelectExprs)
		}
	}
	return nil, fmt.Errorf("IaC accepts INSERT INTO iac.run, DELETE FROM iac.run, or UPDATE iac.recover")
}

func iacInsertParams(node *sqlparser.Insert) (map[string]string, error) {
	rows, ok := node.Rows.(sqlparser.Values)
	if !ok || len(rows) != 1 || len(rows[0]) != len(node.Columns) || len(node.OnDup) != 0 ||
		len(node.Partitions) != 0 || node.Ignore != "" {
		return nil, fmt.Errorf("INSERT INTO iac.run requires a column list and a single VALUES row")
	}
	params := make(map[string]string, len(node.Columns))
	for i, col := range node.Columns {
		if err := addIaCValue(params, col.GetRawVal(), rows[0][i]); err != nil {
			return nil, err
		}
	}
	return params, nil
}

func iacWhereParams(where *sqlparser.Where) (map[string]string, error) {
	params := map[string]string{}
	if where == nil {
		return params, nil
	}
	return params, addIaCPredicates(params, where.Expr)
}

func addIaCPredicates(params map[string]string, expr sqlparser.Expr) error {
	switch node := expr.(type) {
	case *sqlparser.AndExpr:
		if err := addIaCPredicates(params, node.Left); err != nil {
			return err
		}
		return addIaCPredicates(params, node.Right)
	case *sqlparser.ComparisonExpr:
		if col, ok := node.Left.(*sqlparser.ColName); ok && node.Operator == sqlparser.EqualStr {
			return addIaCValue(params, col.Name.GetRawVal(), node.Right)
		}
	}
	return fmt.Errorf("IaC WHERE accepts only literal equality predicates; cannot apply %s", sqlparser.String(expr))
}

func iacRecoveryParams(node *sqlparser.Update) (map[string]string, error) {
	params, err := iacWhereParams(node.Where)
	if err != nil {
		return nil, err
	}
	for name := range params {
		switch name {
		case collectionPredicate, statePredicate, runIDPredicate, "resource_key", "version":
		default:
			return nil, fmt.Errorf("unexpected recovery WHERE parameter %q", name)
		}
	}
	for _, assignment := range node.Exprs {
		name := strings.ToLower(assignment.Name.Name.GetRawVal())
		if name != "outcome" && name != "identity" {
			return nil, fmt.Errorf("recovery SET accepts outcome and identity")
		}
		if valueErr := addIaCValue(params, name, assignment.Expr); valueErr != nil {
			return nil, valueErr
		}
	}
	return params, nil
}

func addIaCValue(params map[string]string, name string, expr sqlparser.Expr) error {
	name = strings.ToLower(name)
	value, ok := expr.(*sqlparser.SQLVal)
	if !ok || (value.Type != sqlparser.StrVal && value.Type != sqlparser.IntVal && value.Type != sqlparser.FloatVal) {
		return fmt.Errorf("IaC parameter %q requires a string or numeric literal", name)
	}
	if _, exists := params[name]; exists {
		return fmt.Errorf("duplicate IaC parameter %q", name)
	}
	params[name] = string(value.Val)
	return nil
}

// Without RETURNING, drain the run to completion and report any deferred failure.
func streamIaCMutation(
	ctx queryContext, plan omnisdk.Plan, relation string, returning sqlparser.SelectExprs,
) internaldto.ExecutorOutput {
	if len(returning) > 0 {
		return streamPlan(ctx, plan, relation, returning)
	}
	rows, err := plan.Open(executionContext(ctx))
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	if relation == convergeRelation {
		rows = &iacRows{Rows: rows}
	}
	for rows.Next() {
	}
	if err = errors.Join(rows.Err(), rows.Close()); err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	return internaldto.NewNopEmptyExecutorOutput(nil)
}
