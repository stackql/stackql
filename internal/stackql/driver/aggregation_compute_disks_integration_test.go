package driver_test

import (
	"bufio"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"github.com/stackql/stackql/internal/stackql/config"
	. "github.com/stackql/stackql/internal/stackql/driver"
	"github.com/stackql/stackql/internal/stackql/entryutil"
	"github.com/stackql/stackql/internal/stackql/querysubmit"
	"github.com/stackql/stackql/internal/stackql/responsehandler"
	"github.com/stackql/stackql/internal/test/stackqltestutil"
	"github.com/stackql/stackql/internal/test/testobjects"

	log "github.com/sirupsen/logrus"

	lrucache "vitess.io/vitess/go/cache"
)

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func TestSelectComputeDisksOrderByCrtTmstpAsc(t *testing.T) {

	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(config.GetGoogleProviderString(), "text", "TestSelectComputeDisksOrderByCrtTmstpAsc")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	sqlEngine, err := stackqltestutil.BuildSQLEngine(*runtimeCtx)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	testSubject := func(t *testing.T, outFile *bufio.Writer) {

		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, strings.NewReader(""), lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), sqlEngine)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		tc, err := entryutil.GetTxnCounterManager(handlerCtx)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.TxnCounterMgr = tc

		handlerCtx.Query = testobjects.SelectGoogleComputeDisksOrderCreationTmstpAsc
		response := querysubmit.SubmitQuery(&handlerCtx)
		handlerCtx.Outfile = outFile
		responsehandler.HandleResponse(&handlerCtx, response)

		ProcessQuery(&handlerCtx)
	}

	stackqltestutil.SetupSimpleSelectGoogleComputeDisks(t, 1)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksOrderCrtTmstpAsc})

}

func TestSelectComputeDisksAggOrderBySizeAsc(t *testing.T) {

	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(config.GetGoogleProviderString(), "text", "TestSelectComputeDisksAggOrderBySizeAsc")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	sqlEngine, err := stackqltestutil.BuildSQLEngine(*runtimeCtx)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	testSubject := func(t *testing.T, outFile *bufio.Writer) {

		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, strings.NewReader(""), lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), sqlEngine)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		tc, err := entryutil.GetTxnCounterManager(handlerCtx)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.TxnCounterMgr = tc

		handlerCtx.Query = testobjects.SelectGoogleComputeDisksAggOrderSizeAsc
		response := querysubmit.SubmitQuery(&handlerCtx)
		handlerCtx.Outfile = outFile
		responsehandler.HandleResponse(&handlerCtx, response)

		ProcessQuery(&handlerCtx)
	}

	stackqltestutil.SetupSimpleSelectGoogleComputeDisks(t, 1)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksAggSizeOrderSizeAsc})

}

func TestSelectComputeDisksAggOrderBySizeDesc(t *testing.T) {

	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(config.GetGoogleProviderString(), "text", "TestSelectComputeDisksAggOrderBySizeDesc")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	sqlEngine, err := stackqltestutil.BuildSQLEngine(*runtimeCtx)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	testSubject := func(t *testing.T, outFile *bufio.Writer) {

		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, strings.NewReader(""), lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), sqlEngine)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		tc, err := entryutil.GetTxnCounterManager(handlerCtx)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.TxnCounterMgr = tc

		handlerCtx.Query = testobjects.SelectGoogleComputeDisksAggOrderSizeDesc
		response := querysubmit.SubmitQuery(&handlerCtx)
		handlerCtx.Outfile = outFile
		responsehandler.HandleResponse(&handlerCtx, response)

		ProcessQuery(&handlerCtx)
	}

	stackqltestutil.SetupSimpleSelectGoogleComputeDisks(t, 1)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksAggSizeOrderSizeDesc})

}

func TestSelectComputeDisksAggTotalSize(t *testing.T) {

	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(config.GetGoogleProviderString(), "text", "TestSelectComputeDisksAggTotalSize")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	sqlEngine, err := stackqltestutil.BuildSQLEngine(*runtimeCtx)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	testSubject := func(t *testing.T, outFile *bufio.Writer) {

		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, strings.NewReader(""), lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), sqlEngine)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		tc, err := entryutil.GetTxnCounterManager(handlerCtx)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.TxnCounterMgr = tc

		handlerCtx.Query = testobjects.SelectGoogleComputeDisksAggSizeTotal
		response := querysubmit.SubmitQuery(&handlerCtx)
		handlerCtx.Outfile = outFile
		responsehandler.HandleResponse(&handlerCtx, response)

		ProcessQuery(&handlerCtx)
	}

	stackqltestutil.SetupSimpleSelectGoogleComputeDisks(t, 1)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksAggSizeTotal})

}

func TestSelectComputeDisksAggTotalString(t *testing.T) {

	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(config.GetGoogleProviderString(), "text", "TestSelectComputeDisksAggTotalString")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	sqlEngine, err := stackqltestutil.BuildSQLEngine(*runtimeCtx)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	testSubject := func(t *testing.T, outFile *bufio.Writer) {

		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, strings.NewReader(""), lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), sqlEngine)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		tc, err := entryutil.GetTxnCounterManager(handlerCtx)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.TxnCounterMgr = tc

		handlerCtx.Query = testobjects.SelectGoogleComputeDisksAggStringTotal
		response := querysubmit.SubmitQuery(&handlerCtx)
		handlerCtx.Outfile = outFile
		responsehandler.HandleResponse(&handlerCtx, response)

		ProcessQuery(&handlerCtx)
	}

	stackqltestutil.SetupSimpleSelectGoogleComputeDisks(t, 1)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSelectComputeDisksAggStringTotal})

}
