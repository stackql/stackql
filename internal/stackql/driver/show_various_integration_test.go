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

	lrucache "vitess.io/vitess/go/cache"
)

func TestSimpleShowResourcesFiltered(t *testing.T) {

	testSubject := func(t *testing.T, outFile *bufio.Writer) {

		runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSimpleShowResourcesFiltered")
		if err != nil {
			t.Fatalf("TestSimpleShowResourcesFiltered failed: %v", err)
		}
		inputBundle, err := stackqltestutil.BuildInputBundle(*runtimeCtx)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
		showInsertFile, err := util.GetFilePathFromRepositoryRoot(testobjects.SimpleShowResourcesFilteredFile)
		if err != nil {
			t.Fatalf("TestSimpleShowResourcesFiltered failed: %v", err)
		}
		runtimeCtx.InfilePath = showInsertFile
		runtimeCtx.OutputFormat = "text"
		runtimeCtx.CSVHeadersDisable = true

		rdr, err := os.Open(runtimeCtx.InfilePath)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, rdr, lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.Outfile = outFile
		handlerCtx.OutErrFile = os.Stderr

		ProcessQuery(&handlerCtx)
	}

	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedShowResourcesFilteredFile})

}

func TestSimpleShowBQDatasets(t *testing.T) {

	testSubject := func(t *testing.T, outFile *bufio.Writer) {

		runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSimpleShowBQDatasets")
		if err != nil {
			t.Fatalf("TestSimpleShowResourcesFiltered failed: %v", err)
		}
		inputBundle, err := stackqltestutil.BuildInputBundle(*runtimeCtx)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
		showInsertFile, err := util.GetFilePathFromRepositoryRoot(testobjects.SimpleShowmethodsGoogleBQDatasetsFile)
		if err != nil {
			t.Fatalf("TestSimpleShowResourcesFiltered failed: %v", err)
		}
		runtimeCtx.InfilePath = showInsertFile
		runtimeCtx.OutputFormat = "csv"
		runtimeCtx.CSVHeadersDisable = false

		rdr, err := os.Open(runtimeCtx.InfilePath)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, rdr, lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.Outfile = outFile
		handlerCtx.OutErrFile = os.Stderr

		ProcessQuery(&handlerCtx)
	}

	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedShowMethodsGoogleBQDatasetsFile})

}

func TestSimpleShowGoogleStorageBuckets(t *testing.T) {

	testSubject := func(t *testing.T, outFile *bufio.Writer) {

		runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSimpleShowGoogleStorageBuckets")
		if err != nil {
			t.Fatalf("TestSimpleShowResourcesFiltered failed: %v", err)
		}
		inputBundle, err := stackqltestutil.BuildInputBundle(*runtimeCtx)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
		showInsertFile, err := util.GetFilePathFromRepositoryRoot(testobjects.SimpleShowmethodsGoogleStorageBucketsFile)
		if err != nil {
			t.Fatalf("TestSimpleShowResourcesFiltered failed: %v", err)
		}
		runtimeCtx.InfilePath = showInsertFile
		runtimeCtx.OutputFormat = "csv"
		runtimeCtx.CSVHeadersDisable = false

		rdr, err := os.Open(runtimeCtx.InfilePath)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, rdr, lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.Outfile = outFile
		handlerCtx.OutErrFile = os.Stderr

		ProcessQuery(&handlerCtx)
	}

	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedShowMethodsGoogleStorageBucketsFile})

}
