package asynccompose

import (
	"fmt"
	"strings"

	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/stackql/internal/stackql/acid/binlog"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/execution"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/provider"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

type IAsyncMonitor interface {
	GetMonitorPrimitive(
		prov provider.IProvider,
		op anysdk.OperationStore,
		precursor primitive.IPrimitive,
		initialCtx primitive.IPrimitiveCtx,
		comments sqlparser.CommentDirectives,
		isReturning bool,
		insertCtx drm.PreparedStatementCtx,
		drmCfg drm.Config,
	) (primitive.IPrimitive, error)
}

//nolint:unused // TODO: refactor
type AsyncHTTPMonitorPrimitive struct {
	handlerCtx          handler.HandlerContext
	prov                provider.IProvider
	op                  anysdk.OperationStore
	initialCtx          primitive.IPrimitiveCtx
	precursor           primitive.IPrimitive
	executor            func(pc primitive.IPrimitiveCtx, initalBody interface{}) internaldto.ExecutorOutput
	elapsedSeconds      int
	pollIntervalSeconds int
	noStatus            bool
	id                  int64
	comments            sqlparser.CommentDirectives
}

func (pr *AsyncHTTPMonitorPrimitive) SetTxnID(_ int) {
}

func (pr *AsyncHTTPMonitorPrimitive) IsReadOnly() bool {
	return false
}

func (pr *AsyncHTTPMonitorPrimitive) GetRedoLog() (binlog.LogEntry, bool) {
	return nil, false
}

func (pr *AsyncHTTPMonitorPrimitive) GetUndoLog() (binlog.LogEntry, bool) {
	return nil, false
}

func (pr *AsyncHTTPMonitorPrimitive) WithDebugName(_ string) primitive.IPrimitive {
	return pr
}

func (pr *AsyncHTTPMonitorPrimitive) SetUndoLog(_ binlog.LogEntry) {
}

func (pr *AsyncHTTPMonitorPrimitive) SetRedoLog(_ binlog.LogEntry) {
}

func (pr *AsyncHTTPMonitorPrimitive) IncidentData(fromID int64, input internaldto.ExecutorOutput) error {
	return pr.precursor.IncidentData(fromID, input)
}

func (pr *AsyncHTTPMonitorPrimitive) SetInputAlias(alias string, id int64) error {
	return pr.precursor.SetInputAlias(alias, id)
}

func (pr *AsyncHTTPMonitorPrimitive) Optimise() error {
	return nil
}

func (pr *AsyncHTTPMonitorPrimitive) Execute(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
	if pr.executor != nil {
		if pc == nil {
			pc = pr.initialCtx
		}
		subPr := pr.precursor.Execute(pc)
		if subPr.GetError() != nil || pr.executor == nil {
			return subPr
		}
		prStr := pr.prov.GetProviderString()
		// seems pointless
		_, err := pr.initialCtx.GetAuthContext(prStr)
		if err != nil {
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, err)
		}
		//
		asyP := internaldto.NewBasicPrimitiveContext(
			pr.initialCtx.GetAuthContext,
			pc.GetWriter(),
			pc.GetErrWriter(),
		)
		return pr.executor(asyP, subPr.GetOutputBody())
	}
	return internaldto.NewExecutorOutput(nil, nil, nil, nil, nil)
}

func (pr *AsyncHTTPMonitorPrimitive) ID() int64 {
	return pr.id
}

func (pr *AsyncHTTPMonitorPrimitive) GetInputFromAlias(string) (internaldto.ExecutorOutput, bool) {
	var rv internaldto.ExecutorOutput
	return rv, false
}

func (pr *AsyncHTTPMonitorPrimitive) SetExecutor(_ func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput) error {
	return fmt.Errorf("AsyncHTTPMonitorPrimitive does not support SetExecutor()")
}

func NewAsyncMonitor(
	handlerCtx handler.HandlerContext,
	prov provider.IProvider,
	op anysdk.OperationStore,
	isReturning bool,
) (IAsyncMonitor, error) {
	//nolint:gocritic //TODO: refactor
	switch prov.GetProviderString() {
	case "google":
		return newGoogleAsyncMonitor(handlerCtx, prov, op, prov.GetVersion(), isReturning)
	}
	return nil, fmt.Errorf(
		"async operation monitor for provider = '%s', api version = '%s' currently not supported",
		prov.GetProviderString(), prov.GetVersion())
}

func newGoogleAsyncMonitor(
	handlerCtx handler.HandlerContext,
	prov provider.IProvider,
	op anysdk.OperationStore,
	version string, //nolint:unparam // TODO: refactor
	isReturning bool, //nolint:unparam,revive // TODO: refactor
) (IAsyncMonitor, error) {
	//nolint:gocritic //TODO: refactor
	switch version {
	default:
		return &DefaultGoogleAsyncMonitor{
			handlerCtx: handlerCtx,
			prov:       prov,
			op:         op,
		}, nil
	}
}

type DefaultGoogleAsyncMonitor struct {
	handlerCtx handler.HandlerContext
	prov       provider.IProvider
	op         anysdk.OperationStore
}

func (gm *DefaultGoogleAsyncMonitor) GetMonitorPrimitive(
	prov provider.IProvider,
	op anysdk.OperationStore,
	precursor primitive.IPrimitive,
	initialCtx primitive.IPrimitiveCtx,
	comments sqlparser.CommentDirectives,
	isReturning bool,
	insertCtx drm.PreparedStatementCtx,
	drmCfg drm.Config,
) (primitive.IPrimitive, error) {
	//nolint:gocritic,staticcheck //TODO: refactor
	switch strings.ToLower(prov.GetVersion()) {
	default:
		return gm.getV1Monitor(prov, op, precursor, initialCtx, comments, isReturning, insertCtx, drmCfg)
	}
}

func (gm *DefaultGoogleAsyncMonitor) getV1Monitor(
	prov provider.IProvider,
	op anysdk.OperationStore,
	precursor primitive.IPrimitive,
	initialCtx primitive.IPrimitiveCtx,
	comments sqlparser.CommentDirectives,
	isReturning bool,
	insertCtx drm.PreparedStatementCtx,
	drmCfg drm.Config,
) (primitive.IPrimitive, error) {
	provider, providerErr := prov.GetProvider()
	if providerErr != nil {
		return nil, providerErr
	}
	ex, exPrepErr := execution.GetMonitorExecutor(
		gm.handlerCtx,
		provider,
		op,
		precursor,
		initialCtx,
		comments,
		isReturning,
		insertCtx,
		drmCfg,
	)
	if exPrepErr != nil {
		return nil, exPrepErr
	}
	return ex, nil
}
