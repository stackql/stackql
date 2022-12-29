package primitivebuilder

import (
	"fmt"
	"strconv"

	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/httpmiddleware"
	"github.com/stackql/stackql/internal/stackql/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"github.com/stackql/stackql/internal/stackql/util"
	"vitess.io/vitess/go/vt/sqlparser"
)

type Delete struct {
	graph             primitivegraph.PrimitiveGraph
	handlerCtx        handler.HandlerContext
	drmCfg            drm.DRMConfig
	root              primitivegraph.PrimitiveNode
	tbl               tablemetadata.ExtendedTableMetadata
	node              sqlparser.SQLNode
	commentDirectives sqlparser.CommentDirectives
	isAwait           bool
}

func NewDelete(
	graph primitivegraph.PrimitiveGraph,
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
	ex := func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
		var target map[string]interface{}
		keys := make(map[string]map[string]interface{})
		httpArmoury, err := tbl.GetHttpArmoury()
		if err != nil {
			return util.PrepareResultSet(internaldto.NewPrepareResultSetDTO(nil, nil, nil, nil, err, nil))
		}
		for _, req := range httpArmoury.GetRequestParams() {
			response, apiErr := httpmiddleware.HttpApiCallFromRequest(handlerCtx.Clone(), prov, m, req.GetRequest())
			if apiErr != nil {
				return util.PrepareResultSet(internaldto.NewPrepareResultSetDTO(nil, nil, nil, nil, apiErr, nil))
			}
			target, err = m.DeprecatedProcessResponse(response)
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
				))
			}
			logging.GetLogger().Infoln(fmt.Sprintf("target = %v", target))
			items, ok := target[prov.GetDefaultKeyForDeleteItems()]
			if ok {
				iArr, ok := items.([]interface{})
				if ok && len(iArr) > 0 {
					for i := range iArr {
						item, ok := iArr[i].(map[string]interface{})
						if ok {
							keys[strconv.Itoa(i)] = item
						}
					}
				}
			}
		}
		msgs := internaldto.BackendMessages{}
		if err == nil {
			msgs.WorkingMessages = generateSuccessMessagesFromHeirarchy(tbl)
		}
		return generateResultIfNeededfunc(keys, target, &msgs, err, false)
	}
	deletePrimitive := primitive.NewHTTPRestPrimitive(
		prov,
		ex,
		nil,
		nil,
	)
	if ss.isAwait {
		deletePrimitive, err = composeAsyncMonitor(handlerCtx, deletePrimitive, tbl, nil)
	}
	if err != nil {
		return err
	}

	graph := ss.graph
	insertNode := graph.CreatePrimitiveNode(deletePrimitive)
	ss.root = insertNode

	return nil
}
