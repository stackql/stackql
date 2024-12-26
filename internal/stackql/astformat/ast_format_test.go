package astformat //nolint:testpackage // don't use an underscore in package name

import (
	"testing"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/parser"

	"github.com/stretchr/testify/assert"
)

// sqlNodes are used for both.
//
//nolint:gochecknoglobals // allows test functions to share test cases, only used by ast_format_test, unique names
var astFormatTestselectNode, astFormatTestdropNode sqlparser.Statement

func TestMain(m *testing.M) {
	p, _ := parser.NewParser()
	astFormatTestselectNode, _ = p.ParseQuery("select 1+2*3 from a WHERE a.v1 = 3")
	astFormatTestdropNode, _ = p.ParseQuery("DROP TABLE tab")
	m.Run()
}

func TestPostgresSelectExprsFormatter(t *testing.T) {
	// only covers SELECT and DROP Statements.
	// Doesn't cover most actual SQL Nodes.
	// no constructor for ColName, ColIdent, ...
	selNode, ok1 := astFormatTestselectNode.(sqlparser.SQLNode)
	droNode, ok2 := astFormatTestdropNode.(sqlparser.SQLNode)
	if !ok1 || !ok2 {
		t.Errorf("selectNode and/or dropNode are not of type sqlparser.SQLNode")
		return
	}
	tests := []struct {
		name     string
		node     sqlparser.SQLNode
		expected string
	}{
		{
			"PostgresSelectExprsFormatter: select",
			selNode,
			"select 1 + 2 * 3 from \"a\" where \"a\".v1 = 3",
		},
		{
			"PostgresSelectExprsFormatter: drop",
			droNode,
			"drop table \"tab\"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := sqlparser.NewTrackedBuffer(nil)

			PostgresSelectExprsFormatter(buf, tt.node)

			// assert.Equal(t, fmt.Sprintf("TYPE: %T\n",tt.node), "").
			switch tt.node.(type) { //nolint:gocritic // switch makes it easily extendable in case we want to add more test cases
			case *sqlparser.GroupConcatExpr:
				assert.Equal(t, true, false)
			}
			assert.Equal(t, tt.expected, buf.String())
		})
	}
}

func TestString(t *testing.T) {
	selNode, ok1 := astFormatTestselectNode.(sqlparser.SQLNode)
	droNode, ok2 := astFormatTestdropNode.(sqlparser.SQLNode)
	if !ok1 || !ok2 {
		t.Errorf("selectNode and/or dropNode are not of type sqlparser.SQLNode")
		return
	}

	tests := []struct {
		name     string
		node     sqlparser.SQLNode
		expected string
	}{
		{
			"string: select query",
			selNode,
			"select 1 + 2 * 3 from \"a\" where \"a\".\"v1\" = 3",
		},
		{
			"string: drop query",
			droNode,
			"drop table \"tab\"",
		},
	}
	var formatter sqlparser.NodeFormatter = PostgresSelectExprsFormatter
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := String(tt.node, formatter)
			assert.Equal(t, tt.expected, got)
		})
	}
}
