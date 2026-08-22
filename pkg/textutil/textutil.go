package textutil

import (
	"regexp"
	"strings"
)

const (
	IndirectQueryPlaceholder   = "__iql__placeholder_indirect_query__"
	ControlOnClausePlaceholder = "__iql__placeholder_control_on_clause__"
)

var (
	namespaceLikeStringRegex *regexp.Regexp = regexp.MustCompile(`{{.*?}}`)
)

func GetTemplateLikeString(templateString string) string {
	return namespaceLikeStringRegex.ReplaceAllString(templateString, "%")
}

func ExpandPlaceholders(template string, placeholder string, replacements []string) string {
	if placeholder == "" || len(replacements) == 0 {
		return template
	}
	var expanded strings.Builder
	remainder := template
	for _, replacement := range replacements {
		idx := strings.Index(remainder, placeholder)
		if idx < 0 {
			break
		}
		expanded.WriteString(remainder[:idx])
		expanded.WriteString(replacement)
		remainder = remainder[idx+len(placeholder):]
	}
	expanded.WriteString(remainder)
	return expanded.String()
}
