package dependencyplanner //nolint:testpackage // tests internal projection mapping

import (
	"io"
	"reflect"
	"testing"

	"github.com/stackql/any-sdk/pkg/streaming"
	"github.com/stackql/any-sdk/public/formulation"
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
