package sqlstream

import (
	"io"

	"github.com/stackql/any-sdk/pkg/streaming"
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

func (ss *StaticMapStream) Write(_ []map[string]interface{}) error {
	return nil
}

func (ss *StaticMapStream) Read() ([]map[string]interface{}, error) {
	return ss.payload, io.EOF
}
