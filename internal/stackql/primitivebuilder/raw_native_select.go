package primitivebuilder

import (
	"fmt"

	"github.com/stackql/any-sdk/pkg/logging"
	"github.com/stackql/stackql/internal/stackql/data_staging/input_data_staging"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
)

type RawNativeSelect struct {
	graph       primitivegraph.PrimitiveGraphHolder
	handlerCtx  handler.HandlerContext
	txnCtrlCtr  internaldto.TxnControlCounters
	root        primitivegraph.PrimitiveNode
	nativeQuery string
}

func NewRawNativeSelect(
	graph primitivegraph.PrimitiveGraphHolder,
	handlerCtx handler.HandlerContext,
	txnCtrlCtr internaldto.TxnControlCounters,
	nativeQuery string,
) Builder {
	return &RawNativeSelect{
		graph:       graph,
		handlerCtx:  handlerCtx,
		txnCtrlCtr:  txnCtrlCtr,
		nativeQuery: nativeQuery,
	}
}

func (ss *RawNativeSelect) GetRoot() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *RawNativeSelect) GetTail() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *RawNativeSelect) Build() error {
	//nolint:revive // no big deal
	selectEx := func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
		// select phase
		logging.GetLogger().Infoln(fmt.Sprintf("running native query: '''%s''' ", ss.nativeQuery))

		rows, err := ss.handlerCtx.GetSQLEngine().Query(ss.nativeQuery)

		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
		defer rows.Close()

		preparator := input_data_staging.NewNaiveNativeResultSetPreparator(
			rows,
			ss.handlerCtx.GetDrmConfig(),
			ss.handlerCtx.GetTypingConfig(),
			nil)

		rv := preparator.PrepareNativeResultSet()
		return rv
	}

	graph := ss.graph
	selectNode := graph.CreatePrimitiveNode(primitive.NewLocalPrimitive(selectEx))
	ss.root = selectNode

	return nil
}
