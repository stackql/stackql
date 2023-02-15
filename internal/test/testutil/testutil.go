package testutil

import (
	"io"
	"io/ioutil"
	"strings"
	"testing"
)

func CreateReadCloserFromString(s string) io.ReadCloser {
	return ioutil.NopCloser(strings.NewReader(s))
}

func StringEqualsFileContents(t *testing.T, s string, filePath string, stringent bool) bool {
	fileContents, err := ioutil.ReadFile(filePath)
	fileContentsString := string(fileContents)
	if err == nil {
		t.Logf("file contents for testing = %s", fileContentsString)
		if stringent {
			return s == fileContentsString
		}
		if s == fileContentsString {
			return true
		}
		lhs := strings.ReplaceAll(strings.TrimSpace(s), "\r\n", "\n")
		rhs := strings.ReplaceAll(strings.TrimSpace(fileContentsString), "\r\n", "\n")
		comparison := strings.Compare(lhs, rhs)
		return comparison == 0
	}
	return false
}
