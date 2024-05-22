package serde

import "strings"

type StringArrayMapSerDe interface {
	Serialize([]string) (string, error)
	Deserialize(string) (map[string]any, error)
}

type stringArrayMapSerDe struct {
}

func NewStringArrayMapSerDe() StringArrayMapSerDe {
	return &stringArrayMapSerDe{}
}

func (s *stringArrayMapSerDe) Serialize(arr []string) (string, error) {
	return strings.Join(arr, ","), nil
}

func (s *stringArrayMapSerDe) Deserialize(str string) (map[string]any, error) {
	strArr := strings.Split(str, ",")
	rv := make(map[string]any, len(strArr))
	for _, strElem := range strArr {
		if strElem == "" {
			continue
		}
		rv[strElem] = struct{}{}
	}
	return rv, nil
}
