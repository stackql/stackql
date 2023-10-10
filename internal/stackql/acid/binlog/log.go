package binlog

var (
	_ LogEntry = &simpleLogEntry{}
)

type LogEntry interface {
	AppendHumanReadable(string)
	AppendRaw([]byte)
	Clone() LogEntry
	Concatenate(...LogEntry)
	GetHumanReadable() []string
	GetRaw() []byte
	Size() int
}

type simpleLogEntry struct {
	raw           []byte
	humanReadable []string
}

func NewSimpleLogEntry(
	raw []byte,
	humanReadable []string,
) LogEntry {
	return newSimpleLogEntry(raw, humanReadable)
}

func newSimpleLogEntry(
	raw []byte,
	humanReadable []string,
) LogEntry {
	return &simpleLogEntry{
		raw:           raw,
		humanReadable: humanReadable,
	}
}

func (l *simpleLogEntry) Concatenate(others ...LogEntry) {
	rSize, hrSize := len(l.raw), len(l.humanReadable)
	for _, entry := range others {
		rSize += entry.Size()
		hrSize += len(entry.GetHumanReadable())
	}
	raw := make([]byte, rSize)
	humanReadable := make([]string, hrSize)
	rawN := copy(raw, l.raw)
	hrN := copy(humanReadable, l.humanReadable)
	for _, entry := range others {
		rawN += copy(raw[rawN:], entry.GetRaw())
		hrN += copy(humanReadable[hrN:], entry.GetHumanReadable())
	}
	l.raw = raw
	l.humanReadable = humanReadable
}

func (l *simpleLogEntry) Size() int {
	return len(l.raw)
}

func (l *simpleLogEntry) Clone() LogEntry {
	rawCopy := make([]byte, len(l.raw))
	copy(rawCopy, l.raw)
	humanReadableCopy := make([]string, len(l.humanReadable))
	copy(humanReadableCopy, l.humanReadable)
	return NewSimpleLogEntry(rawCopy, humanReadableCopy)
}

func (l *simpleLogEntry) GetRaw() []byte {
	return l.raw
}

func (l *simpleLogEntry) AppendRaw(b []byte) {
	l.raw = append(l.raw, b...)
}

func (l *simpleLogEntry) GetHumanReadable() []string {
	return l.humanReadable
}

func (l *simpleLogEntry) AppendHumanReadable(s string) {
	l.humanReadable = append(l.humanReadable, s)
}
