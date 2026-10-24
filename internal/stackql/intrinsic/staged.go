package intrinsic

// A SELECT over document-driven relations whose remaining SQL needs every row
// - ordering, grouping, aggregation, de-duplication, OFFSET - is split in two
// when staging is enabled: omnisdk runs the joins and filters, its final
// relation is staged in one query-owned table, and the RDBMS evaluates the
// rest of the statement over that table. See docs/technical/omnisdk_staging.md.

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql-labs/omnisdk/pkg/query"
	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/psql-wire/pkg/sqldata"
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
		if fn, isFunc := n.(*sqlparser.FuncExpr); isFunc && (fn.IsAggregate() || fn.Over != nil) {
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
	offset        int
}

func newStagedSelect(
	source docQuery,
	columns, outputNames []string,
	outerTemplate string,
	offsets ...int,
) stagedSelect {
	offset := 0
	if len(offsets) > 0 {
		offset = offsets[0]
	}
	return &standardStagedSelect{
		source: source, columns: columns, outputNames: outputNames, outerTemplate: outerTemplate, offset: offset,
	}
}

func (s *standardStagedSelect) getSource() docQuery { return s.source }

func (s *standardStagedSelect) getColumns() []string { return s.columns }

// getOutputNames is the reported name of each select-list item, in order.
func (s *standardStagedSelect) getOutputNames() []string { return s.outputNames }

func (s *standardStagedSelect) renderOuter(tableName string) string {
	return renderStagedTemplate(s.outerTemplate, tableName)
}
func renderStagedTemplate(statement, tableName string) string {
	quoted := `"` + stagedRelationPlaceholder + `"`
	if strings.Contains(statement, quoted) {
		return strings.ReplaceAll(statement, quoted, tableName)
	}
	return strings.ReplaceAll(statement, stagedRelationPlaceholder, tableName)
}

func stagedSelectWithRelations(
	ctx queryContext, node *sqlparser.Select, currentProvider string, relations map[string]omnisdk.Relation,
) func() internaldto.ExecutorOutput {
	dialect, err := backendDialect(ctx)
	if err != nil {
		return refuse(err)
	}
	translator := newDocTranslator(currentProvider, dialect)
	if err = expandDerivedStars(ctx, node, translator, relations); err != nil {
		return refuse(err)
	}
	staged, err := planStagedSelect(node, translator, ctx.GetASTFormatter())
	if err != nil {
		return refuse(err)
	}
	staged = withStagedRelations(staged, relations)
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
		if literal, ok := node.(*sqlparser.SQLVal); ok && literal.Type == sqlparser.StrVal {
			formatted := sqlparser.NewTrackedBuffer(inner)
			formatted.Myprintf("%v", node)
			buf.Myprintf("%s", strings.ReplaceAll(formatted.String(), `\"`, `"`))
			return
		}
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
	anonymous := 0
	for i, expr := range exprs {
		aliased, isAliased := expr.(*sqlparser.AliasedExpr)
		if !isAliased {
			return nil, nil, nil, fmt.Errorf("'%s' cannot be staged for %s relations; name the columns",
				sqlparser.String(expr), UnstablePrefix+"*")
		}
		// Output names match the streamed path: an alias, else a bare
		// column's own name, else its legacy expression label. The last is
		// reported by position, so the outer statement aliases it safely.
		outerExpr := &sqlparser.AliasedExpr{Expr: aliased.Expr, As: aliased.As}
		name, err := projectionName(aliased)
		if err != nil {
			return nil, nil, nil, err
		}
		if name == "" {
			anonymous++
			name = fmt.Sprintf("col_%d", anonymous)
		}
		if aliased.As.IsEmpty() {
			if _, isColumn := aliased.Expr.(*sqlparser.ColName); isColumn {
				outerExpr.As = sqlparser.NewColIdent(name)
			} else {
				outerExpr.As = sqlparser.NewColIdent(stagedColumnPrefix + "out_" + strconv.Itoa(i))
			}
		}
		outputNames = append(outputNames, name)
		selectExprs = append(selectExprs, outerExpr)
		aliases[strings.ToLower(outerExpr.As.GetRawVal())] = true
		if err = r.collect(aliased.Expr, nil); err != nil {
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
	translator docTranslator,
	formatter sqlparser.NodeFormatter,
) (stagedSelect, error) {
	src, err := translator.source(node)
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
	// staged row per source row. Read only the first source so joined sources
	// with duplicate column names do not cause ambiguous wildcard expansion.
	if len(refs.outputs) == 0 {
		refs.outputs = append(refs.outputs, query.NewOutput("", query.NewStar(src.joins()[0].Resource().Alias())))
		refs.names = append(refs.names, stagedColumnPrefix+"0")
	}
	aliasOutputs, err := whereAliasOutputs(node)
	if err != nil {
		return nil, err
	}
	outputs := append(append([]query.Output(nil), refs.outputs...), aliasOutputs...)
	q, err := query.New(src.joins(), src.where(), outputs)
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
	offset := 0
	if node.Limit != nil && node.Limit.Offset != nil {
		offset, _ = rowCount(node.Limit.Offset)
	}
	return newStagedSelect(
		newDocQuery(q, refs.names, 0, src.bundles(), readHints(node)),
		refs.names, outputNames, buf.String()+limit, offset,
	), nil
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
	return runStagedSelectIn(ctx, staged, context.Background())
}

func runStagedSelectIn(ctx queryContext, staged stagedSelect, runCtx context.Context) internaldto.ExecutorOutput {
	manager, err := getStagingManager(ctx)
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(fmt.Errorf("omnisdk staging unavailable: %w", err))
	}
	id, err := manager.Begin(runCtx)
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	output, runErr := stageAndQueryIn(ctx, manager, id, staged, runCtx)
	if releaseErr := manager.Release(context.Background(), id); releaseErr != nil && runErr == nil {
		runErr = releaseErr
	}
	if runErr != nil {
		return internaldto.NewErroneousExecutorOutput(runErr)
	}
	return output
}

//nolint:rowserrcheck // readStagedResult consumes the cursor and checks result.Err before returning.
func stageAndQueryIn(
	ctx queryContext, manager omnistaging.Manager, id omnistaging.QueryID,
	staged stagedSelect, bg context.Context,
) (internaldto.ExecutorOutput, error) {
	read, err := stageRead(ctx, manager, id, staged, bg)
	if err != nil {
		return nil, err
	}
	result := read.Rows()
	defer result.Close()
	output, err := readStagedResult(ctx, result, staged.getOutputNames())
	if err != nil {
		return nil, err
	}
	if terminalErr := read.TerminalError(); terminalErr != nil {
		stream := &terminalErrorStream{ISQLResultStream: output.GetSQLResult(), terminalErr: terminalErr}
		return internaldto.NewExecutorOutput(stream, nil, nil, nil, nil), nil
	}
	return output, nil
}

// stageRead preserves SQL values for relation consumers. Presentation conversion
// belongs only at the outermost client result boundary.
//
//nolint:sqlclosecheck // Ownership of successful SQL rows transfers to the returned read and its consumer.
func stageRead(
	ctx queryContext, manager omnistaging.Manager, id omnistaging.QueryID,
	staged stagedSelect, bg context.Context,
) (stagedRead, error) {
	rows, _, err := openDocQueryIn(ctx, staged.getSource(), bg)
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
	statement := staged.renderOuter(manager.TableName(id))
	if effects, ok := rows.(omnisdk.ReadEffects); ok && effects.AppliedOffset() > 0 {
		if adjusted, hasOffsetRenderer := staged.(interface{ renderAfterOffset(string, int) string }); hasOffsetRenderer {
			statement = adjusted.renderAfterOffset(manager.TableName(id), effects.AppliedOffset())
		}
	}
	result, err := db.QueryContext(bg, statement)
	if err != nil {
		return nil, fmt.Errorf("staged query failed: %w", err)
	}
	return newStagedRead(result, source.terminalErr), nil
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
	case json.Number:
		// omnisdk decodes with UseNumber; bound as a string it would sort as text
		if i, err := typed.Int64(); err == nil {
			return i
		}
		if f, err := typed.Float64(); err == nil {
			return f
		}
		return typed.String()
	default:
		return textValue(value)
	}
}

// omniBatchSource reads the omnisdk cursor a batch at a time, only when the
// stager asks for one.
type omniBatchSource struct {
	rows        omnisdk.Rows
	columns     []string
	size        int
	pending     [][]any
	readRows    int
	terminalErr error
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
		s.readRows++
	}
	if len(batch) > 0 {
		return batch, nil
	}
	if err := s.rows.Err(); err != nil {
		// Preserve completed pages, then surface the failed continuation to the
		// client after the partial result. The upstream failure remains an error.
		s.terminalErr = err
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

func (s *standardStagedSelect) renderAfterOffset(tableName string, applied int) string {
	statement := s.outerTemplate
	suffix := " OFFSET " + strconv.Itoa(s.offset)
	if s.offset > 0 && strings.HasSuffix(statement, suffix) {
		statement = strings.TrimSuffix(statement, suffix)
		if remaining := s.offset - applied; remaining > 0 {
			statement += " OFFSET " + strconv.Itoa(remaining)
		}
	}
	return renderStagedTemplate(statement, tableName)
}

// terminalErrorStream returns completed SQL batches before reporting the source
// failure. Embedding the stream preserves its columns and close behaviour.
type terminalErrorStream struct {
	sqldata.ISQLResultStream
	terminalErr error
	pending     bool
}

func (s *terminalErrorStream) Read() (sqldata.ISQLResult, error) {
	if s.pending {
		s.pending = false
		err := s.terminalErr
		s.terminalErr = nil
		return nil, err
	}
	result, err := s.ISQLResultStream.Read()
	if errors.Is(err, io.EOF) && s.terminalErr != nil {
		if result != nil {
			s.pending = true
			return result, nil
		}
		terminal := s.terminalErr
		s.terminalErr = nil
		return result, terminal
	}
	return result, err
}

type stagedRead interface {
	Rows() *sql.Rows
	TerminalError() error
}
type standardStagedRead struct {
	rows        *sql.Rows
	terminalErr error
}

func newStagedRead(rows *sql.Rows, terminalErr error) stagedRead {
	return standardStagedRead{rows: rows, terminalErr: terminalErr}
}
func (r standardStagedRead) Rows() *sql.Rows      { return r.rows }
func (r standardStagedRead) TerminalError() error { return r.terminalErr }
