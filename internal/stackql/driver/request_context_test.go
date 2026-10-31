package driver_test

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stackql/psql-wire/pkg/sqldata"
	lrucache "github.com/stackql/stackql-parser/go/cache"
	"github.com/stackql/stackql/internal/stackql/driver"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/test/stackqltestutil"
)

func TestCommitUsesCurrentRequestContext(t *testing.T) {
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
	// PG wire cancels each command's context as soon as that command completes.
	for _, query := range []string{
		"begin;",
		"create materialized view request_context_mv as select 'kept' as value;",
		"commit;",
		"select * from request_context_mv;",
		"drop materialized view request_context_mv;",
	} {
		ctx, cancel := context.WithCancel(context.Background())
		result, queryErr := backend.HandleSimpleQuery(ctx, query)
		if queryErr != nil {
			cancel()
			t.Fatalf("%s: %v", query, queryErr)
		}
		if query == "select * from request_context_mv;" {
			assertCommittedRequestResult(t, result)
		}
		if result != nil {
			if err := result.Close(); err != nil {
				cancel()
				t.Fatal(err)
			}
		}
		cancel()
	}
}

func assertCommittedRequestResult(t *testing.T, result sqldata.ISQLResultStream) {
	t.Helper()
	if result == nil {
		t.Fatal("committed materialized view returned no result")
	}
	rows, err := result.Read()
	if (err != nil && !errors.Is(err, io.EOF)) || rows == nil || len(rows.ToArr()) != 1 || rows.ToArr()[0]["value"] != "kept" {
		t.Fatalf("committed materialized view: result=%v, error=%v", rows, err)
	}
}
