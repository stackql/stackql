package driver_test

import (
	"bufio"
	"os"
	"strings"
	"testing"

	. "github.com/stackql/stackql/internal/stackql/driver"
	"github.com/stackql/stackql/internal/stackql/entryutil"
	"github.com/stackql/stackql/internal/stackql/querysubmit"
	"github.com/stackql/stackql/internal/stackql/responsehandler"
	"github.com/stackql/stackql/internal/test/stackqltestutil"
	"github.com/stackql/stackql/internal/test/testobjects"

	lrucache "vitess.io/vitess/go/cache"
)

func TestSelectComputeDisksOrderByCrtTmstpAscPaginated(t *testing.T) {

	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSelectComputeDisksOrderByCrtTmstpAscPaginated")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	runtimeCtx.HTTPMaxResults = 5
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
		response := querysubmit.SubmitQuery(handlerCtx)
		handlerCtx.SetOutfile(outFile)
		responsehandler.HandleResponse(handlerCtx, response)

		ProcessQuery(handlerCtx)
	}

	stackqltestutil.SetupSimpleSelectGoogleComputeDisksPaginated(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksOrderCrtTmstpAscPaginated})

}

func TestSelectComputeDisksAggOrderBySizeAscPaginated(t *testing.T) {

	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSelectComputeDisksAggOrderBySizeAscPaginated")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	runtimeCtx.HTTPMaxResults = 5
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
		response := querysubmit.SubmitQuery(handlerCtx)
		handlerCtx.SetOutfile(outFile)
		responsehandler.HandleResponse(handlerCtx, response)

		ProcessQuery(handlerCtx)
	}

	stackqltestutil.SetupSimpleSelectGoogleComputeDisksPaginated(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksAggPaginatedSizeOrderSizeAsc})

}

func TestSelectComputeDisksAggOrderBySizeDescPaginated(t *testing.T) {

	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSelectComputeDisksAggOrderBySizeDescPaginated")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	runtimeCtx.HTTPMaxResults = 5
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
		response := querysubmit.SubmitQuery(handlerCtx)
		handlerCtx.SetOutfile(outFile)
		responsehandler.HandleResponse(handlerCtx, response)

		ProcessQuery(handlerCtx)
	}

	stackqltestutil.SetupSimpleSelectGoogleComputeDisksPaginated(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksAggPaginatedSizeOrderSizeDesc})

}

func TestSelectComputeDisksAggTotalSizePaginated(t *testing.T) {

	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSelectComputeDisksAggTotalSizePaginated")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	runtimeCtx.HTTPMaxResults = 5
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
		response := querysubmit.SubmitQuery(handlerCtx)
		handlerCtx.SetOutfile(outFile)
		responsehandler.HandleResponse(handlerCtx, response)

		ProcessQuery(handlerCtx)
	}

	stackqltestutil.SetupSimpleSelectGoogleComputeDisksPaginated(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksAggPaginatedSizeTotal})

}

func TestSelectComputeDisksAggTotalStringPaginated(t *testing.T) {

	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSelectComputeDisksAggTotalStringPaginated")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	runtimeCtx.HTTPMaxResults = 5
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
		response := querysubmit.SubmitQuery(handlerCtx)
		handlerCtx.SetOutfile(outFile)
		responsehandler.HandleResponse(handlerCtx, response)

		ProcessQuery(handlerCtx)
	}

	stackqltestutil.SetupSimpleSelectGoogleComputeDisksPaginated(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksAggPaginatedStringTotal})

}
