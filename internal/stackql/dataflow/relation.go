package dataflow

import (
	"fmt"

	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"

	"github.com/stackql/any-sdk/pkg/logging"
)

type Relation interface {
	GetProjection() (string, string, error)
	GetSelectExpr() (sqlparser.SelectExpr, error)
	GetColumnDescriptor() (anysdk.ColumnDescriptor, error)
	GetDestColumnName() string
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

func (dr *standardDataFlowRelation) GetDestColumnName() string {
	return dr.destColumn.Name.GetRawVal()
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

func (dr *standardDataFlowRelation) GetColumnDescriptor() (anysdk.ColumnDescriptor, error) {
	decoratedColumn := fmt.Sprintf(`%s AS %s`, sqlparser.ColDelimitedString(dr.sourceExpr), dr.destColumn.Name.GetRawVal())
	cd := anysdk.NewColumnDescriptor(dr.destColumn.Name.GetRawVal(), "", "", decoratedColumn, nil, nil, nil)
	return cd, nil
}
