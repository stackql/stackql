package intrinsic //nolint:testpackage // Exercises private reversal binding and error contracts.

import (
	"context"
	"errors"
	"testing"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
)

type inverseFixture struct{}

func (inverseFixture) Method() string { return "delete" }
func (inverseFixture) Tokens() map[string]omnisdk.InverseToken {
	return map[string]omnisdk.InverseToken{
		"userKey": {Key: "id", Location: "body", Algorithm: "toString"},
	}
}
func TestInverseRequiresAcceptedResponseIdentity(t *testing.T) {
	c := &inverseCapture{declaration: inverseFixture{}}
	if _, err := c.Complete(context.Background(), "p.s.users", "insert", nil, omnisdk.Row{}); err == nil {
		t.Fatal("missing identity accepted")
	}
	if len(c.rows) != 0 {
		t.Fatal("registered reversal without identity")
	}
	row := omnisdk.Row{"id": "created-user"}
	if _, err := c.Complete(context.Background(), "p.s.users", "insert", map[string]any{"region": "east"}, row); err != nil {
		t.Fatal(err)
	}
	if len(c.rows) != 1 || c.rows[0]["userKey"] != "created-user" || c.rows[0]["region"] != "east" {
		t.Fatalf("bindings=%v", c.rows)
	}
}

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
