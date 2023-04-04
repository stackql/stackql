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

//nolint:govet,lll // legacy test
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
	inputBundle, err := stackqltestutil.BuildInputBundle(*runtimeCtx)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	testSubject := func(t *testing.T, outFile *bufio.Writer) {
		rdr, err := os.Open(runtimeCtx.InfilePath)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, rdr, lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle)
		handlerCtx.SetOutfile(os.Stdout)
		handlerCtx.SetOutErrFile(os.Stderr)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.SetOutfile(outFile)
		handlerCtx.SetOutErrFile(os.Stderr)

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		dr, _ := NewStackQLDriver(handlerCtx)
		dr.ProcessQuery(handlerCtx.GetRawQuery())
	}

	stackqltestutil.SetupDependentInsertGoogleComputeDisks(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedComputeDisksDependentInsertAsyncFile})
}

//nolint:govet,lll // legacy test
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
	inputBundle, err := stackqltestutil.BuildInputBundle(*runtimeCtx)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	testSubject := func(t *testing.T, outFile *bufio.Writer) {
		rdr, err := os.Open(runtimeCtx.InfilePath)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, rdr, lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle)
		handlerCtx.SetOutfile(os.Stdout)
		handlerCtx.SetOutErrFile(os.Stderr)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.SetOutfile(outFile)
		handlerCtx.SetOutErrFile(os.Stderr)

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		dr, _ := NewStackQLDriver(handlerCtx)
		dr.ProcessQuery(handlerCtx.GetRawQuery())
	}

	stackqltestutil.SetupDependentInsertGoogleComputeDisks(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedComputeDisksDependentInsertAsyncFile})
}

//nolint:govet,lll // legacy test
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
	inputBundle, err := stackqltestutil.BuildInputBundle(*runtimeCtx)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	testSubject := func(t *testing.T, outFile *bufio.Writer) {
		rdr, err := os.Open(runtimeCtx.InfilePath)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, rdr, lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle)
		handlerCtx.SetOutfile(os.Stdout)
		handlerCtx.SetOutErrFile(os.Stderr)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.SetOutfile(outFile)
		handlerCtx.SetOutErrFile(os.Stderr)

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		dr, _ := NewStackQLDriver(handlerCtx)
		dr.ProcessQuery(handlerCtx.GetRawQuery())
	}

	stackqltestutil.SetupDependentInsertGoogleBQDatasets(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedBQDatasetsDependentInsertFile})
}

//nolint:govet,lll // legacy test
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
	inputBundle, err := stackqltestutil.BuildInputBundle(*runtimeCtx)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	testSubject := func(t *testing.T, outFile *bufio.Writer) {
		rdr, err := os.Open(runtimeCtx.InfilePath)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}
		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, rdr, lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle)
		handlerCtx.SetOutfile(os.Stdout)
		handlerCtx.SetOutErrFile(os.Stderr)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.SetOutfile(outFile)
		handlerCtx.SetOutErrFile(os.Stderr)

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		dr, _ := NewStackQLDriver(handlerCtx)
		dr.ProcessQuery(handlerCtx.GetRawQuery())
	}

	stackqltestutil.SetupExecGoogleOrganizationsGetIamPolicy(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectExecOrgGetIamPolicyAgg})
}
