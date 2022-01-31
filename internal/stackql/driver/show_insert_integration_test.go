package driver_test

import (
	"os"
	"testing"

	"bufio"

	. "github.com/stackql/stackql/internal/stackql/driver"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/stackql/internal/stackql/config"
	"github.com/stackql/stackql/internal/stackql/entryutil"

	"github.com/stackql/stackql/internal/test/stackqltestutil"
	"github.com/stackql/stackql/internal/test/testobjects"

	lrucache "vitess.io/vitess/go/cache"
)

func TestSimpleShowInsertComputeAddressesRequired(t *testing.T) {

	testSubject := func(t *testing.T, outFile *bufio.Writer) {

		runtimeCtx, err := stackqltestutil.GetRuntimeCtx(config.GetGoogleProviderString(), "text", "TestSimpleShowInsertComputeAddressesRequired")
		if err != nil {
			t.Fatalf("TestSimpleTemplateComputeAddressesRequired failed: %v", err)
		}
		sqlEngine, err := stackqltestutil.BuildSQLEngine(*runtimeCtx)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
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

		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, rdr, lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), sqlEngine)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.Outfile = outFile
		handlerCtx.OutErrFile = os.Stderr

		tc, err := entryutil.GetTxnCounterManager(handlerCtx)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
		handlerCtx.TxnCounterMgr = tc

		ProcessQuery(&handlerCtx)
	}

	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedShowInsertAddressesRequiredFile})

}

func TestSimpleShowInsertBiqueryDatasets(t *testing.T) {

	testSubject := func(t *testing.T, outFile *bufio.Writer) {

		runtimeCtx, err := stackqltestutil.GetRuntimeCtx(config.GetGoogleProviderString(), "text", "TestSimpleShowInsertBiqueryDatasets")
		if err != nil {
			t.Fatalf("TestSimpleShowInsertBiqueryDatasets failed: %v", err)
		}
		sqlEngine, err := stackqltestutil.BuildSQLEngine(*runtimeCtx)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
		showInsertFile, err := util.GetFilePathFromRepositoryRoot(testobjects.ShowInsertBQDatasetsFile)
		if err != nil {
			t.Fatalf("TestSimpleShowInsertBiqueryDatasets failed: %v", err)
		}
		runtimeCtx.InfilePath = showInsertFile
		runtimeCtx.CSVHeadersDisable = true

		rdr, err := os.Open(runtimeCtx.InfilePath)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, rdr, lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), sqlEngine)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.Outfile = outFile
		handlerCtx.OutErrFile = os.Stderr

		tc, err := entryutil.GetTxnCounterManager(handlerCtx)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
		handlerCtx.TxnCounterMgr = tc

		ProcessQuery(&handlerCtx)
	}

	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedShowInsertBQDatasetsFile})

}

func TestSimpleShowInsertBiqueryDatasetsRequired(t *testing.T) {

	testSubject := func(t *testing.T, outFile *bufio.Writer) {

		runtimeCtx, err := stackqltestutil.GetRuntimeCtx(config.GetGoogleProviderString(), "text", "TestSimpleShowInsertBiqueryDatasetsRequired")
		if err != nil {
			t.Fatalf("TestSimpleShowInsertBiqueryDatasetsRequired failed: %v", err)
		}
		sqlEngine, err := stackqltestutil.BuildSQLEngine(*runtimeCtx)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
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

		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, rdr, lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), sqlEngine)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.Outfile = outFile
		handlerCtx.OutErrFile = os.Stderr

		tc, err := entryutil.GetTxnCounterManager(handlerCtx)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
		handlerCtx.TxnCounterMgr = tc

		ProcessQuery(&handlerCtx)
	}

	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedShowInsertBQDatasetsRequiredFile})

}
