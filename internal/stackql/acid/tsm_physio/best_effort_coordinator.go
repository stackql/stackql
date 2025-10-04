package tsm_physio //nolint:stylecheck // prefer this nomenclature

import (
	"fmt"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/acid/acid_dto"
	"github.com/stackql/stackql/internal/stackql/acid/binlog"
	"github.com/stackql/stackql/internal/stackql/acid/tsm"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
)

var (
	_ Coordinator = &basicBestEffortTransactionCoordinator{}
)

type basicBestEffortTransactionCoordinator struct {
	tsmInstance       tsm.TSM
	handlerCtx        handler.HandlerContext
	parent            Coordinator
	statementSequence []Statement
	undoLogs          []binlog.LogEntry
	redoLogs          []binlog.LogEntry
	statementGraphs   []primitivegraph.PrimitiveGraphHolder
	maxTxnDepth       int
	outputs           []internaldto.ExecutorOutput
	isExecuted        bool
	// redoGraphs        []primitivegraph.PrimitiveGraph
	// undoGraphs        []primitivegraph.PrimitiveGraph
}

func newBasicBestEffortTransactionCoordinator(
	tsmInstance tsm.TSM,
	handlerCtx handler.HandlerContext,
	parent Coordinator,
	maxTxnDepth int,
) Coordinator {
	return &basicBestEffortTransactionCoordinator{
		tsmInstance: tsmInstance,
		handlerCtx:  handlerCtx,
		parent:      parent,
		maxTxnDepth: maxTxnDepth,
	}
}

func (m *basicBestEffortTransactionCoordinator) GetQuery() string {
	return ""
}

func (m *basicBestEffortTransactionCoordinator) GetPrimitiveGraphHolder() (primitivegraph.PrimitiveGraphHolder, bool) {
	return nil, false
}

func (m *basicBestEffortTransactionCoordinator) IsReadOnly() bool {
	for _, statement := range m.statementSequence {
		if !statement.IsReadOnly() {
			return false
		}
	}
	return true
}

func (m *basicBestEffortTransactionCoordinator) GetAST() (sqlparser.Statement, bool) {
	return nil, false
}

func (m *basicBestEffortTransactionCoordinator) GetParent() (Coordinator, bool) {
	return m.parent, m.parent != nil
}

func (m *basicBestEffortTransactionCoordinator) AppendRedoLog(log binlog.LogEntry) {
	m.redoLogs = append(m.redoLogs, log)
}

func (m *basicBestEffortTransactionCoordinator) AppendUndoLog(log binlog.LogEntry) {
	m.undoLogs = append(m.undoLogs, log)
}

func (m *basicBestEffortTransactionCoordinator) IsBegin() bool {
	return false
}

func (m *basicBestEffortTransactionCoordinator) IsCommit() bool {
	return false
}

func (m *basicBestEffortTransactionCoordinator) IsRollback() bool {
	return false
}

func (m *basicBestEffortTransactionCoordinator) SetUndoLog(log binlog.LogEntry) {
	m.undoLogs = []binlog.LogEntry{log}
}

func (m *basicBestEffortTransactionCoordinator) GetRedoLog() (binlog.LogEntry, bool) {
	rv := binlog.NewSimpleLogEntry(nil, nil)
	if len(m.redoLogs) == 0 {
		return nil, false
	}
	rv.Concatenate(m.redoLogs...)
	return rv, true
}

func (m *basicBestEffortTransactionCoordinator) GetUndoLog() (binlog.LogEntry, bool) {
	rv := binlog.NewSimpleLogEntry(nil, nil)
	if len(m.undoLogs) == 0 {
		return nil, false
	}
	rv.Concatenate(m.undoLogs...)
	return rv, true
}

func (m *basicBestEffortTransactionCoordinator) Prepare() error {
	var err error
	for _, stmt := range m.statementSequence {
		err = stmt.Prepare()
		if err != nil {
			return err
		}
	}
	return nil
}

// This is an approximation of 2PC.
func (m *basicBestEffortTransactionCoordinator) Execute() internaldto.ExecutorOutput {
	outputs, err := m.votingPhase()
	m.outputs = outputs
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	return internaldto.NewExecutorOutput(
		nil,
		nil,
		nil,
		internaldto.NewBackendMessages([]string{"transaction committed"}),
		nil,
	)
}

func (m *basicBestEffortTransactionCoordinator) IsExecuted() bool {
	return m.outputs != nil
}

func (m *basicBestEffortTransactionCoordinator) votingPhase() ([]internaldto.ExecutorOutput, error) {
	defer func() {
		m.isExecuted = true
	}()
	var rv []internaldto.ExecutorOutput
	for _, stmt := range m.statementSequence {
		coDomain := stmt.Execute()
		rv = append(rv, coDomain)
		err := coDomain.GetError()
		undoLog, undoLogExists := coDomain.GetUndoLog()
		redoLog, redoLogExists := coDomain.GetRedoLog()
		if undoLogExists {
			m.undoLogs = append(m.undoLogs, undoLog)
		}
		if redoLogExists {
			m.redoLogs = append(m.redoLogs, redoLog)
		}
		if err != nil {
			return rv, err
		}
	}
	return rv, nil
}

func (m *basicBestEffortTransactionCoordinator) completionPhase() error {
	return nil
}

func (m *basicBestEffortTransactionCoordinator) Begin() (Coordinator, error) {
	if m.maxTxnDepth >= 0 && m.Depth() >= m.maxTxnDepth {
		return nil, fmt.Errorf("cannot begin nested transaction of depth = %d", m.Depth()+1)
	}
	return newBasicBestEffortTransactionCoordinator(m.tsmInstance, m.handlerCtx, m, m.maxTxnDepth), nil
}

func (m *basicBestEffortTransactionCoordinator) Commit() acid_dto.CommitCoDomain {
	rv, err := m.votingPhase()
	if err != nil {
		return acid_dto.NewCommitCoDomain(rv, err, nil)
	}
	completionErr := m.completionPhase()
	return acid_dto.NewCommitCoDomain(rv, nil, completionErr)
}

// Rollback is best effort and runs in reverse order.
func (m *basicBestEffortTransactionCoordinator) Rollback() acid_dto.CommitCoDomain {
	var coDomains []internaldto.ExecutorOutput
	for i := len(m.statementGraphs) - 1; i >= 0; i-- {
		stmt := m.statementGraphs[i]
		pl := internaldto.NewBasicPrimitiveContext(
			nil,
			m.handlerCtx.GetOutfile(),
			m.handlerCtx.GetOutErrFile(),
		)
		inverseGraph := stmt.GetInversePrimitiveGraph()
		if inverseGraph == nil {
			return acid_dto.NewCommitCoDomain(
				nil,
				nil,
				fmt.Errorf("cannot rollback statement without inverse primitive graph"),
			)
		}
		optimiseErr := inverseGraph.Optimise()
		if optimiseErr != nil {
			return acid_dto.NewCommitCoDomain(
				nil,
				nil,
				optimiseErr,
			)
		}
		coDomain := stmt.GetInversePrimitiveGraph().Execute(pl)
		coDomains = append(coDomains, coDomain)
		if coDomain.GetError() != nil {
			return acid_dto.NewCommitCoDomain(
				coDomains,
				nil,
				coDomain.GetError(),
			)
		}
	}
	return acid_dto.NewCommitCoDomain(
		coDomains,
		nil,
		nil,
	)
}

func (m *basicBestEffortTransactionCoordinator) Enqueue(stmt Statement) error {
	graphHolder, graphHolderExists := stmt.GetPrimitiveGraphHolder()
	if !graphHolderExists {
		return fmt.Errorf("cannot enqueue statement without primitive graph holder")
	}
	reversal := graphHolder.GetInversePrimitiveGraph()
	if reversal == nil {
		return fmt.Errorf("cannot enqueue statement without inverse primitive graph")
	}
	if reversal.Size() < 1 {
		return fmt.Errorf("cannot enqueue statement with empty inverse primitive graph")
	}
	m.statementGraphs = append(m.statementGraphs, graphHolder)
	m.statementSequence = append(m.statementSequence, stmt)
	return nil
}

func (m *basicBestEffortTransactionCoordinator) Depth() int {
	return m.depth()
}

func (m *basicBestEffortTransactionCoordinator) IsRoot() bool {
	return m.parent == nil
}

func (m *basicBestEffortTransactionCoordinator) depth() int {
	if m.parent != nil {
		return m.parent.Depth() + 1
	}
	return 0
}
