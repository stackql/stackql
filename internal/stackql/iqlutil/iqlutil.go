package iqlutil

import (
	"bytes"
	"encoding/json"
	"regexp"
	"sort"
	"strings"
)

func TranslateLikeToRegexPattern(likeString string) string {
	return "^" + strings.ReplaceAll(regexp.QuoteMeta(likeString), "%", ".*") + "$"
}

func SanitisePossibleTickEscapedTerm(term string) string {
	return strings.TrimSuffix(strings.TrimPrefix(term, "`"), "`")
}

func PrettyPrintSomeJSON(body []byte) ([]byte, error) {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, body, "", "  ")
	if err != nil {
		return nil, err
	}
	return prettyJSON.Bytes(), nil
}

func GetSortedKeysStringMap(m map[string]string) []string {
	var retVal []string
	for k := range m {
		retVal = append(retVal, k)
	}
	sort.Strings(retVal)
	return retVal
}
