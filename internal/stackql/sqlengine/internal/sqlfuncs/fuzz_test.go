package sqlfuncs //nolint:testpackage // Exercises private scalar conversion helpers.

import (
	"regexp"
	"strings"
	"testing"
)

func FuzzSplitPart(f *testing.F) {
	f.Add("a/b/c", "/", int64(2))
	f.Add("/a/b/c/", "/", int64(-1))
	f.Add("", "/", int64(1))
	f.Add("a,b,c", ",", int64(0))
	f.Add("aaa", "aa", int64(1))
	f.Fuzz(func(t *testing.T, source, sep string, part int64) {
		got, err := SplitPart(source, sep, part)
		if err != nil {
			t.Fatalf("split_part must never error, got %v", err)
		}
		switch v := got.(type) {
		case nil:
			if sep != "" && part >= 1 && part <= int64(strings.Count(source, sep))+1 {
				t.Fatalf("split_part(%q, %q, %d) returned NULL for an in-range part", source, sep, part)
			}
		case string:
			if sep == "" {
				t.Fatalf("split_part with empty separator must be NULL, got %q", v)
			}
			if !strings.Contains(source, v) {
				t.Fatalf("split_part(%q, %q, %d) = %q is not a substring of the source", source, sep, part, v)
			}
		default:
			t.Fatalf("split_part returned unexpected type %T", got)
		}
	})
}

// fuzzCompiles reports whether the pattern compiles under the production
// translation; fuzz targets use it to decide whether an error is legitimate.
func fuzzCompiles(pattern string) bool {
	_, err := regexp.Compile("(?s)" + pattern)
	return err == nil
}

func FuzzRegexpLike(f *testing.F) {
	f.Add("hello world", "hello")
	f.Add("abracadabra", "abra(cad)?abra")
	f.Add("abc123", "[0-9]+")
	f.Add("a\nb", "a.b")
	f.Add("", "")
	f.Fuzz(func(t *testing.T, source, pattern string) {
		got, err := RegexpLike(source, pattern)
		if err != nil {
			if fuzzCompiles(pattern) {
				t.Fatalf("regexp_like(%q, %q) errored on a valid pattern: %v", source, pattern, err)
			}
			return
		}
		if got != int64(0) && got != int64(1) {
			t.Fatalf("regexp_like(%q, %q) = %#v, want 0 or 1", source, pattern, got)
		}
	})
}

func FuzzRegexp(f *testing.F) {
	f.Add("hello", "hello world")
	f.Add("[0-9]+", "the year is 2021")
	f.Fuzz(func(t *testing.T, pattern, source string) {
		got, err := Regexp(pattern, source)
		if err != nil {
			if fuzzCompiles(pattern) {
				t.Fatalf("regexp(%q, %q) errored on a valid pattern: %v", pattern, source, err)
			}
			return
		}
		want, wantErr := RegexpLike(source, pattern)
		if wantErr != nil || got != want {
			t.Fatalf("regexp(%q, %q) = %#v, want regexp_like result %#v (err %v)", pattern, source, got, want, wantErr)
		}
	})
}

func FuzzRegexpSubstr(f *testing.F) {
	f.Add("hello world", "w.*d")
	f.Add("file.txt", "\\.\\w+$")
	f.Add("abc", "[0-9]")
	f.Add("hello", "")
	f.Fuzz(func(t *testing.T, source, pattern string) {
		got, err := RegexpSubstr(source, pattern)
		if err != nil {
			if fuzzCompiles(pattern) {
				t.Fatalf("regexp_substr(%q, %q) errored on a valid pattern: %v", source, pattern, err)
			}
			return
		}
		switch v := got.(type) {
		case nil:
		case string:
			if !strings.Contains(source, v) {
				t.Fatalf("regexp_substr(%q, %q) = %q is not a substring of the source", source, pattern, v)
			}
		default:
			t.Fatalf("regexp_substr returned unexpected type %T", got)
		}
	})
}

func FuzzRegexpReplace(f *testing.F) {
	f.Add("hello world", "world", "SQLite")
	f.Add("123-456-7890", "[0-9]", "X")
	f.Add("abracadabra", "abra(cad)?abra", "magic")
	f.Add("ab12cd", "([a-z]+)", "<$1>")
	f.Fuzz(func(t *testing.T, source, pattern, replacement string) {
		got, err := RegexpReplace(source, pattern, replacement)
		if err != nil {
			if fuzzCompiles(pattern) {
				t.Fatalf("regexp_replace(%q, %q, %q) errored on a valid pattern: %v", source, pattern, replacement, err)
			}
			return
		}
		if _, ok := got.(string); !ok {
			t.Fatalf("regexp_replace returned unexpected type %T", got)
		}
	})
}

func FuzzJSONEqual(f *testing.F) {
	f.Add(`{"key": "value"}`, `{ "key" : "value" }`)
	f.Add(`[1, 2, 3]`, `[3, 2, 1]`)
	f.Add(`{"a":1}`, `{ "a" : 1.0 }`)
	f.Add(`1`, `2`)
	f.Add(`not json`, `{}`)
	f.Fuzz(func(t *testing.T, a, b string) {
		got, err := JSONEqual(a, b)
		mirror, mirrorErr := JSONEqual(b, a)
		if (err != nil) != (mirrorErr != nil) {
			t.Fatalf("json_equal error asymmetry: (%q,%q) err=%v, mirrored err=%v", a, b, err, mirrorErr)
		}
		if err != nil {
			return
		}
		if got != int64(0) && got != int64(1) {
			t.Fatalf("json_equal(%q, %q) = %#v, want 0 or 1", a, b, got)
		}
		if got != mirror {
			t.Fatalf("json_equal symmetry broken: (%q,%q)=%v but mirrored=%v", a, b, got, mirror)
		}
		if self, selfErr := JSONEqual(a, a); selfErr == nil && self != int64(1) {
			t.Fatalf("json_equal(%q, %q) must be reflexive", a, a)
		}
	})
}

func FuzzAWSPolicyEqual(f *testing.F) {
	f.Add(`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["s3:GetObject"],"Resource":"*"}]}`,
		`{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":"s3:GetObject","Resource":"*"}]}`)
	f.Add(`{"Resource":"arn:aws:s3:::mybucket"}`, `{"Resource":"arn:aws:S3:::mybucket"}`)
	f.Add(`not json`, `not json`)
	f.Add(`{"a": [1]}`, `{"A": 1}`)
	f.Fuzz(func(t *testing.T, a, b string) {
		// Note: no symmetry assertion - the "arn:" case-insensitive rule is
		// deliberately tested on the left operand only, matching the C code.
		got, err := AWSPolicyEqual(a, b)
		if err != nil {
			return
		}
		if got != int64(0) && got != int64(1) {
			t.Fatalf("aws_policy_equal(%q, %q) = %#v, want 0 or 1", a, b, got)
		}
		self, selfErr := AWSPolicyEqual(a, a)
		if selfErr != nil {
			t.Fatalf("aws_policy_equal(%q, %q) must not error (identical-string shortcut): %v", a, a, selfErr)
		}
		if self != int64(1) {
			t.Fatalf("aws_policy_equal(%q, %q) must be reflexive", a, a)
		}
	})
}
