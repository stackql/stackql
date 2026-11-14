package intrinsic //nolint:testpackage // Exercises private reversal binding and error contracts.

import (
	"errors"
	"testing"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
)

type failedMutationError struct{}

func (failedMutationError) Error() string        { return "upstream failed" }
func (failedMutationError) StatusCode() int      { return 500 }
func (failedMutationError) ResponseBody() string { return "server error" }
func (failedMutationError) RequestURL() string   { return "https://example.test" }
func TestMutationPresentationPreservesUnknownOutcome(t *testing.T) {
	cause := errors.Join(omnisdk.ErrOutcomeUnknown, failedMutationError{})
	err := mutationHTTPFailure(nil, "insert", cause)
	if err.Error() != "insert over HTTP error: 500 INTERNAL SERVER ERROR" {
		t.Fatal(err)
	}
	if !errors.Is(err, omnisdk.ErrOutcomeUnknown) {
		t.Fatal("lost unknown outcome")
	}
	var status omnisdk.HTTPStatusError
	if !errors.As(err, &status) || status.StatusCode() != 500 {
		t.Fatal("lost HTTP status")
	}
}
