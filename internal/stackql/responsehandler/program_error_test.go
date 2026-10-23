package responsehandler //nolint:testpackage // Tests the CLI presentation boundary.

import (
	"fmt"
	"testing"
)

type unsupportedTestProgramError struct{}

func (unsupportedTestProgramError) Error() string       { return "unsupported template type: unknown_v1" }
func (unsupportedTestProgramError) ProgramType() string { return "unknown_v1" }
func TestUnsupportedProgramPresentationKeepsLegacyDiagnostic(t *testing.T) {
	err := fmt.Errorf("resource: %w", unsupportedTestProgramError{})
	if got := presentationError(err).Error(); got != "unsupported template type: unknown_v1" {
		t.Fatalf("diagnostic=%q", got)
	}
}
