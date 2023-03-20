package driver_test

import (
	"io/ioutil"
	"net/url"
	"os"
	"strings"
	"testing"

	"bufio"

	. "github.com/stackql/stackql/internal/stackql/driver"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/stackql/internal/stackql/entryutil"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/provider"
	"github.com/stackql/stackql/internal/stackql/querysubmit"
	"github.com/stackql/stackql/internal/stackql/responsehandler"

	"github.com/stackql/stackql/internal/test/stackqltestutil"
	"github.com/stackql/stackql/internal/test/testhttpapi"
	"github.com/stackql/stackql/internal/test/testobjects"

	lrucache "github.com/stackql/stackql-parser/go/cache"
)

//nolint:lll // legacy test
func TestSimpleSelectGoogleComputeInstanceDriver(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSimpleSelectGoogleComputeInstanceDriver")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	path := "/compute/v1/projects/testing-project/zones/australia-southeast1-b/instances"
	url := &url.URL{
		Path: path,
	}
	ex := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", url, "compute.googleapis.com", testobjects.SimpleSelectGoogleComputeInstanceResponse, nil)
	expectations := map[string]testhttpapi.HTTPRequestExpectations{
		"compute.googleapis.com" + path: ex,
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

	handlerCtx, err := handler.GetHandlerCtx(testobjects.SimpleSelectGoogleComputeInstance, *runtimeCtx, lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle)
	handlerCtx.SetOutfile(os.Stdout)
	handlerCtx.SetOutErrFile(os.Stderr)

	dr, _ := NewStackQLDriver(handlerCtx)

	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	dr.ProcessQuery(handlerCtx)

	t.Logf("simple select driver integration test passed")
}

//nolint:lll,errcheck,govet // legacy test
func TestSimpleSelectGoogleComputeInstanceDriverOutput(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSimpleSelectGoogleComputeInstanceDriverOutput")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	inputBundle, err := stackqltestutil.BuildInputBundle(*runtimeCtx)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	testSubject := func(t *testing.T, outFile *bufio.Writer) {
		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, strings.NewReader(""), lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle)
		handlerCtx.SetOutfile(os.Stdout)
		handlerCtx.SetOutErrFile(os.Stderr)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.SetOutfile(outFile)
		handlerCtx.SetOutErrFile(os.Stderr)

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.SetQuery(testobjects.SimpleSelectGoogleComputeInstance)
		querySubmitter := querysubmit.NewQuerySubmitter()
		response := querySubmitter.SubmitQuery(handlerCtx)
		handlerCtx.SetOutfile(outFile)
		responsehandler.HandleResponse(handlerCtx, response)
	}

	stackqltestutil.SetupSimpleSelectGoogleComputeInstance(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSimpleSelectGoogleComputeInstanceTextFile01, testobjects.ExpectedSimpleSelectGoogleComputeInstanceTextFile02})
}

//nolint:lll,errcheck,govet // legacy test
func TestSimpleSelectGoogleComputeInstanceDriverOutputRepeated(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSimpleSelectGoogleComputeInstanceDriverOutputRepeated")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	inputBundle, err := stackqltestutil.BuildInputBundle(*runtimeCtx)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	testSubject := func(t *testing.T, outFile *bufio.Writer) {
		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, strings.NewReader(""), lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle)
		handlerCtx.SetOutfile(os.Stdout)
		handlerCtx.SetOutErrFile(os.Stderr)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.SetOutfile(outFile)
		handlerCtx.SetOutErrFile(os.Stderr)

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.SetQuery(testobjects.SimpleSelectGoogleComputeInstance)
		querySubmitter := querysubmit.NewQuerySubmitter()
		response := querySubmitter.SubmitQuery(handlerCtx)
		handlerCtx.SetOutfile(outFile)
		responsehandler.HandleResponse(handlerCtx, response)
	}

	stackqltestutil.SetupSimpleSelectGoogleComputeInstance(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSimpleSelectGoogleComputeInstanceTextFile01, testobjects.ExpectedSimpleSelectGoogleComputeInstanceTextFile02})
}

//nolint:govet,lll,errcheck // legacy test
func TestSimpleSelectGoogleContainerSubnetworksAllowedDriverOutput(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSimpleSelectGoogleContainerSubnetworksAllowedDriverOutput")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	inputBundle, err := stackqltestutil.BuildInputBundle(*runtimeCtx)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	testSubject := func(t *testing.T, outFile *bufio.Writer) {
		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, strings.NewReader(""), lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle)
		handlerCtx.SetOutfile(os.Stdout)
		handlerCtx.SetOutErrFile(os.Stderr)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.SetOutfile(outFile)
		handlerCtx.SetOutErrFile(os.Stderr)

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.SetQuery(testobjects.SimpleSelectGoogleContainerSubnetworks)
		querySubmitter := querysubmit.NewQuerySubmitter()
		response := querySubmitter.SubmitQuery(handlerCtx)
		handlerCtx.SetOutfile(outFile)
		responsehandler.HandleResponse(handlerCtx, response)
	}

	stackqltestutil.SetupSimpleSelectGoogleContainerAggAllowedSubnetworks(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSimpleSelectGoogleCotainerSubnetworkTextFile01, testobjects.ExpectedSimpleSelectGoogleCotainerSubnetworkTextFile02})
}

//nolint:govet,lll,errcheck // legacy test
func TestSimpleInsertGoogleComputeNetworkAsync(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSimpleInsertGoogleComputeNetworkAsync")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	inputBundle, err := stackqltestutil.BuildInputBundle(*runtimeCtx)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	testSubject := func(t *testing.T, outFile *bufio.Writer) {
		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, strings.NewReader(""), lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle)
		handlerCtx.SetOutfile(os.Stdout)
		handlerCtx.SetOutErrFile(os.Stderr)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.SetOutfile(outFile)
		handlerCtx.SetOutErrFile(os.Stderr)

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.SetQuery(testobjects.SimpleInsertComputeNetwork)
		querySubmitter := querysubmit.NewQuerySubmitter()
		response := querySubmitter.SubmitQuery(handlerCtx)
		handlerCtx.SetOutfile(outFile)
		responsehandler.HandleResponse(handlerCtx, response)
	}

	stackqltestutil.SetupSimpleInsertGoogleComputeNetworks(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedComputeNetworkInsertAsyncFile})
}

//nolint:govet,lll // legacy test
func TestK8sTheHardWayAsync(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestK8sTheHardWayAsync")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	inputBundle, err := stackqltestutil.BuildInputBundle(*runtimeCtx)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	testSubject := func(t *testing.T, outFile *bufio.Writer) {
		k8sthwRenderedFile, err := util.GetFilePathFromRepositoryRoot(testobjects.ExpectedK8STheHardWayRenderedFile)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
		megaQueryConcat, err := ioutil.ReadFile(k8sthwRenderedFile)
		if err != nil {
			t.Fatalf("%v", err)
		}
		runtimeCtx.InfilePath = k8sthwRenderedFile
		runtimeCtx.CSVHeadersDisable = true

		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, strings.NewReader(""), lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle)
		handlerCtx.SetOutfile(os.Stdout)
		handlerCtx.SetOutErrFile(os.Stderr)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.SetOutfile(outFile)
		handlerCtx.SetOutErrFile(os.Stderr)

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		dr, _ := NewStackQLDriver(handlerCtx)

		handlerCtx.SetRawQuery(strings.TrimSpace(string(megaQueryConcat)))
		dr.ProcessQuery(handlerCtx)
	}

	stackqltestutil.SetupK8sTheHardWayE2eSuccess(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedK8STheHardWayAsyncFile})
}

//nolint:lll // legacy test
func TestSimpleDryRunK8sTheHardWayDriver(t *testing.T) {
	testSubject := func(t *testing.T, outFile *bufio.Writer) {
		runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSimpleDryRunK8sTheHardWayDriver")
		if err != nil {
			t.Fatalf("TestSimpleDryRunDriver failed: %v", err)
		}
		inputBundle, err := stackqltestutil.BuildInputBundle(*runtimeCtx)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
		templateFile, err := util.GetFilePathFromRepositoryRoot(testobjects.K8STheHardWayTemplateFile)
		if err != nil {
			t.Fatalf("TestSimpleDryRunDriver failed: %v", err)
		}
		templateCtxFile, err := util.GetFilePathFromRepositoryRoot(testobjects.K8STheHardWayTemplateContextFile)
		if err != nil {
			t.Fatalf("TestSimpleDryRunDriver failed: %v", err)
		}
		runtimeCtx.InfilePath = templateFile
		runtimeCtx.TemplateCtxFilePath = templateCtxFile
		runtimeCtx.DryRunFlag = true
		runtimeCtx.CSVHeadersDisable = true

		rdr, err := os.Open(runtimeCtx.InfilePath)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, rdr, lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle)
		handlerCtx.SetOutfile(os.Stdout)
		handlerCtx.SetOutErrFile(os.Stderr)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.SetOutfile(outFile)
		handlerCtx.SetOutErrFile(os.Stderr)

		dr, _ := NewStackQLDriver(handlerCtx)
		dr.ProcessDryRun(handlerCtx)
	}

	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedK8STheHardWayRenderedFile})
}
