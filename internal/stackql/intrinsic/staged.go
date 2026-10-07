package intrinsic

// A SELECT over document-driven relations whose remaining SQL needs every row
// - ordering, grouping, aggregation, de-duplication, OFFSET - is split in two
// when staging is enabled: omnisdk runs the joins and filters, its final
// relation is staged in one query-owned table, and the RDBMS evaluates the
// rest of the statement over that table. See docs/technical/omnisdk_staging.md.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql-labs/omnisdk/pkg/query"
	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/omnistaging"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

// stagedRelationPlaceholder stands in for the query table while the outer
// statement is formatted; the table only exists once a query ID is allocated.
const stagedRelationPlaceholder = "__omnistaging_relation__"

// stagedColumnPrefix names the staged columns; it keeps them clear of the
// aliases the outer statement reports.
const stagedColumnPrefix = "_omni_"

const (
	stagedTextType    = "text"
	stagedNumericType = "numeric"
)

// stagingManager is created on first use from the SQL backend the session
// already holds, and shared by every query thereafter.
//
//nolint:gochecknoglobals // one manager per backend, created once
var (
	stagingOnce    sync.Once
	stagingManager omnistaging.Manager
	errStaging     error
)

func getStagingManager(ctx queryContext) (omnistaging.Manager, error) {
	stagingOnce.Do(func() {
		raw := ctx.GetRuntimeContext().SQLBackendCfgRaw
		sqlCfg, err := dto.GetSQLBackendCfg(raw)
		if err != nil {
			errStaging = err
			return
		}
		cfg, err := omnistaging.NewConfigFromSQLBackend(raw)
		if err != nil {
			errStaging = err
			return
		}
		dialect, err := omnistaging.NewDialect(sqlCfg.GetSQLDialect(), cfg)
		if err != nil {
			errStaging = err
			return
		}
		db, err := ctx.GetSQLEngine().GetDB()
		if err != nil {
			errStaging = err
			return
		}
		stagingManager, errStaging = omnistaging.NewManager(context.Background(), db, dialect)
	})
	return stagingManager, errStaging
}

// needsStaging reports whether a SELECT carries SQL omnisdk leaves unapplied.
func needsStaging(node *sqlparser.Select) bool {
	if len(unsupportedDocClauses(node)) > 0 {
		return true
	}
	if node.Limit != nil && node.Limit.Offset != nil {
		return true
	}
	hasAggregate := false
	//nolint:errcheck // the visitor returns no error
	_ = sqlparser.Walk(func(n sqlparser.SQLNode) (bool, error) {
		if fn, isFunc := n.(*sqlparser.FuncExpr); isFunc && fn.IsAggregate() {
			hasAggregate = true
			return false, nil
		}
		return true, nil
	}, node.SelectExprs)
	return hasAggregate
}

// stagedSelect is a SELECT split at the staging boundary: the omnisdk query
// producing the staged relation, and the statement the RDBMS runs over it.
type stagedSelect interface {
	getSource() docQuery
	getColumns() []string
	getOutputNames() []string
	renderOuter(tableName string) string
}

type standardStagedSelect struct {
	source        docQuery
	columns       []string
	outputNames   []string
	outerTemplate string
}

func newStagedSelect(source docQuery, columns, outputNames []string, outerTemplate string) stagedSelect {
	return &standardStagedSelect{
		source: source, columns: columns, outputNames: outputNames, outerTemplate: outerTemplate,
	}
}

func (s *standardStagedSelect) getSource() docQuery { return s.source }

func (s *standardStagedSelect) getColumns() []string { return s.columns }

// getOutputNames is the reported name of each select-list item, in order.
func (s *standardStagedSelect) getOutputNames() []string { return s.outputNames }

func (s *standardStagedSelect) renderOuter(tableName string) string {
	quoted := `"` + stagedRelationPlaceholder + `"`
	if strings.Contains(s.outerTemplate, quoted) {
		return strings.ReplaceAll(s.outerTemplate, quoted, tableName)
	}
	return strings.ReplaceAll(s.outerTemplate, stagedRelationPlaceholder, tableName)
}

func stagedSelectFunc(
	ctx queryContext,
	node *sqlparser.Select,
	currentProvider string,
) func() internaldto.ExecutorOutput {
	staged, err := planStagedSelect(node, currentProvider, ctx.GetASTFormatter())
	if err != nil {
		return refuse(err)
	}
	return func() internaldto.ExecutorOutput { return runStagedSelect(ctx, staged) }
}

// stagedRefs assigns a staged column to each distinct column reference. The
// statement is never modified: each reference occurrence is renamed as the
// outer statement is formatted.
type stagedRefs struct {
	byKey   map[string]string
	renamed map[*sqlparser.ColName]string
	outputs []query.Output
	names   []string
}

func newStagedRefs() *stagedRefs {
	return &stagedRefs{byKey: make(map[string]string), renamed: make(map[*sqlparser.ColName]string)}
}

func (r *stagedRefs) collect(expr sqlparser.SQLNode, aliases map[string]bool) error {
	return sqlparser.Walk(func(n sqlparser.SQLNode) (bool, error) {
		switch node := n.(type) {
		case *sqlparser.Subquery:
			return false, fmt.Errorf("a subquery cannot be staged for %s relations", UnstablePrefix+"*")
		case *sqlparser.ColName:
			qualifier := node.Qualifier.Name.GetRawVal()
			name := node.Name.GetRawVal()
			if qualifier == "" && aliases[strings.ToLower(name)] {
				return false, nil
			}
			key := qualifier + "." + name
			staged, seen := r.byKey[key]
			if !seen {
				staged = stagedColumnPrefix + strconv.Itoa(len(r.names))
				r.byKey[key] = staged
				r.outputs = append(r.outputs, query.NewOutput(staged, query.NewColumn(qualifier, name)))
				r.names = append(r.names, staged)
			}
			r.renamed[node] = staged
			return false, nil
		}
		return true, nil
	}, expr)
}

// formatter wraps the backend's formatter, rendering each collected
// reference as its staged column.
func (r *stagedRefs) formatter(inner sqlparser.NodeFormatter) sqlparser.NodeFormatter {
	return func(buf *sqlparser.TrackedBuffer, node sqlparser.SQLNode) {
		if col, isCol := node.(*sqlparser.ColName); isCol {
			if staged, isRenamed := r.renamed[col]; isRenamed {
				node = &sqlparser.ColName{Name: sqlparser.NewColIdent(staged)}
			}
		}
		if inner == nil {
			node.Format(buf)
			return
		}
		inner(buf, node)
	}
}

// collectSelectExprs collects the select list's references and returns the
// outer select list with the reported name of each item and the aliases the
// rest of the statement may name.
func (r *stagedRefs) collectSelectExprs(
	exprs sqlparser.SelectExprs,
) (sqlparser.SelectExprs, []string, map[string]bool, error) {
	aliases := make(map[string]bool, len(exprs))
	outputNames := make([]string, 0, len(exprs))
	selectExprs := make(sqlparser.SelectExprs, 0, len(exprs))
	for i, expr := range exprs {
		aliased, isAliased := expr.(*sqlparser.AliasedExpr)
		if !isAliased {
			return nil, nil, nil, fmt.Errorf("'%s' cannot be staged for %s relations; name the columns",
				sqlparser.String(expr), UnstablePrefix+"*")
		}
		// Output names match the streamed path: an alias, else a bare
		// column's own name, else the expression as written. The last is
		// reported by position, so the outer statement aliases it safely.
		outerExpr := &sqlparser.AliasedExpr{Expr: aliased.Expr, As: aliased.As}
		name := aliased.As.GetRawVal()
		if name == "" {
			if col, isCol := aliased.Expr.(*sqlparser.ColName); isCol {
				name = col.Name.GetRawVal()
				outerExpr.As = sqlparser.NewColIdent(name)
			} else {
				name = sqlparser.String(aliased.Expr)
				outerExpr.As = sqlparser.NewColIdent(stagedColumnPrefix + "out_" + strconv.Itoa(i))
			}
		}
		outputNames = append(outputNames, name)
		selectExprs = append(selectExprs, outerExpr)
		aliases[strings.ToLower(outerExpr.As.GetRawVal())] = true
		if err := r.collect(aliased.Expr, nil); err != nil {
			return nil, nil, nil, err
		}
	}
	return selectExprs, outputNames, aliases, nil
}

// planStagedSelect splits a SELECT: FROM and WHERE go to omnisdk, which
// outputs every column the rest of the statement references; the rest is
// formatted over the staged table.
func planStagedSelect(
	node *sqlparser.Select,
	currentProvider string,
	formatter sqlparser.NodeFormatter,
) (stagedSelect, error) {
	t, where, err := translateSource(node, currentProvider)
	if err != nil {
		return nil, err
	}
	refs := newStagedRefs()
	selectExprs, outputNames, aliases, err := refs.collectSelectExprs(node.SelectExprs)
	if err != nil {
		return nil, err
	}
	// GROUP BY and ORDER BY may name a select-list alias; HAVING may not.
	for _, expr := range node.GroupBy {
		if err = refs.collect(expr, aliases); err != nil {
			return nil, err
		}
	}
	if node.Having != nil {
		if err = refs.collect(node.Having.Expr, nil); err != nil {
			return nil, err
		}
	}
	for _, order := range node.OrderBy {
		if err = refs.collect(order.Expr, aliases); err != nil {
			return nil, err
		}
	}
	// A statement referencing no column, such as count(*), still needs one
	// staged row per source row. omnisdk outputs only columns, so the rows
	// are taken whole and staged as a single null placeholder column.
	if len(refs.outputs) == 0 {
		refs.outputs = append(refs.outputs, query.NewOutput("", query.NewStar("")))
		refs.names = append(refs.names, stagedColumnPrefix+"0")
	}
	q, err := query.New(t.joins, where, refs.outputs)
	if err != nil {
		return nil, err
	}
	limit, err := stagedLimit(node.Limit)
	if err != nil {
		return nil, err
	}
	outer := &sqlparser.Select{
		Distinct:    node.Distinct,
		SelectExprs: selectExprs,
		From: sqlparser.TableExprs{&sqlparser.AliasedTableExpr{
			Expr: sqlparser.TableName{Name: sqlparser.NewTableIdent(stagedRelationPlaceholder)},
		}},
		GroupBy: node.GroupBy,
		Having:  node.Having,
		OrderBy: node.OrderBy,
	}
	buf := sqlparser.NewTrackedBuffer(refs.formatter(formatter))
	outer.Format(buf)
	return newStagedSelect(
		newDocQuery(q, refs.names, 0, t.bundles), refs.names, outputNames, buf.String()+limit), nil
}

// stagedLimit renders LIMIT and OFFSET in the form both backends accept.
func stagedLimit(limit *sqlparser.Limit) (string, error) {
	if limit == nil {
		return "", nil
	}
	var b strings.Builder
	if limit.Rowcount != nil {
		n, err := rowCount(limit.Rowcount)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, " LIMIT %d", n)
	}
	if limit.Offset != nil {
		n, err := rowCount(limit.Offset)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&b, " OFFSET %d", n)
	}
	return b.String(), nil
}

func rowCount(expr sqlparser.Expr) (int, error) {
	val, isVal := expr.(*sqlparser.SQLVal)
	if !isVal || val.Type != sqlparser.IntVal {
		return 0, fmt.Errorf("'%s' is not a row count", sqlparser.String(expr))
	}
	n, err := strconv.Atoi(string(val.Val))
	if err != nil || n < 0 {
		return 0, fmt.Errorf("'%s' is not a row count", string(val.Val))
	}
	return n, nil
}

// runStagedSelect stages the omnisdk relation, runs the outer statement over
// it and releases the table. There is no result cache, so the table is
// released as soon as the outer result has been read.
func runStagedSelect(ctx queryContext, staged stagedSelect) internaldto.ExecutorOutput {
	manager, err := getStagingManager(ctx)
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(fmt.Errorf("omnisdk staging unavailable: %w", err))
	}
	bg := context.Background()
	id, err := manager.Begin(bg)
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	output, runErr := stageAndQuery(ctx, manager, id, staged)
	if releaseErr := manager.Release(bg, id); releaseErr != nil && runErr == nil {
		runErr = releaseErr
	}
	if runErr != nil {
		return internaldto.NewErroneousExecutorOutput(runErr)
	}
	return output
}

func stageAndQuery(
	ctx queryContext,
	manager omnistaging.Manager,
	id omnistaging.QueryID,
	staged stagedSelect,
) (internaldto.ExecutorOutput, error) {
	bg := context.Background()
	rows, _, err := openDocQuery(ctx, staged.getSource())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	source := newOmniBatchSource(rows, staged.getColumns(), previewCfg.getBatchSize())
	// Column types come from the first batch, read before the table exists.
	first, err := source.Next(bg)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	source.pending = first
	if _, err = manager.Stage(bg, id, stagedColumns(staged.getColumns(), first), source); err != nil {
		return nil, err
	}
	db, err := ctx.GetSQLEngine().GetDB()
	if err != nil {
		return nil, err
	}
	result, err := db.QueryContext(bg, staged.renderOuter(manager.TableName(id)))
	if err != nil {
		return nil, fmt.Errorf("staged query failed: %w", err)
	}
	defer result.Close()
	return readStagedResult(ctx, result, staged.getOutputNames())
}

// stagedColumns types each column by its first non-null value. Numbers are
// staged as numeric so they order and aggregate as numbers; everything else
// is text, as the streamed path renders it.
func stagedColumns(names []string, first [][]any) []omnistaging.Column {
	columns := make([]omnistaging.Column, 0, len(names))
	for i, name := range names {
		relationalType := stagedTextType
		for _, row := range first {
			if row[i] == nil {
				continue
			}
			if isNumber(row[i]) {
				relationalType = stagedNumericType
			}
			break
		}
		columns = append(columns, omnistaging.NewColumn(name, relationalType))
	}
	return columns
}

func isNumber(value any) bool {
	switch value.(type) {
	case int64, float64:
		return true
	default:
		return false
	}
}

// stagedValue keeps numbers as numbers and renders everything else as the
// streamed path would.
func stagedValue(value any) any {
	switch typed := value.(type) {
	case int:
		return int64(typed)
	case int32:
		return int64(typed)
	case int64, float64:
		return typed
	case float32:
		return float64(typed)
	default:
		return textValue(value)
	}
}

// omniBatchSource reads the omnisdk cursor a batch at a time, only when the
// stager asks for one.
type omniBatchSource struct {
	rows    omnisdk.Rows
	columns []string
	size    int
	pending [][]any
}

func newOmniBatchSource(rows omnisdk.Rows, columns []string, size int) *omniBatchSource {
	if size < 1 {
		size = defaultBatchSize
	}
	return &omniBatchSource{rows: rows, columns: columns, size: size}
}

func (s *omniBatchSource) Next(context.Context) ([][]any, error) {
	if s.pending != nil {
		batch := s.pending
		s.pending = nil
		return batch, nil
	}
	batch := make([][]any, 0, s.size)
	for len(batch) < s.size && s.rows.Next() {
		row := s.rows.Row()
		values := make([]any, 0, len(s.columns))
		for _, name := range s.columns {
			values = append(values, stagedValue(row[name]))
		}
		batch = append(batch, values)
	}
	if len(batch) > 0 {
		return batch, nil
	}
	if err := s.rows.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

// readStagedResult reads the outer result under the reported output names;
// its columns are the select list, in order.
func readStagedResult(
	ctx queryContext,
	result *sql.Rows,
	columnOrder []string,
) (internaldto.ExecutorOutput, error) {
	var err error
	rowMap := make(map[string]map[string]interface{})
	for i := 0; result.Next(); i++ {
		values := make([]any, len(columnOrder))
		pointers := make([]any, len(columnOrder))
		for j := range values {
			pointers[j] = &values[j]
		}
		if err = result.Scan(pointers...); err != nil {
			return nil, err
		}
		row := make(map[string]interface{}, len(columnOrder))
		for j, name := range columnOrder {
			row[name] = outputValue(values[j])
		}
		rowMap[fmt.Sprintf("%012d", i)] = row
	}
	if err = result.Err(); err != nil {
		return nil, err
	}
	return prepare(ctx, columnOrder, rowMap, util.DefaultRowSort), nil
}

func outputValue(value any) any {
	if raw, isBytes := value.([]byte); isBytes {
		return string(raw)
	}
	return textValue(value)
}
