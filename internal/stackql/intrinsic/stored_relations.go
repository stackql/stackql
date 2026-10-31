package intrinsic

import (
	"context"
	"fmt"
	"maps"
	"strings"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/sql_system"
)

// storedRelationResolver resolves metadata before provider discovery. Backend
// relations become ordinary read handles; views keep their own query boundary.
type storedRelationResolver interface {
	Prepare(*sqlparser.Select) (*sqlparser.Select, map[string]omnisdk.Relation, error)
}
type storedResolver struct {
	ctx       queryContext
	system    sql_system.SQLSystem
	relations map[string]omnisdk.Relation
	active    map[string]bool
	ctes      map[string]sqlparser.SelectStatement
}

func describeStoredFunc(
	ctx queryContext, node *sqlparser.DescribeTable, current string,
) (func() internaldto.ExecutorOutput, bool) {
	host, ok := ctx.(interface{ GetSQLSystem() sql_system.SQLSystem })
	if !ok || host.GetSQLSystem() == nil {
		return nil, false
	}
	system := host.GetSQLSystem()
	name := qualifiedName(node.Table)
	_, view := system.GetViewByName(name)
	materialized, mv := system.GetMaterializedViewByName(name)
	physical, tableExists := system.GetPhysicalTableByName(name)
	if !view && !mv && !tableExists {
		return nil, false
	}
	return func() internaldto.ExecutorOutput {
		var columns []column
		if view {
			selectNode := &sqlparser.Select{
				SelectExprs: sqlparser.SelectExprs{&sqlparser.StarExpr{}},
				From:        sqlparser.TableExprs{&sqlparser.AliasedTableExpr{Expr: node.Table}},
			}
			names, _, err := planStatementRead(ctx, selectNode, current, 0, nil)
			if err != nil {
				return internaldto.NewErroneousExecutorOutput(err)
			}
			for _, name := range names {
				columns = append(columns, column{name: name})
			}
		} else {
			metadata := physical
			if mv {
				metadata = materialized
			}
			for _, declared := range metadata.GetColumns() {
				columns = append(columns, column{name: declared.GetName(), dataType: declared.GetType()})
			}
		}
		return describeTable(ctx, table{columns: columns}, isExtended(node.Extended))
	}, true
}

func newStoredResolver(ctx queryContext, system sql_system.SQLSystem) storedRelationResolver {
	return &storedResolver{ctx: ctx, system: system, relations: map[string]omnisdk.Relation{}, active: map[string]bool{}}
}
func prepareStoredRelations(
	ctx queryContext, node *sqlparser.Select,
) (*sqlparser.Select, map[string]omnisdk.Relation, error) {
	var system sql_system.SQLSystem
	if host, ok := ctx.(interface{ GetSQLSystem() sql_system.SQLSystem }); ok {
		system = host.GetSQLSystem()
	}
	if _, external := ctx.(sqlSourceProvider); system == nil && !external {
		return node, nil, nil
	}
	return newStoredResolver(ctx, system).Prepare(node)
}
func (r *storedResolver) Prepare(node *sqlparser.Select) (*sqlparser.Select, map[string]omnisdk.Relation, error) {
	prepared, err := r.selectNode(node, 0)
	if err == nil {
		err = validateRelationDepth(prepared.From, r.ctx.GetRuntimeContext().IndirectDepthMax, 0)
	}
	return prepared, r.relations, err
}

// validateRelationDepth applies the host's configured limit to the expanded
// tree, including chains of earlier CTE references rather than only definitions.
func validateRelationDepth(tables sqlparser.TableExprs, maximum, depth int) error {
	for _, expression := range tables {
		switch table := expression.(type) {
		case *sqlparser.JoinTableExpr:
			if err := validateRelationDepth(sqlparser.TableExprs{table.LeftExpr, table.RightExpr}, maximum, depth); err != nil {
				return err
			}
		case *sqlparser.ParenTableExpr:
			if err := validateRelationDepth(table.Exprs, maximum, depth); err != nil {
				return err
			}
		case *sqlparser.AliasedTableExpr:
			if err := validateDerivedDepth(table, maximum, depth); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateDerivedDepth(table *sqlparser.AliasedTableExpr, maximum, depth int) error {
	sub, isSub := table.Expr.(*sqlparser.Subquery)
	if !isSub {
		return nil
	}
	if depth+1 > maximum {
		return fmt.Errorf("indirection chain length %d > %d; please do not cite views at too deep a level",
			depth+1, maximum)
	}
	if query, isSelect := sub.Select.(*sqlparser.Select); isSelect {
		return validateRelationDepth(query.From, maximum, depth+1)
	}
	return nil
}

func (r *storedResolver) selectNode(node *sqlparser.Select, depth int) (*sqlparser.Select, error) {
	if err := checkSQLNodes(r.ctx, node); err != nil {
		return nil, err
	}
	if depth >= derivedRelationMaxDepth {
		return nil, fmt.Errorf("stored relation expansion exceeds nesting bound")
	}
	parent := r.ctes
	r.ctes = maps.Clone(parent)
	if r.ctes == nil {
		r.ctes = map[string]sqlparser.SelectStatement{}
	}
	defer func() { r.ctes = parent }()
	if err := r.with(node.With, depth); err != nil {
		return nil, err
	}
	if expanded, handled, err := r.inlineView(node, depth); handled || err != nil {
		return expanded, err
	}

	clone := *node
	clone.With = nil
	clone.From = cloneRelationTables(node.From)
	for _, table := range clone.From {
		if err := r.table(table, depth); err != nil {
			return nil, err
		}
	}
	return &clone, nil
}

// with resolves definitions in lexical order. A nested WITH owns a cloned
// scope, and a CTE name shadows a stored relation only in that scope.
func (r *storedResolver) with(clause *sqlparser.With, depth int) error {
	if clause == nil {
		return nil
	}
	if clause.Recursive {
		return fmt.Errorf("recursive provider CTEs require an iterative execution boundary")
	}
	for _, definition := range clause.CTEs {
		prepared, err := r.selectStatement(definition.Select, depth+1)
		if err != nil {
			return err
		}
		if err = renameCTEColumns(firstSelect(prepared), definition.Columns); err != nil {
			return err
		}
		r.ctes[strings.ToLower(definition.Name.GetRawVal())] = prepared
	}
	return nil
}

func renameCTEColumns(node *sqlparser.Select, columns sqlparser.Columns) error {
	if len(columns) == 0 {
		return nil
	}
	if node == nil {
		return fmt.Errorf("CTE requires a SELECT projection")
	}
	if len(columns) != len(node.SelectExprs) {
		return fmt.Errorf("CTE column count does not match its projection")
	}
	expressions := make(sqlparser.SelectExprs, len(columns))
	for i, name := range columns {
		projected, named := node.SelectExprs[i].(*sqlparser.AliasedExpr)
		if !named {
			return fmt.Errorf("CTE with explicit column names must name its projected expressions")
		}
		clone := *projected
		clone.As = name
		expressions[i] = &clone
	}
	node.SelectExprs = expressions
	return nil
}

func (r *storedResolver) table(expr sqlparser.TableExpr, depth int) error {
	switch table := expr.(type) {
	case *sqlparser.JoinTableExpr:
		if err := r.table(table.LeftExpr, depth); err != nil {
			return err
		}
		return r.table(table.RightExpr, depth)
	case *sqlparser.ParenTableExpr:
		for _, inner := range table.Exprs {
			if err := r.table(inner, depth); err != nil {
				return err
			}
		}
	case *sqlparser.AliasedTableExpr:
		if sub, isSub := table.Expr.(*sqlparser.Subquery); isSub {
			prepared, err := r.selectStatement(sub.Select, depth+1)
			if err != nil {
				return err
			}
			table.Expr = &sqlparser.Subquery{Select: prepared}
			return nil
		}
		name, isName := table.Expr.(sqlparser.TableName)
		if !isName {
			return nil
		}
		return r.named(table, name, depth)
	}
	return nil
}
func (r *storedResolver) named(table *sqlparser.AliasedTableExpr, name sqlparser.TableName, depth int) error {
	address := qualifiedName(name)
	if cache := legacyCache(r.ctx); cache != nil && cache.namespace.IsAllowed(address) {
		canonical := cache.namespace.GetObjectName(address)
		if parts := strings.Split(canonical, "."); len(parts) == 3 && canonical != address {
			name.QualifierSecond = sqlparser.NewTableIdent(parts[0])
			name.Qualifier = sqlparser.NewTableIdent(parts[1])
			name.Name = sqlparser.NewTableIdent(parts[2])
			table.Expr = name
			address = canonical
		}
	}
	if name.Qualifier.IsEmpty() && name.QualifierSecond.IsEmpty() {
		if cte, exists := r.ctes[strings.ToLower(name.Name.GetRawVal())]; exists {
			table.Expr = &sqlparser.Subquery{Select: cte}
			if table.As.IsEmpty() {
				table.As = name.Name
			}
			return nil
		}
	}
	if source, exists := externalSource(r.ctx, name); exists {
		return r.registerExternal(table, name, source)
	}
	if r.system == nil {
		return nil
	}
	if view, exists := r.system.GetViewByName(address); exists {
		return r.expandView(table, name, view, depth)
	}
	relation, exists := r.system.GetMaterializedViewByName(address)
	if !exists {
		relation, exists = r.system.GetPhysicalTableByName(address)
	}
	if !exists {
		return nil
	}
	return r.register(table, name, relation)
}
func (r *storedResolver) register(
	table *sqlparser.AliasedTableExpr, name sqlparser.TableName, metadata internaldto.RelationDTO,
) error {
	if err := checkHost(r.ctx, "host/backend", "read"); err != nil {
		return err
	}
	handleName := fmt.Sprintf("stored%d", len(r.relations))
	handle := UnstablePrefix + "omni_relations.sources." + handleName
	columns := map[string]omnisdk.ColumnType{}
	var names, quoted []string
	for _, column := range metadata.GetColumns() {
		names = append(names, column.GetName())
		quoted = append(quoted, quoteRelationIdentifier(column.GetName()))
		columns[column.GetName()] = storedColumnType(column.GetType())
	}
	if len(names) == 0 {
		return fmt.Errorf("stored relation %s has no declared columns", metadata.GetName())
	}
	actualName := r.system.DelimitFullyQualifiedRelationName(metadata.GetName())
	// Identifiers come from registered metadata and are quoted; data is never interpolated.
	//nolint:gosec // G202: metadata identifiers only.
	statement := "SELECT " + strings.Join(quoted, ",") + " FROM " + actualName
	open := func(runCtx context.Context, _ map[string]any) (omnisdk.Rows, error) {
		db, dbErr := r.ctx.GetSQLEngine().GetDB()
		if dbErr != nil {
			return nil, dbErr
		}
		result, queryErr := db.QueryContext(runCtx, statement)
		if queryErr != nil {
			return nil, queryErr
		}
		return materializeSQLRows(runCtx, result, names, nil)
	}
	relation, err := omnisdk.NewRelation(handle, nil, columns, open)
	if err != nil {
		return err
	}
	r.relations[handle] = newOrderedRelation(relation, names)
	table.Expr = sqlparser.TableName{
		QualifierSecond: sqlparser.NewTableIdent(UnstablePrefix + "omni_relations"),
		Qualifier:       sqlparser.NewTableIdent("sources"), Name: sqlparser.NewTableIdent(handleName),
	}
	if table.As.IsEmpty() {
		table.As = name.Name
	}
	return nil
}
func quoteRelationIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}
func storedColumnType(raw string) omnisdk.ColumnType {
	typ := "string"
	switch strings.ToLower(raw) {
	case "integer", "int", "bigint", "smallint", "int4", "int8":
		typ = "integer"
	case "numeric", "decimal", "number", "double precision", "real", "float", "float8":
		typ = "number"
	case "bool", "boolean":
		typ = "boolean"
	}
	return omnisdk.ColumnType{Type: typ}
}

// containsProviderRead excludes handles already resolved as stored relations.
// Pure database queries retain their established backend execution path.
func containsProviderRead(node *sqlparser.Select, current string, relations map[string]omnisdk.Relation) bool {
	found := false
	_ = sqlparser.Walk(func(n sqlparser.SQLNode) (bool, error) {
		table, isTable := n.(*sqlparser.AliasedTableExpr)
		if !isTable {
			return true, nil
		}
		name, isName := table.Expr.(sqlparser.TableName)
		if !isName {
			return true, nil
		}
		if _, stored := relations[qualifiedName(name)]; stored {
			return false, nil
		}
		if _, provider := docProvider(resolveProvider(name.QualifierSecond.GetRawVal(), current)); provider {
			found = true
		}
		return true, nil
	}, node)
	return found
}

func (r *storedResolver) expandView(
	table *sqlparser.AliasedTableExpr, name sqlparser.TableName, view internaldto.RelationDTO, depth int,
) error {
	address := qualifiedName(name)
	if r.active[qualifiedName(name)] {
		return fmt.Errorf("stored view cycle at %s", address)
	}
	r.active[qualifiedName(name)] = true
	defer delete(r.active, address)
	// Stored view definitions have their own lexical scope; a caller's CTE
	// must not capture a table name inside an independently defined view.
	callerCTEs := r.ctes
	r.ctes = nil
	defer func() { r.ctes = callerCTEs }()
	statement, err := sqlparser.Parse(view.GetRawQuery())
	if err != nil {
		return err
	}
	query, isSelect := statement.(sqlparser.SelectStatement)
	if !isSelect {
		return fmt.Errorf("stored view %s requires SELECT", address)
	}
	prepared, err := r.selectStatement(query, depth+1)
	if err != nil {
		return err
	}
	table.Expr = &sqlparser.Subquery{Select: prepared}
	if table.As.IsEmpty() {
		table.As = name.Name
	}
	return nil
}

func storedSelectFunc(
	ctx queryContext, node *sqlparser.Select, currentProvider string,
) (func() internaldto.ExecutorOutput, bool) {
	prepared, stored, err := prepareStoredRelations(ctx, node)
	if err != nil {
		return refuse(err), true
	}
	if !containsProviderRead(prepared, currentProvider, stored) && len(stored) == 0 {
		return nil, false
	}
	isDoc, err := fromDocProviders(prepared.From, currentProvider)
	if !isDoc {
		return nil, false
	}
	if err != nil {
		return refuse(err), true
	}
	return docSelectFunc(ctx, prepared, currentProvider, stored)
}

func firstSelect(stmt sqlparser.SelectStatement) *sqlparser.Select {
	switch node := stmt.(type) {
	case *sqlparser.Select:
		return node
	case *sqlparser.ParenSelect:
		return firstSelect(node.Select)
	case *sqlparser.Union:
		return firstSelect(node.FirstStatement)
	}
	return nil
}

func (r *storedResolver) selectStatement(stmt sqlparser.SelectStatement, depth int) (sqlparser.SelectStatement, error) {
	if depth >= derivedRelationMaxDepth {
		return nil, fmt.Errorf("stored relation expansion exceeds nesting bound")
	}
	switch node := stmt.(type) {
	case *sqlparser.Select:
		return r.selectNode(node, depth)
	case *sqlparser.ParenSelect:
		clone := *node
		var err error
		clone.Select, err = r.selectStatement(node.Select, depth+1)
		return &clone, err
	case *sqlparser.Union:
		parent := r.ctes
		r.ctes = maps.Clone(parent)
		if r.ctes == nil {
			r.ctes = map[string]sqlparser.SelectStatement{}
		}
		defer func() { r.ctes = parent }()
		clone := *node
		if first := firstSelect(node.FirstStatement); first != nil {
			if err := r.with(first.With, depth+1); err != nil {
				return nil, err
			}
		}
		var err error
		clone.FirstStatement, err = r.selectStatement(node.FirstStatement, depth+1)
		if err != nil {
			return nil, err
		}
		clone.UnionSelects = make([]*sqlparser.UnionSelect, len(node.UnionSelects))
		for i, branch := range node.UnionSelects {
			child := *branch
			child.Statement, err = r.selectStatement(branch.Statement, depth+1)
			if err != nil {
				return nil, err
			}
			clone.UnionSelects[i] = &child
		}
		return &clone, nil
	}
	return nil, fmt.Errorf("unsupported stored read %T", stmt)
}

// Only SDK-declared request inputs can replace a stored view's default bindings.
func bindStoredViewInputs(ctx queryContext, view *sqlparser.Select, where *sqlparser.Where, alias string) error {
	if len(view.From) != 1 || view.Where == nil {
		return nil
	}
	table, ok := view.From[0].(*sqlparser.AliasedTableExpr)
	if !ok {
		return nil
	}
	name, ok := table.Expr.(sqlparser.TableName)
	if !ok || name.Qualifier.IsEmpty() {
		return nil
	}
	bundle, ok := docProvider(resolveProvider(name.QualifierSecond.GetRawVal(), ctx.GetCurrentProvider()))
	if !ok {
		return nil
	}
	options, err := documentOptions(ctx)
	if err != nil {
		return err
	}
	address := UnstablePrefix + bundle + "." + name.Qualifier.GetRawVal() + "." + name.Name.GetRawVal()
	declared, err := omnisdk.DescribeTable(registryRoot(ctx), address, options...)
	if err != nil {
		return err
	}
	inputs := map[string]bool{}
	for _, method := range declared.Methods() {
		for _, param := range method.Params() {
			inputs[param.Name()] = true
		}
	}
	bindings := viewLiteralBindings(where, alias, alias)
	rewritten := sqlparser.Rewrite(view.Where.Expr, func(cursor *sqlparser.Cursor) bool {
		compare, isComparison := cursor.Node().(*sqlparser.ComparisonExpr)
		if !isComparison || !literalViewBinding(compare) {
			return true
		}
		col, isColumn := compare.Left.(*sqlparser.ColName)
		if !isColumn || !inputs[col.Name.GetRawVal()] {
			return true
		}
		if replacement := bindings[col.Name.GetRawVal()]; replacement != nil {
			clone := *replacement
			clone.Left = col
			cursor.Replace(&clone)
			return false
		}
		return true
	}, nil)
	view.Where.Expr, ok = rewritten.(sqlparser.Expr)
	if !ok {
		return fmt.Errorf("view binding rewrite requires an expression")
	}
	return nil
}

type orderedRelation interface {
	omnisdk.Relation
	columnOrder() []string
}
type declaredOrderedRelation struct {
	omnisdk.Relation
	names []string
}

func newOrderedRelation(relation omnisdk.Relation, names []string) orderedRelation {
	return &declaredOrderedRelation{Relation: relation, names: append([]string(nil), names...)}
}
func (r *declaredOrderedRelation) columnOrder() []string { return append([]string(nil), r.names...) }

func (r *declaredOrderedRelation) Methods() []omnisdk.MethodSignature {
	methods := r.Relation.Methods()
	result := make([]omnisdk.MethodSignature, len(methods))
	for i, method := range methods {
		result[i] = orderedSignature{MethodSignature: method, relation: r}
	}
	return result
}

type orderedSignature struct {
	omnisdk.MethodSignature
	relation orderedRelation
}

func (s orderedSignature) Columns() []string { return s.relation.columnOrder() }

// Simple views share the caller's request bindings; aggregations keep their boundary.
func (r *storedResolver) inlineView(node *sqlparser.Select, depth int) (*sqlparser.Select, bool, error) {
	if len(node.From) != 1 || r.system == nil {
		return nil, false, nil
	}
	table, isTable := node.From[0].(*sqlparser.AliasedTableExpr)
	if !isTable {
		return nil, false, nil
	}
	name, isName := table.Expr.(sqlparser.TableName)
	if !isName {
		return nil, false, nil
	}
	address := qualifiedName(name)
	view, exists := r.system.GetViewByName(address)
	if !exists {
		return nil, false, nil
	}
	if r.active[address] {
		return nil, true, fmt.Errorf("stored view cycle at %s", address)
	}
	parsed, err := sqlparser.Parse(view.GetRawQuery())
	if err != nil {
		return nil, true, err
	}
	projection, isSelect := parsed.(*sqlparser.Select)
	if !isSelect || projection.With != nil || needsStaging(projection) {
		return nil, false, nil
	}
	for _, selection := range projection.SelectExprs {
		if _, named := selection.(*sqlparser.AliasedExpr); !named {
			return nil, false, nil
		}
	}
	maximum := r.ctx.GetRuntimeContext().IndirectDepthMax
	if depth+1 > maximum {
		return nil, true, fmt.Errorf(
			"indirection chain length %d > %d; please do not cite views at too deep a level", depth+1, maximum)
	}
	cloned, err := sqlparser.Parse(statementSQL(node))
	if err != nil {
		return nil, true, err
	}
	caller, isCaller := cloned.(*sqlparser.Select)
	if !isCaller {
		return nil, true, fmt.Errorf("view caller must be SELECT")
	}
	alias := table.As.GetRawVal()
	if alias == "" {
		alias = name.Name.GetRawVal()
	}
	if err = bindStoredViewInputs(r.ctx, projection, node.Where, alias); err != nil {
		return nil, true, err
	}
	expanded, err := inlineDocumentView(caller, projection, alias)
	if err != nil {
		return nil, true, err
	}
	r.active[address] = true
	result, err := r.selectNode(expanded, depth+1)
	delete(r.active, address)
	return result, true, err
}
