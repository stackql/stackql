package txn_context //nolint:stylecheck,revive // meaning of package name is clear

var (
	_ ITransactionContext = &transactionContext{}
)

type ITransactionContext interface {
	GetStackDepth() int
}

type transactionContext struct {
	stackDepth int
}

func NewTransactionContext(
	stackDepth int,
) ITransactionContext {
	return &transactionContext{
		stackDepth: stackDepth,
	}
}

func (tc *transactionContext) GetStackDepth() int {
	return tc.stackDepth
}
