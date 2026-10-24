package streaming

import (
	"io"
)

type StackQLReader interface{}

type StackQLWriter interface{}

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

type standardMapStream struct {
	store []map[string]interface{}
}

type MapStream interface {
	MapReader
	MapWriter
}

func NewstandardMapStream() MapStream {
	return &standardMapStream{}
}

func (ss *standardMapStream) Write(input []map[string]interface{}) error {
	ss.store = append(ss.store, input...)
	return nil
}

func (ss *standardMapStream) Read() ([]map[string]interface{}, error) {
	rv := ss.store
	ss.store = nil
	return rv, io.EOF
}
