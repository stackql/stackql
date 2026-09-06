package relational_test

import (
	"errors"
	"regexp"
	"testing"

	"github.com/stackql/any-sdk/public/formulation"
	"github.com/stackql/stackql-parser/go/sqltypes"
	"github.com/stackql/stackql/internal/stackql/relational"
)

var (
	errTestTable = errors.New("test table error")
	errLHS       = errors.New("lhs error")
	errRHS       = errors.New("rhs error")
)

type testTable struct {
	name      string
	values    map[string]any
	sqlValues map[string]sqltypes.Value
	err       error
}

func (t *testTable) GetKey(key string) (any, error) {
	if t.err != nil {
		return nil, t.err
	}
	value, ok := t.values[key]
	if !ok {
		return nil, errTestTable
	}
	return value, nil
}

func (t *testTable) GetKeyAsSqlVal(key string) (sqltypes.Value, error) {
	if t.err != nil {
		return sqltypes.NULL, t.err
	}
	value, ok := t.sqlValues[key]
	if !ok {
		return sqltypes.NULL, errTestTable
	}
	return value, nil
}

func (t *testTable) GetName() string           { return t.name }
func (t *testTable) KeyExists(key string) bool { _, ok := t.values[key]; return ok }

func tableFilter(result formulation.ITable, err error) func(formulation.ITable) (formulation.ITable, error) {
	return func(formulation.ITable) (formulation.ITable, error) {
		return result, err
	}
}

func TestAndTableFilters(t *testing.T) {
	row := &testTable{name: "row"}
	lhsResult := &testTable{name: "lhs"}
	rhsResult := &testTable{name: "rhs"}
	rhs := tableFilter(row, nil)
	if got, err := relational.AndTableFilters(nil, rhs)(row); err != nil || got != row {
		t.Fatalf("nil lhs should return rhs result: got %v, err %v", got, err)
	}

	tests := []struct {
		name    string
		lhs     func(formulation.ITable) (formulation.ITable, error)
		rhs     func(formulation.ITable) (formulation.ITable, error)
		want    formulation.ITable
		wantErr error
	}{
		{name: "both match returns lhs", lhs: tableFilter(lhsResult, nil), rhs: tableFilter(rhsResult, nil), want: lhsResult},
		{name: "lhs misses", lhs: tableFilter(nil, nil), rhs: tableFilter(row, nil)},
		{name: "rhs misses", lhs: tableFilter(row, nil), rhs: tableFilter(nil, nil)},
		{name: "lhs error", lhs: tableFilter(nil, errLHS), rhs: tableFilter(row, nil), wantErr: errLHS},
		{name: "rhs error", lhs: tableFilter(row, nil), rhs: tableFilter(nil, errRHS), wantErr: errRHS},
		{name: "lhs error takes precedence", lhs: tableFilter(nil, errLHS), rhs: tableFilter(nil, errRHS), wantErr: errLHS},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := relational.AndTableFilters(tt.lhs, tt.rhs)(row)
			if !errors.Is(err, tt.wantErr) || got != tt.want {
				t.Fatalf("got (%v, %v), want (%v, %v)", got, err, tt.want, tt.wantErr)
			}
		})
	}
}

func TestOrTableFilters(t *testing.T) {
	row := &testTable{name: "row"}
	lhsResult := &testTable{name: "lhs"}
	rhsResult := &testTable{name: "rhs"}
	rhs := tableFilter(row, nil)
	if got, err := relational.OrTableFilters(nil, rhs)(row); err != nil || got != row {
		t.Fatalf("nil lhs should return rhs result: got %v, err %v", got, err)
	}

	tests := []struct {
		name    string
		lhs     func(formulation.ITable) (formulation.ITable, error)
		rhs     func(formulation.ITable) (formulation.ITable, error)
		want    formulation.ITable
		wantErr error
	}{
		{name: "lhs matches", lhs: tableFilter(lhsResult, nil), rhs: tableFilter(nil, nil), want: lhsResult},
		{name: "rhs matches", lhs: tableFilter(nil, nil), rhs: tableFilter(rhsResult, nil), want: rhsResult},
		{name: "both match returns lhs", lhs: tableFilter(lhsResult, nil), rhs: tableFilter(rhsResult, nil), want: lhsResult},
		{name: "neither matches", lhs: tableFilter(nil, nil), rhs: tableFilter(nil, nil)},
		{name: "lhs error", lhs: tableFilter(nil, errLHS), rhs: tableFilter(row, nil), wantErr: errLHS},
		{name: "rhs error", lhs: tableFilter(row, nil), rhs: tableFilter(nil, errRHS), wantErr: errRHS},
		{name: "lhs error takes precedence", lhs: tableFilter(nil, errLHS), rhs: tableFilter(nil, errRHS), wantErr: errLHS},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := relational.OrTableFilters(tt.lhs, tt.rhs)(row)
			if !errors.Is(err, tt.wantErr) || got != tt.want {
				t.Fatalf("got (%v, %v), want (%v, %v)", got, err, tt.want, tt.wantErr)
			}
		})
	}
}

func TestConstructTablePredicateFilter(t *testing.T) {
	row := &testTable{sqlValues: map[string]sqltypes.Value{
		"age":        sqltypes.NewInt64(42),
		"expression": sqltypes.MakeTrusted(sqltypes.Expression, []byte("x")),
	}}
	tests := []struct {
		name     string
		operator string
		rhs      sqltypes.Value
		want     formulation.ITable
	}{
		{name: "equal", operator: "=", rhs: sqltypes.NewInt64(42), want: row},
		{name: "not equal", operator: "!=", rhs: sqltypes.NewInt64(41), want: row},
		{name: "greater than", operator: ">", rhs: sqltypes.NewInt64(41), want: row},
		{name: "greater than rejects larger rhs", operator: ">", rhs: sqltypes.NewInt64(43)},
		{name: "greater than or equal", operator: ">=", rhs: sqltypes.NewInt64(42), want: row},
		{name: "less than", operator: "<", rhs: sqltypes.NewInt64(43), want: row},
		{name: "less than or equal", operator: "<=", rhs: sqltypes.NewInt64(42), want: row},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			predicate, err := relational.GetOperatorPredicate(tt.operator)
			if err != nil {
				t.Fatal(err)
			}
			got, err := relational.ConstructTablePredicateFilter("age", tt.rhs, predicate)(row)
			if err != nil || got != tt.want {
				t.Fatalf("got (%v, %v), want (%v, nil)", got, err, tt.want)
			}
		})
	}

	equal, err := relational.GetOperatorPredicate("=")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = relational.ConstructTablePredicateFilter("missing", sqltypes.NewInt64(42), equal)(row); !errors.Is(err, errTestTable) {
		t.Fatalf("missing column error = %v, want %v", err, errTestTable)
	}
	if _, err = relational.ConstructTablePredicateFilter("expression", sqltypes.MakeTrusted(sqltypes.Expression, []byte("x")), equal)(row); err == nil {
		t.Fatal("uncomparable values should return an error")
	}
}

func TestConstructLikePredicateFilter(t *testing.T) {
	re := regexp.MustCompile(`^Stack`)
	row := &testTable{values: map[string]any{"name": "StackQL", "other": "SQL", "count": 3}}

	tests := []struct {
		name     string
		column   string
		negating bool
		want     formulation.ITable
		wantErr  bool
	}{
		{name: "like match", column: "name", want: row},
		{name: "like rejects nonmatch", column: "other"},
		{name: "not like rejects match", column: "name", negating: true},
		{name: "not like accepts nonmatch", column: "other", negating: true, want: row},
		{name: "missing column", column: "missing", wantErr: true},
		{name: "non string", column: "count", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := relational.ConstructLikePredicateFilter(tt.column, re, tt.negating)(row)
			if (err != nil) != tt.wantErr || got != tt.want {
				t.Fatalf("got (%v, %v), want (%v, error=%v)", got, err, tt.want, tt.wantErr)
			}
		})
	}
}

func TestGetOperatorPredicate(t *testing.T) {
	tests := []struct {
		operator string
		matches  []int
	}{
		{operator: "=", matches: []int{0}},
		{operator: "!=", matches: []int{-1, 1}},
		{operator: ">=", matches: []int{0, 1}},
		{operator: ">", matches: []int{1}},
		{operator: "<=", matches: []int{-1, 0}},
		{operator: "<", matches: []int{-1}},
	}
	for _, tt := range tests {
		t.Run(tt.operator, func(t *testing.T) {
			predicate, err := relational.GetOperatorPredicate(tt.operator)
			if err != nil {
				t.Fatal(err)
			}
			for _, result := range []int{-1, 0, 1} {
				want := false
				for _, match := range tt.matches {
					want = want || result == match
				}
				if got := predicate(result); got != want {
					t.Errorf("predicate(%d) = %v, want %v", result, got, want)
				}
			}
		})
	}
	if _, err := relational.GetOperatorPredicate("LIKE"); err == nil {
		t.Fatal("unsupported operator should return an error")
	}
}
