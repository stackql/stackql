package primitivebuilder

import (
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/asyncmonitor"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
)

func composeAsyncMonitor(
	handlerCtx handler.HandlerContext,
	precursor primitive.IPrimitive,
	meta tablemetadata.ExtendedTableMetadata,
	commentDirectives sqlparser.CommentDirectives,
) (primitive.IPrimitive, error) {
	prov, err := meta.GetProvider()
	if err != nil {
		return nil, err
	}
	asm, err := asyncmonitor.NewAsyncMonitor(handlerCtx, prov)
	if err != nil {
		return nil, err
	}
	// might be pointless
	_, err = handlerCtx.GetAuthContext(prov.GetProviderString())
	if err != nil {
		return nil, err
	}
	//
	pl := internaldto.NewBasicPrimitiveContext(
		handlerCtx.GetAuthContext,
		handlerCtx.GetOutfile(),
		handlerCtx.GetOutErrFile(),
	)
	primitive, err := asm.GetMonitorPrimitive(meta.GetHeirarchyObjects(), precursor, pl, commentDirectives)
	if err != nil {
		return nil, err
	}
	return primitive, err
}
