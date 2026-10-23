package intrinsic //nolint:testpackage // Exercises the host analyzer boundary.

import (
	"context"
	"testing"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

func TestMutationDerivedSourceUsesStatementLocalHandle(t *testing.T) {
	ctx := newStagingTestCtx(t, "")
	original, err := sqlparser.Parse("insert into stackql_unstable_google.cloudkms.key_rings(projectsId, locationsId, keyRingId) select s.project, 'global', s.id from (select 'p' as project, 'r' as id) s returning name")
	if err != nil {
		t.Fatal(err)
	}
	before := sqlparser.String(original)
	prepared, relations, err := prepareMutationRelations(ctx, original, "")
	if err != nil {
		t.Fatal(err)
	}
	if sqlparser.String(original) != before {
		t.Fatal("planning mutated the caller's statement")
	}
	if len(relations) != 2 {
		t.Fatalf("relations=%v", relations)
	}
	translated, err := newDocTranslator("", sqliteDialect{}).mutation(prepared)
	if err != nil {
		t.Fatal(err)
	}
	if len(translated.getQuery().From()) != 1 {
		t.Fatal("lost mutation source")
	}
	for _, relation := range []omnisdk.Relation{relations[translated.getQuery().From()[0].Resource().Handle()]} {
		rows, err := relation.Open(context.Background(), nil)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		if !rows.Next() || rows.Row()["_omni_insert_0"] != "p" || rows.Row()["_omni_insert_2"] != "r" {
			t.Fatalf("source row=%v error=%v", rows.Row(), rows.Err())
		}
		if rows.Next() || rows.Err() != nil {
			t.Fatalf("source continuation=%v", rows.Err())
		}
	}
}

func TestAwaitDeleteReturningIsRejectedBeforeExecution(t *testing.T) {
	statement, err := sqlparser.Parse("delete /*+ AWAIT */ from stackql_unstable_google.compute.firewalls where project = 'p' and firewall = 'f' returning kind")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := prepareMutationRelations(nil, statement, ""); err == nil || err.Error() != "returning from asynchronous delete is disallowed" {
		t.Fatalf("error=%v", err)
	}
}
