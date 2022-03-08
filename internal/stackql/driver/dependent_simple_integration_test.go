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

func TestSimpleInsertDependentGoogleComputeDiskAsync(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSimpleInsertDependentGoogleComputeDiskAsync")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	inputFile, err := util.GetFilePathFromRepositoryRoot(testobjects.SimpleInsertDependentComputeDisksFile)
	if err != nil {
		t.Fatalf("TestSimpleInsertDependentGoogleComputeNetworkAsync failed: %v", err)
	}
	runtimeCtx.InfilePath = inputFile
	sqlEngine, err := stackqltestutil.BuildSQLEngine(*runtimeCtx)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	testSubject := func(t *testing.T, outFile *bufio.Writer) {

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

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		tc, err := entryutil.GetTxnCounterManager(handlerCtx)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
		handlerCtx.TxnCounterMgr = tc

		ProcessQuery(&handlerCtx)
	}

	stackqltestutil.SetupDependentInsertGoogleComputeDisks(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedComputeDisksDependentInsertAsyncFile})

}

func TestSimpleInsertDependentGoogleComputeDiskAsyncReversed(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSimpleInsertDependentGoogleComputeDiskAsyncReversed")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	inputFile, err := util.GetFilePathFromRepositoryRoot(testobjects.SimpleInsertDependentComputeDisksReversedFile)
	if err != nil {
		t.Fatalf("TestSimpleInsertDependentGoogleComputeNetworkAsync failed: %v", err)
	}
	runtimeCtx.InfilePath = inputFile
	sqlEngine, err := stackqltestutil.BuildSQLEngine(*runtimeCtx)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	testSubject := func(t *testing.T, outFile *bufio.Writer) {

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

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		tc, err := entryutil.GetTxnCounterManager(handlerCtx)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
		handlerCtx.TxnCounterMgr = tc

		ProcessQuery(&handlerCtx)
	}

	stackqltestutil.SetupDependentInsertGoogleComputeDisks(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedComputeDisksDependentInsertAsyncFile})

}

func TestSimpleInsertDependentGoogleBQDatasetAsync(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", "TestSimpleInsertDependentGoogleBQDatasetAsync")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	inputFile, err := util.GetFilePathFromRepositoryRoot(testobjects.SimpleInsertDependentBQDatasetFile)
	if err != nil {
		t.Fatalf("TestSimpleInsertDependentGoogleComputeNetworkAsync failed: %v", err)
	}
	runtimeCtx.InfilePath = inputFile
	sqlEngine, err := stackqltestutil.BuildSQLEngine(*runtimeCtx)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	testSubject := func(t *testing.T, outFile *bufio.Writer) {

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

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		tc, err := entryutil.GetTxnCounterManager(handlerCtx)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
		handlerCtx.TxnCounterMgr = tc

		ProcessQuery(&handlerCtx)
	}

	stackqltestutil.SetupDependentInsertGoogleBQDatasets(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedBQDatasetsDependentInsertFile})

}

func TestSimpleSelectExecDependentGoogleOrganizationsGetIamPolicy(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "csv", "TestSimpleSelectExecDependentGoogleOrganizationsGetIamPolicy")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	inputFile, err := util.GetFilePathFromRepositoryRoot(testobjects.SimpleSelectExecDependentOrgIamPolicyFile)
	if err != nil {
		t.Fatalf("TestSimpleSelectExecDependentGoogleOrganizationsGetIamPolicy failed: %v", err)
	}
	runtimeCtx.InfilePath = inputFile
	sqlEngine, err := stackqltestutil.BuildSQLEngine(*runtimeCtx)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	testSubject := func(t *testing.T, outFile *bufio.Writer) {

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

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		tc, err := entryutil.GetTxnCounterManager(handlerCtx)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
		handlerCtx.TxnCounterMgr = tc

		ProcessQuery(&handlerCtx)
	}

	stackqltestutil.SetupExecGoogleOrganizationsGetIamPolicy(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectExecOrgGetIamPolicyAgg})

}
