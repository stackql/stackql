package driver_test

import (
	"bufio"
	"bytes"
	"os"
	"strings"
	"testing"

	. "github.com/stackql/stackql/internal/stackql/driver"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/stackql/internal/stackql/entryutil"
	"github.com/stackql/stackql/internal/test/stackqltestutil"
	"github.com/stackql/stackql/internal/test/testobjects"

	lrucache "github.com/stackql/stackql-parser/go/cache"
)

// runDescribeMethodFromFile drives a DESCRIBE METHOD test through the same
// path the existing show/describe integration tests use - building the
// handler context from an on-disk .iql input. Output is asserted with
// substring checks rather than byte-level golden comparison because the
// introspection output is verbose and tests target structural invariants
// (column headers, param-type tags, presence of expected fields).
func runDescribeMethodFromFile(t *testing.T, inputRelPath, testName string) string {
	t.Helper()
	runtimeCtx, err := stackqltestutil.GetRuntimeCtx(testobjects.GetGoogleProviderString(), "text", testName)
	if err != nil {
		t.Fatalf("%s: failed to build runtime context: %v", testName, err)
	}
	runtimeCtx.OutputFormat = "csv"
	runtimeCtx.CSVHeadersDisable = false

	inputBundle, err := stackqltestutil.BuildInputBundle(*runtimeCtx)
	if err != nil {
		t.Fatalf("%s: failed to build input bundle: %v", testName, err)
	}

	var b bytes.Buffer
	outFile := bufio.NewWriter(&b)
	inputBundle.WithStdOut(outFile)

	infile, err := util.GetFilePathFromRepositoryRoot(inputRelPath)
	if err != nil {
		t.Fatalf("%s: failed to locate input file %q: %v", testName, inputRelPath, err)
	}
	runtimeCtx.InfilePath = infile

	rdr, err := os.Open(runtimeCtx.InfilePath)
	if err != nil {
		t.Fatalf("%s: failed to open input file %q: %v", testName, runtimeCtx.InfilePath, err)
	}
	defer rdr.Close() //nolint:errcheck // best-effort close in test

	handlerCtx, err := entryutil.BuildHandlerContext(
		*runtimeCtx, rdr,
		lrucache.NewLRUCache(int64(runtimeCtx.QueryCacheSize)),
		inputBundle, true,
	)
	if err != nil {
		t.Fatalf("%s: failed to build handler context: %v", testName, err)
	}

	dr, drErr := NewStackQLDriver(handlerCtx)
	if drErr != nil {
		t.Fatalf("%s: failed to construct driver: %v", testName, drErr)
	}
	dr.ProcessQuery(handlerCtx.GetRawQuery())

	outFile.Flush()
	out := b.String()
	t.Logf("%s output:\n%s", testName, out)
	return out
}

//nolint:lll // legacy test
func TestDescribeMethodSimpleGoogleStorageBucketsGet(t *testing.T) {
	out := runDescribeMethodFromFile(t,
		"test/assets/input/describe-method/describe-method-google-storage-buckets-get.iql",
		"TestDescribeMethodSimpleGoogleStorageBucketsGet",
	)
	// Header columns for the non-extended variant.
	for _, header := range []string{"name", "type", "param_type", "shape"} {
		if !strings.Contains(out, header) {
			t.Fatalf("expected output to contain column header %q, got:\n%s", header, out)
		}
	}
	// The required path parameter for buckets.get is `bucket`.
	if !strings.Contains(out, "bucket") {
		t.Fatalf("expected output to mention 'bucket', got:\n%s", out)
	}
	// Both input_required and output rows must be present.
	for _, want := range []string{"input_required", "output"} {
		if !strings.Contains(out, want) {
			t.Fatalf("expected output to contain param_type %q, got:\n%s", want, out)
		}
	}
}

//nolint:lll // legacy test
func TestDescribeMethodExtendedGoogleStorageBucketsInsert(t *testing.T) {
	out := runDescribeMethodFromFile(t,
		"test/assets/input/describe-method/describe-method-extended-google-storage-buckets-insert.iql",
		"TestDescribeMethodExtendedGoogleStorageBucketsInsert",
	)
	// Extended adds the description column.
	if !strings.Contains(out, "description") {
		t.Fatalf("EXTENDED DESCRIBE METHOD must emit a description column; got:\n%s", out)
	}
	// The buckets.insert method has a method-level request.required: [name]
	// annotation; the body-translation algorithm may rename `name` to
	// `data__name` but it must still surface as input_required.
	if !strings.Contains(out, "name") {
		t.Fatalf("expected output to mention the required 'name' field; got:\n%s", out)
	}
	if !strings.Contains(out, "input_required") {
		t.Fatalf("expected output to contain param_type 'input_required'; got:\n%s", out)
	}
}

//nolint:lll // legacy test
func TestDescribeMethodGoogleStorageBucketsDelete(t *testing.T) {
	out := runDescribeMethodFromFile(t,
		"test/assets/input/describe-method/describe-method-google-storage-buckets-delete.iql",
		"TestDescribeMethodGoogleStorageBucketsDelete",
	)
	// delete returns 204 No Content => only input rows, no output rows.
	if !strings.Contains(out, "input_required") {
		t.Fatalf("expected delete to emit input_required rows; got:\n%s", out)
	}
	// Sanity-check the canonical column headers are still present.
	for _, header := range []string{"name", "type", "param_type"} {
		if !strings.Contains(out, header) {
			t.Fatalf("expected column header %q in output; got:\n%s", header, out)
		}
	}
	// Per the resolver contract, delete methods emit zero output rows.
	for _, line := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(line)
		// CSV form: ...,output,...   or   ...,output<EOL>
		if strings.Contains(trimmed, ",output,") || strings.HasSuffix(trimmed, ",output") {
			t.Fatalf("delete should produce zero output rows, but saw line:\n  %s\nfull output:\n%s", line, out)
		}
	}
}

//nolint:lll // legacy test
func TestDescribeMethodUnknownMethodReturnsError(t *testing.T) {
	out := runDescribeMethodFromFile(t,
		"test/assets/input/describe-method/describe-method-google-storage-buckets-unknown.iql",
		"TestDescribeMethodUnknownMethodReturnsError",
	)
	// The driver renders errors to stdout in the captured stream; the
	// resolver returns an error string referencing the missing method.
	lower := strings.ToLower(out)
	if !strings.Contains(lower, "bogus_method_name") && !strings.Contains(lower, "method") {
		t.Fatalf("expected an error referencing the missing method; got:\n%s", out)
	}
	// On error we must not have produced a populated input_required row.
	if strings.Contains(out, "input_required") {
		t.Fatalf("error path must not emit param_type rows; got:\n%s", out)
	}
}
