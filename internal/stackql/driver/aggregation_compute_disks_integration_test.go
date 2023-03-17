package driver_test

import (
	"bufio"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	. "github.com/stackql/stackql/internal/stackql/driver"
	"github.com/stackql/stackql/internal/stackql/entryutil"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/querysubmit"
	"github.com/stackql/stackql/internal/stackql/responsehandler"
	"github.com/stackql/stackql/internal/test/stackqltestutil"
	"github.com/stackql/stackql/internal/test/testobjects"

	lrucache "github.com/stackql/stackql-parser/go/cache"
)

func TestMain(m *testing.M) {
	logging.GetLogger().SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

//nolint:govet,lll,errcheck // legacy test
func TestSelectComputeDisksOrderByCrtTmstpAsc(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSelectComputeDisksOrderByCrtTmstpAsc")
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

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.SetQuery(testobjects.SelectGoogleComputeDisksOrderCreationTmstpAsc)
		dr, _ := NewStackQLDriver(handlerCtx)
		querySubmitter := querysubmit.NewQuerySubmitter()
		response := querySubmitter.SubmitQuery(handlerCtx)
		handlerCtx.SetOutfile(outFile)
		responsehandler.HandleResponse(handlerCtx, response)

		dr.ProcessQuery(handlerCtx)
	}

	stackqltestutil.SetupSimpleSelectGoogleComputeDisks(t, 1)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksOrderCrtTmstpAsc})
}

//nolint:govet,lll,errcheck // legacy test
func TestSelectComputeDisksAggOrderBySizeAsc(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSelectComputeDisksAggOrderBySizeAsc")
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

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.SetQuery(testobjects.SelectGoogleComputeDisksAggOrderSizeAsc)
		dr, _ := NewStackQLDriver(handlerCtx)
		querySubmitter := querysubmit.NewQuerySubmitter()
		response := querySubmitter.SubmitQuery(handlerCtx)
		handlerCtx.SetOutfile(outFile)
		responsehandler.HandleResponse(handlerCtx, response)

		dr.ProcessQuery(handlerCtx)
	}

	stackqltestutil.SetupSimpleSelectGoogleComputeDisks(t, 1)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksAggSizeOrderSizeAsc})
}

//nolint:govet,lll,errcheck // legacy test
func TestSelectComputeDisksAggOrderBySizeDesc(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSelectComputeDisksAggOrderBySizeDesc")
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

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.SetQuery(testobjects.SelectGoogleComputeDisksAggOrderSizeDesc)
		dr, _ := NewStackQLDriver(handlerCtx)
		querySubmitter := querysubmit.NewQuerySubmitter()
		response := querySubmitter.SubmitQuery(handlerCtx)
		handlerCtx.SetOutfile(outFile)
		responsehandler.HandleResponse(handlerCtx, response)

		dr.ProcessQuery(handlerCtx)
	}

	stackqltestutil.SetupSimpleSelectGoogleComputeDisks(t, 1)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksAggSizeOrderSizeDesc})
}

//nolint:govet,lll,errcheck // legacy test
func TestSelectComputeDisksAggTotalSize(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSelectComputeDisksAggTotalSize")
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

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.SetQuery(testobjects.SelectGoogleComputeDisksAggSizeTotal)
		dr, _ := NewStackQLDriver(handlerCtx)
		querySubmitter := querysubmit.NewQuerySubmitter()
		response := querySubmitter.SubmitQuery(handlerCtx)
		handlerCtx.SetOutfile(outFile)
		responsehandler.HandleResponse(handlerCtx, response)

		dr.ProcessQuery(handlerCtx)
	}

	stackqltestutil.SetupSimpleSelectGoogleComputeDisks(t, 1)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksAggSizeTotal})
}

//nolint:govet,lll,errcheck // legacy test
func TestSelectComputeDisksAggTotalString(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSelectComputeDisksAggTotalString")
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

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.SetQuery(testobjects.SelectGoogleComputeDisksAggStringTotal)
		dr, _ := NewStackQLDriver(handlerCtx)
		querySubmitter := querysubmit.NewQuerySubmitter()
		response := querySubmitter.SubmitQuery(handlerCtx)
		handlerCtx.SetOutfile(outFile)
		responsehandler.HandleResponse(handlerCtx, response)

		dr.ProcessQuery(handlerCtx)
	}

	stackqltestutil.SetupSimpleSelectGoogleComputeDisks(t, 1)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksAggStringTotal})
}
