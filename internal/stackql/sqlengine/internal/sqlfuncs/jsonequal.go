package sqlfuncs

import (
	"encoding/json"
	"errors"
	"strings"
)

// errInvalidJSONArgs preserves the retired C error text for NULL arguments.
var errInvalidJSONArgs = errors.New("Invalid JSON strings") //nolint:staticcheck // C-parity error text

// errJSONParse preserves the retired C error text for unparseable documents.
var errJSONParse = errors.New("Error parsing JSON strings") //nolint:staticcheck // C-parity error text

// JSONEqual implements json_equal(a, b): 1 if the documents are deeply equal,
// else 0. Objects compare unordered, arrays ordered, numbers as doubles with
// the epsilon rule (1 == 1.0). NULL or unparseable arguments are errors.
func JSONEqual(a, b any) (any, error) {
	sa, okA := valueText(a)
	sb, okB := valueText(b)
	if !okA || !okB {
		return nil, errInvalidJSONArgs
	}
	va, errA := parseJSONDocument(sa)
	vb, errB := parseJSONDocument(sb)
	if errA != nil || errB != nil {
		return nil, errJSONParse
	}
	if jsonDeepEqual(va, vb) {
		return int64(1), nil
	}
	return int64(0), nil
}

// parseJSONDocument parses exactly one JSON value, rejecting trailing
// non-whitespace content (a divergence from cJSON, see DIVERGENCES.md).
func parseJSONDocument(s string) (any, error) {
	dec := json.NewDecoder(strings.NewReader(s))
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	if dec.More() {
		return nil, errors.New("trailing content after JSON value")
	}
	return v, nil
}

// jsonDeepEqual ports cJSON_Compare with case-sensitive keys.
//
//nolint:gocognit // Preserve the migrated JSON comparison semantics.
func jsonDeepEqual(a, b any) bool {
	switch av := a.(type) {
	case nil:
		return b == nil
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	case float64:
		bv, ok := b.(float64)
		return ok && compareDouble(av, bv)
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for i := range av {
			if !jsonDeepEqual(av[i], bv[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		for k, v := range av {
			bItem, present := bv[k]
			if !present || !jsonDeepEqual(v, bItem) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
