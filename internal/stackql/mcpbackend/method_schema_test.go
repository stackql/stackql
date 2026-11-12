package mcpbackend_test

import (
	"strings"
	"testing"

	"github.com/stackql/stackql/internal/stackql/mcpbackend"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/pkg/mcp_server/dto"
)

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
