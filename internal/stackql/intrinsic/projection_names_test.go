//nolint:testpackage // Verify host presentation metadata separately from SDK expression evaluation.
package intrinsic

import (
	"testing"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

func TestProjectionLabelsPreserveLegacyMultiArgumentNames(t *testing.T) {
	for _, tc := range []struct{ sql, name string }{
		{"select substr(u.name, 1, instr(u.name, '@') - 1) from p.s.users u", "name"},
		{"select split_part(u.name, '/', -1) from p.s.users u", "name"},
		{"select split_part(u.name, '/', -1) as short from p.s.users u", "short"},
		{"select upper(name) from p.s.users", "upper(name)"},
		{"select sum(size) from p.s.users", "sum(size)"},
	} {
		node := parseSelect(t, tc.sql)
		selected := node.SelectExprs[0].(*sqlparser.AliasedExpr)
		before := sqlparser.String(selected)
		got, err := projectionName(selected)
		if err != nil || got != tc.name {
			t.Fatalf("%s: name=%q err=%v want=%q", tc.sql, got, err, tc.name)
		}
		if sqlparser.String(selected) != before {
			t.Fatal("labeling changed the executable expression")
		}
	}
}
