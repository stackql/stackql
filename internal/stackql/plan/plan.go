package plan

import (
	"time"

	"github.com/stackql/stackql/internal/stackql/acid/binlog"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

var (
	_ Plan = &standardPlan{}
)

type Plan interface {

	// Getters
	GetType() sqlparser.StatementType
	GetStatement() (sqlparser.Statement, bool)
	GetOriginal() string
	GetInstructions() primitivegraph.PrimitiveGraphHolder
	GetBindVarNeeds() sqlparser.BindVarNeeds

	// Signals whether the plan is worthy to place in `cache.LRUCache`.
	IsCacheable() bool

	// Get the redo log entry.
	GetRedoLog() (binlog.LogEntry, bool)
	// Get the undo log entry.
	GetUndoLog() (binlog.LogEntry, bool)

	// Setters
	SetType(t sqlparser.StatementType)
	SetStatement(statement sqlparser.Statement)
	SetOriginal(original string)
	SetInstructions(instructions primitivegraph.PrimitiveGraphHolder)
	SetBindVarNeeds(bindVarNeeds sqlparser.BindVarNeeds)
	SetCacheable(isCacheable bool)
	SetTxnID(txnID int)

	//
	IsReadOnly() bool
	SetReadOnly(bool)

	// Size is defined so that Plan can be given to a cache.LRUCache,
	// which requires its objects to define a Size function.
	Size() int

	GetPrimitiveGraphHolder() (primitivegraph.PrimitiveGraphHolder, bool)
}

type standardPlan struct {
	Type               sqlparser.StatementType // The type of query we have
	RewrittenStatement sqlparser.Statement
	Original           string // Original is the original query.
	// Instructions contains the instructions needed to fulfil the query.
	Instructions primitivegraph.PrimitiveGraphHolder
	// Stores BindVars needed to be provided as part of expression rewriting
	sqlparser.BindVarNeeds

	ExecCount    uint64        // Count of times this plan was executed
	ExecTime     time.Duration // Total execution time
	ShardQueries uint64        // Total number of shard queries
	Rows         uint64        // Total number of rows
	Errors       uint64        // Total number of errors
	isCacheable  bool
	isReadOnly   bool
}

func NewPlan(
	rawQuery string,
) Plan {
	return &standardPlan{
		Original:    rawQuery,
		isCacheable: true,
	}
}

func (p *standardPlan) GetRedoLog() (binlog.LogEntry, bool) {
	if p.Instructions == nil {
		return nil, false
	}
	return p.Instructions.GetPrimitiveGraph().GetRedoLog()
}

func (p *standardPlan) GetPrimitiveGraphHolder() (primitivegraph.PrimitiveGraphHolder, bool) {
	if p.Instructions == nil {
		return nil, false
	}
	return p.Instructions, true
}

func (p *standardPlan) GetUndoLog() (binlog.LogEntry, bool) {
	if p.Instructions == nil {
		return nil, false
	}
	return p.Instructions.GetPrimitiveGraph().GetUndoLog()
}

func (p *standardPlan) SetReadOnly(isReadOnly bool) {
	p.isReadOnly = isReadOnly
}

func (p *standardPlan) IsReadOnly() bool {
	if p.Instructions == nil {
		return true
	}
	if p.isReadOnly {
		return true
	}
	return p.Instructions.GetPrimitiveGraph().IsReadOnly()
}

func (p *standardPlan) SetTxnID(txnID int) {
	p.Instructions.SetTxnID(txnID)
}

func (p *standardPlan) GetType() sqlparser.StatementType {
	return p.Type
}

func (p *standardPlan) GetStatement() (sqlparser.Statement, bool) {
	return p.RewrittenStatement, p.RewrittenStatement != nil
}

func (p *standardPlan) GetOriginal() string {
	return p.Original
}

func (p *standardPlan) GetInstructions() primitivegraph.PrimitiveGraphHolder {
	return p.Instructions
}

func (p *standardPlan) GetBindVarNeeds() sqlparser.BindVarNeeds {
	return p.BindVarNeeds
}

func (p *standardPlan) SetType(t sqlparser.StatementType) {
	p.Type = t
}

func (p *standardPlan) SetStatement(statement sqlparser.Statement) {
	p.RewrittenStatement = statement
}

func (p *standardPlan) SetOriginal(original string) {
	p.Original = original
}

func (p *standardPlan) SetInstructions(instructions primitivegraph.PrimitiveGraphHolder) {
	p.Instructions = instructions
}

func (p *standardPlan) SetBindVarNeeds(bindVarNeeds sqlparser.BindVarNeeds) {
	p.BindVarNeeds = bindVarNeeds
}

// Size is defined so that Plan can be given to a cache.LRUCache,
// which requires its objects to define a Size function.
func (p *standardPlan) Size() int {
	return 1
}

// Signals whether the plan is worthy to place in `cache.LRUCache`.
func (p *standardPlan) IsCacheable() bool {
	return p.isCacheable
}

func (p *standardPlan) SetCacheable(isCacheable bool) {
	p.isCacheable = isCacheable
}
