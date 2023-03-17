package relational

import (
	"fmt"
	"regexp"

	"github.com/stackql/go-openapistackql/openapistackql"

	"github.com/stackql/stackql-parser/go/sqltypes"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql-parser/go/vt/vtgate/evalengine"
)

func AndTableFilters(
	lhs,
	rhs func(openapistackql.ITable) (openapistackql.ITable, error),
) func(openapistackql.ITable) (openapistackql.ITable, error) {
	if lhs == nil {
		return rhs
	}
	return func(t openapistackql.ITable) (openapistackql.ITable, error) {
		lResult, lErr := lhs(t)
		rResult, rErr := rhs(t)
		if lErr != nil {
			return nil, lErr
		}
		if rErr != nil {
			return nil, rErr
		}
		if lResult != nil && rResult != nil {
			return lResult, nil
		}
		return nil, nil
	}
}

func OrTableFilters(
	lhs,
	rhs func(openapistackql.ITable) (openapistackql.ITable, error),
) func(openapistackql.ITable) (openapistackql.ITable, error) {
	if lhs == nil {
		return rhs
	}
	return func(t openapistackql.ITable) (openapistackql.ITable, error) {
		lResult, lErr := lhs(t)
		rResult, rErr := rhs(t)
		if lErr != nil {
			return nil, lErr
		}
		if rErr != nil {
			return nil, rErr
		}
		if lResult != nil {
			return lResult, nil
		}
		if rResult != nil {
			return rResult, nil
		}
		return nil, nil
	}
}

func ConstructTablePredicateFilter(
	colName string, rhs sqltypes.Value,
	operatorPredicate func(int) bool) func(openapistackql.ITable) (openapistackql.ITable, error) {
	return func(row openapistackql.ITable) (openapistackql.ITable, error) {
		v, e := row.GetKeyAsSqlVal(colName)
		if e != nil {
			return nil, e
		}
		result, err := evalengine.NullsafeCompare(v, rhs)
		if err == nil && operatorPredicate(result) {
			return row, nil
		}
		return nil, err
	}
}

func ConstructLikePredicateFilter(
	colName string,
	rhs *regexp.Regexp, isNegating bool) func(openapistackql.ITable) (openapistackql.ITable, error) {
	return func(row openapistackql.ITable) (openapistackql.ITable, error) {
		v, vErr := row.GetKey(colName)
		if vErr != nil {
			return nil, vErr
		}
		s, sOk := v.(string)
		if !sOk {
			return nil, fmt.Errorf("cannot compare non-string type '%T' with regex", v)
		}
		if rhs.MatchString(s) != isNegating {
			return row, nil
		}
		return nil, nil
	}
}

func GetOperatorPredicate(operator string) (func(int) bool, error) {
	switch operator {
	case sqlparser.EqualStr:
		return func(result int) bool {
			return result == 0
		}, nil
	case sqlparser.NotEqualStr:
		return func(result int) bool {
			return result != 0
		}, nil
	case sqlparser.GreaterEqualStr:
		return func(result int) bool {
			return result >= 0
		}, nil
	case sqlparser.GreaterThanStr:
		return func(result int) bool {
			return result > 0
		}, nil
	case sqlparser.LessEqualStr:
		return func(result int) bool {
			return result <= 0
		}, nil
	case sqlparser.LessThanStr:
		return func(result int) bool {
			return result < 0
		}, nil
	}
	return nil, fmt.Errorf("cannot determine predicate")
}
