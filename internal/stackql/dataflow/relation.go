package dataflow

import (
	"fmt"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"

	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql/internal/stackql/logging"
)

type Relation interface {
	GetProjection() (string, string, error)
	GetSelectExpr() (sqlparser.SelectExpr, error)
	GetColumnDescriptor() (openapistackql.ColumnDescriptor, error)
	IsSQL() bool
}

type standardDataFlowRelation struct {
	comparisonExpr *sqlparser.ComparisonExpr
	destColumn     *sqlparser.ColName
	sourceExpr     sqlparser.Expr
}

func NewStandardDataFlowRelation(
	comparisonExpr *sqlparser.ComparisonExpr,
	destColumn *sqlparser.ColName,
	sourceExpr sqlparser.Expr,
) Relation {
	return &standardDataFlowRelation{
		comparisonExpr: comparisonExpr,
		destColumn:     destColumn,
		sourceExpr:     sourceExpr,
	}
}

func (dr *standardDataFlowRelation) GetProjection() (string, string, error) {
	switch se := dr.sourceExpr.(type) {
	case *sqlparser.ColName:
		return se.Name.GetRawVal(), dr.destColumn.Name.GetRawVal(), nil
	default:
		return "", "", fmt.Errorf("cannot project from expression type = '%T'", se)
	}
}

func (dr *standardDataFlowRelation) IsSQL() bool {
	switch se := dr.sourceExpr.(type) {
	case *sqlparser.ColName:
		return false
	default:
		logging.GetLogger().Infof("%v\n", se)
		return true
	}
}

func (dr *standardDataFlowRelation) GetSelectExpr() (sqlparser.SelectExpr, error) {
	rv := &sqlparser.AliasedExpr{
		Expr: dr.sourceExpr,
		As:   dr.destColumn.Name,
	}
	return rv, nil
}

func (dr *standardDataFlowRelation) GetColumnDescriptor() (openapistackql.ColumnDescriptor, error) {
	decoratedColumn := fmt.Sprintf(`%s AS %s`, sqlparser.String(dr.sourceExpr), dr.destColumn.Name.GetRawVal())
	cd := openapistackql.NewColumnDescriptor(dr.destColumn.Name.GetRawVal(), "", "", decoratedColumn, nil, nil, nil)
	return cd, nil
}
