package driver_test

import (
	"os"
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

func TestUnionAllSelectComputeDisksOrderByCrtTmstpAscPlusCoalesceJsonExtract(t *testing.T) {

	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "csv", "TestUnionAllSelectComputeDisksOrderByCrtTmstpAscPlusCoalesceJsonExtract")
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}
	inputBundle, err := stackqltestutil.BuildInputBundle(*runtimeCtx)
	if err != nil {
		t.Fatalf("Test failed: %v", err)
	}

	testSubject := func(t *testing.T, outFile *bufio.Writer) {

		handlerCtx, err := entryutil.BuildHandlerContext(*runtimeCtx, strings.NewReader(""), lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)), inputBundle)
		handlerCtx.SetOutfile(os.Stdout)
		handlerCtx.SetOutErrFile(os.Stderr)
		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		if err != nil {
			t.Fatalf("Test failed: %v", err)
		}

		handlerCtx.SetQuery(testobjects.UnionSelectGoogleComputeDisksOrderCreationTmstpAscPlusJsonExtractCoalesce)
		dr, _ := NewStackQLDriver(handlerCtx)
		querySubmitter := querysubmit.NewQuerySubmitter()
		response := querySubmitter.SubmitQuery(handlerCtx)
		handlerCtx.SetOutfile(outFile)
		responsehandler.HandleResponse(handlerCtx, response)

		dr.ProcessQuery(handlerCtx)
	}

	stackqltestutil.SetupSimpleSelectGoogleComputeDisks(t, 2)
	stackqltestutil.RunCaptureTestAgainstFiles(t, testSubject, []string{testobjects.ExpectedUnionAllSelectComputeDisksOrderCrtTmstpAscPlusJsonExtractCoalesce})

}
