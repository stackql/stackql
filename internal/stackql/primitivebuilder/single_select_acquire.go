package primitivebuilder

import (
	"github.com/stackql/any-sdk/pkg/streaming"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/builder_input"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/tableinsertioncontainer"
)

func NewSingleSelectAcquire(
	graphHolder primitivegraph.PrimitiveGraphHolder,
	handlerCtx handler.HandlerContext,
	insertionContainer tableinsertioncontainer.TableInsertionContainer,
	insertCtx drm.PreparedStatementCtx,
	rowSort func(map[string]map[string]interface{}) []string,
	stream streaming.MapStream,
	bldrInput builder_input.BuilderInput,
) Builder {
	isAwait := bldrInput.IsAwait()
	tableMeta := insertionContainer.GetTableMetadata()
	_, isGraphQL := tableMeta.GetGraphQL()
	if isGraphQL {
		return newGraphQLSingleSelectAcquire(
			graphHolder,
			handlerCtx,
			tableMeta,
			insertCtx,
			insertionContainer,
			rowSort,
			stream,
			bldrInput,
		)
	}
	return newMonoValentBuilder(
		graphHolder,
		handlerCtx,
		tableMeta,
		insertCtx,
		insertionContainer,
		rowSort,
		stream,
		false,
		false,
		isAwait,
		bldrInput,
	)
}
