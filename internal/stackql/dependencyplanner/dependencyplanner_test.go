package dependencyplanner //nolint:testpackage // tests internal projection mapping

import (
	"io"
	"reflect"
	"testing"

	"github.com/stackql/any-sdk/pkg/streaming"
	"github.com/stackql/any-sdk/public/formulation"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

type projectionColumn struct {
	formulation.ColumnDescriptor
	name     string
	wireName string
}

func (column projectionColumn) GetName() string     { return column.name }
func (column projectionColumn) GetWireName() string { return column.wireName }

type projectionTabulation struct {
	formulation.Tabulation
	columns []formulation.ColumnDescriptor
}

func (tab projectionTabulation) GetColumns() []formulation.ColumnDescriptor { return tab.columns }

type projectionParameter struct {
	formulation.Addressable
	name string
}

func (parameter projectionParameter) GetName() string { return parameter.name }

func TestDestinationRequestProjection(t *testing.T) {
	projection := map[string]string{"AccessKeyId": "access_key_id", "other": "unchanged"}
	parameters := map[string]formulation.Addressable{
		"access_key_id": projectionParameter{name: "AccessKeyId"},
	}
	translated := destinationRequestProjection(projection, parameters)
	want := map[string]string{"AccessKeyId": "AccessKeyId", "other": "unchanged"}
	if !reflect.DeepEqual(translated, want) {
		t.Fatalf("projection = %v, want %v", translated, want)
	}
	if projection["AccessKeyId"] != "access_key_id" {
		t.Fatal("source projection was mutated")
	}
}

func TestSourceResponseProjection(t *testing.T) {
	projection := map[string]string{
		"access_key_id": "access_key_id",
		"unchanged":     "destination",
		"unknown":       "unknown_destination",
		"empty_wire":    "empty_destination",
	}
	tab := projectionTabulation{columns: []formulation.ColumnDescriptor{
		projectionColumn{name: "access_key_id", wireName: "AccessKeyId"},
		projectionColumn{name: "unchanged", wireName: "unchanged"},
		projectionColumn{name: "empty_wire"},
	}}
	translated := sourceResponseProjection(projection, tab)
	want := map[string]string{
		"AccessKeyId": "access_key_id", "unchanged": "destination",
		"unknown": "unknown_destination", "empty_wire": "empty_destination",
	}
	if !reflect.DeepEqual(translated, want) {
		t.Fatalf("projection = %v, want %v", translated, want)
	}
	if projection["access_key_id"] != "access_key_id" {
		t.Fatal("source projection was mutated")
	}
	stream := streaming.NewSimpleProjectionMapStream(translated, nil)
	if err := stream.Write([]map[string]interface{}{{
		"AccessKeyId": "test-key", "unchanged": "value", "unknown": "other", "empty_wire": "empty",
	}}); err != nil {
		t.Fatal(err)
	}
	rows, err := stream.Read()
	if err != io.EOF {
		t.Fatalf("stream read: %v", err)
	}
	if len(rows) != 1 || rows[0]["access_key_id"] != "test-key" {
		t.Fatalf("projected rows = %v", rows)
	}
}

func parseSelect(t *testing.T, query string) *sqlparser.Select {
	t.Helper()
	stmt, err := sqlparser.Parse(query)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	selectStmt, ok := stmt.(*sqlparser.Select)
	if !ok {
		t.Fatalf("not a select: %T", stmt)
	}
	return selectStmt
}

func TestIsSourceOnlyResponseExpr(t *testing.T) {
	requestParameters := map[string]struct{}{"org": {}}
	for where, expected := range map[string]bool{
		"r.archived = 0":                 true,
		"r.archived = 0 and r.size > 1":  true,
		"r.org = 'stackql'":              false,
		"archived = 0":                   false,
		"b.type = 'x'":                   false,
		"r.name = b.repo":                false,
		"1 = 1":                          false,
		"r.name in (select name from t)": false,
	} {
		stmt := parseSelect(t, "select 1 from t where "+where)
		if got := isSourceOnlyResponseExpr(stmt.Where.Expr, "r", requestParameters); got != expected {
			t.Errorf("isSourceOnlyResponseExpr(%q) = %t, want %t", where, got, expected)
		}
	}
}

func TestIsNullExtended(t *testing.T) {
	for from, expected := range map[string]bool{
		"a r inner join b on r.x = b.x":                            false,
		"a r left outer join b on r.x = b.x":                       false,
		"b left outer join a r on r.x = b.x":                       true,
		"a r right join b on r.x = b.x":                            true,
		"c inner join (b left join a r on r.x = b.x) on c.y = b.y": true,
	} {
		stmt := parseSelect(t, "select 1 from "+from)
		if got := isNullExtended(stmt.From, "r", false); got != expected {
			t.Errorf("isNullExtended(%q) = %t, want %t", from, got, expected)
		}
	}
}
