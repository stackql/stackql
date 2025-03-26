package driver_test

import (
	"os"
	"testing"

	"bufio"

	. "github.com/stackql/stackql/internal/stackql/driver"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/stackql/internal/stackql/entryutil"

	"github.com/stackql/stackql/internal/test/stackqltestutil"
	"github.com/stackql/stackql/internal/test/testobjects"

	lrucache "github.com/stackql/stackql-parser/go/cache"
)

//nolint:lll // legacy test
func TestSimpleShowInsertComputeAddressesRequired(t *testing.T) {
	testSubject := func(t *testing.T, outFile *bufio.Writer) {
		runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSimpleShowInsertComputeAddressesRequired")
		if err != nil {
			t.Fatalf("TestSimpleTemplateComputeAddressesRequired failed: %v", err)
		}
		inputBundle, err := stackqltestutil.BuildInputBundle(*runtimeCtx)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
		inputBundle.WithStdOut(outFile)
		showInsertFile, err := util.GetFilePathFromRepositoryRoot(testobjects.ShowInsertAddressesRequiredInputFile)
		if err != nil {
			t.Fatalf("TestSimpleTemplateComputeAddressesRequired failed: %v", err)
		}
		runtimeCtx.InfilePath = showInsertFile
		runtimeCtx.CSVHeadersDisable = true

		rdr, err := os.Open(runtimeCtx.InfilePath)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, rdr, lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle, true)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		dr, _ := NewStackQLDriver(handlerCtx)
		dr.ProcessQuery(handlerCtx.GetRawQuery())
	}

	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedShowInsertAddressesRequiredFile})
}

//nolint:lll // legacy test
func TestSimpleShowInsertBiqueryDatasetsRequired(t *testing.T) {
	testSubject := func(t *testing.T, outFile *bufio.Writer) {
		runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSimpleShowInsertBiqueryDatasetsRequired")
		if err != nil {
			t.Fatalf("TestSimpleShowInsertBiqueryDatasetsRequired failed: %v", err)
		}
		inputBundle, err := stackqltestutil.BuildInputBundle(*runtimeCtx)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
		inputBundle.WithStdOut(outFile)
		showInsertFile, err := util.GetFilePathFromRepositoryRoot(testobjects.ShowInsertBQDatasetsRequiredFile)
		if err != nil {
			t.Fatalf("TestSimpleShowInsertBiqueryDatasetsRequired failed: %v", err)
		}
		runtimeCtx.InfilePath = showInsertFile
		runtimeCtx.CSVHeadersDisable = true

		rdr, err := os.Open(runtimeCtx.InfilePath)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, rdr, lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle, true)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		dr, _ := NewStackQLDriver(handlerCtx)
		dr.ProcessQuery(handlerCtx.GetRawQuery())
	}

	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedShowInsertBQDatasetsRequiredFile})
}
