package queryshape //nolint:testpackage // tests unexported extractSingleTableName

import (
	"encoding/json"
	"os"
	"testing"
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

func TestSubstituteParams(t *testing.T) {
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
			got := SubstituteParams(tc.Query, nil, tc.toByteSlices())
			if got != tc.Expected {
				t.Errorf("SubstituteParams(%q, ...) = %q, want %q", tc.Query, got, tc.Expected)
			}
		})
	}
}
