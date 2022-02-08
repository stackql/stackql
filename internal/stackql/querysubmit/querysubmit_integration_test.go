package querysubmit_test

import (
	"net/url"
	"os"
	"testing"

	"github.com/stackql/stackql/internal/stackql/entryutil"
	. "github.com/stackql/stackql/internal/stackql/querysubmit"

	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/provider"

	"github.com/stackql/stackql/internal/test/stackqltestutil"
	"github.com/stackql/stackql/internal/test/testhttpapi"
	"github.com/stackql/stackql/internal/test/testobjects"

	lrucache "vitess.io/vitess/go/cache"
)

func TestSimpleSelectGoogleComputeInstanceQuerySubmit(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSimpleSelectGoogleComputeInstanceQuerySubmit")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	path := "/compute/v1/projects/testing-project/zones/australia-southeast1-b/instances"
	url := &url.URL{
		Path: path,
	}
	ex := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", url, testobjects.GoogleComputeHost, testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	exp := testhttpapi.NewExpectationStore(1)
	exp.Put(testobjects.GoogleComputeHost+path, *ex)

	testhttpapi.StartServer(t, exp)
	provider.DummyAuth = true

	sqlEng, err := stackqltestutil.BuildSQLEngine(*runtimeCtx)

	handlerCtx, err := handler.GetHandlerCtx(testobjects.SimpleSelectGoogleComputeInstance, *runtimeCtx, lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), sqlEng)
	handlerCtx.Outfile = os.Stdout
	handlerCtx.OutErrFile = os.Stderr

	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	tc, err := entryutil.GetTxnCounterManager(handlerCtx)

	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	handlerCtx.TxnCounterMgr = tc

	handlerCtx.Query = testobjects.SimpleSelectGoogleComputeInstance
	response := SubmitQuery(&handlerCtx)

	if response.GetSQLResult() == nil {
		t.Fatalf("response is unexpectedly nil")
	}

	if len(response.GetSQLResult().Rows) != 2 {
		t.Fatalf("response size not as expected, actual != expected: %d != %d", len(response.GetSQLResult().Rows), 2)
	}

	t.Logf("simple select driver integration test passed")
}
