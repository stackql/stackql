package sqlfuncs

import "strings"

// SplitPart implements split_part(source, separator, part): the one-based
// part of source split on separator; negative indexes count from the end.
// NULL source, NULL or empty separator, and out-of-range indexes yield NULL.
func SplitPart(source, separator, part any) (any, error) {
	src, srcOK := valueText(source)
	sep, sepOK := valueText(separator)
	if !srcOK || !sepOK || sep == "" {
		return nil, nil //nolint:nilnil // A SQL NULL is a valid scalar result.
	}
	idx := valueInt(part)
	parts := strings.Split(src, sep)
	n := int64(len(parts))
	if idx < 0 {
		idx = n + idx
	} else {
		idx--
	}
	if idx >= 0 && idx < n {
		return parts[idx], nil
	}
	return nil, nil //nolint:nilnil // A SQL NULL is a valid scalar result.
}
