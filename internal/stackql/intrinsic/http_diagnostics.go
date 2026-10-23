package intrinsic

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
)

type httpDiagnostics struct {
	writer io.Writer
	mu     sync.Mutex
}

func newHTTPDiagnostics(writer io.Writer, enabled bool) omnisdk.HTTPObserver {
	if !enabled {
		return nil
	}
	return &httpDiagnostics{writer: writer}
}
func (d *httpDiagnostics) HTTPRequest(method, url string, body []byte) {
	d.mu.Lock()
	defer d.mu.Unlock()
	//nolint:errcheck // Debug output does not alter request execution.
	fmt.Fprintf(d.writer, "http request url: '%s', method: '%s'\n", url, method)
	if len(body) > 0 {
		//nolint:errcheck // Debug output does not alter request execution.
		fmt.Fprintf(d.writer, "http request body = '%s'\n", body)
	}
}
func (d *httpDiagnostics) HTTPResponse(status int, body []byte) {
	d.mu.Lock()
	defer d.mu.Unlock()
	//nolint:errcheck // Debug output does not alter request execution.
	fmt.Fprintf(d.writer, "http response status code: %d, response body: %s\n", status, body)
}
func (d *httpDiagnostics) HTTPRows(rows any) {
	// Legacy diagnostics marshal JSON numbers through a float representation.
	// This conversion belongs only to debug formatting, never to query values.
	encoded, err := json.Marshal(rows)
	if err != nil {
		return
	}
	var formatted any
	if err = json.Unmarshal(encoded, &formatted); err != nil {
		return
	}
	encoded, err = json.MarshalIndent(formatted, "", "  ")
	if err != nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	//nolint:errcheck // Debug output does not alter request execution.
	fmt.Fprintf(d.writer, "processed http response body object: %s\n", encoded)
}
