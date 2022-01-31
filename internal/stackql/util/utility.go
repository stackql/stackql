package util

import (
	"path/filepath"
	"runtime"
	"strings"
)

func GetFilePathFromRepositoryRoot(relativePath string) (string, error) {
	_, filename, _, _ := runtime.Caller(0)
	curDir := filepath.Dir(filename)
	rv, err := filepath.Abs(filepath.Join(curDir, "../../..", relativePath))
	return strings.ReplaceAll(rv, `\`, `\\`), err
}

func MaxMapKey(numbers map[int]interface{}) int {
	var maxNumber int
	for maxNumber = range numbers {
		break
	}
	for n := range numbers {
		if n > maxNumber {
			maxNumber = n
		}
	}
	return maxNumber
}
