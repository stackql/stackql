package dataflow

import (
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
	"gonum.org/v1/gonum/graph"
)

type DataFlowVertex interface {
	graph.Node
	DataFlowUnit
	GetAnnotation() taxonomy.AnnotationCtx
	GetTableExpr() sqlparser.TableExpr
}

type standardDataFlowVertex struct {
	id         int64
	annotation taxonomy.AnnotationCtx
	tableExpr  sqlparser.TableExpr
}

func NewStandardDataFlowVertex(
	annotation taxonomy.AnnotationCtx,
	tableExpr sqlparser.TableExpr,
	id int64) DataFlowVertex {
	return &standardDataFlowVertex{
		annotation: annotation,
		tableExpr:  tableExpr,
		id:         id,
	}
}

func (dv *standardDataFlowVertex) iDataFlowUnit() {}

func (dv *standardDataFlowVertex) ID() int64 {
	return dv.id
}

func (dv *standardDataFlowVertex) GetAnnotation() taxonomy.AnnotationCtx {
	return dv.annotation
}

func (dv *standardDataFlowVertex) GetTableExpr() sqlparser.TableExpr {
	return dv.tableExpr
}
