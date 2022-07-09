package streaming

import (
	"io"
)

type StackQLReader interface {
	// iStackQLReader()
}

type StackQLWriter interface {
	// iStackQLWriter()
}

type StackQLReadWriter interface {
	StackQLReader
	StackQLWriter
}

type MapReader interface {
	StackQLReader
	Read() ([]map[string]interface{}, error)
}

type MapWriter interface {
	StackQLWriter
	Write([]map[string]interface{}) error
}

type StandardMapStream struct {
	store []map[string]interface{}
}

type MapStream interface {
	MapReader
	MapWriter
}

func NewStandardMapStream() MapStream {
	return &StandardMapStream{}
}

func (ss *StandardMapStream) iStackQLReader() {}

func (ss *StandardMapStream) iStackQLWriter() {}

func (ss *StandardMapStream) Write(input []map[string]interface{}) error {
	ss.store = append(ss.store, input...)
	return nil
}

func (ss *StandardMapStream) Read() ([]map[string]interface{}, error) {
	rv := ss.store
	ss.store = nil
	return rv, io.EOF
}
