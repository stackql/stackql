package driver_test

import (
	"bufio"
	"os"
	"strings"
	"testing"

	"github.com/stackql/stackql/internal/stackql/config"
	"github.com/stackql/stackql/internal/stackql/entryutil"
	"github.com/stackql/stackql/internal/stackql/querysubmit"
	"github.com/stackql/stackql/internal/stackql/responsehandler"
	"github.com/stackql/stackql/internal/test/stackqltestutil"
	"github.com/stackql/stackql/internal/test/testobjects"

	lrucache "vitess.io/vitess/go/cache"
)

func TestSimpleAggGoogleContainerSubnetworksGroupedAllowedDriverOutputAsc(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(config.GetGoogleProviderString(), "table", "TestSimpleAggGoogleContainerSubnetworksGroupedAllowedDriverOutputAsc")
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

		handlerCtx.Query = testobjects.SimpleAggCountGroupedGoogleContainerSubnetworkAsc
		response := querysubmit.SubmitQuery(&handlerCtx)
		handlerCtx.Outfile = outFile
		responsehandler.HandleResponse(&handlerCtx, response)
	}

	stackqltestutil.SetupSimpleSelectGoogleContainerAggAllowedSubnetworks(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSimpleAggCountGroupedGoogleCotainerSubnetworkTableFileAsc})

}

func TestSimpleAggGoogleContainerSubnetworksGroupedAllowedDriverOutputDesc(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(config.GetGoogleProviderString(), "table", "TestSimpleAggGoogleContainerSubnetworksGroupedAllowedDriverOutputDesc")
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

		handlerCtx.Query = testobjects.SimpleAggCountGroupedGoogleContainerSubnetworkDesc
		response := querysubmit.SubmitQuery(&handlerCtx)
		handlerCtx.Outfile = outFile
		responsehandler.HandleResponse(&handlerCtx, response)
	}

	stackqltestutil.SetupSimpleSelectGoogleContainerAggAllowedSubnetworks(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSimpleAggCountGroupedGoogleCotainerSubnetworkTableFileDesc})

}
