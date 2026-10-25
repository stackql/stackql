package sqlfuncs

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// dblEpsilon is C's DBL_EPSILON, the tolerance the retired cJSON-based
// implementations used for number equality.
const dblEpsilon = 2.220446049250313e-16

// valueText mirrors sqlite3_value_text() coercion; the boolean reports
// whether the value was non-NULL.
func valueText(v any) (string, bool) {
	switch t := v.(type) {
	case nil:
		return "", false
	case string:
		return t, true
	case []byte:
		return string(t), true
	case int64:
		return strconv.FormatInt(t, 10), true
	case float64:
		return formatFloatText(t), true
	default:
		return fmt.Sprintf("%v", t), true
	}
}

// formatFloatText renders a float the way SQLite renders REAL as text:
// shortest round-trip form always carrying a decimal point or exponent.
func formatFloatText(f float64) string {
	s := strconv.FormatFloat(f, 'g', -1, 64)
	if !strings.ContainsAny(s, ".eE") && !math.IsInf(f, 0) && !math.IsNaN(f) {
		s += ".0"
	}
	return s
}

// valueInt mirrors sqlite3_value_int() coercion: NULL is 0, floats truncate
// toward zero, text parses as a signed leading-digit prefix.
func valueInt(v any) int64 {
	switch t := v.(type) {
	case nil:
		return 0
	case int64:
		return t
	case float64:
		if math.IsNaN(t) {
			return 0
		}
		return int64(math.Trunc(t))
	case string:
		return parseIntPrefix(t)
	case []byte:
		return parseIntPrefix(string(t))
	default:
		return 0
	}
}

// parseIntPrefix parses an optionally signed run of leading ASCII digits
// after whitespace, the way SQLite coerces text to an integer.
func parseIntPrefix(s string) int64 {
	i := 0
	for i < len(s) && (s[i] == ' ' || s[i] == '\t' || s[i] == '\n' || s[i] == '\r' || s[i] == '\v' || s[i] == '\f') {
		i++
	}
	start := i
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	digitsStart := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		i++
	}
	if i == digitsStart {
		return 0
	}
	n, err := strconv.ParseInt(s[start:i], 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// compareDouble is the retired implementations' epsilon comparison:
// |a-b| <= max(|a|,|b|) * DBL_EPSILON.
func compareDouble(a, b float64) bool {
	maxVal := math.Abs(a)
	if abs := math.Abs(b); abs > maxVal {
		maxVal = abs
	}
	return math.Abs(a-b) <= maxVal*dblEpsilon
}

// asciiEqualFold reports byte-wise ASCII case-insensitive equality.
func asciiEqualFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
