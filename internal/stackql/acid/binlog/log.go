package binlog

import "strings"

var (
	_ LogEntry = &simpleLogEntry{}
)

type LogEntry interface {
	AppendHumanReadable(string)
	AppendRaw([]byte)
	Clone() LogEntry
	GetHumanReadable() []string
	GetRaw() []byte
}

type simpleLogEntry struct {
	raw           []byte
	humanReadable []string
}

func NewSimpleLogEntry(
	raw []byte,
	humanReadable []string,
) LogEntry {
	return &simpleLogEntry{
		raw:           raw,
		humanReadable: humanReadable,
	}
}

func (l *simpleLogEntry) Clone() LogEntry {
	rawCopy := make([]byte, len(l.raw))
	copy(rawCopy, l.raw)
	var humanReadableCopy []string
	for _, s := range l.humanReadable {
		humanReadableCopy = append(humanReadableCopy, strings.Clone(s))
	}
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
