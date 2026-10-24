package jsonpath

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/PaesslerAG/gval"
	jp "github.com/PaesslerAG/jsonpath"

	"context"
)

func Get(path string, value any) (interface{}, error) {
	return jp.Get(path, value)
}

func SplitSearchPath(pathStr string) ([]string, error) {
	return splitSearchPath(map[string]interface{}{}, pathStr)
}

func splitSearchPath(inputMap map[string]interface{}, pathStr string) ([]string, error) {
	pathStr = strings.TrimPrefix(pathStr, "$.")
	if pathStr == "" {
		return []string{}, nil
	}
	vs := gval.VariableSelector(func(path gval.Evaluables) gval.Evaluable {
		return func(c context.Context, v interface{}) (interface{}, error) {
			keys, err := path.EvalStrings(c, v)
			if err != nil {
				return nil, err
			}
			return keys, nil
		}
	})
	lang := gval.NewLanguage(append(
		[]gval.Language{gval.Base()},
		vs,
	)...)
	rawVal, err := lang.Evaluate(
		pathStr,
		inputMap,
	)
	conformedVal, isStringSlice := rawVal.([]string)
	if !isStringSlice {
		return nil, fmt.Errorf("cannot accomodate inferred JSON path of type %T", conformedVal)
	}
	if err != nil {
		return nil, err
	}
	return conformedVal, nil
}

//nolint:gocognit // Preserve nested JSON path assignment semantics.
func Set(input any, pathStr string, rhs interface{}) error {
	if pathStr == "" {
		switch input := input.(type) {
		case map[string]interface{}:
			rhsMap, rhsIsMap := rhs.(map[string]interface{})
			if !rhsIsMap {
				return fmt.Errorf("cannot overwrite map with type %T", rhs)
			}
			for k, _ := range input {
				delete(input, k)
			}
			for k, v := range rhsMap {
				input[k] = v
			}
			return nil
		default:
			mutatedSer, mutatedSerErr := json.Marshal(rhs)
			if mutatedSerErr != nil {
				return mutatedSerErr
			}
			overwriteErr := json.Unmarshal(mutatedSer, input)
			if overwriteErr != nil {
				return overwriteErr
			}
			return nil
		}
	}
	switch input := input.(type) {
	case map[string]interface{}:
		return setMap(input, pathStr, rhs)
	default:
		ser, serErr := json.Marshal(input)
		if serErr != nil {
			return serErr
		}
		m := map[string]interface{}{}
		deserErr := json.Unmarshal(ser, &m)
		if deserErr != nil {
			return deserErr
		}
		mutateErr := setMap(m, pathStr, rhs)
		if mutateErr != nil {
			return mutateErr
		}
		mutatedSer, mutatedSerErr := json.Marshal(m)
		if mutatedSerErr != nil {
			return mutatedSerErr
		}
		overwriteErr := json.Unmarshal(mutatedSer, input)
		if overwriteErr != nil {
			return overwriteErr
		}
	}
	return nil
}

func setMap(input map[string]interface{}, pathStr string, rhs interface{}) error {
	conformedVal, err := splitSearchPath(map[string]interface{}{}, pathStr)
	if err != nil {
		return err
	}
	lhs := input
	var lhsOk bool
	var k string
	for i := range conformedVal {
		k = conformedVal[i]
		if i == len(conformedVal)-1 {
			break
		}
		lhs, lhsOk = lhs[k].(map[string]interface{})
		if !lhsOk {
			return fmt.Errorf("disallowed map traversal into type %T on key %s at index %d", lhs[k], k, i)
		}
	}
	lhs[k] = rhs
	return nil
}
