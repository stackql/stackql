package driver_test

import (
	"context"
	"testing"

	lrucache "github.com/stackql/stackql-parser/go/cache"
	"github.com/stackql/stackql/internal/stackql/driver"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/test/stackqltestutil"
)

func transactionTestBackend(t *testing.T) (driver.StackQLDriver, handler.HandlerContext) {
	t.Helper()
	runtime, err := stackqltestutil.GetRuntimeCtx("", "json", t.Name())
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := stackqltestutil.BuildInputBundle(*runtime)
	if err != nil {
		t.Fatal(err)
	}
	hc, err := handler.NewHandlerCtx("", *runtime, lrucache.NewLRUCache(0), bundle, "test")
	if err != nil {
		t.Fatal(err)
	}
	backend, err := driver.NewStackQLDriver(hc)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(hc.ClearProviderTransaction)
	return backend, hc
}
func transactionTestQuery(backend driver.StackQLDriver, sql string) error {
	rows, err := backend.HandleSimpleQuery(context.Background(), sql)
	if rows != nil {
		if closeErr := rows.Close(); err == nil {
			err = closeErr
		}
	}
	return err
}

func TestTransactionParseFailureBlocksCommitUntilRollback(t *testing.T) {
	backend, hc := transactionTestBackend(t)
	for _, badSQL := range []string{"SELEC 1;", "SELECT 'unterminated;"} {
		if err := transactionTestQuery(backend, "BEGIN;"); err != nil {
			t.Fatal(err)
		}
		if err := transactionTestQuery(backend, badSQL); err == nil {
			t.Fatal("syntax error accepted")
		}
		if err := transactionTestQuery(backend, "COMMIT;"); err == nil {
			t.Fatal("committed after syntax error")
		}
		session, handle, active := hc.ProviderTransaction()
		if !active {
			t.Fatal("aborted transaction lost recovery state")
		}
		state, err := session.Status(context.Background(), handle)
		if err != nil || state.Status != "aborted" {
			t.Fatal(state, err)
		}
		if err = transactionTestQuery(backend, "ROLLBACK;"); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTransactionRejectsLocalDDLWithoutTouchingOtherSession(t *testing.T) {
	backend, hc := transactionTestBackend(t)
	other, err := driver.NewStackQLDriver(hc.NewClientContext())
	if err != nil {
		t.Fatal(err)
	}
	if err = transactionTestQuery(other, "CREATE TABLE protected_local (value text);"); err != nil {
		t.Fatal(err)
	}
	for _, ddl := range []string{
		"CREATE TABLE rejected_local (value text);",
		"CREATE VIEW rejected_local AS SELECT 1 AS value;",
		"CREATE MATERIALIZED VIEW rejected_local AS SELECT 1 AS value;",
		"DROP TABLE protected_local;",
	} {
		if err = transactionTestQuery(backend, "BEGIN;"); err != nil {
			t.Fatal(err)
		}
		if err = transactionTestQuery(backend, ddl); err == nil {
			t.Fatalf("DDL accepted: %s", ddl)
		}
		if err = transactionTestQuery(backend, "COMMIT;"); err == nil {
			t.Fatal("committed rejected DDL")
		}
		if err = transactionTestQuery(backend, "ROLLBACK;"); err != nil {
			t.Fatal(err)
		}
		if err = transactionTestQuery(other, "SELECT * FROM protected_local;"); err != nil {
			t.Fatal("other session's relation lost", err)
		}
		if err = transactionTestQuery(other, "SELECT * FROM rejected_local;"); err == nil {
			t.Fatal("rejected DDL created a relation")
		}
	}
	if err = transactionTestQuery(other, "DROP TABLE protected_local;"); err != nil {
		t.Fatal(err)
	}
}
