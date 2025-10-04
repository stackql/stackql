package driver_test

import (
	"bufio"
	"strings"
	"testing"

	. "github.com/stackql/stackql/internal/stackql/driver"
	"github.com/stackql/stackql/internal/stackql/entryutil"
	"github.com/stackql/stackql/internal/stackql/querysubmit"
	"github.com/stackql/stackql/internal/stackql/responsehandler"
	"github.com/stackql/stackql/internal/test/stackqltestutil"
	"github.com/stackql/stackql/internal/test/testobjects"

	lrucache "github.com/stackql/stackql-parser/go/cache"
)

//nolint:govet,lll // legacy test
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
		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, strings.NewReader(""), lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle.WithStdOut(outFile), true)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.SetQuery(testobjects.SelectGoogleComputeDisksOrderCreationTmstpAsc)
		dr, _ := NewStackQLDriver(handlerCtx)
		querySubmitter := querysubmit.NewQuerySubmitter()
		prepareErr := querySubmitter.PrepareQuery(handlerCtx)
		if prepareErr != nil {
			t.Fatalf("Test failed: %v", prepareErr)
		}
		response := querySubmitter.SubmitQuery()
		responsehandler.HandleResponse(handlerCtx, response)

		dr.ProcessQuery(handlerCtx.GetRawQuery())
	}

	stackqltestutil.SetupSimpleSelectGoogleComputeDisksPaginated(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksOrderCrtTmstpAscPaginated})
}

//nolint:govet,lll,errcheck // legacy test
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
		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, strings.NewReader(""), lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle.WithStdOut(outFile), true)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.SetQuery(testobjects.SelectGoogleComputeDisksAggOrderSizeAsc)
		dr, _ := NewStackQLDriver(handlerCtx)
		querySubmitter := querysubmit.NewQuerySubmitter()
		prepareErr := querySubmitter.PrepareQuery(handlerCtx)
		if prepareErr != nil {
			t.Fatalf("Test failed: %v", prepareErr)
		}
		response := querySubmitter.SubmitQuery()
		responsehandler.HandleResponse(handlerCtx, response)

		dr.ProcessQuery(handlerCtx.GetRawQuery())
	}

	stackqltestutil.SetupSimpleSelectGoogleComputeDisksPaginated(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksAggPaginatedSizeOrderSizeAsc})
}

//nolint:govet,lll // legacy test
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
		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, strings.NewReader(""), lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle.WithStdOut(outFile), true)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.SetQuery(testobjects.SelectGoogleComputeDisksAggOrderSizeDesc)
		dr, _ := NewStackQLDriver(handlerCtx)
		querySubmitter := querysubmit.NewQuerySubmitter()
		prepareErr := querySubmitter.PrepareQuery(handlerCtx)
		if prepareErr != nil {
			t.Fatalf("Test failed: %v", prepareErr)
		}
		response := querySubmitter.SubmitQuery()
		responsehandler.HandleResponse(handlerCtx, response)

		dr.ProcessQuery(handlerCtx.GetRawQuery())
	}

	stackqltestutil.SetupSimpleSelectGoogleComputeDisksPaginated(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksAggPaginatedSizeOrderSizeDesc})
}

//nolint:govet,lll // legacy test
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
		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, strings.NewReader(""), lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle.WithStdOut(outFile), true)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.SetQuery(testobjects.SelectGoogleComputeDisksAggSizeTotal)
		dr, _ := NewStackQLDriver(handlerCtx)
		querySubmitter := querysubmit.NewQuerySubmitter()
		prepareErr := querySubmitter.PrepareQuery(handlerCtx)
		if prepareErr != nil {
			t.Fatalf("Test failed: %v", prepareErr)
		}
		response := querySubmitter.SubmitQuery()
		responsehandler.HandleResponse(handlerCtx, response)

		dr.ProcessQuery(handlerCtx.GetRawQuery())
	}

	stackqltestutil.SetupSimpleSelectGoogleComputeDisksPaginated(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksAggPaginatedSizeTotal})
}

//nolint:govet,lll // legacy test
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
		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, strings.NewReader(""), lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle.WithStdOut(outFile), true)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.SetQuery(testobjects.SelectGoogleComputeDisksAggStringTotal)
		dr, _ := NewStackQLDriver(handlerCtx)
		querySubmitter := querysubmit.NewQuerySubmitter()
		prepareErr := querySubmitter.PrepareQuery(handlerCtx)
		if prepareErr != nil {
			t.Fatalf("Test failed: %v", prepareErr)
		}
		response := querySubmitter.SubmitQuery()
		responsehandler.HandleResponse(handlerCtx, response)

		dr.ProcessQuery(handlerCtx.GetRawQuery())
	}

	stackqltestutil.SetupSimpleSelectGoogleComputeDisksPaginated(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksAggPaginatedStringTotal})
}
