package sqlstream

import (
	"io"

	"github.com/stackql/stackql/internal/stackql/streaming"
)

type StaticMapStream struct {
	payload []map[string]interface{}
}

func NewStaticMapStream(
	payload []map[string]interface{},
) streaming.MapStream {
	return &StaticMapStream{
		payload: payload,
	}
}

func (ss *StaticMapStream) Write(input []map[string]interface{}) error {
	return nil
}

func (ss *StaticMapStream) Read() ([]map[string]interface{}, error) {

	return ss.payload, io.EOF
}
