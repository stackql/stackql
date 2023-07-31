package constants

type RollbackType int

// Rollback algorithms.
const (
	NopRollback RollbackType = iota
	EagerRollback
)

const (
	NopRollbackStr   = "nop"
	EagerRollbackStr = "eager"
)
