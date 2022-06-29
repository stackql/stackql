package streaming

import (
	"io"
)

type NopMapStream struct {
}

func NewNopMapStream() MapStream {
	return &StandardMapStream{}
}

func (ss *NopMapStream) iStackQLReader() {}

func (ss *NopMapStream) iStackQLWriter() {}

func (ss *NopMapStream) Write(input []map[string]interface{}) error {
	return nil
}

func (ss *NopMapStream) Read() ([]map[string]interface{}, error) {
	return nil, io.EOF
}
