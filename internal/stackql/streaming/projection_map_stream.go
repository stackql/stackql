package streaming

import (
	"fmt"
	"io"
)

type SimpleProjectionMapStream struct {
	staticStore  map[string]interface{}
	dynamicStore []map[string]interface{}
	projection   map[string]string
}

func NewSimpleProjectionMapStream(projection map[string]string, staticStore map[string]interface{}) MapStream {
	return &SimpleProjectionMapStream{
		projection:  projection,
		staticStore: staticStore,
	}
}

func (ss *SimpleProjectionMapStream) Write(input []map[string]interface{}) error {
	ss.dynamicStore = append(ss.dynamicStore, input...)
	return nil
}

func (ss *SimpleProjectionMapStream) Read() ([]map[string]interface{}, error) {
	var rv []map[string]interface{}
	for _, row := range ss.dynamicStore {
		rowTransformed := map[string]interface{}{}
		for k, v := range ss.projection {
			captured, ok := row[k]
			if !ok {
				return nil, fmt.Errorf("streaming: cannot project response data: missing key '%s'", k)
			}
			rowTransformed[v] = captured
		}
		for k, v := range ss.staticStore {
			rowTransformed[k] = v
		}
		rv = append(rv, rowTransformed)
	}
	ss.dynamicStore = nil
	return rv, io.EOF
}
