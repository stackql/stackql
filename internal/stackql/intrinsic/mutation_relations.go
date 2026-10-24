package intrinsic

import (
	"fmt"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

// INSERT sources use the same opaque relation handles as ordinary SELECTs.
// Clone the mutation so planning does not modify the caller's syntax tree.
func prepareMutationRelations(
	ctx queryContext, stmt sqlparser.Statement, current string,
) (sqlparser.Statement, map[string]omnisdk.Relation, error) {
	if deletion, ok := stmt.(*sqlparser.Delete); ok && len(deletion.SelectExprs) > 0 {
		if sqlparser.ExtractCommentDirectives(deletion.Comments).IsSet("AWAIT") {
			return nil, nil, fmt.Errorf("returning from asynchronous delete is disallowed")
		}
	}
	insert, ok := stmt.(*sqlparser.Insert)
	if !ok {
		return stmt, nil, nil
	}
	source, ok := insert.Rows.(*sqlparser.Select)
	if !ok || literalSource(source.From) {
		return stmt, nil, nil
	}
	prepared, relations, err := prepareMutationSource(ctx, source, current)
	if err != nil {
		return nil, nil, err
	}

	dialect, err := backendDialect(ctx)
	if err != nil {
		return nil, nil, err
	}
	translator := newDocTranslator(current, dialect)
	if err = expandDerivedStars(ctx, prepared, translator, relations); err != nil {
		return nil, nil, err
	}
	local := fmt.Sprintf("r%d", len(relations))
	sourcePlan := *prepared
	sourcePlan.SelectExprs = make(sqlparser.SelectExprs, len(prepared.SelectExprs))
	projected := make(sqlparser.SelectExprs, len(prepared.SelectExprs))
	for i, selection := range prepared.SelectExprs {
		expression, isExpression := selection.(*sqlparser.AliasedExpr)
		if !isExpression {
			return nil, nil, fmt.Errorf("insert source must name its columns")
		}
		field := expression.As.GetRawVal()
		if field == "" {
			field = fmt.Sprintf("_omni_insert_%d", i)
		}
		cloned := *expression
		cloned.As = sqlparser.NewColIdent(field)
		sourcePlan.SelectExprs[i] = &cloned
		projected[i] = &sqlparser.AliasedExpr{Expr: &sqlparser.ColName{
			Name:      sqlparser.NewColIdent(field),
			Qualifier: sqlparser.TableName{Name: sqlparser.NewTableIdent(local)},
		}}
	}
	names, open, err := planDerivedRead(ctx, &sourcePlan, translator, relations)
	if err != nil {
		return nil, nil, err
	}
	columns, err := derivedColumns(names)
	if err != nil {
		return nil, nil, err
	}
	handle := UnstablePrefix + "omni_relations.sources." + local
	relation, err := omnisdk.NewRelation(handle, nil, columns, open)
	if err != nil {
		return nil, nil, err
	}
	relations[handle] = relation
	clone := *insert
	clone.Rows = &sqlparser.Select{SelectExprs: projected,
		From: sqlparser.TableExprs{&sqlparser.AliasedTableExpr{Expr: sqlparser.TableName{
			QualifierSecond: sqlparser.NewTableIdent(UnstablePrefix + "omni_relations"),
			Qualifier:       sqlparser.NewTableIdent("sources"), Name: sqlparser.NewTableIdent(local),
		}}}}
	return &clone, relations, nil
}

func prepareMutationSource(ctx queryContext, source *sqlparser.Select, current string) (
	*sqlparser.Select, map[string]omnisdk.Relation, error,
) {
	storedSource, stored, err := prepareStoredRelations(ctx, source)
	if err != nil {
		return nil, nil, err
	}
	expanded, err := expandDocumentViews(ctx, storedSource, current)
	if err != nil {
		return nil, nil, err
	}
	return resolveDerivedRelations(ctx, expanded, current, 0, stored)
}
