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
		{
			name:  "no placeholders",
			input: "hello world",
			want:  "hello world",
		},
		{
			name:  "single placeholder",
			input: "hello {{name}}",
			want:  "hello %",
		},
		{
			name:  "multiple placeholders",
			input: "{{first}} and {{second}}",
			want:  "% and %",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "only placeholder",
			input: "{{}}",
			want:  "%",
		},
		{
			name:  "placeholder at start",
			input: "{{foo}}bar",
			want:  "%bar",
		},
		{
			name:  "placeholder at end",
			input: "foo{{bar}}",
			want:  "foo%",
		},
		{
			name:  "consecutive placeholders",
			input: "{{a}}{{b}}{{c}}",
			want:  "%%%",
		},
		{
			name:  "placeholder with spaces",
			input: "{{ hello world }}",
			want:  "%",
		},
		{
			name:  "nested braces not placeholder",
			input: "{{foo{{bar}}}}",
			want:  "%}}",
		},
		{
			name:  "unmatched opening brace",
			input: "foo{{bar",
			want:  "foo{{bar",
		},
		{
			name:  "unmatched closing brace",
			input: "foo}bar}}",
			want:  "foo}bar}}",
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
		{
			name:         "single replacement",
			template:     "hello {{name}}",
			placeholder:  "{{name}}",
			replacements: []string{"world"},
			want:         "hello world",
		},
		{
			name:         "multiple replacements",
			template:     "{{}} and {{}}",
			placeholder:  "{{}}",
			replacements: []string{"foo", "bar"},
			want:         "foo and bar",
		},
		{
			name:         "more replacements than placeholders",
			template:     "{{}} and {{}}",
			placeholder:  "{{}}",
			replacements: []string{"foo", "bar", "baz"},
			want:         "foo and bar",
		},
		{
			name:         "fewer replacements than placeholders",
			template:     "{{}} and {{}} and {{}}",
			placeholder:  "{{}}",
			replacements: []string{"foo"},
			want:         "foo and {{}} and {{}}",
		},
		{
			name:         "empty template",
			template:     "",
			placeholder:  "{{}}",
			replacements: []string{"foo"},
			want:         "",
		},
		{
			name:         "empty placeholder",
			template:     "a {{}} b",
			placeholder:  "",
			replacements: []string{"foo"},
			want:         "a {{}} b",
		},
		{
			name:         "empty replacements",
			template:     "hello {{name}}",
			placeholder:  "{{name}}",
			replacements: []string{},
			want:         "hello {{name}}",
		},
		{
			name:         "no placeholder in template",
			template:     "hello world",
			placeholder:  "{{name}}",
			replacements: []string{"foo"},
			want:         "hello world",
		},
		{
			name:         "adjacent placeholders",
			template:     "{{}}{{}}",
			placeholder:  "{{}}",
			replacements: []string{"x", "y"},
			want:         "xy",
		},
		{
			name:         "placeholder with special chars",
			template:     "url: {{url}}",
			placeholder:  "{{url}}",
			replacements: []string{"http://example.com?a=1&b=2"},
			want:         "url: http://example.com?a=1&b=2",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := textutil.ExpandPlaceholders(tc.template, tc.placeholder, tc.replacements)
			if got != tc.want {
				t.Errorf("ExpandPlaceholders(%q, %q, %v) = %q, want %q",
					tc.template, tc.placeholder, tc.replacements, got, tc.want)
			}
		})
	}
}

func TestExpandPlaceholdersReplacesInOrder(t *testing.T) {
	// Verify replacements happen left-to-right
	got := textutil.ExpandPlaceholders("x{{}}x{{}}x", "{{}}", []string{"a", "b"})
	expected := "xaxbx"
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestExpandPlaceholdersLeftoverText(t *testing.T) {
	// After all replacements, remainder should be appended
	got := textutil.ExpandPlaceholders("{{}}extra", "{{}}", []string{"x"})
	expected := "xextra"
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}

func TestExpandPlaceholdersEmptyReplacementsMidTemplate(t *testing.T) {
	// Replacements run out before template ends — remainder should be preserved
	got := textutil.ExpandPlaceholders("start {{}} middle {{}} end", "{{}}", []string{"x"})
	expected := "start x middle {{}} end"
	if got != expected {
		t.Errorf("got %q, want %q", got, expected)
	}
}
