package iqlutil_test

import (
	"testing"

	"github.com/stackql/stackql/internal/stackql/iqlutil"
)

func TestTranslateLikeToRegexPattern(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "exact match no wildcards",
			input:    "hello",
			expected: "^hello$",
		},
		{
			name:     "single percent wildcard",
			input:    "hello%",
			expected: "^hello.*$",
		},
		{
			name:     "trailing percent wildcard",
			input:    "%hello",
			expected: "^.*hello$",
		},
		{
			name:     "percent wildcard on both sides",
			input:    "%hello%",
			expected: "^.*hello.*$",
		},
		{
			name:     "multiple percent wildcards",
			input:    "a%b%c",
			expected: "^a.*b.*c$",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "^$",
		},
		{
			name:     "only percent wildcard",
			input:    "%",
			expected: "^.*$",
		},
		{
			name:     "multiple consecutive percent wildcards",
			input:    "%%",
			expected: "^.*.*$",
		},
		{
			name:     "regexp special characters are escaped",
			input:    "hello.world",
			expected: "^hello\\.world$",
		},
		{
			name:     "regexp special characters escaped with wildcard",
			input:    "hello%world.test",
			expected: "^hello.*world\\.test$",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := iqlutil.TranslateLikeToRegexPattern(tt.input)
			if got != tt.expected {
				t.Errorf("TranslateLikeToRegexPattern(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestSanitisePossibleTickEscapedTerm(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "term with backticks",
			input:    "`hello`",
			expected: "hello",
		},
		{
			name:     "term with only leading backtick",
			input:    "`hello",
			expected: "hello",
		},
		{
			name:     "term with only trailing backtick",
			input:    "hello`",
			expected: "hello",
		},
		{
			name:     "term without backticks",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only backticks",
			input:    "``",
			expected: "",
		},
		{
			name:     "multiple backticks",
			input:    "```hello```",
			expected: "``hello``",
		},
		{
			name:     "backtick in middle",
			input:    "hel`lo",
			expected: "hel`lo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := iqlutil.SanitisePossibleTickEscapedTerm(tt.input)
			if got != tt.expected {
				t.Errorf("SanitisePossibleTickEscapedTerm(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestPrettyPrintSomeJSON(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    string
		expectError bool
	}{
		{
			name:        "simple valid JSON",
			input:       `{"key":"value"}`,
			expected:    "{\n  \"key\": \"value\"\n}",
			expectError: false,
		},
		{
			name:        "nested valid JSON",
			input:       `{"outer":{"inner":"value"}}`,
			expected:    "{\n  \"outer\": {\n    \"inner\": \"value\"\n  }\n}",
			expectError: false,
		},
		{
			name:        "empty JSON object",
			input:       `{}`,
			expected:    "{}",
			expectError: false,
		},
		{
			name:        "JSON array",
			input:       `["a","b"]`,
			expected:    "[\n  \"a\",\n  \"b\"\n]",
			expectError: false,
		},
		{
			name:        "invalid JSON",
			input:       `{invalid}`,
			expectError: true,
		},
		{
			name:        "empty string",
			input:       ``,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := iqlutil.PrettyPrintSomeJSON([]byte(tt.input))
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got nil for input %q", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error for input %q: %v", tt.input, err)
				return
			}
			if string(got) != tt.expected {
				t.Errorf("PrettyPrintSomeJSON(%q) = %q, want %q", tt.input, string(got), tt.expected)
			}
		})
	}
}

func TestGetSortedKeysStringMap(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]string
		expected []string
	}{
		{
			name:     "empty map",
			input:    map[string]string{},
			expected: []string{},
		},
		{
			name:  "single element",
			input: map[string]string{"key1": "value1"},
			expected: []string{"key1"},
		},
		{
			name:  "multiple elements",
			input: map[string]string{"zebra": "z", "apple": "a", "mango": "m"},
			expected: []string{"apple", "mango", "zebra"},
		},
		{
			name:  "already sorted",
			input: map[string]string{"a": "1", "b": "2", "c": "3"},
			expected: []string{"a", "b", "c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := iqlutil.GetSortedKeysStringMap(tt.input)
			if len(got) != len(tt.expected) {
				t.Errorf("GetSortedKeysStringMap() returned %d elements, want %d", len(got), len(tt.expected))
				return
			}
			for i, v := range got {
				if v != tt.expected[i] {
					t.Errorf("GetSortedKeysStringMap()[%d] = %q, want %q", i, v, tt.expected[i])
				}
			}
		})
	}
}
