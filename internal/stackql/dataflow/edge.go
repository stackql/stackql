package dataflow

import (
	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"gonum.org/v1/gonum/graph"
)

type Edge interface {
	graph.WeightedEdge
	AddRelation(Relation)
	GetColumnDescriptors() ([]openapistackql.ColumnDescriptor, error)
	GetDest() Vertex
	GetProjection() (map[string]string, error)
	GetSelectExprs() (sqlparser.SelectExprs, error)
	GetSource() Vertex
	IsSQL() bool
}

type standardDataFlowEdge struct {
	source, dest Vertex
	relations    []Relation
}

func NewStandardDataFlowEdge(
	source Vertex,
	dest Vertex,
	comparisonExpr *sqlparser.ComparisonExpr,
	sourceExpr sqlparser.Expr,
	destColumn *sqlparser.ColName,
) Edge {
	return &standardDataFlowEdge{
		source: source,
		dest:   dest,
		relations: []Relation{
			NewStandardDataFlowRelation(
				comparisonExpr,
				destColumn,
				sourceExpr,
			),
		},
	}
}

func (de *standardDataFlowEdge) AddRelation(rel Relation) {
	de.relations = append(de.relations, rel)
}

func (de *standardDataFlowEdge) From() graph.Node {
	return de.source
}

func (de *standardDataFlowEdge) To() graph.Node {
	return de.dest
}

func (de *standardDataFlowEdge) ReversedEdge() graph.Edge {
	// Reversal is invalid given the assymetric
	// expressions, therefore returning unaltered
	// as per library recommmendation.
	return de
}

func (de *standardDataFlowEdge) Weight() float64 {
	return 1.0
}

func (de *standardDataFlowEdge) GetSource() Vertex {
	return de.source
}

func (de *standardDataFlowEdge) IsSQL() bool {
	for _, rel := range de.relations {
		if rel.IsSQL() {
			return true
		}
	}
	return false
}

func (de *standardDataFlowEdge) GetDest() Vertex {
	return de.dest
}

func (de *standardDataFlowEdge) GetProjection() (map[string]string, error) {
	rv := make(map[string]string)
	for _, rel := range de.relations {
		src, dst, err := rel.GetProjection()
		if err != nil {
			return nil, err
		}
		rv[src] = dst
	}
	return rv, nil
}

func (de *standardDataFlowEdge) GetSelectExprs() (sqlparser.SelectExprs, error) {
	var rv sqlparser.SelectExprs
	for _, rel := range de.relations {
		selExpr, err := rel.GetSelectExpr()
		if err != nil {
			return nil, err
		}
		rv = append(rv, selExpr)
	}
	return rv, nil
}

func (de *standardDataFlowEdge) GetColumnDescriptors() ([]openapistackql.ColumnDescriptor, error) {
	var rv []openapistackql.ColumnDescriptor
	for _, rel := range de.relations {
		d, err := rel.GetColumnDescriptor()
		if err != nil {
			return nil, err
		}
		rv = append(rv, d)
	}
	return rv, nil
}
