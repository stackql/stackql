package driver_test

import (
	"bufio"
	"strings"
	"testing"

	"github.com/stackql/stackql/internal/stackql/entryutil"
	"github.com/stackql/stackql/internal/stackql/querysubmit"
	"github.com/stackql/stackql/internal/stackql/responsehandler"
	"github.com/stackql/stackql/internal/test/stackqltestutil"
	"github.com/stackql/stackql/internal/test/testobjects"

	lrucache "github.com/stackql/stackql-parser/go/cache"
)

//nolint:govet,lll,errcheck // legacy test
func TestSimpleAggGoogleContainerSubnetworksGroupedAllowedDriverOutputAsc(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "table", "TestSimpleAggGoogleContainerSubnetworksGroupedAllowedDriverOutputAsc")
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

		handlerCtx.SetQuery(testobjects.SimpleAggCountGroupedGoogleContainerSubnetworkAsc)
		querySubmitter := querysubmit.NewQuerySubmitter()
		prepareErr := querySubmitter.PrepareQuery(handlerCtx)
		if prepareErr != nil {
			t.Fatalf("Test failed: %v", prepareErr)
		}
		response := querySubmitter.SubmitQuery()
		responsehandler.HandleResponse(handlerCtx, response)
	}

	stackqltestutil.SetupSimpleSelectGoogleContainerAggAllowedSubnetworks(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSimpleAggCountGroupedGoogleCotainerSubnetworkTableFileAsc})
}

//nolint:govet,lll // legacy test
func TestSimpleAggGoogleContainerSubnetworksGroupedAllowedDriverOutputDesc(t *testing.T) {
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "table", "TestSimpleAggGoogleContainerSubnetworksGroupedAllowedDriverOutputDesc")
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

		handlerCtx.SetQuery(testobjects.SimpleAggCountGroupedGoogleContainerSubnetworkDesc)
		querySubmitter := querysubmit.NewQuerySubmitter()
		prepareErr := querySubmitter.PrepareQuery(handlerCtx)
		if prepareErr != nil {
			t.Fatalf("Test failed: %v", prepareErr)
		}
		response := querySubmitter.SubmitQuery()
		responsehandler.HandleResponse(handlerCtx, response)
	}

	stackqltestutil.SetupSimpleSelectGoogleContainerAggAllowedSubnetworks(t)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedSimpleAggCountGroupedGoogleCotainerSubnetworkTableFileDesc})
}
