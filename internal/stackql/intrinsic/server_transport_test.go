package intrinsic //nolint:testpackage // Verifies host-only transport defaults and explicit scope.

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql-labs/omnisdk/pkg/query"
)

func TestTransportDefaultsPreserveExplicitInputsAndScope(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"provider.yaml": "name: p\nproviderServices:\n  s:\n    service: {$ref: s.yaml}\n",
		"s.yaml": `openapi: 3.0.0
servers:
  - url: '{scheme}://{host}/{region}'
    variables:
      scheme: {default: https}
      host: {default: ignored.example.test}
      region: {default: ignored-region}
paths:
  /items:
    get:
      responses:
        '200':
          content:
            application/json:
              schema: {type: array, items: {type: object, properties: {id: {type: string}}}}
components:
  x-stackQL-resources:
    items:
      methods:
        select: {operation: {$ref: '#/paths/~1items/get'}, response: {mediaType: application/json}}
`,
	}
	for name, body := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	table, err := omnisdk.DescribeTable(dir, "stackql_unstable_p.s.items")
	if err != nil {
		t.Fatal(err)
	}
	from := []query.Join{query.NewJoin(query.NewResource("i", table.Address()), query.Base)}
	selected := []query.Output{query.NewOutput("id", query.NewColumn("i", "id"))}
	for _, explicit := range []bool{false, true} {
		where := []query.Predicate{query.NewEq(query.NewColumn("i", "host"), query.NewLiteral("caller.test")), query.NewEq(query.NewColumn("i", "region"), query.NewLiteral("caller-region"))}
		expected := "https"
		if explicit {
			expected = "http"
			where = append(where, query.NewEq(query.NewColumn("i", "scheme"), query.NewLiteral(expected)))
		}
		original, queryErr := query.New(from, where, selected)
		if queryErr != nil {
			t.Fatal(queryErr)
		}
		bound, bindingErr := bindServerEnvironment(original, map[string]omnisdk.Table{"i": table})
		if bindingErr != nil {
			t.Fatal(bindingErr)
		}
		resolved, resolveErr := omnisdk.Resolve(bound, map[string]omnisdk.Table{"i": table})
		if resolveErr != nil {
			t.Fatal(resolveErr)
		}
		if got := resolved.Graph().Nodes()[0].Params()["scheme"]; got != expected {
			t.Fatalf("scheme=%s want=%s", got, expected)
		}
	}
	original, err := query.New(from, nil, selected)
	if err != nil {
		t.Fatal(err)
	}
	bound, err := bindServerEnvironment(original, map[string]omnisdk.Table{"i": table})
	if err != nil {
		t.Fatal(err)
	}
	if len(bound.Where()) != 1 {
		t.Fatal("host substituted scope defaults")
	}
	methods, err := omnisdk.DocMethods(dir, "p", "s", "items")
	if err != nil {
		t.Fatal(err)
	}
	if got := unresolvedServerParams(methods[0]); !reflect.DeepEqual(got, []string{"host", "region"}) {
		t.Fatalf("required=%v", got)
	}
}
