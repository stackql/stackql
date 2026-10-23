package intrinsic //nolint:testpackage // Verifies host read-absence policy and diagnostics.

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stackql/any-sdk/pkg/dto"
)

func TestReadFailuresRemainLocalAndPreserveDiagnostics(t *testing.T) {
	var diagnostic bytes.Buffer
	policy := &readErrorPolicy{writer: &diagnostic}
	for _, status := range []int{400, 401, 403, 429, 500, 503} {
		if !policy.IsIgnored(status) {
			t.Fatalf("read status %d aborts sibling reads", status)
		}
	}
	if !policy.IsIgnored(404) {
		t.Fatal("absent resource is not an empty read")
	}
	policy.IgnoredStatus(404, `{"msg":"Project Not Found"}`)
	if diagnostic.String() != "http response status code: 404, response body: {\"msg\":\"Project Not Found\"}\n" {
		t.Fatalf("absence diagnostics=%q", diagnostic.String())
	}
	policy.HTTPUpstreamErrorCfg = dto.HTTPUpstreamErrorCfg{IgnoreStatusCodes: []string{"503"}}
	if !policy.IsIgnored(503) {
		t.Fatal("explicit policy lost")
	}
	policy.IgnoredStatus(503, "temporarily unavailable")
	if !strings.Contains(diagnostic.String(), "ignoring upstream response: 503 SERVICE UNAVAILABLE") {
		t.Fatalf("diagnostics=%q", diagnostic.String())
	}
}
