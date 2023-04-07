package txn_context //nolint:revive,stylecheck // meaning of package name is clear

var (
	_ ITransactionCoordinatorContext = &transactionCoordinatorContext{}
)

type ITransactionCoordinatorContext interface {
	GetMaxStackDepth() int
}

type transactionCoordinatorContext struct {
	maxStackDepth int
}

func NewTransactionCoordinatorContext(
	maxStackDepth int,
) ITransactionCoordinatorContext {
	return &transactionCoordinatorContext{
		maxStackDepth: maxStackDepth,
	}
}

func (tc *transactionCoordinatorContext) GetMaxStackDepth() int {
	return tc.maxStackDepth
}
