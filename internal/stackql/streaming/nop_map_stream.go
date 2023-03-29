package streaming

import (
	"io"
)

type NopMapStream struct {
}

func NewNopMapStream() MapStream {
	return &StandardMapStream{}
}

func (ss *NopMapStream) Write(_ []map[string]interface{}) error {
	return nil
}

func (ss *NopMapStream) Read() ([]map[string]interface{}, error) {
	return nil, io.EOF
}
