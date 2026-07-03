package mcpbackend //nolint:testpackage // exercise unexported classifier

import (
	"fmt"
	"strings"
	"testing"
)

func TestClassifyBackendError_UpstreamStatusVariants(t *testing.T) {
	cases := []struct {
		name      string
		err       error
		status    int
		retryable bool
	}{
		{
			name: "any-sdk detail form 403",
			err: fmt.Errorf(
				"upstream provider error: HTTP response error.  Status code 403.  Detail: 'rate limit exceeded'"),
			status:    403,
			retryable: false,
		},
		{
			name:      "any-sdk body form 429",
			err:       fmt.Errorf(`upstream provider error: Response error.  Status code 429.  Body: {"message":"slow down"}`),
			status:    429,
			retryable: true,
		},
		{
			name:      "client log form 503",
			err:       fmt.Errorf("http response status code: 503, response body is nil"),
			status:    503,
			retryable: true,
		},
		{
			name:      "request timeout 408",
			err:       fmt.Errorf("upstream provider error: HTTP response error.  Status code 408.  Detail: 'timeout'"),
			status:    408,
			retryable: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyBackendError(tc.err)
			wantFragment := fmt.Sprintf(`{"http_status": %d, "retryable": %t}`, tc.status, tc.retryable)
			if !strings.Contains(got.Error(), wantFragment) {
				t.Errorf("expected %q in classified error, got %q", wantFragment, got.Error())
			}
			if !strings.Contains(got.Error(), tc.err.Error()) {
				t.Errorf("expected underlying detail preserved, got %q", got.Error())
			}
		})
	}
}

func TestClassifyBackendError_NonHTTPKeepsLegacyPrefix(t *testing.T) {
	err := fmt.Errorf("'registry list' is meaningless in local mode")
	got := classifyBackendError(err)
	if !strings.Contains(got.Error(), "failed to extract query results") {
		t.Errorf("expected legacy prefix, got %q", got.Error())
	}
	if !strings.Contains(got.Error(), "meaningless in local mode") {
		t.Errorf("expected underlying detail preserved, got %q", got.Error())
	}
}
