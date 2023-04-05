package transact

import (
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/acid/binlog"
	"github.com/stackql/stackql/internal/stackql/acid/txn_context"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/querysubmit"
)

type Statement interface {
	Prepare() error
	Execute() internaldto.ExecutorOutput
	GetAST() (sqlparser.Statement, bool)
	GetUndoLog() (binlog.LogEntry, bool)
	GetRedoLog() (binlog.LogEntry, bool)
	IsReadOnly() bool
	SetUndoLog(binlog.LogEntry)
	SetRedoLog(binlog.LogEntry)
	IsBegin() bool
	IsCommit() bool
	IsRollback() bool
}

type basicStatement struct {
	query              string
	handlerCtx         handler.HandlerContext
	querySubmitter     querysubmit.QuerySubmitter
	transactionContext txn_context.ITransactionContext
	undoLog            binlog.LogEntry
	redoLog            binlog.LogEntry
}

func NewStatement(
	query string,
	handlerCtx handler.HandlerContext,
	transactionContext txn_context.ITransactionContext,
) Statement {
	return &basicStatement{
		query:              query,
		handlerCtx:         handlerCtx,
		querySubmitter:     querysubmit.NewQuerySubmitter(),
		transactionContext: transactionContext,
	}
}

func (st *basicStatement) IsBegin() bool {
	ast, hasAst := st.GetAST()
	if hasAst {
		_, isBegin := ast.(*sqlparser.Begin)
		return isBegin
	}
	return false
}

func (st *basicStatement) IsCommit() bool {
	ast, hasAst := st.GetAST()
	if hasAst {
		_, isCommit := ast.(*sqlparser.Commit)
		return isCommit
	}
	return false
}

func (st *basicStatement) IsRollback() bool {
	ast, hasAst := st.GetAST()
	if hasAst {
		_, isRollback := ast.(*sqlparser.Rollback)
		return isRollback
	}
	return false
}

func (st *basicStatement) IsReadOnly() bool {
	if st.querySubmitter == nil {
		return true
	}
	return st.querySubmitter.IsReadOnly()
}

func (st *basicStatement) SetUndoLog(log binlog.LogEntry) {
	st.undoLog = log
}

func (st *basicStatement) SetRedoLog(log binlog.LogEntry) {
	st.redoLog = log
}

func (st *basicStatement) GetUndoLog() (binlog.LogEntry, bool) {
	return st.undoLog, st.undoLog != nil
}

func (st *basicStatement) GetRedoLog() (binlog.LogEntry, bool) {
	return st.redoLog, st.redoLog != nil
}

func (st *basicStatement) Prepare() error {
	cmdString := st.query
	clonedCtx := st.handlerCtx.Clone()
	clonedCtx.SetQuery(cmdString)
	if st.transactionContext != nil {
		st.querySubmitter = st.querySubmitter.WithTransactionContext(st.transactionContext)
	}
	return st.querySubmitter.PrepareQuery(clonedCtx)
}

func (st *basicStatement) Execute() internaldto.ExecutorOutput {
	return st.querySubmitter.SubmitQuery()
}

func (st *basicStatement) GetAST() (sqlparser.Statement, bool) {
	return st.querySubmitter.GetStatement()
}
