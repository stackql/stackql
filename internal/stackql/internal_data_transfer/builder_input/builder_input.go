package builder_input //nolint:revive,stylecheck // permissable deviation from norm

import (
	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/astanalysis/annotatedast"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/provider"
	"github.com/stackql/stackql/internal/stackql/streaming"
	"github.com/stackql/stackql/internal/stackql/streaming/http_preparator_stream.go"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
)

var (
	_ BuilderInput = &builderInput{}
)

type BuilderInput interface {
	GetGraphHolder() (primitivegraph.PrimitiveGraphHolder, bool)
	GetHandlerContext() (handler.HandlerContext, bool)
	GetParamMap() (map[int]map[string]interface{}, bool)
	GetParamMapStream() (streaming.MapStream, bool)
	GetTableMetadata() (tablemetadata.ExtendedTableMetadata, bool)
	GetDependencyNode() (primitivegraph.PrimitiveNode, bool)
	GetCommentDirectives() (sqlparser.CommentDirectives, bool)
	GetParserNode() (sqlparser.SQLNode, bool)
	GetProvider() (provider.IProvider, bool)
	SetProvider(provider.IProvider)
	GetOperationStore() (openapistackql.OperationStore, bool)
	SetOperationStore(op openapistackql.OperationStore)
	IsAwait() bool
	GetVerb() string
	GetInputAlias() string
	IsUndo() bool
	SetInputAlias(inputAlias string)
	SetIsAwait(isAwait bool)
	SetCommentDirectives(commentDirectives sqlparser.CommentDirectives)
	SetIsUndo(isUndo bool)
	SetDependencyNode(dependencyNode primitivegraph.PrimitiveNode)
	SetParserNode(node sqlparser.SQLNode)
	SetParamMap(paramMap map[int]map[string]interface{})
	GetAnnotatedAST() (annotatedast.AnnotatedAst, bool)
	SetAnnotatedAST(annotatedAST annotatedast.AnnotatedAst)
	SetParamMapStream(streaming.MapStream)
	SetVerb(verb string)
	Clone() BuilderInput
	GetHTTPPreparatorStream() (http_preparator_stream.HttpPreparatorStream, bool)
	SetHTTPPreparatorStream(prepStream http_preparator_stream.HttpPreparatorStream)
}

type builderInput struct {
	graphHolder       primitivegraph.PrimitiveGraphHolder
	handlerCtx        handler.HandlerContext
	paramMap          map[int]map[string]interface{}
	tbl               tablemetadata.ExtendedTableMetadata
	dependencyNode    primitivegraph.PrimitiveNode
	commentDirectives sqlparser.CommentDirectives
	isAwait           bool
	verb              string
	inputAlias        string
	isUndo            bool
	node              sqlparser.SQLNode
	paramMapStream    streaming.MapStream
	httpPrepStream    http_preparator_stream.HttpPreparatorStream
	op                openapistackql.OperationStore
	prov              provider.IProvider
	annotatedAst      annotatedast.AnnotatedAst
}

func NewBuilderInput(
	graphHolder primitivegraph.PrimitiveGraphHolder,
	handlerCtx handler.HandlerContext,
	tbl tablemetadata.ExtendedTableMetadata,
) BuilderInput {
	return &builderInput{
		graphHolder:       graphHolder,
		handlerCtx:        handlerCtx,
		tbl:               tbl,
		commentDirectives: sqlparser.CommentDirectives{},
		inputAlias:        "", // this default is explicit for emphasisis
	}
}

func (bi *builderInput) SetAnnotatedAST(annotatedAST annotatedast.AnnotatedAst) {
	bi.annotatedAst = annotatedAST
}

func (bi *builderInput) GetAnnotatedAST() (annotatedast.AnnotatedAst, bool) {
	return bi.annotatedAst, bi.annotatedAst != nil
}

func (bi *builderInput) GetProvider() (provider.IProvider, bool) {
	return bi.prov, bi.prov != nil
}

func (bi *builderInput) SetProvider(prov provider.IProvider) {
	bi.prov = prov
}

func (bi *builderInput) GetOperationStore() (openapistackql.OperationStore, bool) {
	return bi.op, bi.op != nil
}

func (bi *builderInput) SetOperationStore(op openapistackql.OperationStore) {
	bi.op = op
}

func (bi *builderInput) GetParamMapStream() (streaming.MapStream, bool) {
	return bi.paramMapStream, bi.paramMapStream != nil
}

func (bi *builderInput) GetHTTPPreparatorStream() (http_preparator_stream.HttpPreparatorStream, bool) {
	return bi.httpPrepStream, bi.httpPrepStream != nil
}

func (bi *builderInput) SetHTTPPreparatorStream(prepStream http_preparator_stream.HttpPreparatorStream) {
	bi.httpPrepStream = prepStream
}

func (bi *builderInput) SetParamMapStream(s streaming.MapStream) {
	bi.paramMapStream = s
}

func (bi *builderInput) GetGraphHolder() (primitivegraph.PrimitiveGraphHolder, bool) {
	return bi.graphHolder, bi.graphHolder != nil
}

func (bi *builderInput) GetParserNode() (sqlparser.SQLNode, bool) {
	return bi.node, bi.node != nil
}

func (bi *builderInput) SetParserNode(node sqlparser.SQLNode) {
	bi.node = node
}

func (bi *builderInput) GetHandlerContext() (handler.HandlerContext, bool) {
	return bi.handlerCtx, bi.handlerCtx != nil
}

func (bi *builderInput) GetParamMap() (map[int]map[string]interface{}, bool) {
	return bi.paramMap, bi.paramMap != nil
}

func (bi *builderInput) GetTableMetadata() (tablemetadata.ExtendedTableMetadata, bool) {
	return bi.tbl, bi.tbl != nil
}

func (bi *builderInput) GetDependencyNode() (primitivegraph.PrimitiveNode, bool) {
	return bi.dependencyNode, bi.dependencyNode != nil
}

func (bi *builderInput) GetCommentDirectives() (sqlparser.CommentDirectives, bool) {
	return bi.commentDirectives, len(bi.commentDirectives) > 0
}

func (bi *builderInput) IsAwait() bool {
	return bi.isAwait
}

func (bi *builderInput) GetVerb() string {
	return bi.verb
}

func (bi *builderInput) GetInputAlias() string {
	return bi.inputAlias
}

func (bi *builderInput) IsUndo() bool {
	return bi.isUndo
}

func (bi *builderInput) SetGraphHolder(graphHolder primitivegraph.PrimitiveGraphHolder) {
	bi.graphHolder = graphHolder
}

func (bi *builderInput) SetHandlerContext(handlerCtx handler.HandlerContext) {
	bi.handlerCtx = handlerCtx
}

func (bi *builderInput) SetParamMap(paramMap map[int]map[string]interface{}) {
	bi.paramMap = paramMap
}

func (bi *builderInput) SetTableMetadata(tbl tablemetadata.ExtendedTableMetadata) {
	bi.tbl = tbl
}

func (bi *builderInput) SetDependencyNode(dependencyNode primitivegraph.PrimitiveNode) {
	bi.dependencyNode = dependencyNode
}

func (bi *builderInput) SetCommentDirectives(commentDirectives sqlparser.CommentDirectives) {
	bi.commentDirectives = commentDirectives
}

func (bi *builderInput) SetIsAwait(isAwait bool) {
	bi.isAwait = isAwait
}

func (bi *builderInput) SetVerb(verb string) {
	bi.verb = verb
}

func (bi *builderInput) SetInputAlias(inputAlias string) {
	bi.inputAlias = inputAlias
}

func (bi *builderInput) SetIsUndo(isUndo bool) {
	bi.isUndo = isUndo
}

func (bi *builderInput) Clone() BuilderInput {
	return &builderInput{
		graphHolder:       bi.graphHolder,
		handlerCtx:        bi.handlerCtx,
		paramMap:          bi.paramMap,
		tbl:               bi.tbl,
		node:              bi.node,
		dependencyNode:    bi.dependencyNode,
		commentDirectives: bi.commentDirectives,
		isAwait:           bi.isAwait,
		verb:              bi.verb,
		inputAlias:        bi.inputAlias,
		isUndo:            bi.isUndo,
	}
}
