package intrinsic

// A SELECT over the document-driven providers is handed to omnisdk whole. This
// file turns the parsed statement into omnisdk's query.Unresolved: FROM becomes
// joins over registry addresses, WHERE and each ON become conjuncts, and the
// select list becomes named outputs. omnisdk decides which conditions become
// request parameters, edges between tables, or row filters.

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/stackql-labs/omnisdk/pkg/query"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

// starOutput marks a select-list slot the first row's columns fill.
const starOutput = "*"

// docQuery is a SELECT translated for omnisdk: the query itself, the output
// names in select-list order, and the LIMIT to push down.
type docQuery interface {
	getQuery() query.Unresolved
	getOutputs() []string
	getLimit() int
	getBundles() []string
}

type standardDocQuery struct {
	q       query.Unresolved
	outputs []string
	limit   int
	bundles []string
}

func newDocQuery(q query.Unresolved, outputs []string, limit int, bundles []string) docQuery {
	return &standardDocQuery{q: q, outputs: outputs, limit: limit, bundles: bundles}
}

func (d *standardDocQuery) getQuery() query.Unresolved { return d.q }

func (d *standardDocQuery) getOutputs() []string { return d.outputs }

func (d *standardDocQuery) getLimit() int { return d.limit }

func (d *standardDocQuery) getBundles() []string { return d.bundles }

// fromDocProviders reports whether a FROM clause reads the document-driven
// providers, and refuses one mixing them with anything else: omnisdk runs the
// whole query, so every relation in it has to be one it can resolve.
func fromDocProviders(from sqlparser.TableExprs, currentProvider string) (bool, error) {
	var docs, others []string
	var walk func(expr sqlparser.TableExpr)
	walk = func(expr sqlparser.TableExpr) {
		switch node := expr.(type) {
		case *sqlparser.AliasedTableExpr:
			tableName, isName := node.Expr.(sqlparser.TableName)
			if !isName {
				others = append(others, sqlparser.String(node))
				return
			}
			if _, isDoc := docProvider(
				resolveProvider(tableName.QualifierSecond.GetRawVal(), currentProvider)); isDoc {
				docs = append(docs, tableName.Name.GetRawVal())
				return
			}
			others = append(others, qualifiedName(tableName))
		case *sqlparser.JoinTableExpr:
			walk(node.LeftExpr)
			walk(node.RightExpr)
		case *sqlparser.ParenTableExpr:
			for _, inner := range node.Exprs {
				walk(inner)
			}
		default:
			others = append(others, sqlparser.String(node))
		}
	}
	for _, expr := range from {
		walk(expr)
	}
	if len(docs) == 0 {
		return false, nil
	}
	if len(others) > 0 {
		return true, fmt.Errorf(
			"%s relations cannot be combined with %s in one query",
			UnstablePrefix+"*", strings.Join(others, ", "))
	}
	return true, nil
}

// translateSelect builds the omnisdk query for a SELECT over document-driven
// relations. What omnisdk leaves to the caller - ordering, grouping,
// aggregation, de-duplication - is refused until stackql applies it over the
// streamed rows.
func translateSelect(node *sqlparser.Select, currentProvider string) (docQuery, error) {
	if unsupported := unsupportedDocClauses(node); len(unsupported) > 0 {
		return nil, fmt.Errorf("%s cannot be applied to %s relations; remove %s from the query",
			strings.Join(unsupported, ", "), UnstablePrefix+"*", pluralClause(len(unsupported)))
	}
	limit, err := pushedLimit(node.Limit)
	if err != nil {
		return nil, err
	}
	if len(node.From) != 1 {
		return nil, fmt.Errorf("a comma-separated FROM cannot be applied to %s relations; use JOIN ... ON",
			UnstablePrefix+"*")
	}
	t := &translator{currentProvider: currentProvider}
	if err = t.from(node.From[0], query.Base, nil); err != nil {
		return nil, err
	}
	var where []query.Predicate
	if node.Where != nil {
		if where, err = conjuncts(node.Where.Expr); err != nil {
			return nil, err
		}
	}
	outputs, names, err := selectOutputs(node.SelectExprs)
	if err != nil {
		return nil, err
	}
	q, err := query.New(t.joins, where, outputs)
	if err != nil {
		return nil, err
	}
	return newDocQuery(q, names, limit, t.bundles), nil
}

// unsupportedDocClauses names what omnisdk returns unapplied: its row stream is
// unordered, ungrouped and not de-duplicated. LIMIT is pushed down instead.
func unsupportedDocClauses(node *sqlparser.Select) []string {
	var out []string
	if len(node.OrderBy) > 0 {
		out = append(out, "ORDER BY")
	}
	if len(node.GroupBy) > 0 {
		out = append(out, "GROUP BY")
	}
	if node.Having != nil {
		out = append(out, "HAVING")
	}
	if node.Distinct {
		out = append(out, "DISTINCT")
	}
	return out
}

// pushedLimit is the row cap omnisdk applies. Without ORDER BY a LIMIT needs
// no sorted input, so it can stop the stream early.
func pushedLimit(limit *sqlparser.Limit) (int, error) {
	if limit == nil {
		return 0, nil
	}
	if limit.Offset != nil {
		return 0, fmt.Errorf("OFFSET cannot be applied to %s relations", UnstablePrefix+"*")
	}
	val, isVal := limit.Rowcount.(*sqlparser.SQLVal)
	if !isVal || val.Type != sqlparser.IntVal {
		return 0, fmt.Errorf("LIMIT %s is not a row count", sqlparser.String(limit.Rowcount))
	}
	n, err := strconv.Atoi(string(val.Val))
	if err != nil || n < 1 {
		return 0, fmt.Errorf("LIMIT %s is not a positive row count", string(val.Val))
	}
	return n, nil
}

type translator struct {
	currentProvider string
	joins           []query.Join
	bundles         []string
}

// from appends a FROM item's joins in source order. A join's right side takes
// the join's form and ON; everything on its left keeps its own.
func (t *translator) from(expr sqlparser.TableExpr, form query.JoinForm, on []query.Predicate) error {
	switch node := expr.(type) {
	case *sqlparser.AliasedTableExpr:
		tableName, isName := node.Expr.(sqlparser.TableName)
		if !isName {
			return fmt.Errorf("'%s' cannot be read from %s relations", sqlparser.String(node), UnstablePrefix+"*")
		}
		if len(t.joins) == 0 {
			form = query.Base
		}
		t.joins = append(t.joins, query.NewJoin(t.resource(tableName, node.As), form, on...))
		return nil
	case *sqlparser.ParenTableExpr:
		if len(node.Exprs) != 1 {
			return fmt.Errorf("'%s' cannot be read from %s relations", sqlparser.String(node), UnstablePrefix+"*")
		}
		return t.from(node.Exprs[0], form, on)
	case *sqlparser.JoinTableExpr:
		rightForm, err := joinForm(node.Join)
		if err != nil {
			return err
		}
		if len(node.Condition.Using) > 0 {
			return fmt.Errorf("JOIN ... USING cannot be applied to %s relations; use ON", UnstablePrefix+"*")
		}
		if err = t.from(node.LeftExpr, form, on); err != nil {
			return err
		}
		var rightOn []query.Predicate
		if node.Condition.On != nil {
			if rightOn, err = conjuncts(node.Condition.On); err != nil {
				return err
			}
		}
		return t.from(node.RightExpr, rightForm, rightOn)
	default:
		return fmt.Errorf("'%s' cannot be read from %s relations", sqlparser.String(node), UnstablePrefix+"*")
	}
}

// resource addresses a relation by its registry handle, aliased as written or
// else by its own name, and records the bundle whose credential it runs under.
func (t *translator) resource(tableName sqlparser.TableName, as sqlparser.TableIdent) query.Resource {
	bundle, _ := docProvider(resolveProvider(tableName.QualifierSecond.GetRawVal(), t.currentProvider))
	t.bundles = append(t.bundles, bundle)
	alias := as.GetRawVal()
	if alias == "" {
		alias = tableName.Name.GetRawVal()
	}
	address := fmt.Sprintf("%s%s.%s.%s", UnstablePrefix, bundle,
		tableName.Qualifier.GetRawVal(), tableName.Name.GetRawVal())
	return query.NewResource(alias, address)
}

func qualifiedName(tableName sqlparser.TableName) string {
	var parts []string
	for _, part := range []string{
		tableName.QualifierSecond.GetRawVal(), tableName.Qualifier.GetRawVal(), tableName.Name.GetRawVal(),
	} {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, ".")
}

func joinForm(join string) (query.JoinForm, error) {
	switch strings.ToLower(join) {
	case sqlparser.JoinStr:
		return query.Inner, nil
	case sqlparser.LeftJoinStr, sqlparser.LeftOuterJoinStr:
		return query.Left, nil
	default:
		return query.Base, fmt.Errorf("%s cannot be applied to %s relations",
			strings.ToUpper(join), UnstablePrefix+"*")
	}
}

// conjuncts splits a condition at its top-level ANDs.
func conjuncts(expr sqlparser.Expr) ([]query.Predicate, error) {
	if and, isAnd := expr.(*sqlparser.AndExpr); isAnd {
		left, err := conjuncts(and.Left)
		if err != nil {
			return nil, err
		}
		right, err := conjuncts(and.Right)
		if err != nil {
			return nil, err
		}
		return append(left, right...), nil
	}
	p, err := predicate(expr)
	if err != nil {
		return nil, err
	}
	return []query.Predicate{p}, nil
}

var compareOps = map[string]query.CompareOp{ //nolint:gochecknoglobals // fixed mapping
	sqlparser.EqualStr:        query.Eq,
	sqlparser.NotEqualStr:     query.Ne,
	sqlparser.LessThanStr:     query.Lt,
	sqlparser.LessEqualStr:    query.Le,
	sqlparser.GreaterThanStr:  query.Gt,
	sqlparser.GreaterEqualStr: query.Ge,
}

func predicate(expr sqlparser.Expr) (query.Predicate, error) {
	switch node := expr.(type) {
	case *sqlparser.AndExpr:
		parts, err := conjuncts(node)
		if err != nil {
			return nil, err
		}
		// Under OR or NOT an AND is a single predicate: every part must hold.
		negated := make([]query.Predicate, 0, len(parts))
		for _, part := range parts {
			negated = append(negated, query.NewNot(part))
		}
		return query.NewNot(query.NewOr(negated...)), nil
	case *sqlparser.OrExpr:
		left, err := predicate(node.Left)
		if err != nil {
			return nil, err
		}
		right, err := predicate(node.Right)
		if err != nil {
			return nil, err
		}
		return query.NewOr(left, right), nil
	case *sqlparser.NotExpr:
		inner, err := predicate(node.Expr)
		if err != nil {
			return nil, err
		}
		return query.NewNot(inner), nil
	case *sqlparser.ComparisonExpr:
		return comparison(node)
	case *sqlparser.FuncExpr:
		call, err := expression(node)
		if err != nil {
			return nil, err
		}
		return query.NewTest(call), nil
	default:
		return nil, fmt.Errorf("condition '%s' cannot be applied to %s relations",
			sqlparser.String(expr), UnstablePrefix+"*")
	}
}

func comparison(node *sqlparser.ComparisonExpr) (query.Predicate, error) {
	left, err := expression(node.Left)
	if err != nil {
		return nil, err
	}
	right, err := expression(node.Right)
	if err != nil {
		return nil, err
	}
	switch node.Operator {
	case sqlparser.InStr:
		return query.NewIn(left, right), nil
	case sqlparser.NotInStr:
		return query.NewNot(query.NewIn(left, right)), nil
	}
	op, known := compareOps[node.Operator]
	if !known {
		return nil, fmt.Errorf("condition '%s' cannot be applied to %s relations",
			sqlparser.String(node), UnstablePrefix+"*")
	}
	return query.NewCompare(op, left, right), nil
}

func expression(expr sqlparser.Expr) (query.Expr, error) {
	switch node := expr.(type) {
	case *sqlparser.ColName:
		return query.NewColumn(node.Qualifier.Name.GetRawVal(), node.Name.GetRawVal()), nil
	case *sqlparser.SQLVal:
		return literal(node)
	case sqlparser.BoolVal:
		return query.NewLiteral(bool(node)), nil
	case *sqlparser.NullVal:
		return query.NewLiteral(nil), nil
	case sqlparser.ValTuple:
		items := make([]query.Expr, 0, len(node))
		for _, item := range node {
			translated, err := expression(item)
			if err != nil {
				return nil, err
			}
			items = append(items, translated)
		}
		return query.NewCollection(items...), nil
	case *sqlparser.FuncExpr:
		if node.IsAggregate() || node.Distinct || node.Over != nil {
			return nil, fmt.Errorf("'%s' cannot be applied to %s relations",
				sqlparser.String(node), UnstablePrefix+"*")
		}
		args := make([]query.Expr, 0, len(node.Exprs))
		for _, arg := range node.Exprs {
			aliased, isAliased := arg.(*sqlparser.AliasedExpr)
			if !isAliased {
				return nil, fmt.Errorf("argument '%s' cannot be applied to %s relations",
					sqlparser.String(arg), UnstablePrefix+"*")
			}
			translated, err := expression(aliased.Expr)
			if err != nil {
				return nil, err
			}
			args = append(args, translated)
		}
		return query.NewCall(node.Name.Lowered(), args...), nil
	default:
		return nil, fmt.Errorf("'%s' cannot be applied to %s relations",
			sqlparser.String(expr), UnstablePrefix+"*")
	}
}

func literal(val *sqlparser.SQLVal) (query.Expr, error) {
	switch val.Type { //nolint:exhaustive // hex, bit and bind values have no omnisdk literal
	case sqlparser.StrVal:
		return query.NewLiteral(string(val.Val)), nil
	case sqlparser.IntVal:
		n, err := strconv.ParseInt(string(val.Val), 10, 64)
		if err != nil {
			return nil, err
		}
		return query.NewLiteral(n), nil
	case sqlparser.FloatVal:
		f, err := strconv.ParseFloat(string(val.Val), 64)
		if err != nil {
			return nil, err
		}
		return query.NewLiteral(f), nil
	default:
		return nil, fmt.Errorf("'%s' cannot be applied to %s relations",
			sqlparser.String(val), UnstablePrefix+"*")
	}
}

// selectOutputs names every output: an alias, else a bare column's own name,
// else the expression as written. A star is left unnamed for omnisdk to expand.
func selectOutputs(exprs sqlparser.SelectExprs) ([]query.Output, []string, error) {
	outputs := make([]query.Output, 0, len(exprs))
	names := make([]string, 0, len(exprs))
	for _, expr := range exprs {
		switch node := expr.(type) {
		case *sqlparser.StarExpr:
			outputs = append(outputs, query.NewOutput("", query.NewStar(node.TableName.Name.GetRawVal())))
			names = append(names, starOutput)
		case *sqlparser.AliasedExpr:
			translated, err := expression(node.Expr)
			if err != nil {
				return nil, nil, err
			}
			name := node.As.GetRawVal()
			if name == "" {
				if col, isCol := node.Expr.(*sqlparser.ColName); isCol {
					name = col.Name.GetRawVal()
				} else {
					name = sqlparser.String(node.Expr)
				}
			}
			outputs = append(outputs, query.NewOutput(name, translated))
			names = append(names, name)
		default:
			return nil, nil, fmt.Errorf("'%s' cannot be applied to %s relations",
				sqlparser.String(expr), UnstablePrefix+"*")
		}
	}
	return outputs, names, nil
}

// mutationTables is every relation a mutation reads or writes: the target
// first, then its sources.
func mutationTables(stmt sqlparser.Statement) (sqlparser.TableExprs, bool) {
	switch node := stmt.(type) {
	case *sqlparser.Insert:
		tables := sqlparser.TableExprs{&sqlparser.AliasedTableExpr{Expr: node.Table}}
		if sel, isSelect := node.Rows.(*sqlparser.Select); isSelect {
			tables = append(tables, sel.From...)
		}
		return tables, true
	case *sqlparser.Update:
		return append(append(sqlparser.TableExprs{}, node.TableExprs...), updateSources(node)...), true
	case *sqlparser.Delete:
		return node.TableExprs, true
	default:
		return nil, false
	}
}

// translateMutation builds the omnisdk mutation for an INSERT, UPDATE or
// DELETE whose target is a document-driven relation. A RETURNING list becomes
// the mutation's outputs.
func translateMutation(stmt sqlparser.Statement, currentProvider string) (docQuery, error) {
	t := &translator{currentProvider: currentProvider}
	var (
		target    query.Target
		where     []query.Predicate
		returning sqlparser.SelectExprs
		err       error
	)
	switch node := stmt.(type) {
	case *sqlparser.Insert:
		target, where, err = t.insert(node)
		returning = node.SelectExprs
	case *sqlparser.Update:
		target, where, err = t.update(node)
		returning = node.SelectExprs
	case *sqlparser.Delete:
		target, where, err = t.delete(node)
		returning = node.SelectExprs
	default:
		return nil, fmt.Errorf("'%s' cannot be applied to %s relations", sqlparser.String(stmt), UnstablePrefix+"*")
	}
	if err != nil {
		return nil, err
	}
	var outputs []query.Output
	var names []string
	if len(returning) > 0 {
		if outputs, names, err = selectOutputs(returning); err != nil {
			return nil, err
		}
	}
	q, err := query.NewMutation(target, t.joins, where, outputs)
	if err != nil {
		return nil, err
	}
	return newDocQuery(q, names, 0, t.bundles), nil
}

// insert takes its values from a single VALUES row, or from a SELECT whose
// relations become the mutation's sources and whose list is matched to the
// columns by position.
func (t *translator) insert(node *sqlparser.Insert) (query.Target, []query.Predicate, error) {
	switch {
	case !strings.EqualFold(node.Action, sqlparser.InsertStr):
		return nil, nil, fmt.Errorf("%s cannot be applied to %s relations; use INSERT",
			strings.ToUpper(node.Action), UnstablePrefix+"*")
	case node.Ignore != "" || len(node.OnDup) > 0:
		return nil, nil, fmt.Errorf("INSERT IGNORE and ON DUPLICATE KEY cannot be applied to %s relations",
			UnstablePrefix+"*")
	case len(node.Columns) == 0:
		return nil, nil, fmt.Errorf("an INSERT into %s relations must name its columns", UnstablePrefix+"*")
	}
	resource := t.resource(node.Table, sqlparser.NewTableIdent(""))
	var values []sqlparser.Expr
	var where []query.Predicate
	switch rows := node.Rows.(type) {
	case sqlparser.Values:
		if len(rows) != 1 {
			return nil, nil, fmt.Errorf("an INSERT into %s relations takes exactly one VALUES row; got %d",
				UnstablePrefix+"*", len(rows))
		}
		values = rows[0]
	case *sqlparser.Select:
		var err error
		if values, where, err = t.insertSelect(rows); err != nil {
			return nil, nil, err
		}
	default:
		return nil, nil, fmt.Errorf("'%s' cannot be inserted into %s relations",
			sqlparser.String(node.Rows), UnstablePrefix+"*")
	}
	if len(values) != len(node.Columns) {
		return nil, nil, fmt.Errorf("INSERT names %d columns but supplies %d values", len(node.Columns), len(values))
	}
	assignments := make([]query.Assignment, 0, len(values))
	for i, value := range values {
		translated, err := expression(value)
		if err != nil {
			return nil, nil, err
		}
		assignments = append(assignments, query.NewAssignment(node.Columns[i].GetRawVal(), translated))
	}
	return query.NewInsert(resource, assignments...), where, nil
}

// insertSelect reads an INSERT ... SELECT: its relations become the mutation's
// sources, and its list supplies the values, matched to the columns by position.
func (t *translator) insertSelect(rows *sqlparser.Select) ([]sqlparser.Expr, []query.Predicate, error) {
	if unsupported := unsupportedDocClauses(rows); len(unsupported) > 0 || rows.Limit != nil {
		return nil, nil, fmt.Errorf("an INSERT ... SELECT into %s relations takes no "+
			"ORDER BY, GROUP BY, HAVING, DISTINCT or LIMIT", UnstablePrefix+"*")
	}
	if len(rows.From) != 1 {
		return nil, nil, fmt.Errorf("a comma-separated FROM cannot be applied to %s relations; use JOIN ... ON",
			UnstablePrefix+"*")
	}
	if err := t.from(rows.From[0], query.Base, nil); err != nil {
		return nil, nil, err
	}
	where, err := whereConjuncts(rows.Where)
	if err != nil {
		return nil, nil, err
	}
	values := make([]sqlparser.Expr, 0, len(rows.SelectExprs))
	for _, expr := range rows.SelectExprs {
		aliased, isAliased := expr.(*sqlparser.AliasedExpr)
		if !isAliased {
			return nil, nil, fmt.Errorf("'%s' cannot be inserted into %s relations; name each value",
				sqlparser.String(expr), UnstablePrefix+"*")
		}
		values = append(values, aliased.Expr)
	}
	return values, where, nil
}

func (t *translator) update(node *sqlparser.Update) (query.Target, []query.Predicate, error) {
	if !strings.EqualFold(node.Action, sqlparser.UpdateStr) {
		return nil, nil, fmt.Errorf("%s cannot be applied to %s relations; use UPDATE",
			strings.ToUpper(node.Action), UnstablePrefix+"*")
	}
	if len(node.OrderBy) > 0 || node.Limit != nil {
		return nil, nil, fmt.Errorf("ORDER BY and LIMIT cannot be applied to an UPDATE of %s relations",
			UnstablePrefix+"*")
	}
	resource, err := t.targetResource(node.TableExprs)
	if err != nil {
		return nil, nil, err
	}
	if err = t.sources(updateSources(node)); err != nil {
		return nil, nil, err
	}
	assignments := make([]query.Assignment, 0, len(node.Exprs))
	for _, set := range node.Exprs {
		translated, translateErr := expression(set.Expr)
		if translateErr != nil {
			return nil, nil, translateErr
		}
		assignments = append(assignments, query.NewAssignment(set.Name.Name.GetRawVal(), translated))
	}
	where, err := whereConjuncts(node.Where)
	if err != nil {
		return nil, nil, err
	}
	return query.NewUpdate(resource, assignments...), where, nil
}

func (t *translator) delete(node *sqlparser.Delete) (query.Target, []query.Predicate, error) {
	switch {
	case len(node.Targets) > 0:
		return nil, nil, fmt.Errorf("a multi-table DELETE cannot be applied to %s relations", UnstablePrefix+"*")
	case len(node.OrderBy) > 0 || node.Limit != nil:
		return nil, nil, fmt.Errorf("ORDER BY and LIMIT cannot be applied to a DELETE of %s relations",
			UnstablePrefix+"*")
	}
	resource, err := t.targetResource(node.TableExprs)
	if err != nil {
		return nil, nil, err
	}
	where, err := whereConjuncts(node.Where)
	if err != nil {
		return nil, nil, err
	}
	return query.NewDelete(resource), where, nil
}

// targetResource is the single relation an UPDATE or DELETE writes.
func (t *translator) targetResource(exprs sqlparser.TableExprs) (query.Resource, error) {
	if len(exprs) == 1 {
		if aliased, isAliased := exprs[0].(*sqlparser.AliasedTableExpr); isAliased {
			if tableName, isName := aliased.Expr.(sqlparser.TableName); isName {
				return t.resource(tableName, aliased.As), nil
			}
		}
	}
	return nil, fmt.Errorf("'%s' cannot be written as one %s relation", sqlparser.String(exprs), UnstablePrefix+"*")
}

// updateSources is an UPDATE's FROM clause. The parser stands in "dual" for an
// absent one, which reads nothing.
func updateSources(node *sqlparser.Update) sqlparser.TableExprs {
	if len(node.From) == 1 {
		if aliased, isAliased := node.From[0].(*sqlparser.AliasedTableExpr); isAliased {
			if tableName, isName := aliased.Expr.(sqlparser.TableName); isName &&
				tableName.Qualifier.IsEmpty() && tableName.Name.GetRawVal() == "dual" {
				return nil
			}
		}
	}
	return node.From
}

// sources reads an UPDATE ... FROM clause as the mutation's source joins.
func (t *translator) sources(from sqlparser.TableExprs) error {
	switch len(from) {
	case 0:
		return nil
	case 1:
		return t.from(from[0], query.Base, nil)
	default:
		return fmt.Errorf("a comma-separated FROM cannot be applied to %s relations; use JOIN ... ON",
			UnstablePrefix+"*")
	}
}

func whereConjuncts(where *sqlparser.Where) ([]query.Predicate, error) {
	if where == nil {
		return nil, nil
	}
	return conjuncts(where.Expr)
}
