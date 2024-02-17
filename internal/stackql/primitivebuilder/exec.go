package primitivebuilder

import (
	"fmt"
	"strconv"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
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

type Exec struct {
	graph         primitivegraph.PrimitiveGraphHolder
	handlerCtx    handler.HandlerContext
	drmCfg        drm.Config
	root          primitivegraph.PrimitiveNode
	tbl           tablemetadata.ExtendedTableMetadata
	isAwait       bool
	isShowResults bool
}

func NewExec(
	graph primitivegraph.PrimitiveGraphHolder,
	handlerCtx handler.HandlerContext,
	node sqlparser.SQLNode, //nolint:revive // future proofing
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

//nolint:gocognit // probably a headache no matter which way you slice it
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
		var columnOrder []string
		keys := make(map[string]map[string]interface{})
		httpArmoury, httpArmouryErr := tbl.GetHTTPArmoury()
		if httpArmouryErr != nil {
			return internaldto.NewErroneousExecutorOutput(httpArmouryErr)
		}
		for i, req := range httpArmoury.GetRequestParams() {
			response, apiErr := httpmiddleware.HTTPApiCallFromRequest(handlerCtx.Clone(), prov, m, req.GetRequest())
			if apiErr != nil {
				return util.PrepareResultSet(internaldto.NewPrepareResultSetDTO(nil, nil, nil, nil, apiErr, nil,
					handlerCtx.GetTypingConfig(),
				))
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
					handlerCtx.GetTypingConfig(),
				))
			}
			logging.GetLogger().Infoln(fmt.Sprintf("target = %v", target))
			items, ok := target[tbl.LookupSelectItemsKey()]
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
			} else {
				keys[fmt.Sprintf("%d", i)] = target
			}
			// optional data return pattern to be included in grammar subsequently
			// return util.PrepareResultSet(internaldto.NewPrepareResultSetDTO(nil, keys, columnOrder, nil, err, nil))
			logging.GetLogger().Debugln(fmt.Sprintf("keys = %v", keys))
			logging.GetLogger().Debugln(fmt.Sprintf("columnOrder = %v", columnOrder))
		}
		return generateResultIfNeededfunc(
			keys, target,
			internaldto.NewBackendMessages(
				generateSuccessMessagesFromHeirarchy(tbl, ss.isAwait),
			),
			err, ss.isShowResults,
			ss.handlerCtx.GetTypingConfig(),
		)
	}
	execPrimitive := primitive.NewHTTPRestPrimitive(
		prov,
		ex,
		nil,
		nil,
		primitive_context.NewPrimitiveContext(),
	)
	if !ss.isAwait {
		ss.graph.CreatePrimitiveNode(execPrimitive)
		return nil
	}
	pr, err := composeAsyncMonitor(handlerCtx, execPrimitive, prov, m, nil)
	if err != nil {
		return err
	}
	ss.graph.CreatePrimitiveNode(pr)
	return nil
}
