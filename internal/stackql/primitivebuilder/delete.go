package primitivebuilder

import (
	"fmt"
	"strconv"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/acid/binlog"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/httpmiddleware"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/primitive_context"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"github.com/stackql/stackql/internal/stackql/util"
)

type Delete struct {
	graph             primitivegraph.PrimitiveGraphHolder
	handlerCtx        handler.HandlerContext
	drmCfg            drm.Config
	root              primitivegraph.PrimitiveNode
	tbl               tablemetadata.ExtendedTableMetadata
	node              sqlparser.SQLNode
	commentDirectives sqlparser.CommentDirectives
	isAwait           bool
}

func NewDelete(
	graph primitivegraph.PrimitiveGraphHolder,
	handlerCtx handler.HandlerContext,
	node sqlparser.SQLNode,
	tbl tablemetadata.ExtendedTableMetadata,
	commentDirectives sqlparser.CommentDirectives,
	isAwait bool,
) Builder {
	return &Delete{
		graph:             graph,
		handlerCtx:        handlerCtx,
		drmCfg:            handlerCtx.GetDrmConfig(),
		tbl:               tbl,
		node:              node,
		commentDirectives: commentDirectives,
		isAwait:           isAwait,
	}
}

func (ss *Delete) GetRoot() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *Delete) GetTail() primitivegraph.PrimitiveNode {
	return ss.root
}

//nolint:gocognit,funlen // probably a headache no matter which way you slice it
func (ss *Delete) Build() error {
	tbl := ss.tbl
	handlerCtx := ss.handlerCtx
	prov, err := tbl.GetProvider()
	if err != nil {
		return err
	}
	m, err := tbl.GetMethod()
	if err != nil {
		return err
	}
	tableName, _ := tbl.GetTableName()
	ex := func(_ primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
		var target map[string]interface{}
		keys := make(map[string]map[string]interface{})
		httpArmoury, httpErr := tbl.GetHTTPArmoury()
		if httpErr != nil {
			return util.PrepareResultSet(internaldto.NewPrepareResultSetDTO(nil, nil, nil, nil, httpErr, nil,
				ss.handlerCtx.GetTypingConfig(),
			))
		}
		for _, req := range httpArmoury.GetRequestParams() {
			response, apiErr := httpmiddleware.HTTPApiCallFromRequest(handlerCtx.Clone(), prov, m, req.GetRequest())
			if apiErr != nil {
				return util.PrepareResultSet(internaldto.NewPrepareResultSetDTO(nil, nil, nil, nil, apiErr, nil,
					ss.handlerCtx.GetTypingConfig(),
				))
			}
			target, err = m.DeprecatedProcessResponse(response)
			if response.StatusCode < 300 && len(target) < 1 {
				return util.PrepareResultSet(internaldto.NewPrepareResultSetDTO(
					nil,
					nil,
					nil,
					nil,
					nil,
					internaldto.NewBackendMessages(
						generateSuccessMessagesFromHeirarchy(tbl, ss.isAwait),
					),
					ss.handlerCtx.GetTypingConfig(),
				)).WithUndoLog(
					binlog.NewSimpleLogEntry(
						nil,
						[]string{
							"Undo the delete on " + tableName,
						},
					),
				)
			}
			handlerCtx.LogHTTPResponseMap(target)

			logging.GetLogger().Infoln(fmt.Sprintf("DeleteExecutor() target = %v", target))
			if err != nil {
				return util.PrepareResultSet(internaldto.NewPrepareResultSetDTO(
					nil,
					nil,
					nil,
					nil,
					err,
					nil,
					ss.handlerCtx.GetTypingConfig(),
				))
			}
			logging.GetLogger().Infoln(fmt.Sprintf("target = %v", target))
			items, ok := target[prov.GetDefaultKeyForDeleteItems()]
			if ok {
				iArr, iOk := items.([]interface{})
				if iOk && len(iArr) > 0 {
					for i := range iArr {
						item, itemOk := iArr[i].(map[string]interface{})
						if itemOk {
							keys[strconv.Itoa(i)] = item
						}
					}
				}
			}
		}
		return generateResultIfNeededfunc(
			keys,
			target,
			internaldto.NewBackendMessages(
				generateSuccessMessagesFromHeirarchy(tbl, ss.isAwait)),
			err,
			false,
			ss.handlerCtx.GetTypingConfig(),
		)
	}
	deletePrimitive := primitive.NewHTTPRestPrimitive(
		prov,
		ex,
		nil,
		nil,
		primitive_context.NewPrimitiveContext(),
	)
	if ss.isAwait {
		deletePrimitive, err = composeAsyncMonitor(handlerCtx, deletePrimitive, prov, m, nil)
	}
	if err != nil {
		return err
	}

	graph := ss.graph
	insertNode := graph.CreatePrimitiveNode(deletePrimitive)
	ss.root = insertNode

	return nil
}
