package parserutil_test

import (
	"testing"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/pkg/astformat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInferColNameFromExpr_WindowFunctions(t *testing.T) {
	formatter := astformat.DefaultSelectExprsFormatter

	t.Run("ROW_NUMBER with OVER clause is marked as aggregate", func(t *testing.T) {
		query := "SELECT ROW_NUMBER() OVER (ORDER BY id) as row_num FROM t"
		stmt, err := sqlparser.Parse(query)
		require.NoError(t, err)

		sel := stmt.(*sqlparser.Select)
		require.Len(t, sel.SelectExprs, 1)

		aliasedExpr := sel.SelectExprs[0].(*sqlparser.AliasedExpr)
		colHandle, err := parserutil.InferColNameFromExpr(aliasedExpr, formatter)
		require.NoError(t, err)

		assert.True(t, colHandle.IsAggregateExpr, "ROW_NUMBER() with OVER should be marked as aggregate expression")
		assert.Equal(t, "row_num", colHandle.Alias)
		// Verify DecoratedColumn includes OVER clause.
		assert.Contains(t, colHandle.DecoratedColumn, "over", "DecoratedColumn should contain OVER clause")
	})

	t.Run("SUM with OVER PARTITION BY is marked as aggregate", func(t *testing.T) {
		query := "SELECT SUM(amount) OVER (PARTITION BY category) as running_sum FROM t"
		stmt, err := sqlparser.Parse(query)
		require.NoError(t, err)

		sel := stmt.(*sqlparser.Select)
		require.Len(t, sel.SelectExprs, 1)

		aliasedExpr := sel.SelectExprs[0].(*sqlparser.AliasedExpr)
		colHandle, err := parserutil.InferColNameFromExpr(aliasedExpr, formatter)
		require.NoError(t, err)

		assert.True(t, colHandle.IsAggregateExpr, "SUM() with OVER PARTITION BY should be marked as aggregate expression")
		assert.Equal(t, "running_sum", colHandle.Alias)
	})

	t.Run("RANK with OVER ORDER BY is marked as aggregate", func(t *testing.T) {
		query := "SELECT RANK() OVER (ORDER BY score DESC) as ranking FROM t"
		stmt, err := sqlparser.Parse(query)
		require.NoError(t, err)

		sel := stmt.(*sqlparser.Select)
		require.Len(t, sel.SelectExprs, 1)

		aliasedExpr := sel.SelectExprs[0].(*sqlparser.AliasedExpr)
		colHandle, err := parserutil.InferColNameFromExpr(aliasedExpr, formatter)
		require.NoError(t, err)

		assert.True(t, colHandle.IsAggregateExpr, "RANK() with OVER should be marked as aggregate expression")
		assert.Equal(t, "ranking", colHandle.Alias)
	})

	t.Run("DENSE_RANK with OVER is marked as aggregate", func(t *testing.T) {
		query := "SELECT DENSE_RANK() OVER (PARTITION BY dept ORDER BY salary DESC) as dense_rank FROM t"
		stmt, err := sqlparser.Parse(query)
		require.NoError(t, err)

		sel := stmt.(*sqlparser.Select)
		require.Len(t, sel.SelectExprs, 1)

		aliasedExpr := sel.SelectExprs[0].(*sqlparser.AliasedExpr)
		colHandle, err := parserutil.InferColNameFromExpr(aliasedExpr, formatter)
		require.NoError(t, err)

		assert.True(t, colHandle.IsAggregateExpr, "DENSE_RANK() with OVER should be marked as aggregate expression")
	})

	t.Run("NTILE with OVER is marked as aggregate", func(t *testing.T) {
		query := "SELECT NTILE(4) OVER (ORDER BY id) as quartile FROM t"
		stmt, err := sqlparser.Parse(query)
		require.NoError(t, err)

		sel := stmt.(*sqlparser.Select)
		require.Len(t, sel.SelectExprs, 1)

		aliasedExpr := sel.SelectExprs[0].(*sqlparser.AliasedExpr)
		colHandle, err := parserutil.InferColNameFromExpr(aliasedExpr, formatter)
		require.NoError(t, err)

		assert.True(t, colHandle.IsAggregateExpr, "NTILE() with OVER should be marked as aggregate expression")
	})

	t.Run("LAG with OVER is marked as aggregate", func(t *testing.T) {
		query := "SELECT LAG(value, 1) OVER (ORDER BY date) as prev_value FROM t"
		stmt, err := sqlparser.Parse(query)
		require.NoError(t, err)

		sel := stmt.(*sqlparser.Select)
		require.Len(t, sel.SelectExprs, 1)

		aliasedExpr := sel.SelectExprs[0].(*sqlparser.AliasedExpr)
		colHandle, err := parserutil.InferColNameFromExpr(aliasedExpr, formatter)
		require.NoError(t, err)

		assert.True(t, colHandle.IsAggregateExpr, "LAG() with OVER should be marked as aggregate expression")
	})

	t.Run("LEAD with OVER is marked as aggregate", func(t *testing.T) {
		query := "SELECT LEAD(value, 1) OVER (ORDER BY date) as next_value FROM t"
		stmt, err := sqlparser.Parse(query)
		require.NoError(t, err)

		sel := stmt.(*sqlparser.Select)
		require.Len(t, sel.SelectExprs, 1)

		aliasedExpr := sel.SelectExprs[0].(*sqlparser.AliasedExpr)
		colHandle, err := parserutil.InferColNameFromExpr(aliasedExpr, formatter)
		require.NoError(t, err)

		assert.True(t, colHandle.IsAggregateExpr, "LEAD() with OVER should be marked as aggregate expression")
	})

	t.Run("FIRST_VALUE with OVER is marked as aggregate", func(t *testing.T) {
		query := "SELECT FIRST_VALUE(name) OVER (PARTITION BY category ORDER BY date) as first_name FROM t"
		stmt, err := sqlparser.Parse(query)
		require.NoError(t, err)

		sel := stmt.(*sqlparser.Select)
		require.Len(t, sel.SelectExprs, 1)

		aliasedExpr := sel.SelectExprs[0].(*sqlparser.AliasedExpr)
		colHandle, err := parserutil.InferColNameFromExpr(aliasedExpr, formatter)
		require.NoError(t, err)

		assert.True(t, colHandle.IsAggregateExpr, "FIRST_VALUE() with OVER should be marked as aggregate expression")
	})

	t.Run("LAST_VALUE with OVER is marked as aggregate", func(t *testing.T) {
		query := "SELECT LAST_VALUE(name) OVER (PARTITION BY category ORDER BY date) as last_name FROM t"
		stmt, err := sqlparser.Parse(query)
		require.NoError(t, err)

		sel := stmt.(*sqlparser.Select)
		require.Len(t, sel.SelectExprs, 1)

		aliasedExpr := sel.SelectExprs[0].(*sqlparser.AliasedExpr)
		colHandle, err := parserutil.InferColNameFromExpr(aliasedExpr, formatter)
		require.NoError(t, err)

		assert.True(t, colHandle.IsAggregateExpr, "LAST_VALUE() with OVER should be marked as aggregate expression")
	})

	t.Run("Regular aggregate function without OVER", func(t *testing.T) {
		query := "SELECT COUNT(*) as total FROM t"
		stmt, err := sqlparser.Parse(query)
		require.NoError(t, err)

		sel := stmt.(*sqlparser.Select)
		require.Len(t, sel.SelectExprs, 1)

		aliasedExpr := sel.SelectExprs[0].(*sqlparser.AliasedExpr)
		colHandle, err := parserutil.InferColNameFromExpr(aliasedExpr, formatter)
		require.NoError(t, err)

		assert.True(t, colHandle.IsAggregateExpr, "COUNT() should be marked as aggregate expression")
		assert.Equal(t, "total", colHandle.Alias)
	})

	t.Run("Regular function without OVER is not aggregate", func(t *testing.T) {
		query := "SELECT UPPER(name) as upper_name FROM t"
		stmt, err := sqlparser.Parse(query)
		require.NoError(t, err)

		sel := stmt.(*sqlparser.Select)
		require.Len(t, sel.SelectExprs, 1)

		aliasedExpr := sel.SelectExprs[0].(*sqlparser.AliasedExpr)
		colHandle, err := parserutil.InferColNameFromExpr(aliasedExpr, formatter)
		require.NoError(t, err)

		assert.False(t, colHandle.IsAggregateExpr, "UPPER() should not be marked as aggregate expression")
	})
}

func TestInferColNameFromExpr_BinaryExpressions(t *testing.T) {
	formatter := astformat.DefaultSelectExprsFormatter

	t.Run("String concatenation with || operator", func(t *testing.T) {
		query := "SELECT 'homebrew://' || formula_name as formula_uri FROM t"
		stmt, err := sqlparser.Parse(query)
		require.NoError(t, err)

		sel := stmt.(*sqlparser.Select)
		require.Len(t, sel.SelectExprs, 1)

		aliasedExpr := sel.SelectExprs[0].(*sqlparser.AliasedExpr)
		colHandle, err := parserutil.InferColNameFromExpr(aliasedExpr, formatter)
		require.NoError(t, err)

		assert.Equal(t, "formula_name", colHandle.Name, "Should extract column name from right operand")
		assert.Equal(t, "formula_uri", colHandle.Alias)
		assert.Contains(t, colHandle.DecoratedColumn, "||", "DecoratedColumn should contain concatenation operator")
		assert.False(t, colHandle.IsAggregateExpr, "Concatenation should not be marked as aggregate")
	})

	t.Run("String concatenation with literal on right side", func(t *testing.T) {
		query := "SELECT col_name || '_suffix' as modified_col FROM t"
		stmt, err := sqlparser.Parse(query)
		require.NoError(t, err)

		sel := stmt.(*sqlparser.Select)
		require.Len(t, sel.SelectExprs, 1)

		aliasedExpr := sel.SelectExprs[0].(*sqlparser.AliasedExpr)
		colHandle, err := parserutil.InferColNameFromExpr(aliasedExpr, formatter)
		require.NoError(t, err)

		assert.Equal(t, "col_name", colHandle.Name, "Should extract column name from left operand")
		assert.Equal(t, "modified_col", colHandle.Alias)
	})

	t.Run("Arithmetic expression with columns", func(t *testing.T) {
		query := "SELECT col_a + col_b as sum_cols FROM t"
		stmt, err := sqlparser.Parse(query)
		require.NoError(t, err)

		sel := stmt.(*sqlparser.Select)
		require.Len(t, sel.SelectExprs, 1)

		aliasedExpr := sel.SelectExprs[0].(*sqlparser.AliasedExpr)
		colHandle, err := parserutil.InferColNameFromExpr(aliasedExpr, formatter)
		require.NoError(t, err)

		// Should extract first available column name
		assert.Equal(t, "col_a", colHandle.Name, "Should extract column name from left operand")
		assert.Equal(t, "sum_cols", colHandle.Alias)
	})
}
