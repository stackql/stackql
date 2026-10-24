package intrinsic //nolint:testpackage // Tests host environment policy before SDK resolution.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql-labs/omnisdk/pkg/query"
)

//nolint:gocognit // End-to-end metadata, precedence, and missing-input assertions share one fixture.
func TestServerEnvironmentBindingsPreserveSQLPrecedence(t *testing.T) {
	dir := t.TempDir()
	document := `openapi: 3.0.0
servers:
  - url: 'https://{deployment}.example.test'
    variables:
      deployment: {default: ignored, x-stackQL-envVar: OMNI_EXPLICIT_TEST_DEPLOYMENT}
paths:
  /items:
    get:
      responses:
        '200':
          content:
            application/json:
              schema: {type: array, items: {type: object, properties: {name: {type: string}}}}
components:
  x-stackQL-resources:
    items:
      methods:
        select:
          operation: {$ref: '#/paths/~1items/get'}
          response: {mediaType: application/json}
`
	if err := os.WriteFile(filepath.Join(dir, "provider.yaml"), []byte("name: p\nproviderServices:\n  s:\n    service: {$ref: s.yaml}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "s.yaml"), []byte(document), 0600); err != nil {
		t.Fatal(err)
	}
	table, err := omnisdk.DescribeTable(dir, "stackql_unstable_p.s.items")
	if err != nil {
		t.Fatal(err)
	}
	tables := map[string]omnisdk.Table{"i": table}
	methods, err := omnisdk.DocMethods(dir, "p", "s", "items")
	if err != nil || len(methods) != 1 {
		t.Fatalf("methods=%v err=%v", methods, err)
	}
	t.Setenv("OMNI_EXPLICIT_TEST_DEPLOYMENT", "from-env")
	for _, explicit := range []bool{false, true} {
		where := []query.Predicate{}
		expected := "from-env"
		if explicit {
			expected = "from-sql"
			where = append(where, query.NewEq(query.NewColumn("i", "deployment"), query.NewLiteral(expected)))
		}
		original, err := query.New([]query.Join{query.NewJoin(query.NewResource("i", table.Address()), query.Base)}, where, []query.Output{query.NewOutput("name", query.NewColumn("i", "name"))})
		if err != nil {
			t.Fatal(err)
		}
		bound, err := bindServerEnvironment(original, tables)
		if err != nil {
			t.Fatal(err)
		}
		if len(original.Where()) != len(where) {
			t.Fatal("host mutated original query")
		}
		if len(bound.Where()) != 1 {
			t.Fatalf("bindings=%v", bound.Where())
		}
		predicate := bound.Where()[0].(query.Compare)
		if got := predicate.Right().(query.Literal).Value(); got != expected {
			t.Fatalf("binding=%v want=%v", got, expected)
		}
	}
	if got := unresolvedServerParams(methods[0]); len(got) != 0 {
		t.Fatalf("env-resolved required=%v", got)
	}
	t.Setenv("OMNI_EXPLICIT_TEST_DEPLOYMENT", "")
	original, err := query.New([]query.Join{query.NewJoin(query.NewResource("i", table.Address()), query.Base)}, nil, []query.Output{query.NewOutput("name", query.NewColumn("i", "name"))})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bindServerEnvironment(original, tables); err == nil {
		t.Fatal("missing environment silently used document default")
	}
	if got := unresolvedServerParams(methods[0]); len(got) != 1 || got[0] != "deployment" {
		t.Fatalf("unresolved required=%v", got)
	}
}
