package driver_test

import (
	"bufio"
	"io/ioutil"
	"net/url"
	"os"
	"strings"
	"testing"

	. "github.com/stackql/stackql/internal/stackql/driver"
	"github.com/stackql/stackql/internal/stackql/querysubmit"
	"github.com/stackql/stackql/internal/stackql/responsehandler"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/stackql/internal/stackql/entryutil"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/provider"

	"github.com/stackql/stackql/internal/test/stackqltestutil"
	"github.com/stackql/stackql/internal/test/testhttpapi"
	"github.com/stackql/stackql/internal/test/testobjects"

	lrucache "vitess.io/vitess/go/cache"
)

func TestSelectOktaApplicationAppsDriver(t *testing.T) {
	// SimpleOktaApplicationsAppsListResponseFile

	responseFile1, err := util.GetFilePathFromRepositoryRoot(testobjects.SimpleOktaApplicationsAppsListResponseFile)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	responseBytes1, err := ioutil.ReadFile(responseFile1)
	if err != nil {
		t.Fatalf("%v", err)
	}

	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSelectOktaApplicationAppsDriver")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	path := "/api/v1/apps"
	url := &url.URL{
		Path: path,
	}
	ex := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", url, "some-silly-subdomain.okta.com", string(responseBytes1), nil)
	expectations := map[string]testhttpapi.HTTPRequestExpectations{
		"some-silly-subdomain.okta.com" + path: *ex,
	}
	exp := testhttpapi.NewExpectationStore(1)
	for k, v := range expectations {
		exp.Put(k, v)
	}
	testhttpapi.StartServer(t, exp)
	provider.DummyAuth = true

	sqlEng, err := stackqltestutil.BuildSQLEngine(*runtimeCtx)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	handlerCtx, err := handler.GetHandlerCtx(testobjects.SimpleSelectOktaApplicationApps, *runtimeCtx, lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), sqlEng)
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

	ProcessQuery(&handlerCtx)

	t.Logf("simple select driver integration test passed")
}

func TestSimpleSelectOktaApplicationAppsDriverOutput(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSimpleSelectOktaApplicationAppsDriverOutput")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	sqlEngine, err := stackqltestutil.BuildSQLEngine(*runtimeCtx)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	testSubject := func(t *testing.T, outFile *bufio.Writer) {

		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, strings.NewReader(""), lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), sqlEngine)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		tc, err := entryutil.GetTxnCounterManager(handlerCtx)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
		handlerCtx.TxnCounterMgr = tc

		handlerCtx.Outfile = outFile
		handlerCtx.OutErrFile = os.Stderr

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.Query = testobjects.SimpleSelectOktaApplicationApps
		response := querysubmit.SubmitQuery(&handlerCtx)
		handlerCtx.Outfile = outFile
		responsehandler.HandleResponse(&handlerCtx, response)
	}

	stackqltestutil.SetupSelectOktaApplicationApps(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectOktaApplicationAppsJson})

}
