package primitivebuilder

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
)

type SingleSelect struct {
	graph                      *primitivegraph.PrimitiveGraph
	handlerCtx                 *handler.HandlerContext
	drmCfg                     drm.DRMConfig
	selectPreparedStatementCtx *drm.PreparedStatementCtx
	txnCtrlCtr                 *dto.TxnControlCounters
	rowSort                    func(map[string]map[string]interface{}) []string
	root                       primitivegraph.PrimitiveNode
}

func NewSingleSelect(graph *primitivegraph.PrimitiveGraph, handlerCtx *handler.HandlerContext, selectCtx *drm.PreparedStatementCtx, rowSort func(map[string]map[string]interface{}) []string) Builder {
	return &SingleSelect{
		graph:                      graph,
		handlerCtx:                 handlerCtx,
		rowSort:                    rowSort,
		drmCfg:                     handlerCtx.DrmConfig,
		selectPreparedStatementCtx: selectCtx,
		txnCtrlCtr:                 selectCtx.GetGCCtrlCtrs(),
	}
}

func (ss *SingleSelect) GetRoot() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *SingleSelect) GetTail() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *SingleSelect) Build() error {

	selectEx := func(pc primitive.IPrimitiveCtx) dto.ExecutorOutput {

		// select phase
		log.Infoln(fmt.Sprintf("running select with control parameters: %v", ss.selectPreparedStatementCtx.GetGCCtrlCtrs()))

		return prepareGolangResult(ss.handlerCtx.SQLEngine, ss.handlerCtx.OutErrFile, drm.NewPreparedStatementParameterized(ss.selectPreparedStatementCtx, nil, true), ss.selectPreparedStatementCtx.GetNonControlColumns(), ss.drmCfg)
	}
	graph := ss.graph
	selectNode := graph.CreatePrimitiveNode(primitive.NewLocalPrimitive(selectEx))
	ss.root = selectNode

	return nil
}
