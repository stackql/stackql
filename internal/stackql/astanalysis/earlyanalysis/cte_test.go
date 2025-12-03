package earlyanalysis_test

import (
	"testing"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCTEParsing(t *testing.T) {
	t.Run("Simple CTE is parsed correctly", func(t *testing.T) {
		query := "WITH cte AS (SELECT id, name FROM users) SELECT * FROM cte"
		stmt, err := sqlparser.Parse(query)
		require.NoError(t, err)

		sel, ok := stmt.(*sqlparser.Select)
		require.True(t, ok, "Statement should be a SELECT")

		// Check that With clause exists
		require.NotNil(t, sel.With, "WITH clause should exist")
		require.Len(t, sel.With.CTEs, 1, "Should have 1 CTE")

		// Check CTE name
		cte := sel.With.CTEs[0]
		assert.Equal(t, "cte", cte.Name.GetRawVal(), "CTE name should be 'cte'")

		// Check that CTE has a select statement
		require.NotNil(t, cte.Select, "CTE should have a select statement")
	})

	t.Run("Multiple CTEs are parsed correctly", func(t *testing.T) {
		query := "WITH a AS (SELECT 1 as x), b AS (SELECT 2 as y) SELECT * FROM a, b"
		stmt, err := sqlparser.Parse(query)
		require.NoError(t, err)

		sel, ok := stmt.(*sqlparser.Select)
		require.True(t, ok, "Statement should be a SELECT")

		// Check that With clause exists
		require.NotNil(t, sel.With, "WITH clause should exist")
		require.Len(t, sel.With.CTEs, 2, "Should have 2 CTEs")

		// Check CTE names
		assert.Equal(t, "a", sel.With.CTEs[0].Name.GetRawVal(), "First CTE name should be 'a'")
		assert.Equal(t, "b", sel.With.CTEs[1].Name.GetRawVal(), "Second CTE name should be 'b'")
	})

	t.Run("Recursive CTE is parsed correctly", func(t *testing.T) {
		query := "WITH RECURSIVE cte AS (SELECT 1 as n UNION ALL SELECT n + 1 FROM cte WHERE n < 10) SELECT * FROM cte"
		stmt, err := sqlparser.Parse(query)
		require.NoError(t, err)

		sel, ok := stmt.(*sqlparser.Select)
		require.True(t, ok, "Statement should be a SELECT")

		// Check that With clause exists with Recursive flag
		require.NotNil(t, sel.With, "WITH clause should exist")
		assert.True(t, sel.With.Recursive, "WITH clause should be RECURSIVE")
		require.Len(t, sel.With.CTEs, 1, "Should have 1 CTE")

		// Check CTE name
		assert.Equal(t, "cte", sel.With.CTEs[0].Name.GetRawVal(), "CTE name should be 'cte'")
	})

	t.Run("CTE with column aliases", func(t *testing.T) {
		query := "WITH cte(col1, col2) AS (SELECT id, name FROM users) SELECT * FROM cte"
		stmt, err := sqlparser.Parse(query)
		require.NoError(t, err)

		sel, ok := stmt.(*sqlparser.Select)
		require.True(t, ok, "Statement should be a SELECT")

		require.NotNil(t, sel.With, "WITH clause should exist")
		require.Len(t, sel.With.CTEs, 1, "Should have 1 CTE")

		cte := sel.With.CTEs[0]
		assert.Equal(t, "cte", cte.Name.GetRawVal(), "CTE name should be 'cte'")

		// Check column aliases if present
		require.Len(t, cte.Columns, 2, "CTE should have 2 column aliases")
		assert.Equal(t, "col1", cte.Columns[0].GetRawVal(), "First column alias should be 'col1'")
		assert.Equal(t, "col2", cte.Columns[1].GetRawVal(), "Second column alias should be 'col2'")
	})

	t.Run("Nested CTEs - CTE referencing another CTE", func(t *testing.T) {
		query := "WITH a AS (SELECT 1 as x), b AS (SELECT x * 2 as y FROM a) SELECT * FROM b"
		stmt, err := sqlparser.Parse(query)
		require.NoError(t, err)

		sel, ok := stmt.(*sqlparser.Select)
		require.True(t, ok, "Statement should be a SELECT")

		require.NotNil(t, sel.With, "WITH clause should exist")
		require.Len(t, sel.With.CTEs, 2, "Should have 2 CTEs")
	})
}

func TestCTERegistry(t *testing.T) {
	t.Run("CTE registry stores CTEs correctly", func(t *testing.T) {
		registry := make(map[string]*sqlparser.CommonTableExpr)

		query := "WITH cte1 AS (SELECT 1), cte2 AS (SELECT 2) SELECT * FROM cte1, cte2"
		stmt, err := sqlparser.Parse(query)
		require.NoError(t, err)

		sel := stmt.(*sqlparser.Select)
		require.NotNil(t, sel.With)

		// Simulate what the visitor does - register CTEs
		for _, cte := range sel.With.CTEs {
			cteName := cte.Name.GetRawVal()
			registry[cteName] = cte
		}

		// Verify registry contents
		assert.Len(t, registry, 2, "Registry should have 2 CTEs")
		assert.Contains(t, registry, "cte1", "Registry should contain 'cte1'")
		assert.Contains(t, registry, "cte2", "Registry should contain 'cte2'")
	})

	t.Run("CTE lookup works correctly", func(t *testing.T) {
		registry := make(map[string]*sqlparser.CommonTableExpr)

		query := "WITH my_cte AS (SELECT id, name FROM users) SELECT * FROM my_cte"
		stmt, err := sqlparser.Parse(query)
		require.NoError(t, err)

		sel := stmt.(*sqlparser.Select)
		require.NotNil(t, sel.With)

		// Register the CTE
		for _, cte := range sel.With.CTEs {
			cteName := cte.Name.GetRawVal()
			registry[cteName] = cte
		}

		// Verify we can look up the CTE
		_, isCTE := registry["my_cte"]
		assert.True(t, isCTE, "'my_cte' should be found in registry")

		// Verify non-CTE names are not found
		_, isNotCTE := registry["users"]
		assert.False(t, isNotCTE, "'users' should not be found in registry")
	})
}

func TestWindowFunctionParsing(t *testing.T) {
	t.Run("Window function with OVER clause is parsed correctly", func(t *testing.T) {
		query := "SELECT ROW_NUMBER() OVER (ORDER BY id) as row_num FROM t"
		stmt, err := sqlparser.Parse(query)
		require.NoError(t, err)

		sel, ok := stmt.(*sqlparser.Select)
		require.True(t, ok, "Statement should be a SELECT")

		require.Len(t, sel.SelectExprs, 1, "Should have 1 select expression")

		aliased, ok := sel.SelectExprs[0].(*sqlparser.AliasedExpr)
		require.True(t, ok, "Select expression should be aliased")

		funcExpr, ok := aliased.Expr.(*sqlparser.FuncExpr)
		require.True(t, ok, "Expression should be a FuncExpr")

		assert.Equal(t, "row_number", funcExpr.Name.Lowered(), "Function name should be 'row_number'")
		assert.NotNil(t, funcExpr.Over, "FuncExpr should have OVER clause")
	})

	t.Run("Window function with PARTITION BY is parsed correctly", func(t *testing.T) {
		query := "SELECT SUM(amount) OVER (PARTITION BY category ORDER BY date) as running_sum FROM t"
		stmt, err := sqlparser.Parse(query)
		require.NoError(t, err)

		sel := stmt.(*sqlparser.Select)
		aliased := sel.SelectExprs[0].(*sqlparser.AliasedExpr)
		funcExpr := aliased.Expr.(*sqlparser.FuncExpr)

		assert.Equal(t, "sum", funcExpr.Name.Lowered())
		require.NotNil(t, funcExpr.Over, "FuncExpr should have OVER clause")

		// Check partition by exists
		require.NotNil(t, funcExpr.Over.PartitionBy, "OVER clause should have PARTITION BY")
	})

	t.Run("Window function with frame specification", func(t *testing.T) {
		query := "SELECT SUM(value) OVER (ORDER BY date ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) as cumsum FROM t"
		stmt, err := sqlparser.Parse(query)
		require.NoError(t, err)

		sel := stmt.(*sqlparser.Select)
		aliased := sel.SelectExprs[0].(*sqlparser.AliasedExpr)
		funcExpr := aliased.Expr.(*sqlparser.FuncExpr)

		assert.NotNil(t, funcExpr.Over, "FuncExpr should have OVER clause")
	})

	t.Run("Multiple window functions in query", func(t *testing.T) {
		query := "SELECT ROW_NUMBER() OVER (ORDER BY id) as rn, RANK() OVER (ORDER BY score DESC) as rank FROM t"
		stmt, err := sqlparser.Parse(query)
		require.NoError(t, err)

		sel := stmt.(*sqlparser.Select)
		require.Len(t, sel.SelectExprs, 2, "Should have 2 select expressions")

		// Check first window function
		aliased1 := sel.SelectExprs[0].(*sqlparser.AliasedExpr)
		funcExpr1 := aliased1.Expr.(*sqlparser.FuncExpr)
		assert.NotNil(t, funcExpr1.Over, "First FuncExpr should have OVER clause")

		// Check second window function
		aliased2 := sel.SelectExprs[1].(*sqlparser.AliasedExpr)
		funcExpr2 := aliased2.Expr.(*sqlparser.FuncExpr)
		assert.NotNil(t, funcExpr2.Over, "Second FuncExpr should have OVER clause")
	})

	t.Run("Regular function without OVER clause", func(t *testing.T) {
		query := "SELECT UPPER(name) as upper_name FROM t"
		stmt, err := sqlparser.Parse(query)
		require.NoError(t, err)

		sel := stmt.(*sqlparser.Select)
		aliased := sel.SelectExprs[0].(*sqlparser.AliasedExpr)
		funcExpr := aliased.Expr.(*sqlparser.FuncExpr)

		assert.Equal(t, "upper", funcExpr.Name.Lowered())
		assert.Nil(t, funcExpr.Over, "UPPER() should not have OVER clause")
	})
}
