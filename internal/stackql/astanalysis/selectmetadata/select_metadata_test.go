package selectmetadata //nolint:testpackage // to test unexported methods

import (
	"testing"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/dataflow"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
	"github.com/stretchr/testify/assert"
)

func TestSelectMetadata(t *testing.T) {
	var onConditionDataFlows dataflow.Collection
	onConditionsToRewrite := make(map[*sqlparser.ComparisonExpr]struct{})
	tableMap := taxonomy.TblMap{}
	annotations := taxonomy.AnnotationCtxMap{}
	t.Run("NewSelectMetadata", func(t *testing.T) {
		sm := NewSelectMetadata(onConditionDataFlows, onConditionsToRewrite, tableMap, annotations)
		assert.NotNil(t, sm)
	})

	t.Run("GetTableMap", func(t *testing.T) {
		sm := NewSelectMetadata(onConditionDataFlows, onConditionsToRewrite, tableMap, annotations)
		val, exp := sm.GetTableMap()
		assert.Equal(t, val, tableMap)
		assert.Equal(t, exp, true)
	})

	t.Run("GetAnnotations", func(t *testing.T) {
		sm := NewSelectMetadata(onConditionDataFlows, onConditionsToRewrite, tableMap, annotations)
		val, exp := sm.GetAnnotations()
		assert.Equal(t, val, annotations)
		assert.Equal(t, exp, true)
	})

	t.Run("GetOnConditionsToRewrite", func(t *testing.T) {
		sm := NewSelectMetadata(onConditionDataFlows, onConditionsToRewrite, tableMap, annotations)
		val := sm.GetOnConditionsToRewrite()
		assert.Equal(t, val, onConditionsToRewrite)
	})

	t.Run("GetOnConditionDataFlows", func(t *testing.T) {
		sm := NewSelectMetadata(onConditionDataFlows, onConditionsToRewrite, tableMap, annotations)
		val, exp := sm.GetOnConditionDataFlows()
		assert.Equal(t, val, onConditionDataFlows)
		assert.Equal(t, exp, false)
	})
}
