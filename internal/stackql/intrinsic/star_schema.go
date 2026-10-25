package intrinsic

import (
	"slices"
	"strings"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql-labs/omnisdk/pkg/query"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

// resolvedStarColumns shares operation selection with streamed execution. The
// host only turns the resulting projection metadata back into SQL AST columns.
func resolvedStarColumns(ctx queryContext, source translatedSource, star *sqlparser.StarExpr,
	relations map[string]omnisdk.Relation, aliasOutputs ...query.Output) (sqlparser.SelectExprs, error) {
	tables, err := starTables(ctx, source, relations)
	if err != nil {
		return nil, err
	}
	outputs := []query.Output{query.NewOutput("", query.NewStar(star.TableName.Name.GetRawVal()))}
	outputs = append(outputs, aliasOutputs...)
	q, err := query.New(source.joins(), source.where(), outputs)
	if err != nil {
		return nil, err
	}
	q, err = bindServerEnvironment(q, tables)
	if err != nil {
		return nil, err
	}
	dialect, err := backendDialect(ctx)
	if err != nil {
		return nil, err
	}
	resolved, err := omnisdk.ResolveIn(q, tables, dialect.catalogue())
	if err != nil {
		return nil, err
	}
	var methods map[string]omnisdk.MethodSignature
	if selected, ok := resolved.(omnisdk.SelectedMethods); ok {
		methods = selected.SelectedMethods()
	}
	if methods == nil {
		methods = map[string]omnisdk.MethodSignature{}
	}
	for alias, table := range tables {
		if _, ordered := table.(orderedRelation); ordered {
			methods[alias] = table.Methods()[0]
		}
	}
	excluded := map[string]bool{}
	for _, output := range aliasOutputs {
		excluded[output.Name()] = true
	}
	return starSQLColumns(resolved.Graph(), methods, excluded), nil
}

func starTables(
	ctx queryContext, source translatedSource, relations map[string]omnisdk.Relation,
) (map[string]omnisdk.Table, error) {
	tables := map[string]omnisdk.Table{}
	for _, join := range source.joins() {
		resource := join.Resource()
		if _, function := resource.(query.TableFunction); function {
			continue
		}
		if relation, present := relations[resource.Handle()]; present {
			tables[resource.Alias()] = relation
			continue
		}
		table, err := omnisdk.DescribeTable(registryRoot(ctx), resource.Handle())
		if err != nil {
			return nil, err
		}
		tables[resource.Alias()] = table
	}
	return tables, nil
}

func starScopeNames(method omnisdk.MethodSignature) []string {
	if declared, ok := method.(omnisdk.ServerBindings); ok {
		return declared.ServerVariables()
	}
	return nil
}

func starSQLColumns(
	graph omnisdk.Graph, methods map[string]omnisdk.MethodSignature, excluded ...map[string]bool,
) sqlparser.SelectExprs {
	var expressions sqlparser.SelectExprs
	for _, projection := range graph.Projections() {
		var names []string
		for _, column := range projection.Columns() {
			if !strings.HasPrefix(column.Out(), "\x00") && (len(excluded) == 0 || !excluded[0][column.Out()]) {
				names = append(names, column.Out())
			}
		}
		if len(names) == 0 {
			continue
		}
		// Scope columns describe host output; they do not introduce value producers.
		names = append(names, starScopeNames(methods[projection.Alias()])...)
		if method, ok := methods[projection.Alias()].(orderedSignature); ok {
			names = method.Columns()
		} else {
			slices.Sort(names)
		}
		for _, name := range slices.Compact(names) {
			expressions = append(expressions, &sqlparser.AliasedExpr{Expr: &sqlparser.ColName{
				Name:      sqlparser.NewColIdent(name),
				Qualifier: sqlparser.TableName{Name: sqlparser.NewTableIdent(projection.Alias())},
			}})
		}
	}
	return expressions
}
