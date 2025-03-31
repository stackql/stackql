package querysubmit_test

import (
	"errors"
	"io"
	"net/url"
	"testing"

	. "github.com/stackql/stackql/internal/stackql/querysubmit"
	"gotest.tools/assert"

	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/provider"

	"github.com/stackql/stackql/internal/test/stackqltestutil"
	"github.com/stackql/stackql/internal/test/testhttpapi"
	"github.com/stackql/stackql/internal/test/testobjects"

	lrucache "github.com/stackql/stackql-parser/go/cache"
)

func TestSimpleSelectGoogleComputeInstanceQuerySubmit(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(
		testobjects.GetGoogleProviderString(), "text", "TestSimpleSelectGoogleComputeInstanceQuerySubmit")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	path := "/compute/v1/projects/testing-project/zones/australia-southeast1-b/instances"
	url := &url.URL{
		Path: path,
	}
	ex := testhttpapi.NewHTTPRequestExpectations(
		nil, nil, "GET", url, testobjects.GoogleComputeHost, testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	exp := testhttpapi.NewExpectationStore(1)
	exp.Put(testobjects.GoogleComputeHost+path, ex)

	testhttpapi.StartServer(t, exp)
	provider.DummyAuth = true

	inputBundle, err := stackqltestutil.BuildInputBundle(*runtimeCtx)

	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	handlerCtx, err := handler.NewHandlerCtx(
		testobjects.SimpleSelectGoogleComputeInstance,
		*runtimeCtx,
		lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle)

	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	handlerCtx.SetQuery(testobjects.SimpleSelectGoogleComputeInstance)
	qs := NewQuerySubmitter()
	prepareErr := qs.PrepareQuery(handlerCtx)
	if prepareErr != nil {
		t.Fatalf("Test failed: %v", prepareErr)
	}
	response := qs.SubmitQuery()

	if response.GetSQLResult() == nil {
		t.Fatalf("response is unexpectedly nil")
	}

	r, err := response.GetSQLResult().Read()

	assert.Assert(t, errors.Is(err, io.EOF))

	assert.Assert(t, len(r.GetRows()) == 2)

	t.Logf("simple select driver integration test passed")
}
