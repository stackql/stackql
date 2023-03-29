package plan

import (
	"time"

	"github.com/stackql/stackql/internal/stackql/primitive"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

var (
	_ Plan = &standardPlan{}
)

type Plan interface {

	// Getters
	GetType() sqlparser.StatementType
	GetStatement() sqlparser.Statement
	GetOriginal() string
	GetInstructions() primitive.IPrimitive
	GetBindVarNeeds() sqlparser.BindVarNeeds

	// Signals whether the plan is worthy to place in `cache.LRUCache`.
	IsCacheable() bool

	// Setters
	SetType(t sqlparser.StatementType)
	SetStatement(statement sqlparser.Statement)
	SetOriginal(original string)
	SetInstructions(instructions primitive.IPrimitive)
	SetBindVarNeeds(bindVarNeeds sqlparser.BindVarNeeds)
	SetCacheable(isCacheable bool)
	SetTxnID(txnID int)

	// Size is defined so that Plan can be given to a cache.LRUCache,
	// which requires its objects to define a Size function.
	Size() int
}

type standardPlan struct {
	Type                   sqlparser.StatementType // The type of query we have
	RewrittenStatement     sqlparser.Statement
	Original               string               // Original is the original query.
	Instructions           primitive.IPrimitive // Instructions contains the instructions needed to fulfil the query.
	sqlparser.BindVarNeeds                      // Stores BindVars needed to be provided as part of expression rewriting

	ExecCount    uint64        // Count of times this plan was executed
	ExecTime     time.Duration // Total execution time
	ShardQueries uint64        // Total number of shard queries
	Rows         uint64        // Total number of rows
	Errors       uint64        // Total number of errors
	isCacheable  bool
}

func NewPlan(
	rawQuery string,
) Plan {
	return &standardPlan{
		Original:    rawQuery,
		isCacheable: true,
	}
}

func (p *standardPlan) SetTxnID(txnID int) {
	p.Instructions.SetTxnID(txnID)
}

func (p *standardPlan) GetType() sqlparser.StatementType {
	return p.Type
}

func (p *standardPlan) GetStatement() sqlparser.Statement {
	return p.RewrittenStatement
}

func (p *standardPlan) GetOriginal() string {
	return p.Original
}

func (p *standardPlan) GetInstructions() primitive.IPrimitive {
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

func (p *standardPlan) SetInstructions(instructions primitive.IPrimitive) {
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
