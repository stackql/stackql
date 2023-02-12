package selectmetadata

import (
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/dataflow"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
)

var (
	_ SelectMetadata = &standardSelectMetadata{}
)

func NewSelectMetadata(
	onConditionDataFlows dataflow.DataFlowCollection,
	onConditionsToRewrite map[*sqlparser.ComparisonExpr]struct{},
	tableMap taxonomy.TblMap,
	annotations taxonomy.AnnotationCtxMap,
) SelectMetadata {
	return &standardSelectMetadata{
		onConditionDataFlows:  onConditionDataFlows,
		onConditionsToRewrite: onConditionsToRewrite,
		tableMap:              tableMap,
		annotations:           annotations,
	}
}

type SelectMetadata interface {
	GetAnnotations() (taxonomy.AnnotationCtxMap, bool)
	GetOnConditionDataFlows() (dataflow.DataFlowCollection, bool)
	GetOnConditionsToRewrite() map[*sqlparser.ComparisonExpr]struct{}
	GetTableMap() (taxonomy.TblMap, bool)
}

type standardSelectMetadata struct {
	onConditionDataFlows  dataflow.DataFlowCollection
	onConditionsToRewrite map[*sqlparser.ComparisonExpr]struct{}
	tableMap              taxonomy.TblMap
	annotations           taxonomy.AnnotationCtxMap
}

func (sm *standardSelectMetadata) GetTableMap() (taxonomy.TblMap, bool) {
	return sm.tableMap, sm.tableMap != nil
}

func (sm *standardSelectMetadata) GetAnnotations() (taxonomy.AnnotationCtxMap, bool) {
	return sm.annotations, sm.annotations != nil
}

func (sm *standardSelectMetadata) GetOnConditionsToRewrite() map[*sqlparser.ComparisonExpr]struct{} {
	return sm.onConditionsToRewrite
}

func (sm *standardSelectMetadata) GetOnConditionDataFlows() (dataflow.DataFlowCollection, bool) {
	return sm.onConditionDataFlows, sm.onConditionDataFlows != nil
}
