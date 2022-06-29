package streaming

import (
	"fmt"
	"io"
)

type SimpleProjectionMapStream struct {
	store      []map[string]interface{}
	projection map[string]string
}

func NewSimpleProjectionMapStream(projection map[string]string) MapStream {
	return &SimpleProjectionMapStream{
		projection: projection,
	}
}

func (ss *SimpleProjectionMapStream) iStackQLReader() {}

func (ss *SimpleProjectionMapStream) iStackQLWriter() {}

func (ss *SimpleProjectionMapStream) Write(input []map[string]interface{}) error {
	ss.store = append(ss.store, input...)
	return nil
}

func (ss *SimpleProjectionMapStream) Read() ([]map[string]interface{}, error) {
	var rv []map[string]interface{}
	for _, row := range ss.store {
		rowTransformed := map[string]interface{}{}
		for k, v := range ss.projection {
			captured, ok := row[k]
			if !ok {
				return nil, fmt.Errorf("streaming: cannot project response data: missing key '%s'", k)
			}
			rowTransformed[v] = captured

		}
		rv = append(rv, rowTransformed)
	}
	ss.store = nil
	return rv, io.EOF
}
