package execution //nolint:testpackage // exercise unexported helper

import "testing"

func TestIsUpstreamNotFound(t *testing.T) {
	cases := []struct {
		name     string
		messages []string
		want     bool
	}{
		{
			name:     "single 404 detail form",
			messages: []string{"HTTP response error.  Status code 404.  Detail: 'not found'"},
			want:     true,
		},
		{
			name:     "single 404 body form",
			messages: []string{`Response error.  Status code 404.  Body: {"message":"Not Found"}`},
			want:     true,
		},
		{
			name:     "403 is not absence",
			messages: []string{"Response error.  Status code 403.  Body: {}"},
			want:     false,
		},
		{
			name: "mixed 404 and 500 is not absence",
			messages: []string{
				"Response error.  Status code 404.  Body: {}",
				"Response error.  Status code 500.  Body: {}",
			},
			want: false,
		},
		{
			name:     "no status code fragment",
			messages: []string{"something else entirely"},
			want:     false,
		},
		{
			name:     "empty messages",
			messages: nil,
			want:     false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isUpstreamNotFound(tc.messages); got != tc.want {
				t.Errorf("isUpstreamNotFound(%v) = %t, want %t", tc.messages, got, tc.want)
			}
		})
	}
}
