package annotatedast

import (
	"testing"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/astindirect"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stretchr/testify/assert"
)

// ─────────────────────────────────────────────────────────────
// Core coverage test
// ─────────────────────────────────────────────────────────────
func TestAnnotatedAstCoverage(t *testing.T) {

	sql := "SELECT * FROM google.compute.instances"
	stmt, err := sqlparser.Parse(sql)
	assert.NoError(t, err)

	aa, err := NewAnnotatedAst(nil, stmt)
	assert.NoError(t, err)
	assert.NotNil(t, aa)

	// Init internal maps
	if concrete, ok := aa.(*standardAnnotatedAst); ok {
		concrete.physicalTableRefs = make(map[string]astindirect.Indirect)
		concrete.materializedViewRefs = make(map[string]astindirect.Indirect)
		concrete.tableIndirects = make(map[string]astindirect.Indirect)
		concrete.whereParamMaps = make(map[*sqlparser.Where]parserutil.ParameterMap)
	}

	// --- IsAwait ---
	t.Run("IsAwait", func(t *testing.T) {
		assert.False(t, aa.IsAwait(stmt))
		assert.False(t, aa.IsAwait(&sqlparser.Insert{}))
		assert.False(t, aa.IsAwait(&sqlparser.Exec{}))
	})

	// --- Tables / Views / Indirects ---
	t.Run("Tables_Views_Indirects", func(t *testing.T) {
		tab := sqlparser.TableName{
			Name: sqlparser.NewTableIdent("test_table"),
		}

		var ind astindirect.Indirect = nil

		aa.SetPhysicalTable(tab, ind)
		_, ok := aa.GetPhysicalTable(tab)
		assert.True(t, ok)

		aa.SetMaterializedView(tab, ind)
		_, ok = aa.GetMaterializedView(tab)
		assert.True(t, ok)

		aa.SetIndirect(tab, ind)
		assert.NotNil(t, aa.GetIndirects())
	})
}

// ─────────────────────────────────────────────────────────────
// Getter / Setter coverage
// ─────────────────────────────────────────────────────────────
func TestAnnotatedAst_GettersAndSetters(t *testing.T) {

	stmt, err := sqlparser.Parse("SELECT * FROM test.t")
	assert.NoError(t, err)

	aa, err := NewAnnotatedAst(nil, stmt)
	assert.NoError(t, err)

	// AST
	assert.Equal(t, stmt, aa.GetAST())

	// Exec indirect
	exec := &sqlparser.Exec{}
	aa.SetExecIndirect(exec, nil)
	_, ok := aa.GetExecIndirect(exec)
	assert.True(t, ok)

	// Select indirect
	sel := &sqlparser.Select{}
	aa.SetSelectIndirect(sel, nil)
	_, ok = aa.GetSelectIndirect(sel)
	assert.True(t, ok)

	// Insert rows indirect
	ins := &sqlparser.Insert{}
	aa.SetInsertRowsIndirect(ins, nil)
	_, ok = aa.GetInsertRowsIndirect(ins)
	assert.True(t, ok)

	// Where params
	where := &sqlparser.Where{}
	aa.SetWhereParamMapsEntry(where, nil)
	_, ok = aa.GetWhereParamMapsEntry(where)
	assert.True(t, ok)

	// Select metadata
	aa.SetSelectMetadata(sel, nil)
	_, ok = aa.GetSelectMetadata(sel)
	assert.True(t, ok)

	// Read only
	assert.False(t, aa.IsReadOnly())
}

// ─────────────────────────────────────────────────────────────
// Indirect + counter coverage
// ─────────────────────────────────────────────────────────────
func TestAnnotatedAst_IndirectAndCounter(t *testing.T) {

	stmt, err := sqlparser.Parse("SELECT * FROM test.t")
	assert.NoError(t, err)

	aa, err := NewAnnotatedAst(nil, stmt)
	assert.NoError(t, err)

	concrete := aa.(*standardAnnotatedAst)
	concrete.tableIndirects = make(map[string]astindirect.Indirect)

	tab := sqlparser.TableName{
		Name: sqlparser.NewTableIdent("my_table"),
	}

	aa.SetIndirect(tab, nil)
	ind, ok := aa.GetIndirect(tab)
	assert.True(t, ok)
	assert.Nil(t, ind)

	// Subquery counter
	assert.Equal(t, 0, concrete.GetSubequeryTableCount())
}
