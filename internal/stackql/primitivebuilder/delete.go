package primitivebuilder

import (
	"github.com/stackql/any-sdk/pkg/streaming"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/asynccompose"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/execution"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/builder_input"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/primitive_context"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/tableinsertioncontainer"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
)

type Delete struct {
	graph             primitivegraph.PrimitiveGraphHolder
	handlerCtx        handler.HandlerContext
	drmCfg            drm.Config
	root              primitivegraph.PrimitiveNode
	tail              primitivegraph.PrimitiveNode
	tbl               tablemetadata.ExtendedTableMetadata
	node              sqlparser.SQLNode
	commentDirectives sqlparser.CommentDirectives
	isAwait           bool
	insertCtx         drm.PreparedStatementCtx
	selectCtx         drm.PreparedStatementCtx
	bldrInput         builder_input.BuilderInput
}

func NewDelete(
	graph primitivegraph.PrimitiveGraphHolder,
	handlerCtx handler.HandlerContext,
	insertCtx drm.PreparedStatementCtx,
	selectCtx drm.PreparedStatementCtx,
	node sqlparser.SQLNode,
	tbl tablemetadata.ExtendedTableMetadata,
	commentDirectives sqlparser.CommentDirectives,
	isAwait bool,
	bldrInput builder_input.BuilderInput,
) Builder {
	return &Delete{
		graph:             graph,
		handlerCtx:        handlerCtx,
		drmCfg:            handlerCtx.GetDrmConfig(),
		bldrInput:         bldrInput,
		tbl:               tbl,
		node:              node,
		commentDirectives: commentDirectives,
		isAwait:           isAwait,
		insertCtx:         insertCtx,
		selectCtx:         selectCtx,
	}
}

func (ss *Delete) GetRoot() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *Delete) GetTail() primitivegraph.PrimitiveNode {
	return ss.tail
}

func (ss *Delete) Build() error {
	tbl := ss.tbl
	handlerCtx := ss.handlerCtx
	prov, err := tbl.GetProvider()
	if err != nil {
		return err
	}
	method, methodErr := tbl.GetMethod()
	if methodErr != nil {
		return methodErr
	}
	insertContainer, err := tableinsertioncontainer.NewTableInsertionContainer(
		tbl,
		ss.handlerCtx.GetSQLEngine(),
		handlerCtx.GetTxnCounterMgr(),
	)
	mvb := execution.NewMonoValentExecutorFactory(
		ss.graph,
		handlerCtx,
		tbl,
		ss.insertCtx,
		insertContainer,
		nil,
		streaming.NewNopMapStream(),
		!ss.isAwait,
		true,
		ss.isAwait,
		ss.bldrInput,
	)
	ex, exErr := mvb.GetExecutor()
	if exErr != nil {
		return exErr
	}
	deletePrimitive := primitive.NewGenericPrimitive(
		ex,
		nil,
		nil,
		primitive_context.NewPrimitiveContext(),
	)
	if ss.isAwait {
		deletePrimitive, err = asynccompose.ComposeAsyncMonitor(
			handlerCtx, deletePrimitive, prov, method, nil, false, nil, nil) // isReturning hardcoded to false for now
	}
	if err != nil {
		return err
	}

	graph := ss.graph
	insertNode := graph.CreatePrimitiveNode(deletePrimitive)
	ss.root = insertNode
	ss.tail = insertNode
	if ss.selectCtx != nil {
		selectionBldr := NewSingleSelect(
			ss.graph,
			handlerCtx,
			ss.selectCtx,
			[]tableinsertioncontainer.TableInsertionContainer{insertContainer},
			nil,
			streaming.NewNopMapStream(),
		)
		err = selectionBldr.Build()
		if err != nil {
			return err
		}
		ss.graph.NewDependency(ss.tail, selectionBldr.GetRoot(), 1.0)
		ss.tail = selectionBldr.GetTail()
	}
	return nil
}
