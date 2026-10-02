package sqlsplit_test

import (
	"reflect"
	"testing"

	"github.com/stackql/stackql/pkg/sqlsplit"
)

func TestStatements(t *testing.T) {
	cases := []struct {
		sql  string
		want []string
	}{
		{"", []string{}},
		{"   ", []string{}},
		{";", []string{}},
		{"select 1", []string{"select 1"}},
		{"select 1;", []string{"select 1"}},
		{" select 1 ;\n delete from t ; ", []string{"select 1", "delete from t"}},
		{"select ';' as semi", []string{"select ';' as semi"}}, // quoted separator does not split
		{"select 1 /* ; */; select 2", []string{"select 1 /* ; */", "select 2"}},
	}
	for _, c := range cases {
		t.Run(c.sql, func(t *testing.T) {
			if got := sqlsplit.Statements(c.sql); !reflect.DeepEqual(got, c.want) {
				t.Errorf("Statements(%q) = %q, want %q", c.sql, got, c.want)
			}
		})
	}
}
