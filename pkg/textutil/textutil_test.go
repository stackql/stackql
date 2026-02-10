package textutil

import "testing"

func TestGetTemplateLikeString(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "single placeholder keeps existing behavior",
			input: "stackql_analytics_{{ .objectName }}",
			want:  "stackql_analytics_%",
		},
		{
			name:  "multiple placeholders preserve intervening literal text",
			input: "cache_{{ .provider }}_mid_{{ .objectName }}",
			want:  "cache_%_mid_%",
		},
		{
			name:  "no placeholders unchanged",
			input: "plain_literal_text",
			want:  "plain_literal_text",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := GetTemplateLikeString(tc.input)
			if got != tc.want {
				t.Errorf("GetTemplateLikeString(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
