package dataflow

import (
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
	"gonum.org/v1/gonum/graph"
)

type Vertex interface {
	graph.Node
	Unit
	GetAnnotation() taxonomy.AnnotationCtx
	GetEquivalencyGroup() int64
	SetEquivalencyGroup(id int64)
	GetTableExpr() sqlparser.TableExpr
}

type standardDataFlowVertex struct {
	id                 int64
	equiValencyGroupID int64
	annotation         taxonomy.AnnotationCtx
	tableExpr          sqlparser.TableExpr
}

func (dv *standardDataFlowVertex) iDataFlowUnit() {}

func (dv *standardDataFlowVertex) ID() int64 {
	return dv.id
}

func (dv *standardDataFlowVertex) GetEquivalencyGroup() int64 {
	return dv.equiValencyGroupID
}

func (dv *standardDataFlowVertex) SetEquivalencyGroup(id int64) {
	dv.equiValencyGroupID = id
}

func (dv *standardDataFlowVertex) GetAnnotation() taxonomy.AnnotationCtx {
	return dv.annotation
}

func (dv *standardDataFlowVertex) GetTableExpr() sqlparser.TableExpr {
	return dv.tableExpr
}
