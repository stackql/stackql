package name_mangle //nolint:revive,stylecheck // preference

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	_ NameMangler   = &standardNameMutator{}
	_ NameUnmangler = &standardNameMutator{}
)

var (
	//nolint:revive // prefer explicit
	standardUnmangleRegexp *regexp.Regexp = regexp.MustCompile(`([\S]+)_([0-9]+)$`)
)

type NameMangler interface {
	MangleName(string, ...any) string
}

type NameUnmangler interface {
	UnmangleName(string) (string, error)
}

func NewViewNameMangler() NameMangler {
	return &standardNameMutator{}
}

func NewViewNameUnmangler() NameUnmangler {
	return &standardNameMutator{}
}

type standardNameMutator struct {
}

func (v *standardNameMutator) MangleName(base string, varargs ...any) string {
	if len(varargs) == 0 {
		return base
	}
	if len(varargs) == 1 {
		i, isInt := varargs[0].(int)
		if isInt && i == 0 {
			return base
		}
	}
	var sb strings.Builder
	sb.WriteString(base)
	for _, arg := range varargs {
		sb.WriteString(fmt.Sprintf("_%v", arg))
	}
	return sb.String()
}

func (v *standardNameMutator) UnmangleName(base string) (string, error) {
	matches := standardUnmangleRegexp.FindStringSubmatch(base)
	if len(matches) == 0 {
		return base, nil
	}
	if len(matches) != 3 { //nolint:mnd // 3 is the expected length
		return "", fmt.Errorf("could not unmangle %s", base)
	}
	return matches[1], nil
}

// func (v *standardNameMutator) GetWhereClause(base string) string {
// 	matches := standardUnmangleRegexp.FindStringSubmatch(base)
// 	if len(matches) == 0 {
// 		return base, nil
// 	}
// 	if len(matches) != 3 { //nolint:mnd // 3 is the expected length
// 		return "", fmt.Errorf("could not unmangle %s", base)
// 	}
// 	return matches[1], nil
// }
