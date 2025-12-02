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

//nolint:govet,lll // test file
func TestSelectComputeDisksWindowRowNumber(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSelectComputeDisksWindowRowNumber")
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

		handlerCtx.SetQuery(testobjects.SelectGoogleComputeDisksWindowRowNumber)
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

	stackqltestutil.SetupSimpleSelectGoogleComputeDisks(t, 1)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksWindowRowNumber})
}

//nolint:govet,lll // test file
func TestSelectComputeDisksWindowRank(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSelectComputeDisksWindowRank")
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

		handlerCtx.SetQuery(testobjects.SelectGoogleComputeDisksWindowRank)
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

	stackqltestutil.SetupSimpleSelectGoogleComputeDisks(t, 1)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksWindowRank})
}

//nolint:govet,lll // test file
func TestSelectComputeDisksWindowSum(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSelectComputeDisksWindowSum")
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

		handlerCtx.SetQuery(testobjects.SelectGoogleComputeDisksWindowSum)
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

	stackqltestutil.SetupSimpleSelectGoogleComputeDisks(t, 1)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksWindowSum})
}

//nolint:govet,lll // test file
func TestSelectComputeDisksCTESimple(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSelectComputeDisksCTESimple")
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

		handlerCtx.SetQuery(testobjects.SelectGoogleComputeDisksCTESimple)
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

	stackqltestutil.SetupSimpleSelectGoogleComputeDisks(t, 1)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksCTESimple})
}

//nolint:govet,lll // test file
func TestSelectComputeDisksCTEWithAgg(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSelectComputeDisksCTEWithAgg")
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

		handlerCtx.SetQuery(testobjects.SelectGoogleComputeDisksCTEWithAgg)
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

	stackqltestutil.SetupSimpleSelectGoogleComputeDisks(t, 1)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksCTEWithAgg})
}

//nolint:govet,lll // test file
func TestSelectComputeDisksCTEMultiple(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSelectComputeDisksCTEMultiple")
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

		handlerCtx.SetQuery(testobjects.SelectGoogleComputeDisksCTEMultiple)
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

	// Multiple CTEs need two API calls (one for each CTE that queries the provider)
	stackqltestutil.SetupSimpleSelectGoogleComputeDisks(t, 2)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksCTEMultiple})
}
