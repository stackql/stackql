package binlog

type LogEntry interface {
	// Get bytes
	GetRaw() []byte
}
