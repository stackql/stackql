package sqlfuncs //nolint:testpackage // Exercises private scalar conversion helpers.

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
)

// goldenVector is one recorded input/output pair. Vectors are sourced from
// the sqlite-ext-functions test suite, the cgo golden samples in stackql
// PR #783, and the retired C reference implementations, as recorded in each
// vector's source field.
type goldenVector struct {
	ID             string `json:"id"`
	Source         string `json:"source"`
	Args           []any  `json:"args"`
	Result         any    `json:"result"`
	Error          bool   `json:"error"`
	RetiredCResult any    `json:"retiredCResult"`
	Note           string `json:"note"`
}

type goldenFile struct {
	Function string         `json:"function"`
	Vectors  []goldenVector `json:"vectors"`
}

// normalizeVectorValue converts JSON-decoded values to the types the
// functions produce: integral float64 becomes int64 (SQLite INTEGER), all
// other kinds pass through.
func normalizeVectorValue(v any) any {
	if f, ok := v.(float64); ok {
		if math.Trunc(f) == f && !math.IsInf(f, 0) {
			return int64(f)
		}
	}
	return v
}

func loadGoldenVectors(t *testing.T, name string) []goldenVector {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name+".json"))
	if err != nil {
		t.Fatalf("cannot read golden vectors for %s: %v", name, err)
	}
	var doc goldenFile
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("cannot parse golden vectors for %s: %v", name, err)
	}
	if doc.Function != name {
		t.Fatalf("golden file %s declares function %q", name, doc.Function)
	}
	seen := map[string]struct{}{}
	for i := range doc.Vectors {
		vec := &doc.Vectors[i]
		if vec.ID == "" || vec.Source == "" {
			t.Fatalf("golden vector %d in %s lacks id or source", i, name)
		}
		if _, dup := seen[vec.ID]; dup {
			t.Fatalf("duplicate golden vector id %q in %s", vec.ID, name)
		}
		seen[vec.ID] = struct{}{}
		for j := range vec.Args {
			vec.Args[j] = normalizeVectorValue(vec.Args[j])
		}
		vec.Result = normalizeVectorValue(vec.Result)
	}
	return doc.Vectors
}

// runGoldenVectors drives fn over every vector and asserts the recorded
// outcome.
func runGoldenVectors(t *testing.T, name string, fn func(args []any) (any, error)) {
	t.Helper()
	vectors := loadGoldenVectors(t, name)
	if len(vectors) == 0 {
		t.Fatalf("no golden vectors for %s", name)
	}
	for _, vec := range vectors {
		vec := vec
		t.Run(vec.ID, func(t *testing.T) {
			got, err := fn(vec.Args)
			if vec.Error {
				if err == nil {
					t.Fatalf("%s(%s): expected an error, got %v", name, formatArgs(vec.Args), got)
				}
				return
			}
			if err != nil {
				t.Fatalf("%s(%s): unexpected error: %v", name, formatArgs(vec.Args), err)
			}
			if got != vec.Result {
				t.Fatalf("%s(%s) = %#v, want %#v (source: %s)", name, formatArgs(vec.Args), got, vec.Result, vec.Source)
			}
		})
	}
}

func formatArgs(args []any) string {
	out := ""
	for i, a := range args {
		if i > 0 {
			out += ", "
		}
		out += fmt.Sprintf("%#v", a)
	}
	return out
}

func TestSplitPartGoldenVectors(t *testing.T) {
	runGoldenVectors(t, "split_part", func(args []any) (any, error) {
		return SplitPart(args[0], args[1], args[2])
	})
}

func TestRegexpLikeGoldenVectors(t *testing.T) {
	runGoldenVectors(t, "regexp_like", func(args []any) (any, error) {
		return RegexpLike(args[0], args[1])
	})
}

func TestRegexpGoldenVectors(t *testing.T) {
	runGoldenVectors(t, "regexp", func(args []any) (any, error) {
		return Regexp(args[0], args[1])
	})
}

func TestRegexpSubstrGoldenVectors(t *testing.T) {
	runGoldenVectors(t, "regexp_substr", func(args []any) (any, error) {
		return RegexpSubstr(args[0], args[1])
	})
}

func TestRegexpReplaceGoldenVectors(t *testing.T) {
	runGoldenVectors(t, "regexp_replace", func(args []any) (any, error) {
		return RegexpReplace(args[0], args[1], args[2])
	})
}

func TestJSONEqualGoldenVectors(t *testing.T) {
	runGoldenVectors(t, "json_equal", func(args []any) (any, error) {
		return JSONEqual(args[0], args[1])
	})
}

func TestAWSPolicyEqualGoldenVectors(t *testing.T) {
	runGoldenVectors(t, "aws_policy_equal", func(args []any) (any, error) {
		return AWSPolicyEqual(args[0], args[1])
	})
}

// TestRegexpInvalidPattern pins the error contract for unparseable patterns,
// mirroring the "Invalid regular expression" error in the retired C code.
func TestRegexpInvalidPattern(t *testing.T) {
	if _, err := RegexpLike("abc", "["); err == nil {
		t.Fatal("regexp_like with an invalid pattern must error")
	}
	if _, err := RegexpSubstr("abc", "["); err == nil {
		t.Fatal("regexp_substr with an invalid pattern must error")
	}
	if _, err := RegexpReplace("abc", "[", "x"); err == nil {
		t.Fatal("regexp_replace with an invalid pattern must error")
	}
}

// TestRegexpDotMatchesNewline pins the RE_DOT_MATCHES_NEWLINE=1 behavior of
// the retired tiny-regex-c build: '.' matches any byte including newline.
func TestRegexpDotMatchesNewline(t *testing.T) {
	got, err := RegexpLike("a\nb", "a.b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != int64(1) {
		t.Fatalf("dot must match newline (RE_DOT_MATCHES_NEWLINE parity), got %#v", got)
	}
	sub, err := RegexpSubstr("a\nb", "a.b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sub != "a\nb" {
		t.Fatalf("regexp_substr dot-newline parity broken, got %#v", sub)
	}
}

// TestValueTextCoercion pins the sqlite3_value_text-style coercions applied
// to non-text arguments.
func TestValueTextCoercion(t *testing.T) {
	cases := []struct {
		in     any
		want   string
		wantOK bool
	}{
		{nil, "", false},
		{"abc", "abc", true},
		{[]byte("xyz"), "xyz", true},
		{int64(42), "42", true},
		{float64(2.5), "2.5", true},
		{float64(3), "3.0", true},
	}
	for _, c := range cases {
		got, ok := valueText(c.in)
		if got != c.want || ok != c.wantOK {
			t.Fatalf("valueText(%#v) = (%q, %v), want (%q, %v)", c.in, got, ok, c.want, c.wantOK)
		}
	}
}

// TestValueIntCoercion pins the sqlite3_value_int-style coercions applied to
// the part index of split_part.
func TestValueIntCoercion(t *testing.T) {
	cases := []struct {
		in   any
		want int64
	}{
		{nil, 0},
		{int64(-3), -3},
		{float64(2.9), 2},
		{float64(-2.9), -2},
		{"11", 11},
		{" -4abc", -4},
		{"abc", 0},
		{[]byte("7x"), 7},
	}
	for _, c := range cases {
		if got := valueInt(c.in); got != c.want {
			t.Fatalf("valueInt(%#v) = %d, want %d", c.in, got, c.want)
		}
	}
}
