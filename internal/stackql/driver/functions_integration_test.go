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

	lrucache "vitess.io/vitess/go/cache"
)

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

		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, strings.NewReader(""), lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.Query = testobjects.SelectGoogleComputeDisksOrderCreationTmstpAscPlusJsonExtract
		response := querysubmit.SubmitQuery(&handlerCtx)
		handlerCtx.Outfile = outFile
		responsehandler.HandleResponse(&handlerCtx, response)

		ProcessQuery(&handlerCtx)
	}

	stackqltestutil.SetupSimpleSelectGoogleComputeDisks(t, 1)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksOrderCrtTmstpAscPlusJsonExtract})

}

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

		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, strings.NewReader(""), lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.Query = testobjects.SelectGoogleComputeDisksOrderCreationTmstpAscPlusJsonExtractCoalesce
		response := querysubmit.SubmitQuery(&handlerCtx)
		handlerCtx.Outfile = outFile
		responsehandler.HandleResponse(&handlerCtx, response)

		ProcessQuery(&handlerCtx)
	}

	stackqltestutil.SetupSimpleSelectGoogleComputeDisks(t, 1)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksOrderCrtTmstpAscPlusJsonExtractCoalesce})

}

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

		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, strings.NewReader(""), lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.Query = testobjects.SelectGoogleComputeDisksOrderCreationTmstpAscPlusJsonExtractInstr
		response := querysubmit.SubmitQuery(&handlerCtx)
		handlerCtx.Outfile = outFile
		responsehandler.HandleResponse(&handlerCtx, response)

		ProcessQuery(&handlerCtx)
	}

	stackqltestutil.SetupSimpleSelectGoogleComputeDisks(t, 1)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksOrderCrtTmstpAscPlusJsonExtractInstr})

}
