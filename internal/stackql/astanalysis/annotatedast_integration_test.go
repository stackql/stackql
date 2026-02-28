package astanalysis

import (
	"testing"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/astanalysis/annotatedast"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Covers hasAwaitComment + IsAwait (major boost)

// Covers INSERT branch + GetInsertRowsIndirect
func TestAnnotatedAst_InsertRowsFlow(t *testing.T) {
	sql := `
	INSERT INTO test.table VALUES (1, 2)
	`

	stmt, err := sqlparser.Parse(sql)
	require.NoError(t, err)

	aa, err := annotatedast.NewAnnotatedAst(nil, stmt)
	require.NoError(t, err)

	insert := stmt.(*sqlparser.Insert)
	_, ok := aa.GetInsertRowsIndirect(insert)

	assert.False(t, ok)
}

// Covers SELECT indirect path
func TestAnnotatedAst_SelectIndirectFlow(t *testing.T) {
	sql := `
	SELECT * FROM test.table
	`

	stmt, err := sqlparser.Parse(sql)
	require.NoError(t, err)

	aa, err := annotatedast.NewAnnotatedAst(nil, stmt)
	require.NoError(t, err)

	sel := stmt.(*sqlparser.Select)
	_, ok := aa.GetSelectIndirect(sel)

	assert.False(t, ok)
}

// Covers GetPhysicalTable + GetMaterializedView branches
func TestAnnotatedAst_TableResolutionBranches(t *testing.T) {
	sql := `
	SELECT * FROM test.table
	`

	stmt, err := sqlparser.Parse(sql)
	require.NoError(t, err)

	aa, err := annotatedast.NewAnnotatedAst(nil, stmt)
	require.NoError(t, err)

	tab := sqlparser.TableName{
		Name: sqlparser.NewTableIdent("test"),
	}

	_, ok1 := aa.GetPhysicalTable(tab)
	_, ok2 := aa.GetMaterializedView(tab)

	assert.False(t, ok1)
	assert.False(t, ok2)
}
