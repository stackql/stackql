package textutil

import (
	"regexp"
)

//nolint:revive // Explicit type declaration removes any ambiguity
var (
	namespaceLikeStringRegex *regexp.Regexp = regexp.MustCompile(`{{.*}}`)
)

func GetTemplateLikeString(templateString string) string {
	return namespaceLikeStringRegex.ReplaceAllString(templateString, "%")
}
