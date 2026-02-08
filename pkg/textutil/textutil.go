package textutil

import (
	"regexp"
)

var (
	namespaceLikeStringRegex *regexp.Regexp = regexp.MustCompile(`{{.*?}}`)
)

func GetTemplateLikeString(templateString string) string {
	return namespaceLikeStringRegex.ReplaceAllString(templateString, "%")
}
