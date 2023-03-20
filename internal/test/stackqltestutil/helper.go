package stackqltestutil

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stackql/stackql/internal/stackql/util"
	"github.com/stackql/stackql/internal/test/testutil"
)

func RunStdOutTestAgainstFiles(t *testing.T, testSubject func(*testing.T), possibleExpectedOutputFiles []string) {
	old := os.Stdout // keep backup of the real stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	outC := make(chan string)

	testSubject(t)

	// copy the output in a separate goroutine so printing can't block indefinitely
	go func() {
		var buf bytes.Buffer
		io.Copy(&buf, r) //nolint:errcheck // ok for testing
		outC <- buf.String()
	}()
	w.Close()
	os.Stdout = old // restoring the real stdout
	out := <-outC
	t.Logf("outC = %s", out)

	checkPossibleMatchFiles(t, out, possibleExpectedOutputFiles)
}

func checkPossibleMatchFiles(t *testing.T, subject string, possibleExpectedOutputFiles []string) {
	hasMatchedExpected := false
	for _, expectedOpFile := range possibleExpectedOutputFiles {
		expF, err := util.GetFilePathFromRepositoryRoot(expectedOpFile)
		if err != nil {
			t.Fatalf("test failed: %v", err)
		}
		if !testutil.StringEqualsFileContents(t, subject, expF, false) {
			t.Logf("NOT THIS TIME: processed response did NOT match expected file contents as per: %s, contents = '%s'",
				expectedOpFile, subject)
			continue
		}
		t.Logf("SUCCESS: processed response did match expected file contents as per: %s, contents = '%s'",
			expectedOpFile, subject)
		hasMatchedExpected = true
		break
	}

	if !hasMatchedExpected {
		t.Fatalf("FAIL: output does NOT match any possibility")
	}

	t.Logf("simple select integration test passed")
}

func RunCaptureTestAgainstFiles(
	t *testing.T, testSubject func(*testing.T, *bufio.Writer),
	possibleExpectedOutputFiles []string) {
	var b bytes.Buffer
	outFile := bufio.NewWriter(&b)

	testSubject(t, outFile)

	outFile.Flush()
	outStr := b.String()
	t.Logf("outStr = '%s'\n", outStr)

	checkPossibleMatchFiles(t, outStr, possibleExpectedOutputFiles)
}
