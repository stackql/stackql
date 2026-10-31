package querysubmit

import (
	"sync"

	"github.com/stackql/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/accessmanagement"
	"github.com/stackql/stackql/internal/stackql/acid/binlog"
	"github.com/stackql/stackql/internal/stackql/acid/txn_context"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/plan"
	"github.com/stackql/stackql/internal/stackql/planbuilder"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
)

var (
	_ QuerySubmitter = &basicQuerySubmitter{}
)

type QuerySubmitter interface {
	GetStatement() (sqlparser.Statement, bool)
	PrepareQuery(handlerCtx handler.HandlerContext) error
	SubmitQuery() internaldto.ExecutorOutput
	WithTransactionContext(transactionContext txn_context.ITransactionContext) QuerySubmitter
	IsReadOnly() bool
	// Get the redo log entry.
	GetRedoLog() (binlog.LogEntry, bool)
	// Get the undo log entry.
	GetUndoLog() (binlog.LogEntry, bool)
	GetPrimitiveGraphHolder() (primitivegraph.PrimitiveGraphHolder, bool)
}

func NewQuerySubmitter() QuerySubmitter {
	return &basicQuerySubmitter{}
}

type basicQuerySubmitter struct {
	queryPlan          plan.Plan
	undoPlan           plan.Plan
	undoHandlerCtx     handler.HandlerContext
	handlerCtx         handler.HandlerContext
	transactionContext txn_context.ITransactionContext
	cancel             func()
}

func (qs *basicQuerySubmitter) GetRedoLog() (binlog.LogEntry, bool) {
	if qs.queryPlan == nil {
		return nil, false
	}
	return qs.queryPlan.GetRedoLog()
}

func (qs *basicQuerySubmitter) GetPrimitiveGraphHolder() (primitivegraph.PrimitiveGraphHolder, bool) {
	return qs.queryPlan.GetInstructions(), true
}

func (qs *basicQuerySubmitter) GetUndoLog() (binlog.LogEntry, bool) {
	if qs.queryPlan == nil {
		return nil, false
	}
	return qs.queryPlan.GetUndoLog()
}

func (qs *basicQuerySubmitter) IsReadOnly() bool {
	if qs.queryPlan == nil {
		return true
	}
	return qs.queryPlan.IsReadOnly()
}

func (qs *basicQuerySubmitter) GetStatement() (sqlparser.Statement, bool) {
	if qs.queryPlan == nil {
		return nil, false
	}
	return qs.queryPlan.GetStatement()
}

func (qs *basicQuerySubmitter) WithTransactionContext(
	transactionContext txn_context.ITransactionContext,
) QuerySubmitter {
	qs.transactionContext = transactionContext
	return qs
}

func (qs *basicQuerySubmitter) PrepareQuery(handlerCtx handler.HandlerContext) error {
	if manager := handlerCtx.GetAccessManager(); manager != nil {
		caller, runCtx, cancel, err := manager.Establish(handlerCtx.GetExecutionContext())
		if err != nil {
			return err
		}
		qs.cancel = cancel
		handlerCtx.SetCaller(caller)
		handlerCtx.SetExecutionContext(runCtx)
		statement, err := sqlparser.Parse(handlerCtx.GetQuery())
		if err == nil {
			err = accessmanagement.CheckStatement(runCtx, caller, statement)
		}
		if err != nil {
			cancel()
			return err
		}
	}
	qs.handlerCtx = handlerCtx
	logging.GetLogger().Debugln("PrepareQuery() invoked...")
	pb := planbuilder.NewPlanBuilder(qs.transactionContext)
	plan, err := pb.BuildPlanFromContext(handlerCtx)
	qs.queryPlan = plan
	if err != nil && qs.cancel != nil {
		qs.cancel()
	}
	return err
}

func (qs *basicQuerySubmitter) SubmitQuery() internaldto.ExecutorOutput {
	logging.GetLogger().Debugln("SubmitQuery() invoked...")
	pl := internaldto.NewBasicPrimitiveContext(
		nil,
		qs.handlerCtx.GetOutfile(),
		qs.handlerCtx.GetOutErrFile(),
	)
	result := qs.queryPlan.GetInstructions().GetPrimitiveGraph().Execute(pl)
	// A cancelled graph must not be reported as a successful empty result.
	if err := qs.handlerCtx.GetExecutionContext().Err(); err != nil && result.GetError() == nil {
		if stream := result.GetSQLResult(); stream != nil {
			_ = stream.Close()
		}
		if qs.cancel != nil {
			qs.cancel()
		}
		return internaldto.NewErroneousExecutorOutput(err)
	}
	if qs.cancel != nil && qs.queryPlan.IsReadOnly() {
		if stream := result.GetSQLResult(); stream != nil {
			bounded := &callerStream{ISQLResultStream: stream, cancel: qs.cancel}
			result.SetSQLResultFn(func() sqldata.ISQLResultStream { return bounded })
		} else {
			qs.cancel()
		}
	}
	return result
}

func (qs *basicQuerySubmitter) PrepareUndoQuery(handlerCtx handler.HandlerContext) error {
	qs.undoHandlerCtx = handlerCtx
	logging.GetLogger().Debugln("PrepareQuery() invoked...")
	pb := planbuilder.NewPlanBuilder(qs.transactionContext)
	plan, err := pb.BuildPlanFromContext(handlerCtx)
	qs.undoPlan = plan
	return err
}

func (qs *basicQuerySubmitter) UndoQuery() internaldto.ExecutorOutput {
	logging.GetLogger().Debugln("UndoQuery() invoked...")
	pl := internaldto.NewBasicPrimitiveContext(
		nil,
		qs.handlerCtx.GetOutfile(),
		qs.handlerCtx.GetOutErrFile(),
	)
	return qs.queryPlan.GetInstructions().GetInversePrimitiveGraph().Execute(pl)
}

// Read-only snapshots live through streaming, and release their deadline timer at EOF/close.
type callerStream struct {
	sqldata.ISQLResultStream
	cancel func()
	once   sync.Once
}

func (s *callerStream) Read() (sqldata.ISQLResult, error) {
	row, err := s.ISQLResultStream.Read()
	if err != nil {
		s.once.Do(s.cancel)
	}
	return row, err
}
func (s *callerStream) Close() error { s.once.Do(s.cancel); return s.ISQLResultStream.Close() }
