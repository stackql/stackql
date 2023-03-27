package transact

type LogEntry interface {
	// Get bytes
	GetRaw() []byte
}
