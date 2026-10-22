package astfuncrewrite_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/astfuncrewrite"
)

func TestNopRewriteFunc(t *testing.T) {
	tests := []struct {
		name string
		expr *sqlparser.FuncExpr
	}{
		{name: "nil"},
		{
			name: "unrelated function",
			expr: &sqlparser.FuncExpr{
				Name:  sqlparser.NewColIdent("sum"),
				Exprs: sqlparser.SelectExprs{&sqlparser.AliasedExpr{Expr: sqlparser.NewIntVal([]byte("1"))}},
			},
		},
		{
			name: "json extract without arguments",
			expr: &sqlparser.FuncExpr{Name: sqlparser.NewColIdent("json_extract")},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var before string
			if tt.expr != nil {
				before = sqlparser.String(tt.expr)
			}
			got, err := astfuncrewrite.GetNopFuncRewriter().RewriteFunc(tt.expr)
			if err != nil {
				t.Fatalf("RewriteFunc() error = %v", err)
			}
			if got != tt.expr {
				t.Fatal("RewriteFunc() did not return the original expression")
			}
			if got != nil && sqlparser.String(got) != before {
				t.Errorf("RewriteFunc() changed expression: got %s, want %s", sqlparser.String(got), before)
			}
		})
	}
}

func TestPostgresRewriteFuncUnchanged(t *testing.T) {
	tests := []struct {
		name string
		expr *sqlparser.FuncExpr
	}{
		{name: "nil"},
		{
			name: "unrelated function",
			expr: &sqlparser.FuncExpr{
				Name:  sqlparser.NewColIdent("sum"),
				Exprs: sqlparser.SelectExprs{&sqlparser.AliasedExpr{Expr: sqlparser.NewIntVal([]byte("1"))}},
			},
		},
		{
			name: "postgres json extract",
			expr: &sqlparser.FuncExpr{
				Name: sqlparser.NewColIdent("json_extract_path_text"),
				Exprs: sqlparser.SelectExprs{
					&sqlparser.AliasedExpr{Expr: &sqlparser.ColName{Name: sqlparser.NewColIdent("document")}},
					&sqlparser.AliasedExpr{Expr: sqlparser.NewStrVal([]byte("name"))},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var before string
			if tt.expr != nil {
				before = sqlparser.String(tt.expr)
			}
			got, err := astfuncrewrite.GetPostgresASTFuncRewriter().RewriteFunc(tt.expr)
			if err != nil {
				t.Fatalf("RewriteFunc() error = %v", err)
			}
			if got != tt.expr {
				t.Fatal("RewriteFunc() did not return the original expression")
			}
			if got != nil && sqlparser.String(got) != before {
				t.Errorf("RewriteFunc() changed expression: got %s, want %s", sqlparser.String(got), before)
			}
		})
	}
}

func TestPostgresRewriteJSONExtract(t *testing.T) {
	tests := []struct {
		name     string
		funcName string
		path     string
		wantPath []string
	}{
		{name: "single key", funcName: "json_extract", path: "$.name", wantPath: []string{"name"}},
		{name: "nested keys", funcName: "json_extract", path: "$.items.name", wantPath: []string{"items", "name"}},
		{name: "root", funcName: "json_extract", path: "$"},
		{name: "without root", funcName: "json_extract", path: "items.name", wantPath: []string{"items", "name"}},
		{name: "mixed case", funcName: "JsOn_ExTrAcT", path: "$.name", wantPath: []string{"name"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			firstArg := &sqlparser.AliasedExpr{Expr: &sqlparser.ColName{Name: sqlparser.NewColIdent("document")}}
			expr := &sqlparser.FuncExpr{
				Name: sqlparser.NewColIdent(tt.funcName),
				Exprs: sqlparser.SelectExprs{
					firstArg,
					&sqlparser.AliasedExpr{Expr: sqlparser.NewStrVal([]byte(tt.path))},
				},
			}
			want := &sqlparser.FuncExpr{
				Name: sqlparser.NewColIdent("json_extract_path_text"),
				Exprs: sqlparser.SelectExprs{
					&sqlparser.AliasedExpr{Expr: &sqlparser.ColName{Name: sqlparser.NewColIdent("document")}},
				},
			}
			for _, key := range tt.wantPath {
				want.Exprs = append(want.Exprs, &sqlparser.AliasedExpr{Expr: sqlparser.NewStrVal([]byte(key))})
			}

			got, err := astfuncrewrite.GetPostgresASTFuncRewriter().RewriteFunc(expr)
			if err != nil {
				t.Fatalf("RewriteFunc() error = %v", err)
			}
			if got != expr {
				t.Fatal("RewriteFunc() did not return the original expression")
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("RewriteFunc() = %#v, want %#v", got, want)
			}
			if got.Exprs[0] != firstArg {
				t.Error("RewriteFunc() did not preserve the first argument")
			}
		})
	}
}

func TestPostgresRewriteJSONExtractInvalidArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    sqlparser.SelectExprs
		wantErr string
	}{
		{name: "no arguments", wantErr: "arg count = 0"},
		{
			name:    "one argument",
			args:    sqlparser.SelectExprs{&sqlparser.AliasedExpr{Expr: sqlparser.NewStrVal([]byte("{}"))}},
			wantErr: "arg count = 1",
		},
		{
			name: "too many arguments",
			args: sqlparser.SelectExprs{
				&sqlparser.AliasedExpr{Expr: sqlparser.NewStrVal([]byte("{}"))},
				&sqlparser.AliasedExpr{Expr: sqlparser.NewStrVal([]byte("$.name"))},
				&sqlparser.AliasedExpr{Expr: sqlparser.NewStrVal([]byte("$.other"))},
			},
			wantErr: "arg count = 3",
		},
		{
			name: "non-aliased path",
			args: sqlparser.SelectExprs{
				&sqlparser.AliasedExpr{Expr: sqlparser.NewStrVal([]byte("{}"))},
				&sqlparser.StarExpr{},
			},
			wantErr: "path expression of type",
		},
		{
			name: "non-literal path",
			args: sqlparser.SelectExprs{
				&sqlparser.AliasedExpr{Expr: sqlparser.NewStrVal([]byte("{}"))},
				&sqlparser.AliasedExpr{Expr: &sqlparser.ColName{Name: sqlparser.NewColIdent("path")}},
			},
			wantErr: "path val of type",
		},
		{
			name: "non-string path",
			args: sqlparser.SelectExprs{
				&sqlparser.AliasedExpr{Expr: sqlparser.NewStrVal([]byte("{}"))},
				&sqlparser.AliasedExpr{Expr: sqlparser.NewIntVal([]byte("1"))},
			},
			wantErr: "path val with value type",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := &sqlparser.FuncExpr{Name: sqlparser.NewColIdent("json_extract"), Exprs: tt.args}
			got, err := astfuncrewrite.GetPostgresASTFuncRewriter().RewriteFunc(expr)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("RewriteFunc() error = %v, want error containing %q", err, tt.wantErr)
			}
			if got != nil {
				t.Errorf("RewriteFunc() = %#v, want nil on error", got)
			}
		})
	}
}
