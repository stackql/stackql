package driver_test

import (
	"strings"
	"testing"

	"bufio"

	. "github.com/stackql/stackql/internal/stackql/driver"

	"github.com/stackql/stackql/internal/stackql/entryutil"
	"github.com/stackql/stackql/internal/stackql/querysubmit"
	"github.com/stackql/stackql/internal/stackql/responsehandler"

	"github.com/stackql/stackql/internal/test/stackqltestutil"
	"github.com/stackql/stackql/internal/test/testobjects"

	lrucache "github.com/stackql/stackql-parser/go/cache"
)

//nolint:govet,lll // legacy test
func TestSelectComputeDisksOrderByCrtTmstpAscPlusJsonExtract(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "csv", "TestSelectComputeDisksOrderByCrtTmstpAscPlusJsonExtract")
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

		handlerCtx.SetQuery(testobjects.SelectGoogleComputeDisksOrderCreationTmstpAscPlusJSONExtract)
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
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksOrderCrtTmstpAscPlusJsonExtract})
}

//nolint:govet,lll // legacy test
func TestSelectComputeDisksOrderByCrtTmstpAscPlusCoalesceJsonExtract(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "csv", "TestSelectComputeDisksOrderByCrtTmstpAscPlusCoalesceJsonExtract")
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

		handlerCtx.SetQuery(testobjects.SelectGoogleComputeDisksOrderCreationTmstpAscPlusJSONExtractCoalesce)
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
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksOrderCrtTmstpAscPlusJsonExtractCoalesce})
}

//nolint:govet,lll // legacy test
func TestSelectComputeDisksOrderByCrtTmstpAscPlusCoalesceJsonInstr(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "csv", "TestSelectComputeDisksOrderByCrtTmstpAscPlusCoalesceJsonInstr")
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

		handlerCtx.SetQuery(testobjects.SelectGoogleComputeDisksOrderCreationTmstpAscPlusJsonExtractInstr)
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
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksOrderCrtTmstpAscPlusJsonExtractInstr})
}
