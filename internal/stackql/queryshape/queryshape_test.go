package queryshape //nolint:testpackage // tests unexported extractSingleTableName

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/paramdecoder"
)

type extractTableCase struct {
	Description string `json:"description"`
	Query       string `json:"query"`
	Expected    string `json:"expected"`
}

func TestExtractSingleTableName(t *testing.T) {
	data, err := os.ReadFile("testdata/extract_table_cases.json")
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}
	var cases []extractTableCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("failed to parse testdata: %v", err)
	}
	for _, tc := range cases {
		t.Run(tc.Description, func(t *testing.T) {
			got := extractSingleTableName(tc.Query)
			if got != tc.Expected {
				t.Errorf("extractSingleTableName(%q) = %q, want %q", tc.Query, got, tc.Expected)
			}
		})
	}
}

type SubstituteParamsCase struct {
	Description string    `json:"description"`
	Query       string    `json:"query"`
	ParamValues []*string `json:"paramValues"` // nil entries represent SQL NULL
	Expected    string    `json:"expected"`
}

func (c *SubstituteParamsCase) toByteSlices() [][]byte {
	result := make([][]byte, len(c.ParamValues))
	for i, v := range c.ParamValues {
		if v == nil {
			result[i] = nil
		} else {
			result[i] = []byte(*v)
		}
	}
	return result
}

// decodeTextParams runs text-format wire values through the real decoder.
func decodeTextParams(t *testing.T, paramValues [][]byte) []paramdecoder.DecodedParam {
	t.Helper()
	decoded, err := paramdecoder.NewDecoder().DecodeParams(nil, nil, paramValues)
	if err != nil {
		t.Fatalf("failed to decode params: %v", err)
	}
	return decoded
}

func TestSubstituteDecodedParams(t *testing.T) {
	data, err := os.ReadFile("testdata/substitute_params_cases.json")
	if err != nil {
		t.Fatalf("failed to read testdata: %v", err)
	}
	var cases []SubstituteParamsCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("failed to parse testdata: %v", err)
	}
	for _, tc := range cases {
		t.Run(tc.Description, func(t *testing.T) {
			got := SubstituteDecodedParams(tc.Query, decodeTextParams(t, tc.toByteSlices()))
			if got != tc.Expected {
				t.Errorf("SubstituteDecodedParams(%q, ...) = %q, want %q", tc.Query, got, tc.Expected)
			}
		})
	}
}

// A substituted value must reach the parser as exactly one string literal.
func TestSubstituteDecodedParamsStaysOneLiteral(t *testing.T) {
	for _, val := range []string{
		`x\`, `\'`, `'`, `\\`, `a' OR 1 = 1 -- `, `"; select 1; "`, "tab\there", `\n`, "$1",
	} {
		resolved := SubstituteDecodedParams("SELECT $1", decodeTextParams(t, [][]byte{[]byte(val)}))
		tokenizer := sqlparser.NewStringTokenizer(resolved)
		tokenizer.Scan() // SELECT
		token, got := tokenizer.Scan()
		next, _ := tokenizer.Scan()
		if token != sqlparser.STRING || string(got) != val || next != 0 {
			t.Errorf("value %q became %q: token %d %q, then token %d", val, resolved, token, got, next)
		}
	}
}
