package asynccompose

import (
	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/provider"
)

func ComposeAsyncMonitor(
	handlerCtx handler.HandlerContext,
	precursor primitive.IPrimitive,
	prov provider.IProvider,
	method anysdk.OperationStore,
	commentDirectives sqlparser.CommentDirectives,
	isReturning bool,
	insertCtx drm.PreparedStatementCtx,
	drmCfg drm.Config,
) (primitive.IPrimitive, error) {
	asm, err := NewAsyncMonitor(handlerCtx, prov, method, isReturning)
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
	primitive, err := asm.GetMonitorPrimitive(
		prov, method, precursor, pl, commentDirectives, isReturning, insertCtx, drmCfg)
	if err != nil {
		return nil, err
	}
	return primitive, err
}
