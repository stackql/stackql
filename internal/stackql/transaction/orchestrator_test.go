//nolint:testpackage // Tests the private SQL snapshot formatter without exposing it as API.
package transaction

import (
	"testing"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

func TestQueuedSQLPreservesQualifiedIdentifiers(t *testing.T) {
	for _, query := range []string{
		"INSERT INTO stackql_unstable_accessdemo.service.items(name) SELECT 'transaction-item'",
		"SELECT * FROM a.b.c.d",
		"SELECT * FROM a.b.\"name.with.dot\"",
	} {
		original, err := sqlparser.Parse(query)
		if err != nil {
			t.Fatal(err)
		}
		buf := sqlparser.NewTrackedBuffer(formatQueuedSQL)
		buf.Myprintf("%v", original)
		restored, err := sqlparser.Parse(buf.String())
		if err != nil {
			t.Fatalf("snapshot %q: %v", buf.String(), err)
		}
		var before, after []sqlparser.TableName
		collect := func(dst *[]sqlparser.TableName) sqlparser.Visit {
			return func(node sqlparser.SQLNode) (bool, error) {
				if name, ok := node.(sqlparser.TableName); ok {
					*dst = append(*dst, name)
				}
				return true, nil
			}
		}
		if err := sqlparser.Walk(collect(&before), original); err != nil {
			t.Fatal(err)
		}
		if err := sqlparser.Walk(collect(&after), restored); err != nil {
			t.Fatal(err)
		}
		if len(before) != len(after) {
			t.Fatalf("table count changed: %q", buf.String())
		}
		for i := range before {
			if before[i] != after[i] {
				t.Fatalf("identifier changed: %#v -> %#v", before[i], after[i])
			}
		}
	}
}
