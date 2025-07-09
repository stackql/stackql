package primitivebuilder

import (
	"fmt"

	"github.com/stackql/any-sdk/pkg/streaming"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/execution"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/primitive_context"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/tableinsertioncontainer"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
)

// monoValentBuilder implements the Builder interface
// and represents the action of acquiring data from an endpoint
// and then persisting that data into a table.
// This data would then subsequently be queried by later execution phases.
type monoValentBuilder struct {
	graphHolder                primitivegraph.PrimitiveGraphHolder
	handlerCtx                 handler.HandlerContext
	tableMeta                  tablemetadata.ExtendedTableMetadata
	drmCfg                     drm.Config
	insertPreparedStatementCtx drm.PreparedStatementCtx
	insertionContainer         tableinsertioncontainer.TableInsertionContainer
	txnCtrlCtr                 internaldto.TxnControlCounters
	rowSort                    func(map[string]map[string]interface{}) []string
	root                       primitivegraph.PrimitiveNode
	stream                     streaming.MapStream
	isReadOnly                 bool //nolint:unused // TODO: build out
	isAwait                    bool //nolint:unused // TODO: build out
	monoValentExecutorFactory  execution.MonoValentExecutorFactory
}

func newMonoValentBuilder(
	graphHolder primitivegraph.PrimitiveGraphHolder,
	handlerCtx handler.HandlerContext,
	tableMeta tablemetadata.ExtendedTableMetadata,
	insertCtx drm.PreparedStatementCtx,
	insertionContainer tableinsertioncontainer.TableInsertionContainer,
	rowSort func(map[string]map[string]interface{}) []string,
	stream streaming.MapStream,
	isSkipResponse bool,
	isMutation bool,
	isAwait bool,
) Builder {
	var tcc internaldto.TxnControlCounters
	if insertCtx != nil {
		tcc = insertCtx.GetGCCtrlCtrs()
	}
	if stream == nil {
		stream = streaming.NewNopMapStream()
	}
	return &monoValentBuilder{
		graphHolder:                graphHolder,
		handlerCtx:                 handlerCtx,
		tableMeta:                  tableMeta,
		rowSort:                    rowSort,
		drmCfg:                     handlerCtx.GetDrmConfig(),
		insertPreparedStatementCtx: insertCtx,
		insertionContainer:         insertionContainer,
		txnCtrlCtr:                 tcc,
		stream:                     stream,
		monoValentExecutorFactory: execution.NewMonoValentExecutorFactory(
			graphHolder,
			handlerCtx,
			tableMeta,
			insertCtx,
			insertionContainer,
			rowSort,
			stream,
			isSkipResponse,
			isMutation,
			isAwait,
		),
	}
}

func (mv *monoValentBuilder) GetRoot() primitivegraph.PrimitiveNode {
	return mv.root
}

func (mv *monoValentBuilder) GetTail() primitivegraph.PrimitiveNode {
	return mv.root
}

func (mv *monoValentBuilder) Build() error {
	tableName, err := mv.tableMeta.GetTableName()
	if err != nil {
		return err
	}
	ex, err := mv.monoValentExecutorFactory.GetExecutor()

	if err != nil {
		return err
	}

	prep := func() drm.PreparedStatementCtx {
		return mv.insertPreparedStatementCtx
	}
	primitiveCtx := primitive_context.NewPrimitiveContext()
	primitiveCtx.SetIsReadOnly(true)
	insertPrim := primitive.NewGenericPrimitive(
		ex,
		prep,
		mv.txnCtrlCtr,
		primitiveCtx,
	).WithDebugName(fmt.Sprintf("insert_%s_%s", tableName, mv.tableMeta.GetAlias()))
	graphHolder := mv.graphHolder
	insertNode := graphHolder.CreatePrimitiveNode(insertPrim)
	mv.root = insertNode

	return nil
}
