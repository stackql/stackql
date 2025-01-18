package primitivebuilder

import (
	"errors"
	"fmt"
	"io"
	"strconv"

	"github.com/stackql/any-sdk/pkg/logging"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/nativedb"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/util"
)

type NativeSelect struct {
	graph       primitivegraph.PrimitiveGraphHolder
	handlerCtx  handler.HandlerContext
	drmCfg      drm.Config
	selectQuery nativedb.Select
	root        primitivegraph.PrimitiveNode
}

func NewNativeSelect(
	graph primitivegraph.PrimitiveGraphHolder,
	handlerCtx handler.HandlerContext,
	selectQuery nativedb.Select,
) Builder {
	return &NativeSelect{
		graph:       graph,
		handlerCtx:  handlerCtx,
		drmCfg:      handlerCtx.GetDrmConfig(),
		selectQuery: selectQuery,
	}
}

func (ss *NativeSelect) GetRoot() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *NativeSelect) GetTail() primitivegraph.PrimitiveNode {
	return ss.root
}

//nolint:gocognit,revive // probably a headache no matter which way you slice it
func (ss *NativeSelect) Build() error {
	selectEx := func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
		// select phase
		logging.GetLogger().Infoln(fmt.Sprintf("running empty select with columns: %v", ss.selectQuery))

		var colz []string
		for _, col := range ss.selectQuery.GetColumns() {
			colz = append(colz, col.GetName())
		}
		rowStream := ss.selectQuery.GetRows()
		rowMap := make(map[string]map[string]interface{})
		if rowStream != nil {
			i := 0
			for {
				rows, err := rowStream.Read()
				if err != nil && !errors.Is(err, io.EOF) {
					return internaldto.NewErroneousExecutorOutput(err)
				}
				for _, row := range rows {
					rowMap[strconv.Itoa(i)] = row
					i++
				}
				if errors.Is(err, io.EOF) {
					break
				}
			}
		}
		rv := util.PrepareResultSet(internaldto.NewPrepareResultSetDTO(nil, rowMap, colz, nil, nil, nil,
			ss.handlerCtx.GetTypingConfig(),
		))
		if len(rowMap) > 0 {
			return rv
		}
		return util.EmptyProtectResultSet(
			rv,
			colz,
			ss.handlerCtx.GetTypingConfig(),
		)
	}
	graph := ss.graph
	selectNode := graph.CreatePrimitiveNode(primitive.NewLocalPrimitive(selectEx))
	ss.root = selectNode

	return nil
}
