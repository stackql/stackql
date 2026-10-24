package sqlfuncs

import (
	"fmt"
	"regexp"
)

// compilePattern compiles pattern in (?s) mode: the retired C engine was
// built with RE_DOT_MATCHES_NEWLINE=1, so '.' must match any byte.
func compilePattern(pattern string) (*regexp.Regexp, error) {
	re, err := regexp.Compile("(?s)" + pattern)
	if err != nil {
		return nil, fmt.Errorf("invalid regular expression: %w", err)
	}
	return re, nil
}

// RegexpLike implements regexp_like(source, pattern): 1 on match, else 0.
// NULL source or pattern yields NULL; an invalid pattern is an error.
func RegexpLike(source, pattern any) (any, error) {
	src, srcOK := valueText(source)
	pat, patOK := valueText(pattern)
	if !srcOK || !patOK {
		return nil, nil //nolint:nilnil // A SQL NULL is a valid scalar result.
	}
	re, err := compilePattern(pat)
	if err != nil {
		return nil, err
	}
	if re.MatchString(src) {
		return int64(1), nil
	}
	return int64(0), nil
}

// Regexp implements regexp(pattern, source), the function SQLite invokes for
// the REGEXP operator: `X REGEXP Y` evaluates regexp(Y, X).
func Regexp(pattern, source any) (any, error) {
	return RegexpLike(source, pattern)
}

// RegexpSubstr implements regexp_substr(source, pattern): the leftmost match
// of pattern in source, or NULL if none. NULL source or pattern yields NULL.
func RegexpSubstr(source, pattern any) (any, error) {
	src, srcOK := valueText(source)
	pat, patOK := valueText(pattern)
	if !srcOK || !patOK {
		return nil, nil //nolint:nilnil // A SQL NULL is a valid scalar result.
	}
	re, err := compilePattern(pat)
	if err != nil {
		return nil, err
	}
	loc := re.FindStringIndex(src)
	if loc == nil {
		return nil, nil //nolint:nilnil // A SQL NULL is a valid scalar result.
	}
	return src[loc[0]:loc[1]], nil
}

// RegexpReplace implements regexp_replace(source, pattern, replacement):
// replaces every match with replacement taken literally ($1 is not expanded,
// matching the retired C implementation). Any NULL argument yields NULL.
func RegexpReplace(source, pattern, replacement any) (any, error) {
	src, srcOK := valueText(source)
	pat, patOK := valueText(pattern)
	rep, repOK := valueText(replacement)
	if !srcOK || !patOK || !repOK {
		return nil, nil //nolint:nilnil // A SQL NULL is a valid scalar result.
	}
	re, err := compilePattern(pat)
	if err != nil {
		return nil, err
	}
	return re.ReplaceAllLiteralString(src, rep), nil
}
