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

type Exec struct {
	graph         *primitivegraph.PrimitiveGraph
	handlerCtx    handler.HandlerContext
	drmCfg        drm.DRMConfig
	root          primitivegraph.PrimitiveNode
	tbl           tablemetadata.ExtendedTableMetadata
	isAwait       bool
	isShowResults bool
}

func NewExec(
	graph *primitivegraph.PrimitiveGraph,
	handlerCtx handler.HandlerContext,
	node sqlparser.SQLNode,
	tbl tablemetadata.ExtendedTableMetadata,
	isAwait bool,
	isShowResults bool,
) Builder {
	return &Exec{
		graph:         graph,
		handlerCtx:    handlerCtx,
		drmCfg:        handlerCtx.GetDrmConfig(),
		tbl:           tbl,
		isAwait:       isAwait,
		isShowResults: isShowResults,
	}
}

func (ss *Exec) GetRoot() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *Exec) GetTail() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *Exec) Build() error {

	handlerCtx := ss.handlerCtx
	tbl := ss.tbl
	prov, err := tbl.GetProvider()
	if err != nil {
		return err
	}
	m, err := tbl.GetMethod()
	if err != nil {
		return err
	}
	var target map[string]interface{}
	ex := func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
		var err error
		var columnOrder []string
		keys := make(map[string]map[string]interface{})
		httpArmoury, err := tbl.GetHttpArmoury()
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		for i, req := range httpArmoury.GetRequestParams() {
			response, apiErr := httpmiddleware.HttpApiCallFromRequest(handlerCtx.Clone(), prov, m, req.GetRequest())
			if apiErr != nil {
				return util.PrepareResultSet(internaldto.NewPrepareResultSetDTO(nil, nil, nil, nil, apiErr, nil))
			}
			target, err = m.DeprecatedProcessResponse(response)
			handlerCtx.LogHTTPResponseMap(target)
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
			items, ok := target[tbl.LookupSelectItemsKey()]
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
			} else {
				keys[fmt.Sprintf("%d", i)] = target
			}
			// optional data return pattern to be included in grammar subsequently
			// return util.PrepareResultSet(internaldto.NewPrepareResultSetDTO(nil, keys, columnOrder, nil, err, nil))
			logging.GetLogger().Debugln(fmt.Sprintf("keys = %v", keys))
			logging.GetLogger().Debugln(fmt.Sprintf("columnOrder = %v", columnOrder))
		}
		msgs := internaldto.BackendMessages{}
		if err == nil {
			msgs.WorkingMessages = generateSuccessMessagesFromHeirarchy(tbl)
		}
		return generateResultIfNeededfunc(keys, target, &msgs, err, ss.isShowResults)
	}
	execPrimitive := primitive.NewHTTPRestPrimitive(
		prov,
		ex,
		nil,
		nil,
	)
	if !ss.isAwait {
		ss.graph.CreatePrimitiveNode(execPrimitive)
		return nil
	}
	pr, err := composeAsyncMonitor(handlerCtx, execPrimitive, tbl, nil)
	if err != nil {
		return err
	}
	ss.graph.CreatePrimitiveNode(pr)
	return nil
}
