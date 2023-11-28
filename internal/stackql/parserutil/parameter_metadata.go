package parserutil

import (
	"fmt"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

var (
	_ ParameterMetadata = (*StandardComparisonParameterMetadata)(nil)
	_ ParameterMetadata = (*PlaceholderParameterMetadata)(nil)
)

type ParameterMetadata interface {
	iParameterMetadata()
	GetParent() *sqlparser.ComparisonExpr
	GetVal() interface{}
	GetTable() sqlparser.SQLNode
	SetTable(sqlparser.SQLNode) error
}

type StandardComparisonParameterMetadata struct {
	Parent *sqlparser.ComparisonExpr
	Val    interface{}
	table  sqlparser.SQLNode
}

type PlaceholderParameterMetadata struct {
	placeholderVal struct{}
}

func NewComparisonParameterMetadata(parent *sqlparser.ComparisonExpr, val interface{}) ParameterMetadata {
	return &StandardComparisonParameterMetadata{
		Parent: parent,
		Val:    val,
	}
}

func NewPlaceholderParameterMetadata() ParameterMetadata {
	return PlaceholderParameterMetadata{}
}

func (pm *StandardComparisonParameterMetadata) iParameterMetadata() {}

func (pm *StandardComparisonParameterMetadata) GetParent() *sqlparser.ComparisonExpr {
	return pm.Parent
}

func (pm *StandardComparisonParameterMetadata) GetVal() interface{} {
	return pm.Val
}

func (pm *StandardComparisonParameterMetadata) GetTable() sqlparser.SQLNode {
	return pm.table
}

func (pm *StandardComparisonParameterMetadata) SetTable(tb sqlparser.SQLNode) error {
	pm.table = tb
	return nil
}

func (pm PlaceholderParameterMetadata) iParameterMetadata() {}

func (pm PlaceholderParameterMetadata) GetVal() interface{} {
	return pm.placeholderVal
}

func (pm PlaceholderParameterMetadata) GetParent() *sqlparser.ComparisonExpr {
	return nil
}

func (pm PlaceholderParameterMetadata) GetTable() sqlparser.SQLNode {
	return nil
}

//nolint:revive // The unused cmd is retained as a future proofing measure
func (pm PlaceholderParameterMetadata) SetTable(tb sqlparser.SQLNode) error {
	return fmt.Errorf("placeholder parameter metadata does not support set table")
}
