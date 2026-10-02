// Package queryshape provides result column metadata inference for SQL queries
// without executing them.
//
// Type inference sources vary by relation kind:
//
//   - Materialized views and user space tables: column metadata is stored
//     alongside the DDL in system tables (__iql__.materialized_views.columns,
//     __iql__.tables.columns).  OIDs, widths, and types are directly available.
//
//   - Views: the view DDL (a SELECT) is stored in __iql__.views.  Parsing the
//     DDL and recursively analysing the projection list yields column shapes.
//     This currently delegates to the plan builder.
//
//   - Direct queries and subqueries: column types are a function of provider
//     method schemas, applied SQL function signatures, and RDBMS expression
//     rules.  This currently delegates to the plan builder.
package queryshape

import (
	"math"
	"strconv"
	"strings"

	"github.com/lib/pq/oid"
	"github.com/stackql/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/planbuilder"
	"github.com/stackql/stackql/internal/stackql/sql_system"
	"github.com/stackql/stackql/internal/stackql/typing"
)

// Inferrer analyses SQL queries and returns result column metadata
// without executing them.
type Inferrer interface {
	// InferResultColumns returns wire-protocol column metadata for the
	// given query.  Returns nil when columns cannot be derived (DDL,
	// mutations, planning failures).
	InferResultColumns(query string) []sqldata.ISQLColumn
}

// NewInferrer creates a new query shape inferrer backed by the given
// handler context.
func NewInferrer(handlerCtx handler.HandlerContext) Inferrer {
	return &standardInferrer{
		handlerCtx: handlerCtx,
		sqlSystem:  handlerCtx.GetSQLSystem(),
	}
}

type standardInferrer struct {
	handlerCtx handler.HandlerContext
	sqlSystem  sql_system.SQLSystem
}

func (si *standardInferrer) InferResultColumns(query string) []sqldata.ISQLColumn {
	// Try stored relation metadata first (cheapest path).
	if cols := si.inferFromStoredRelation(query); cols != nil {
		return cols
	}
	// Fall back to plan-based inference for direct queries, subqueries,
	// and views whose columns require provider schema resolution.
	return si.inferFromPlan(query)
}

// inferFromStoredRelation checks whether the query is a simple
// SELECT against a single materialized view or user space table
// whose column metadata is already stored.  If so, the columns
// are returned directly from the DTO without planning.
func (si *standardInferrer) inferFromStoredRelation(query string) []sqldata.ISQLColumn {
	tableName := extractSingleTableName(query)
	if tableName == "" {
		return nil
	}
	// Materialized views carry stored column metadata with OIDs.
	if dto, ok := si.sqlSystem.GetMaterializedViewByName(tableName); ok {
		return relationalColumnsToSQLColumns(dto.GetColumns())
	}
	// User space tables also carry stored column metadata.
	if dto, ok := si.sqlSystem.GetPhysicalTableByName(tableName); ok {
		return relationalColumnsToSQLColumns(dto.GetColumns())
	}
	return nil
}

// inferFromPlan builds a query plan (without executing) and extracts
// column metadata from it.  This handles views, subqueries, and
// direct provider queries where types derive from method schemas
// and SQL function signatures.
func (si *standardInferrer) inferFromPlan(query string) []sqldata.ISQLColumn {
	clonedCtx := si.handlerCtx.Clone()
	clonedCtx.SetQuery(query)
	clonedCtx.SetRawQuery(query)
	pb := planbuilder.NewPlanBuilder(nil)
	qPlan, err := pb.BuildPlanFromContext(clonedCtx)
	if err != nil || qPlan == nil {
		return nil
	}
	colMeta := qPlan.GetColumnMetadata()
	if len(colMeta) == 0 {
		return nil
	}
	return columnMetadataToSQLColumns(colMeta)
}

// extractSingleTableName does a lightweight parse to detect queries
// of the form "SELECT ... FROM <single_table> ..." and returns the
// table name.  Returns "" for anything more complex (joins, subqueries, etc).
func extractSingleTableName(query string) string {
	stmt, err := sqlparser.Parse(query)
	if err != nil {
		return ""
	}
	sel, ok := stmt.(*sqlparser.Select)
	if !ok || sel == nil {
		return ""
	}
	if len(sel.From) != 1 {
		return ""
	}
	aliased, ok := sel.From[0].(*sqlparser.AliasedTableExpr)
	if !ok {
		return ""
	}
	tableName, ok := aliased.Expr.(sqlparser.TableName)
	if !ok {
		return ""
	}
	// Only unqualified table names (no provider.service.resource).
	if !tableName.Qualifier.IsEmpty() {
		return ""
	}
	return tableName.Name.GetRawVal()
}

// relationalColumnsToSQLColumns converts stored RelationalColumn
// metadata to wire protocol ISQLColumn objects.
func relationalColumnsToSQLColumns(cols []typing.RelationalColumn) []sqldata.ISQLColumn {
	if len(cols) == 0 {
		return nil
	}
	table := sqldata.NewSQLTable(0, "")
	result := make([]sqldata.ISQLColumn, len(cols))
	for i, col := range cols {
		colOID := oid.T_text
		if storedOID, ok := col.GetOID(); ok {
			colOID = storedOID
		}
		w := col.GetWidth()
		if w > math.MaxInt16 || w < math.MinInt16 {
			w = -1
		}
		result[i] = sqldata.NewSQLColumn(
			table,
			col.GetIdentifier(),
			0,
			uint32(colOID),
			int16(w), //nolint:gosec // bounds checked above
			0,
			"text",
		)
	}
	return result
}

// columnMetadataToSQLColumns converts internal ColumnMetadata to
// wire protocol ISQLColumn objects.
func columnMetadataToSQLColumns(cols []typing.ColumnMetadata) []sqldata.ISQLColumn {
	table := sqldata.NewSQLTable(0, "")
	result := make([]sqldata.ISQLColumn, len(cols))
	for i, col := range cols {
		result[i] = sqldata.NewSQLColumn(
			table,
			col.GetIdentifier(),
			0,
			uint32(col.GetColumnOID()),
			-1,
			0,
			"text",
		)
	}
	return result
}

// SubstituteDecodedParams replaces $1, $2, ... placeholders with their
// decoded values as string literals; nil values become NULL.  The query is
// scanned with the parser's own tokenizer, so a $n inside a string literal,
// quoted identifier or comment is left alone.
func SubstituteDecodedParams(query string, decodedValues []*string) string {
	if len(decodedValues) == 0 {
		return query
	}
	var sb strings.Builder
	copied := 0
	tokenizer := sqlparser.NewStringTokenizer(query)
	// Special comments would otherwise yield tokens with their own positions.
	tokenizer.SkipSpecialComments = true
	for {
		token, val := tokenizer.Scan()
		if token == 0 {
			break
		}
		idx, isPlaceholder := placeholderIndex(token, val)
		if !isPlaceholder || idx > len(decodedValues) {
			continue
		}
		// Position sits one past the tokenizer's lookahead character.
		end := tokenizer.Position - 1
		start := end - len(val)
		// A quoted identifier yields the same token but not the same source text.
		if start < copied || query[start:end] != string(val) {
			continue
		}
		sb.WriteString(query[copied:start])
		sb.WriteString(encodeParam(decodedValues[idx-1]))
		copied = end
	}
	sb.WriteString(query[copied:])
	return sb.String()
}

// placeholderIndex returns n for a $n identifier token.
func placeholderIndex(token int, val []byte) (int, bool) {
	if token != sqlparser.ID || len(val) < 2 || val[0] != '$' {
		return 0, false
	}
	idx, err := strconv.Atoi(string(val[1:]))
	if err != nil || idx < 1 {
		return 0, false
	}
	return idx, true
}

// encodeParam renders a value as a string literal.  Backslashes are doubled
// for the stackql parser, which treats them as escapes; quotes are doubled
// rather than backslash-escaped because internally routed queries reach the
// backing RDBMS verbatim, where only a doubled quote stays inside the literal.
func encodeParam(val *string) string {
	if val == nil {
		return "NULL"
	}
	escaped := strings.ReplaceAll(*val, `\`, `\\`)
	escaped = strings.ReplaceAll(escaped, "'", "''")
	return "'" + escaped + "'"
}
