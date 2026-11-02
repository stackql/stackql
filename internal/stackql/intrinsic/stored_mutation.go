package intrinsic

import (
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/sql_system"
)

// Stored table writes use backend SQL, including typed resolved expressions.
// Provider writes and queries with nested sources retain their existing planners.
//
//nolint:gocognit // Resolve the target before constructing its single backend executor.
func storedMutationFunc(ctx queryContext, statement sqlparser.Statement) (func() internaldto.ExecutorOutput, bool) {
	var tables sqlparser.TableExprs
	switch node := statement.(type) {
	case *sqlparser.Update:
		tables = node.TableExprs
	case *sqlparser.Delete:
		tables = node.TableExprs
	default:
		return nil, false
	}
	if len(tables) != 1 {
		return nil, false
	}
	table, ok := tables[0].(*sqlparser.AliasedTableExpr)
	if !ok {
		return nil, false
	}
	name, ok := table.Expr.(sqlparser.TableName)
	if !ok {
		return nil, false
	}
	host, ok := ctx.(interface{ GetSQLSystem() sql_system.SQLSystem })
	if !ok || host.GetSQLSystem() == nil {
		return nil, false
	}
	system := host.GetSQLSystem()
	metadata, exists := system.GetPhysicalTableByName(qualifiedName(name))
	if !exists {
		return nil, false
	}
	nested := false
	_ = sqlparser.Walk(func(node sqlparser.SQLNode) (bool, error) {
		if _, subquery := node.(*sqlparser.Subquery); subquery {
			nested = true
			return false, nil
		}
		return true, nil
	}, statement)
	if nested {
		return nil, false
	}
	inner := backendExpressionFormatter(ctx.GetASTFormatter())
	formatted := sqlparser.NewTrackedBuffer(func(buf *sqlparser.TrackedBuffer, node sqlparser.SQLNode) {
		if object, isTable := node.(sqlparser.TableName); isTable && object == name {
			buf.WriteString(system.DelimitFullyQualifiedRelationName(metadata.GetName()))
			return
		}
		inner(buf, node)
	})
	formatted.Myprintf("%v", statement)
	return func() internaldto.ExecutorOutput {
		if err := checkHost(ctx, "host/backend", "write"); err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		db, err := ctx.GetSQLEngine().GetDB()
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		_, err = db.ExecContext(executionContext(ctx), formatted.String())
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		messages := internaldto.NewBackendMessages([]string{"exec completed"})
		return internaldto.NewExecutorOutput(nil, nil, nil, messages, nil)
	}, true
}
