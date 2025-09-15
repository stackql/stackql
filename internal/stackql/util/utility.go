package util

import (
	"path"
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

func GetForwardSlashFilePathFromRepositoryRoot(relativePath string) (string, error) {
	_, filename, _, _ := runtime.Caller(0)
	curDir := path.Dir(filename)
	rv, err := filepath.Abs(path.Join(curDir, "../../..", relativePath))
	return filepath.ToSlash(rv), err
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

func TrimSelectItemsKey(selectItemsKey string) string {
	splitSet := strings.Split(selectItemsKey, "/")
	if len(splitSet) == 0 {
		return ""
	}
	return splitSet[len(splitSet)-1]
}
