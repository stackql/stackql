package accessmanagement //nolint:testpackage // Exercises the private directory implementation.

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stackql-labs/omnisdk/public/principal"
	_ "modernc.org/sqlite"
)

func TestRDBMSDirectory(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	ctx := context.Background()
	if err := initializeSchema(ctx, db); err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{`INSERT INTO stackql_access_principals VALUES ('alice','{"team":"ops"}'), ('bob','{}')`, `INSERT INTO stackql_access_groups VALUES ('readers')`, `INSERT INTO stackql_access_identities VALUES ('one','same','alice'),('two','same','bob')`, `INSERT INTO stackql_access_memberships VALUES ('alice','readers')`} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	d, err := newDirectory(db)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		issuer, id string
		groups     int
	}{{"one", "alice", 1}, {"two", "bob", 0}} {
		i, _ := principal.NewIdentity(tc.issuer, "same")
		p, err := d.Resolve(ctx, i)
		if err != nil {
			t.Fatal(err)
		}
		if p.ID() != tc.id || len(p.Groups()) != tc.groups {
			t.Fatalf("principal=%s groups=%v", p.ID(), p.Groups())
		}
	}
	i, _ := principal.NewIdentity("missing", "same")
	if _, err := d.Resolve(ctx, i); err == nil {
		t.Fatal("missing identity accepted")
	}
	// No cached membership: a new resolution sees the current directory.
	if _, err := db.Exec(`DELETE FROM stackql_access_memberships`); err != nil {
		t.Fatal(err)
	}
	i, _ = principal.NewIdentity("one", "same")
	p, err := d.Resolve(ctx, i)
	if err != nil || len(p.Groups()) != 0 {
		t.Fatalf("stale directory: %v %v", p, err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := d.Resolve(canceled, i); err == nil {
		t.Fatal("ignored cancellation")
	}
}
