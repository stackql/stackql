package sqlfuncs

import (
	"errors"
	"sort"
	"strings"
)

// errInvalidPolicyArgs preserves the retired C error text for NULL arguments.
var errInvalidPolicyArgs = errors.New("Invalid policy strings") //nolint:staticcheck // C-parity error text

// errPolicyParse preserves the retired C error text for unparseable documents.
var errPolicyParse = errors.New("Error parsing policy JSON strings") //nolint:staticcheck // C-parity error text

// awsUnorderedFields are the policy fields whose arrays compare as unordered
// sets and whose "arn:" strings compare case-insensitively. "Tags" and "tags"
// are fork-only additions shipped in the stackql-go-sqlite3 fork's
// SQLITE_ENABLE_STACKQL amalgamation, never backported to
// sqlite-ext-functions (see DIVERGENCES.md).
//
//nolint:gochecknoglobals // Immutable policy field lookup.
var awsUnorderedFields = map[string]struct{}{
	"Action":       {},
	"NotAction":    {},
	"Resource":     {},
	"NotResource":  {},
	"Principal":    {},
	"NotPrincipal": {},
	"AWS":          {},
	"Service":      {},
	"Tags":         {},
	"tags":         {},
}

// AWSPolicyEqual implements aws_policy_equal(a, b): 1 if the two IAM policy
// documents are equivalent under the retired C normalization rules (see
// awsPolicyCompare and DIVERGENCES.md), else 0. NULL or unparseable
// arguments are errors.
func AWSPolicyEqual(a, b any) (any, error) {
	sa, okA := valueText(a)
	sb, okB := valueText(b)
	if !okA || !okB {
		return nil, errInvalidPolicyArgs
	}
	if sa == sb {
		return int64(1), nil
	}
	va, errA := parseJSONDocument(sa)
	vb, errB := parseJSONDocument(sb)
	if errA != nil || errB != nil {
		return nil, errPolicyParse
	}
	// Fork-only behavior (see DIVERGENCES.md): two top-level array documents
	// compare unordered, as in the stackql-go-sqlite3 fork's amalgamation.
	_, aIsArray := va.([]any)
	_, bIsArray := vb.([]any)
	rootUnordered := aIsArray && bIsArray
	if awsPolicyCompare(va, vb, rootUnordered) {
		return int64(1), nil
	}
	return int64(0), nil
}

// awsPolicyCompare ports aws_policy_compare_items: single-element arrays
// equal their lone string element, awsUnorderedFields arrays compare as sets,
// keys are case-insensitive, and the "arn:" case-fold is tested on the left
// operand only - a deliberate C asymmetry.
//
//nolint:gocognit,gocyclo,cyclop // Preserve the migrated policy comparison semantics.
func awsPolicyCompare(a, b any, parentUnordered bool) bool {
	if aArr, ok := a.([]any); ok {
		if bStr, isStr := b.(string); isStr {
			return len(aArr) == 1 && awsPolicyCompare(aArr[0], bStr, parentUnordered)
		}
	}
	if bArr, ok := b.([]any); ok {
		if aStr, isStr := a.(string); isStr {
			return len(bArr) == 1 && awsPolicyCompare(bArr[0], aStr, parentUnordered)
		}
	}
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
		if !ok {
			return false
		}
		if parentUnordered && strings.HasPrefix(av, "arn:") {
			return asciiEqualFold(av, bv)
		}
		return av == bv
	case []any:
		bv, ok := b.([]any)
		if !ok || len(av) != len(bv) {
			return false
		}
		if parentUnordered {
			for _, aElem := range av {
				if !awsFindMatchingElement(bv, aElem) {
					return false
				}
			}
			return true
		}
		for i := range av {
			if !awsPolicyCompare(av[i], bv[i], parentUnordered) {
				return false
			}
		}
		return true
	case map[string]any:
		bv, ok := b.(map[string]any)
		if !ok {
			return false
		}
		for k, v := range av {
			_, fieldUnordered := awsUnorderedFields[k]
			bItem, present := awsObjectGet(bv, k)
			if !present || !awsPolicyCompare(v, bItem, fieldUnordered) {
				return false
			}
		}
		for k := range bv {
			if _, present := awsObjectGet(av, k); !present {
				return false
			}
		}
		return true
	default:
		return false
	}
}

// awsFindMatchingElement reports whether item matches any element of array
// under unordered comparison.
func awsFindMatchingElement(array []any, item any) bool {
	for _, elem := range array {
		if awsPolicyCompare(elem, item, true) {
			return true
		}
	}
	return false
}

// awsObjectGet is a case-insensitive key lookup: an exact match wins, then
// the lexicographically smallest folding key, keeping lookups deterministic.
func awsObjectGet(obj map[string]any, key string) (any, bool) {
	if v, ok := obj[key]; ok {
		return v, true
	}
	var candidates []string
	for k := range obj {
		if asciiEqualFold(k, key) {
			candidates = append(candidates, k)
		}
	}
	if len(candidates) == 0 {
		return nil, false
	}
	sort.Strings(candidates)
	return obj[candidates[0]], true
}
