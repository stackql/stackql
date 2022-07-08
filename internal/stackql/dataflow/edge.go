package dataflow

import (
	"fmt"

	"gonum.org/v1/gonum/graph"
	"vitess.io/vitess/go/vt/sqlparser"
)

type DataFlowEdge interface {
	graph.WeightedEdge
	AddRelation(DataFlowRelation)
	GetDest() DataFlowVertex
	GetProjection() (map[string]string, error)
	GetSource() DataFlowVertex
}

type DataFlowRelation interface {
	GetProjection() (string, string, error)
}

type StandardDataFlowRelation struct {
	comparisonExpr *sqlparser.ComparisonExpr
	destColumn     *sqlparser.ColName
	sourceExpr     sqlparser.Expr
}

func (dr *StandardDataFlowRelation) GetProjection() (string, string, error) {
	switch se := dr.sourceExpr.(type) {
	case *sqlparser.ColName:
		return se.Name.GetRawVal(), dr.destColumn.Name.GetRawVal(), nil
	default:
		return "", "", fmt.Errorf("cannot project from expression type = '%T'", se)
	}
}

func NewStandardDataFlowRelation(
	comparisonExpr *sqlparser.ComparisonExpr,
	destColumn *sqlparser.ColName,
	sourceExpr sqlparser.Expr,
) DataFlowRelation {
	return &StandardDataFlowRelation{
		comparisonExpr: comparisonExpr,
		destColumn:     destColumn,
		sourceExpr:     sourceExpr,
	}
}

type StandardDataFlowEdge struct {
	source, dest DataFlowVertex
	relations    []DataFlowRelation
}

func NewStandardDataFlowEdge(
	source DataFlowVertex,
	dest DataFlowVertex,
	comparisonExpr *sqlparser.ComparisonExpr,
	sourceExpr sqlparser.Expr,
	destColumn *sqlparser.ColName,
) DataFlowEdge {
	return &StandardDataFlowEdge{
		source: source,
		dest:   dest,
		relations: []DataFlowRelation{
			NewStandardDataFlowRelation(
				comparisonExpr,
				destColumn,
				sourceExpr,
			),
		},
	}
}

func (de *StandardDataFlowEdge) AddRelation(rel DataFlowRelation) {
	de.relations = append(de.relations, rel)
}

func (de *StandardDataFlowEdge) From() graph.Node {
	return de.source
}

func (de *StandardDataFlowEdge) To() graph.Node {
	return de.dest
}

func (de *StandardDataFlowEdge) ReversedEdge() graph.Edge {
	// Reversal is invalid given the assymetric
	// expressions, therefore returning unaltered
	// as per library recommmendation.
	return de
}

func (de *StandardDataFlowEdge) Weight() float64 {
	return 1.0
}

func (de *StandardDataFlowEdge) GetSource() DataFlowVertex {
	return de.source
}

func (de *StandardDataFlowEdge) GetDest() DataFlowVertex {
	return de.dest
}

func (dv *StandardDataFlowEdge) GetProjection() (map[string]string, error) {
	rv := make(map[string]string)
	for _, rel := range dv.relations {
		src, dst, err := rel.GetProjection()
		if err != nil {
			return nil, err
		}
		rv[src] = dst
	}
	return rv, nil
}
