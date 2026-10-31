package intrinsic

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"maps"
	"sort"
	"strings"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/omnistaging"
)

// resolvedDocQuery retains analyzer-owned handles independently of SQL names.
// Handles are scoped to this statement, never installed in a global catalogue.
type resolvedDocQuery struct {
	docQuery
	relations map[string]omnisdk.Relation
}

func (q resolvedDocQuery) getRelations() map[string]omnisdk.Relation { return q.relations }
func withRelations(q docQuery, relations map[string]omnisdk.Relation) docQuery {
	if len(relations) == 0 {
		return q
	}
	return resolvedDocQuery{docQuery: q, relations: relations}
}
func queryRelations(q docQuery) map[string]omnisdk.Relation {
	if r, ok := q.(interface {
		getRelations() map[string]omnisdk.Relation
	}); ok {
		return r.getRelations()
	}
	return nil
}
func (q resolvedDocQuery) getReadHints() map[string]omnisdk.ReadHints {
	if h, ok := q.docQuery.(interface {
		getReadHints() map[string]omnisdk.ReadHints
	}); ok {
		return h.getReadHints()
	}
	return nil
}

const derivedRelationMaxDepth = 32

// resolveDerivedRelations replaces FROM subqueries with opaque read handles.
// Planning performs no reads; each child retains its own relational boundary.
func resolveDerivedRelations(
	ctx queryContext, original *sqlparser.Select, current string, depth int,
	inherited ...map[string]omnisdk.Relation,
) (*sqlparser.Select, map[string]omnisdk.Relation, error) {
	if depth >= derivedRelationMaxDepth {
		return nil, nil, fmt.Errorf("derived relations exceed nesting bound")
	}
	clone := *original
	clone.From = cloneRelationTables(original.From)
	relations := map[string]omnisdk.Relation{}
	if len(inherited) > 0 {
		for handle, relation := range inherited[0] {
			relations[handle] = relation
		}
	}
	var walk func(sqlparser.TableExpr) error
	walk = func(expr sqlparser.TableExpr) error {
		switch table := expr.(type) {
		case *sqlparser.JoinTableExpr:
			if err := walk(table.LeftExpr); err != nil {
				return err
			}
			return walk(table.RightExpr)
		case *sqlparser.ParenTableExpr:
			for _, child := range table.Exprs {
				if err := walk(child); err != nil {
					return err
				}
			}
		case *sqlparser.AliasedTableExpr:
			return resolveDerivedTable(ctx, table, current, depth, relations)
		}
		return nil
	}
	for _, expr := range clone.From {
		if err := walk(expr); err != nil {
			return nil, nil, err
		}
	}
	return &clone, relations, nil
}

func resolveDerivedTable(
	ctx queryContext, table *sqlparser.AliasedTableExpr, current string, depth int,
	relations map[string]omnisdk.Relation,
) error {
	sub, isSubquery := table.Expr.(*sqlparser.Subquery)
	if !isSubquery {
		return nil
	}
	if table.As.IsEmpty() {
		return fmt.Errorf("derived relation requires an alias")
	}
	names, open, err := planStatementRead(ctx, sub.Select, current, depth+1, maps.Clone(relations))
	if err != nil {
		return err
	}
	columns, err := derivedColumns(names)
	if err != nil {
		return err
	}
	localName := fmt.Sprintf("r%d", len(relations))
	handle := UnstablePrefix + "omni_relations.sources." + localName
	relation, err := omnisdk.NewRelation(handle, nil, columns, open)
	if err != nil {
		return err
	}
	relations[handle] = newOrderedRelation(relation, names)
	table.Expr = sqlparser.TableName{
		QualifierSecond: sqlparser.NewTableIdent(UnstablePrefix + "omni_relations"),
		Qualifier:       sqlparser.NewTableIdent("sources"),
		Name:            sqlparser.NewTableIdent(localName),
	}
	return nil
}

type derivedOpener = func(context.Context, map[string]any) (omnisdk.Rows, error)

func planDerivedRead(
	ctx queryContext, node *sqlparser.Select, translator docTranslator, children map[string]omnisdk.Relation,
) ([]string, derivedOpener, error) {
	if backendOnlySelect(node) {
		return planBackendRead(ctx, node)
	}
	if needsStaging(node) {
		staged, err := planStagedSelect(node, translator, ctx.GetASTFormatter())
		if err != nil {
			return nil, nil, err
		}
		staged = withStagedRelations(staged, children)
		if _, _, planErr := planDocQuery(ctx, staged.getSource()); planErr != nil {
			return nil, nil, planErr
		}
		open := func(runCtx context.Context, _ map[string]any) (omnisdk.Rows, error) {
			if cancelErr := runCtx.Err(); cancelErr != nil {
				return nil, cancelErr
			}
			return openStagedRelation(ctx, staged, runCtx)
		}
		return staged.getOutputNames(), open, nil
	}
	translated, err := translator.selectQuery(node)
	if err != nil {
		return nil, nil, err
	}
	translated = withRelations(translated, children)
	plan, _, err := planDocQuery(ctx, translated)
	if err != nil {
		return nil, nil, err
	}
	open := func(runCtx context.Context, _ map[string]any) (omnisdk.Rows, error) {
		return plan.Open(runCtx)
	}
	return translated.getOutputs(), open, nil
}

func derivedColumns(names []string) (map[string]omnisdk.ColumnType, error) {
	columns := map[string]omnisdk.ColumnType{}
	for _, name := range names {
		if name == starOutput {
			return nil, fmt.Errorf("derived relation must name its projected columns")
		}
		if _, duplicate := columns[name]; duplicate {
			return nil, fmt.Errorf("derived relation has duplicate column %s", name)
		}
		columns[name] = omnisdk.ColumnType{}
	}
	return columns, nil
}

// openStagedRelation consumes the child SQL result before releasing its query-owned
// table, retaining native numbers and NULL rather than client-formatted bytes.
func openStagedRelation(ctx queryContext, staged stagedSelect, runCtx context.Context) (omnisdk.Rows, error) {
	manager, err := getStagingManager(ctx)
	if err != nil {
		return nil, err
	}
	id, err := manager.Begin(runCtx)
	if err != nil {
		return nil, err
	}
	rows, readErr := materializeStagedRelation(ctx, manager, id, staged, runCtx)
	if releaseErr := manager.Release(context.WithoutCancel(runCtx), id); releaseErr != nil && readErr == nil {
		return nil, releaseErr
	}
	return rows, readErr
}

//nolint:rowserrcheck // materializeSQLRows consumes the transferred cursor and checks result.Err.
func materializeStagedRelation(ctx queryContext, manager omnistaging.Manager, id omnistaging.QueryID,
	staged stagedSelect, runCtx context.Context) (omnisdk.Rows, error) {
	read, err := stageRead(ctx, manager, id, staged, runCtx)
	if err != nil {
		return nil, err
	}
	result := read.Rows()
	return materializeSQLRows(runCtx, result, staged.getOutputNames(), read.TerminalError())
}

func materializeSQLRows(
	runCtx context.Context, result *sql.Rows, names []string, terminalErr error,
) (omnisdk.Rows, error) {
	defer result.Close()
	var err error
	var rows []omnisdk.Row
	numeric, boolean, err := sqlColumnKinds(result)
	if err != nil {
		return nil, err
	}
	for result.Next() {
		values := make([]any, len(names))
		pointers := make([]any, len(names))
		for i := range values {
			pointers[i] = &values[i]
		}
		if err = result.Scan(pointers...); err != nil {
			return nil, err
		}
		row := omnisdk.Row{}
		for i, name := range names {
			row[name] = nativeSQLValue(values[i], numeric[i], boolean[i])
		}
		rows = append(rows, row)
	}
	if err = result.Err(); err != nil {
		return nil, err
	}
	if err = runCtx.Err(); err != nil {
		return nil, err
	}
	return &materializedRelationRows{ctx: runCtx, rows: rows, terminalErr: terminalErr}, nil
}

type materializedRelationRows struct {
	ctx                     context.Context
	rows                    []omnisdk.Row
	current                 omnisdk.Row
	terminalErr, errorState error
	closed                  bool
}

func (r *materializedRelationRows) Next() bool {
	if r.closed || r.errorState != nil {
		return false
	}
	if err := r.ctx.Err(); err != nil {
		r.errorState = err
		return false
	}
	if len(r.rows) == 0 {
		r.errorState = r.terminalErr
		return false
	}
	r.current = r.rows[0]
	r.rows = r.rows[1:]
	return true
}
func (r *materializedRelationRows) Row() omnisdk.Row { return r.current }
func (r *materializedRelationRows) Err() error       { return r.errorState }
func (r *materializedRelationRows) Close() error     { r.closed = true; r.rows = nil; return nil }

func cloneRelationTables(tables sqlparser.TableExprs) sqlparser.TableExprs {
	result := make(sqlparser.TableExprs, len(tables))
	for i, table := range tables {
		switch node := table.(type) {
		case *sqlparser.AliasedTableExpr:
			clone := *node
			result[i] = &clone
		case *sqlparser.JoinTableExpr:
			clone := *node
			children := cloneRelationTables(sqlparser.TableExprs{node.LeftExpr, node.RightExpr})
			clone.LeftExpr, clone.RightExpr = children[0], children[1]
			result[i] = &clone
		case *sqlparser.ParenTableExpr:
			clone := *node
			clone.Exprs = cloneRelationTables(node.Exprs)
			result[i] = &clone
		default:
			result[i] = table
		}
	}
	return result
}

type resolvedStagedSelect struct {
	stagedSelect
	source docQuery
}

func (s resolvedStagedSelect) getSource() docQuery { return s.source }
func (s resolvedStagedSelect) renderAfterOffset(tableName string, applied int) string {
	if adjusted, ok := s.stagedSelect.(interface{ renderAfterOffset(string, int) string }); ok {
		return adjusted.renderAfterOffset(tableName, applied)
	}
	return s.renderOuter(tableName)
}
func withStagedRelations(s stagedSelect, relations map[string]omnisdk.Relation) stagedSelect {
	if len(relations) == 0 {
		return s
	}
	return resolvedStagedSelect{stagedSelect: s, source: withRelations(s.getSource(), relations)}
}

// Expand stars from declared metadata before crossing a derived relation's
// schema boundary, including empty reads. The statement already belongs to us.
func expandDerivedStars(
	ctx queryContext, node *sqlparser.Select, translator docTranslator, relations map[string]omnisdk.Relation,
) error {
	hasStar := false
	for _, expr := range node.SelectExprs {
		if _, isStar := expr.(*sqlparser.StarExpr); isStar {
			hasStar = true
		}
	}
	if !hasStar {
		return nil
	}
	source, err := translator.source(node)
	if err != nil {
		return err
	}
	aliases, err := whereAliasOutputs(node)
	if err != nil {
		return err
	}
	var expressions sqlparser.SelectExprs
	for _, expr := range node.SelectExprs {
		star, isStar := expr.(*sqlparser.StarExpr)
		if !isStar {
			expressions = append(expressions, expr)
			continue
		}
		expanded, expandErr := resolvedStarColumns(ctx, source, star, relations, aliases...)
		if expandErr != nil {
			return expandErr
		}
		expressions = append(expressions, expanded...)
	}
	node.SelectExprs = expressions
	return nil
}

func declaredRelationColumns(
	ctx queryContext, handle string, relations map[string]omnisdk.Relation,
) ([]string, error) {
	var table omnisdk.Table
	if relation, present := relations[handle]; present {
		if ordered, ok := relation.(orderedRelation); ok {
			return ordered.columnOrder(), nil
		}
		table = relation
	} else {
		opts, err := documentOptions(ctx)
		if err != nil {
			return nil, err
		}
		table, err = omnisdk.DescribeTable(registryRoot(ctx), handle, opts...)
		if err != nil {
			return nil, err
		}
	}
	columns := map[string]bool{}
	for _, method := range table.Methods() {
		for _, column := range method.Columns() {
			columns[column] = true
		}
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("derived relation star requires a declared schema for %s", handle)
	}
	names := make([]string, 0, len(columns))
	for name := range columns {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// backendOnlySelect recognizes the parser's synthetic one-row relation. It
// remains a database read handle when joined to provider or derived relations.
func backendOnlySelect(node *sqlparser.Select) bool {
	if len(node.From) != 1 {
		return false
	}
	table, isTable := node.From[0].(*sqlparser.AliasedTableExpr)
	if !isTable {
		return false
	}
	name, isName := table.Expr.(sqlparser.TableName)
	return isName && name.Name.GetRawVal() == "dual" && name.Qualifier.IsEmpty() && name.QualifierSecond.IsEmpty()
}
func planBackendRead(ctx queryContext, node *sqlparser.Select) ([]string, derivedOpener, error) {
	_, names, err := selectOutputs(node.SelectExprs)
	if err != nil {
		return nil, nil, err
	}
	formatter := sqlparser.NewTrackedBuffer(ctx.GetASTFormatter())
	if node.With != nil {
		formatter.Myprintf("%v ", node.With)
	}
	formatter.Myprintf("select ")
	if node.Distinct {
		formatter.Myprintf("distinct ")
	}
	formatter.Myprintf("%v%v%v%v%v%v", node.SelectExprs, node.Where, node.GroupBy, node.Having, node.OrderBy, node.Limit)
	statement := formatter.String()
	open := func(runCtx context.Context, _ map[string]any) (omnisdk.Rows, error) {
		db, dbErr := ctx.GetSQLEngine().GetDB()
		if dbErr != nil {
			return nil, dbErr
		}
		result, queryErr := db.QueryContext(runCtx, statement)
		if queryErr != nil {
			return nil, queryErr
		}
		return materializeSQLRows(runCtx, result, names, nil)
	}
	return names, open, nil
}

func sqlColumnKinds(rows *sql.Rows) ([]bool, []bool, error) {
	types, err := rows.ColumnTypes()
	if err != nil {
		return nil, nil, err
	}
	numeric := make([]bool, len(types))
	boolean := make([]bool, len(types))
	for i, column := range types {
		switch strings.ToLower(column.DatabaseTypeName()) {
		case "bool", "boolean":
			boolean[i] = true
		case "numeric", "decimal", "integer", "int", "int2", "int4", "int8", "bigint", "smallint",
			"float4", "float8", "real", "double precision":
			numeric[i] = true
		}
	}
	return numeric, boolean, nil
}

// Restore declared SQL types at the database boundary: SQLite returns booleans
// as integers, and drivers may return NUMERIC as bytes or text.
func nativeSQLValue(value any, numeric, boolean bool) any {
	if boolean {
		if n, ok := value.(int64); ok {
			return n != 0
		}
	}
	if raw, ok := value.([]byte); ok {
		value = string(raw)
	}
	if text, ok := value.(string); ok && numeric {
		return json.Number(text)
	}
	return value
}
