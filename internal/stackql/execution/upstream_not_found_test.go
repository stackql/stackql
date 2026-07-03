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
			name:     "no status code fragment and no not-found phrasing",
			messages: []string{"something else entirely"},
			want:     false,
		},
		{
			name:     "empty messages",
			messages: nil,
			want:     false,
		},
		{
			name:     "unstructured go default body",
			messages: []string{"404 page not found"},
			want:     true,
		},
		{
			name:     "unstructured html body",
			messages: []string{"<html><head><title>404 Not Found</title></head><body><h1>NOT FOUND</h1></body></html>"},
			want:     true,
		},
		{
			name:     "unstructured xml fault",
			messages: []string{"<?xml version=\"1.0\"?><Error><Code>404</Code><Message>The specified key does not exist.</Message></Error>"},
			want:     true,
		},
		{
			name:     "unstructured mixed case phrase only",
			messages: []string{"the requested thing was Not FOUND on this server"},
			want:     true,
		},
		{
			name:     "structured 403 with not-found wording in body is still an error",
			messages: []string{`Response error.  Status code 403.  Body: {"message":"secret not found for caller"}`},
			want:     false,
		},
		{
			name:     "digits embedded in larger number are not a 404",
			messages: []string{"upstream failure, trace id 14042"},
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
