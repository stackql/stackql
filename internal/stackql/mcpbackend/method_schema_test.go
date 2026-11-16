package mcpbackend_test

import (
	"strings"
	"testing"

	"github.com/stackql/stackql/internal/stackql/mcpbackend"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/pkg/mcp_server/dto"
)

func TestDescribeMethodQuotesKeywordMethodNames(t *testing.T) {
	// stackql/stackql#926: `list` lexes as a keyword, so the parser rejects
	// it bare in this position. The host-native branch must quote its
	// identifiers, and the quoted statement must still resolve to the same
	// method name downstream.
	input := dto.HierarchyInput{Provider: "stackql_preview", Service: "google", Resource: "buckets", Method: "list", Descriptions: true}
	query, err := mcpbackend.NewSimpleStackqlInterrogator().GetDescribeMethod(input)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := sqlparser.Parse(query)
	if err != nil {
		t.Fatalf("quoted DESCRIBE METHOD does not parse: %v (%s)", err, query)
	}
	dm, ok := parsed.(*sqlparser.DescribeMethod)
	if !ok {
		t.Fatalf("parsed statement is %T, not DescribeMethod", parsed)
	}
	if got := dm.Method.GetRawVal(); got != "list" {
		t.Fatalf("method parsed as %q, want %q", got, "list")
	}
	// A name carrying a quote must not escape the quoted identifier.
	input.Method = `we"ird`
	query, err = mcpbackend.NewSimpleStackqlInterrogator().GetDescribeMethod(input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sqlparser.Parse(query); err != nil {
		t.Fatalf("quoted identifier with an embedded quote does not parse: %v (%s)", err, query)
	}
}

func TestDescribeMethodSerializesControlsForBothBackends(t *testing.T) {
	input := dto.HierarchyInput{Provider: "google", Service: "compute", Resource: "instances", Method: "insert", Depth: 2, Path: "data__settings.mode", Descriptions: true, ParamType: "input", IncludeGlobal: true}
	query, err := mcpbackend.NewSimpleStackqlInterrogator().GetDescribeMethod(input)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sqlparser.Parse(query); err != nil {
		t.Fatal(err)
	}
	for _, fragment := range []string{"stackql_preview.metadata.method_fields", "depth = 2", "path = 'data__settings.mode'", "descriptions = 'true'", "param_type = 'input'", "include_global = 'true'"} {
		if !strings.Contains(query, fragment) {
			t.Fatalf("missing %s: %s", fragment, query)
		}
	}
	input.Path = "data__settings.' OR 1=1 --"
	query, err = mcpbackend.NewSimpleStackqlInterrogator().GetDescribeMethod(input)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := sqlparser.Parse(query)
	if err != nil {
		t.Fatal(err)
	}
	where := parsed.(*sqlparser.Select).Where
	if strings.Contains(sqlparser.String(where.Expr), " OR 1 = 1") {
		t.Fatal("path escaped its literal", query)
	}
	input.Depth = 33
	if _, err := mcpbackend.NewSimpleStackqlInterrogator().GetDescribeMethod(input); err == nil {
		t.Fatal("excessive depth accepted")
	}
}
