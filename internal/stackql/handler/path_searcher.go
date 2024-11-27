package handler

import (
	"fmt"
	"strings"

	"github.com/stackql/any-sdk/pkg/jsonpath"
)

type simpleSystemPathSearcher struct {
	system    string
	remainder string
}

type systemPathSearcher interface {
	GetSystem() string
	GetRemainder() string
}

func composeSystemSearchPath(path string) (systemPathSearcher, error) {
	pSplit, splitErr := jsonpath.SplitSearchPath(path)
	if splitErr != nil {
		return nil, splitErr
	}
	if len(pSplit) < 1 {
		return nil, fmt.Errorf("path '%s' is insufficient", path)
	}
	remainder := ""
	if len(pSplit) > 1 {
		remainder = strings.TrimPrefix(path, pSplit[0]+".")
	}
	return &simpleSystemPathSearcher{
		system:    pSplit[0],
		remainder: remainder,
	}, nil
}

func (ss *simpleSystemPathSearcher) GetSystem() string {
	return ss.system
}

func (ss *simpleSystemPathSearcher) GetRemainder() string {
	return ss.remainder
}
