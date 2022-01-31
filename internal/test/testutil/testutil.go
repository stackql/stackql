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
	if err == nil {
		t.Logf("file contents for testing = %s", string(fileContents))
		if stringent {
			return s == string(fileContents)
		}
		return strings.ReplaceAll(strings.TrimSpace(s), "\r\n", "\n") == strings.ReplaceAll(strings.TrimSpace(string(fileContents)), "\r\n", "\n")
	}
	return false
}
