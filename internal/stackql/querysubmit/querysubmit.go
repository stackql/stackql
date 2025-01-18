package querysubmit

import (
	"github.com/stackql/any-sdk/pkg/logging"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/acid/binlog"
	"github.com/stackql/stackql/internal/stackql/acid/txn_context"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
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
	qs.handlerCtx = handlerCtx
	logging.GetLogger().Debugln("PrepareQuery() invoked...")
	pb := planbuilder.NewPlanBuilder(qs.transactionContext)
	plan, err := pb.BuildPlanFromContext(handlerCtx)
	qs.queryPlan = plan
	return err
}

func (qs *basicQuerySubmitter) SubmitQuery() internaldto.ExecutorOutput {
	logging.GetLogger().Debugln("SubmitQuery() invoked...")
	pl := internaldto.NewBasicPrimitiveContext(
		nil,
		qs.handlerCtx.GetOutfile(),
		qs.handlerCtx.GetOutErrFile(),
	)
	return qs.queryPlan.GetInstructions().GetPrimitiveGraph().Execute(pl)
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
