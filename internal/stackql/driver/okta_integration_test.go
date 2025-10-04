package driver_test

import (
	"bufio"
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

	lrucache "github.com/stackql/stackql-parser/go/cache"
)

//nolint:lll // legacy test
func TestSelectOktaApplicationAppsDriver(t *testing.T) {
	// SimpleOktaApplicationsAppsListResponseFile

	responseFile1, err := util.GetFilePathFromRepositoryRoot(testobjects.SimpleOktaApplicationsAppsListResponseFile)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	responseBytes1, err := os.ReadFile(responseFile1)
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
		"some-silly-subdomain.okta.com" + path: ex,
	}
	exp := testhttpapi.NewExpectationStore(1)
	for k, v := range expectations {
		exp.Put(k, v)
	}
	testhttpapi.StartServer(t, exp)
	provider.DummyAuth = true

	inputBundle, err := stackqltestutil.BuildInputBundle(*runtimeCtx)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	handlerCtx, err := handler.NewHandlerCtx(testobjects.SimpleSelectOktaApplicationApps, *runtimeCtx, lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle)

	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	dr, _ := NewStackQLDriver(handlerCtx)
	dr.ProcessQuery(handlerCtx.GetRawQuery())

	t.Logf("simple select driver integration test passed")
}

//nolint:govet,lll // legacy test
func TestSimpleSelectOktaApplicationAppsDriverOutput(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSimpleSelectOktaApplicationAppsDriverOutput")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	inputBundle, err := stackqltestutil.BuildInputBundle(*runtimeCtx)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	testSubject := func(t *testing.T, outFile *bufio.Writer) {
		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, strings.NewReader(""), lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle.WithStdOut(outFile), true)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
		querySubmitter := querysubmit.NewQuerySubmitter()
		handlerCtx.SetQuery(testobjects.SimpleSelectOktaApplicationApps)

		prepareErr := querySubmitter.PrepareQuery(handlerCtx)
		if prepareErr != nil {
			t.Fatalf("Test failed: %v", prepareErr)
		}
		response := querySubmitter.SubmitQuery()
		responsehandler.HandleResponse(handlerCtx, response)
	}

	stackqltestutil.SetupSelectOktaApplicationApps(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectOktaApplicationAppsJSON})
}
