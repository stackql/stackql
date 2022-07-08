package dataflow

import (
	"github.com/stackql/stackql/internal/stackql/taxonomy"
	"gonum.org/v1/gonum/graph"
	"vitess.io/vitess/go/vt/sqlparser"
)

type DataFlowVertex interface {
	graph.Node
	DataFlowUnit
	GetAnnotation() taxonomy.AnnotationCtx
	GetTableExpr() sqlparser.TableExpr
}

type StandardDataFlowVertex struct {
	id         int64
	collection DataFlowCollection
	annotation taxonomy.AnnotationCtx
	tableExpr  sqlparser.TableExpr
}

func NewStandardDataFlowVertex(
	annotation taxonomy.AnnotationCtx,
	tableExpr sqlparser.TableExpr,
	id int64) DataFlowVertex {
	return &StandardDataFlowVertex{
		annotation: annotation,
		tableExpr:  tableExpr,
		id:         id,
	}
}

func (dv *StandardDataFlowVertex) iDataFlowUnit() {}

func (dv *StandardDataFlowVertex) ID() int64 {
	return dv.id
}

func (dv *StandardDataFlowVertex) GetAnnotation() taxonomy.AnnotationCtx {
	return dv.annotation
}

func (dv *StandardDataFlowVertex) GetTableExpr() sqlparser.TableExpr {
	return dv.tableExpr
}
