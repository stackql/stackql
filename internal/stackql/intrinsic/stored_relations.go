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
	ctes      map[string]*sqlparser.Select
}

func newStoredResolver(ctx queryContext, system sql_system.SQLSystem) storedRelationResolver {
	return &storedResolver{ctx: ctx, system: system, relations: map[string]omnisdk.Relation{}, active: map[string]bool{}}
}
func prepareStoredRelations(
	ctx queryContext, node *sqlparser.Select,
) (*sqlparser.Select, map[string]omnisdk.Relation, error) {
	if !IsUnstableEnabled() {
		return node, nil, nil
	}
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
	if depth >= derivedRelationMaxDepth {
		return nil, fmt.Errorf("stored relation expansion exceeds nesting bound")
	}
	parent := r.ctes
	r.ctes = maps.Clone(parent)
	if r.ctes == nil {
		r.ctes = map[string]*sqlparser.Select{}
	}
	defer func() { r.ctes = parent }()
	if err := r.with(node.With, depth); err != nil {
		return nil, err
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
		query, isSelect := definition.Select.(*sqlparser.Select)
		if !isSelect {
			return fmt.Errorf("CTE %s requires a SELECT boundary", definition.Name.GetRawVal())
		}
		prepared, err := r.selectNode(query, depth+1)
		if err != nil {
			return err
		}
		if err = renameCTEColumns(prepared, definition.Columns); err != nil {
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
			if inner, isSelect := sub.Select.(*sqlparser.Select); isSelect {
				prepared, err := r.selectNode(inner, depth+1)
				if err != nil {
					return err
				}
				table.Expr = &sqlparser.Subquery{Select: prepared}
			}
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
	actualName, err := r.system.GetFullyQualifiedTableName(metadata.GetName())
	if err != nil {
		return err
	}
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
	r.relations[handle] = relation
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
	query, isSelect := statement.(*sqlparser.Select)
	if !isSelect {
		return fmt.Errorf("stored view %s requires SELECT", address)
	}
	prepared, err := r.selectNode(query, depth+1)
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
	// Preserve established view expansion for queries without a direct provider
	// or stored-table read. Those paths also resolve parameterized views and unions.
	if !containsProviderRead(node, currentProvider, nil) && len(stored) == 0 {
		return nil, false
	}
	if err != nil {
		return refuse(err), true
	}
	if !containsProviderRead(prepared, currentProvider, stored) && !hasExternalRelations(stored) {
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
