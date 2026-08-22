package textutil_test

import (
	"testing"

	"github.com/stackql/stackql/pkg/textutil"
)

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
			got := textutil.GetTemplateLikeString(tc.input)
			if got != tc.want {
				t.Errorf("GetTemplateLikeString(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestExpandPlaceholders(t *testing.T) {
	cases := []struct {
		name         string
		template     string
		placeholder  string
		replacements []string
		want         string
	}{
		{
			name:         "percent signs in like patterns survive verbatim",
			template:     `SELECT a FROM ( ` + textutil.IndirectQueryPlaceholder + ` ) AS "v" WHERE a NOT LIKE '%TLS12%'`,
			placeholder:  textutil.IndirectQueryPlaceholder,
			replacements: []string{"SELECT b AS a FROM t"},
			want:         `SELECT a FROM ( SELECT b AS a FROM t ) AS "v" WHERE a NOT LIKE '%TLS12%'`,
		},
		{
			name:         "successive placeholders expand left to right",
			template:     textutil.IndirectQueryPlaceholder + " union " + textutil.IndirectQueryPlaceholder + " ",
			placeholder:  textutil.IndirectQueryPlaceholder,
			replacements: []string{"SELECT 1", "SELECT 2"},
			want:         "SELECT 1 union SELECT 2 ",
		},
		{
			name:         "replacement text is not rescanned for placeholders",
			template:     "a " + textutil.IndirectQueryPlaceholder + " b",
			placeholder:  textutil.IndirectQueryPlaceholder,
			replacements: []string{textutil.IndirectQueryPlaceholder, "surplus"},
			want:         "a " + textutil.IndirectQueryPlaceholder + " b",
		},
		{
			name:         "surplus placeholders are left intact",
			template:     "a " + textutil.IndirectQueryPlaceholder + " b " + textutil.IndirectQueryPlaceholder,
			placeholder:  textutil.IndirectQueryPlaceholder,
			replacements: []string{"x"},
			want:         "a x b " + textutil.IndirectQueryPlaceholder,
		},
		{
			name:         "no replacements returns template unchanged",
			template:     "a " + textutil.IndirectQueryPlaceholder,
			placeholder:  textutil.IndirectQueryPlaceholder,
			replacements: nil,
			want:         "a " + textutil.IndirectQueryPlaceholder,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := textutil.ExpandPlaceholders(tc.template, tc.placeholder, tc.replacements)
			if got != tc.want {
				t.Errorf("ExpandPlaceholders() = %q, want %q", got, tc.want)
			}
		})
	}
}
