package nativedb_test

import (
	"testing"

	"github.com/stackql/stackql/internal/stackql/nativedb"
)

func TestNewColumn(t *testing.T) {
	col := nativedb.NewColumn("id", "INT")
	if col == nil {
		t.Fatal("NewColumn returned nil")
	}
}

func TestColumn_GetName(t *testing.T) {
	col := nativedb.NewColumn("my_column", "VARCHAR")
	if got := col.GetName(); got != "my_column" {
		t.Errorf("GetName() = %q, want 'my_column'", got)
	}
}

func TestColumn_GetType(t *testing.T) {
	col := nativedb.NewColumn("col", "BIGINT")
	if got := col.GetType(); got != "BIGINT" {
		t.Errorf("GetType() = %q, want 'BIGINT'", got)
	}
}

func TestColumn_GetWidth_DefaultZero(t *testing.T) {
	col := nativedb.NewColumn("col", "INT")
	width, ok := col.GetWidth()
	if ok {
		t.Errorf("GetWidth() first return = %v, want false for unset width", ok)
	}
	if width != 0 {
		t.Errorf("GetWidth() width = %d, want 0 for unset", width)
	}
}

func TestColumn_SetWidth(t *testing.T) {
	col := nativedb.NewColumn("col", "VARCHAR")
	col.SetWidth(255)
	width, ok := col.GetWidth()
	if !ok {
		t.Errorf("GetWidth() second return = %v, want true after SetWidth", ok)
	}
	if width != 255 {
		t.Errorf("GetWidth() width = %d, want 255", width)
	}
}

func TestColumn_SetWidth_MultipleTimes(t *testing.T) {
	col := nativedb.NewColumn("col", "CHAR")
	col.SetWidth(10)
	col.SetWidth(20)
	col.SetWidth(30)
	width, _ := col.GetWidth()
	if width != 30 {
		t.Errorf("GetWidth() = %d, want 30 (last SetWidth value)", width)
	}
}

func TestColumn_GetNameAndType_Independent(t *testing.T) {
	// Name and type should not affect each other
	col := nativedb.NewColumn("name_field", "TEXT")
	if got := col.GetName(); got != "name_field" {
		t.Errorf("GetName() = %q, want 'name_field'", got)
	}
	if got := col.GetType(); got != "TEXT" {
		t.Errorf("GetType() = %q, want 'TEXT'", got)
	}

	col.SetWidth(100)
	width, ok := col.GetWidth()
	if !ok || width != 100 {
		t.Errorf("after SetWidth(100), GetWidth() = (%d, %v), want (100, true)", width, ok)
	}
}

func TestColumn_EmptyName(t *testing.T) {
	col := nativedb.NewColumn("", "INT")
	if got := col.GetName(); got != "" {
		t.Errorf("GetName() = %q, want ''", got)
	}
}

func TestColumn_EmptyType(t *testing.T) {
	col := nativedb.NewColumn("col", "")
	if got := col.GetType(); got != "" {
		t.Errorf("GetType() = %q, want ''", got)
	}
}

func TestColumn_ZeroWidth(t *testing.T) {
	// Setting width to 0 should report as unset
	col := nativedb.NewColumn("col", "INT")
	col.SetWidth(0)
	_, ok := col.GetWidth()
	if ok {
		t.Error("GetWidth() should return false after SetWidth(0)")
	}
}

func TestNewSelect(t *testing.T) {
	cols := []nativedb.Column{
		nativedb.NewColumn("id", "INT"),
		nativedb.NewColumn("name", "VARCHAR"),
	}
	sel := nativedb.NewSelect(cols)
	if sel == nil {
		t.Fatal("NewSelect returned nil")
	}
	gotCols := sel.GetColumns()
	if len(gotCols) != 2 {
		t.Errorf("GetColumns() len = %d, want 2", len(gotCols))
	}
	if gotCols[0].GetName() != "id" {
		t.Errorf("GetColumns()[0].GetName() = %q, want 'id'", gotCols[0].GetName())
	}
	if gotCols[1].GetName() != "name" {
		t.Errorf("GetColumns()[1].GetName() = %q, want 'name'", gotCols[1].GetName())
	}
}

func TestNewSelect_EmptyColumns(t *testing.T) {
	sel := nativedb.NewSelect(nil)
	gotCols := sel.GetColumns()
	if len(gotCols) != 0 {
		t.Errorf("GetColumns() len = %d, want 0", len(gotCols))
	}
}

func TestNewSelect_NilRows(t *testing.T) {
	sel := nativedb.NewSelect(nil)
	if sel.GetRows() != nil {
		t.Error("GetRows() should return nil when no rows provided")
	}
}

func TestNewSelectWithRows(t *testing.T) {
	cols := []nativedb.Column{
		nativedb.NewColumn("col1", "TEXT"),
	}
	sel := nativedb.NewSelectWithRows(cols, nil)
	if sel == nil {
		t.Fatal("NewSelectWithRows returned nil")
	}
	if len(sel.GetColumns()) != 1 {
		t.Errorf("GetColumns() len = %d, want 1", len(sel.GetColumns()))
	}
	// rows is nil in this call
	if sel.GetRows() != nil {
		t.Error("GetRows() should return nil when nil rows provided")
	}
}
