package constants

type IsolationLevel int

// Isolation levels.
const (
	ReadUncommitted IsolationLevel = iota
	ReadCommitted
	RepeatableRead
	Serializable
)

const (
	ReadCommittedStr   = "read committed"
	RepeatableReadStr  = "repeatable read"
	SerializableStr    = "serializable"
	ReadUncommittedStr = "read uncommitted"
)
