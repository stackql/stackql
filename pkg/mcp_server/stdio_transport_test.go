package mcp_server //nolint:testpackage,revive // exercise internal transport

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
)

// syncBuffer is a goroutine-safe WriteCloser capturing transport output.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) Close() error { return nil }

func (b *syncBuffer) Lines() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	raw := strings.TrimRight(b.buf.String(), "\n")
	if raw == "" {
		return nil
	}
	return strings.Split(raw, "\n")
}

func newTestStdioConn(t *testing.T, input string, maxFrameBytes int) (*resilientStdioConn, *syncBuffer) {
	t.Helper()
	out := &syncBuffer{}
	conn := newResilientStdioConn(io.NopCloser(strings.NewReader(input)), out, nil, maxFrameBytes)
	t.Cleanup(func() { conn.Close() }) //nolint:errcheck // test cleanup
	return conn, out
}

func mustReadPing(t *testing.T, conn *resilientStdioConn) {
	t.Helper()
	msg, err := conn.Read(context.Background())
	if err != nil {
		t.Fatalf("Read returned error %v, want ping request", err)
	}
	req, ok := msg.(*jsonrpc.Request)
	if !ok {
		t.Fatalf("Read returned %T, want *jsonrpc.Request", msg)
	}
	if req.Method != "ping" {
		t.Fatalf("Read returned method %q, want \"ping\"", req.Method)
	}
}

func mustReadEOF(t *testing.T, conn *resilientStdioConn) {
	t.Helper()
	msg, err := conn.Read(context.Background())
	if err != io.EOF {
		t.Fatalf("Read returned (%v, %v), want io.EOF", msg, err)
	}
}

func assertErrorLine(t *testing.T, line string, wantCode int) {
	t.Helper()
	var resp struct {
		Jsonrpc string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id"`
		Error   *struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		t.Fatalf("error response %q is not valid JSON: %v", line, err)
	}
	if resp.Jsonrpc != "2.0" {
		t.Fatalf("error response %q has jsonrpc %q, want 2.0", line, resp.Jsonrpc)
	}
	if string(resp.ID) != "null" {
		t.Fatalf("error response %q has id %s, want null", line, resp.ID)
	}
	if resp.Error == nil || resp.Error.Code != wantCode {
		t.Fatalf("error response %q lacks error code %d", line, wantCode)
	}
}

// A malformed frame must be answered with -32700 and must not poison the
// session: the following valid request is still delivered (issue #701).
func TestStdioConnSurvivesMalformedFrames(t *testing.T) {
	malformedFrames := []string{
		`{this is not json`,
		`{"jsonrpc":`,
		`garbage-token`,
	}
	for _, malformed := range malformedFrames {
		t.Run(malformed, func(t *testing.T) {
			input := malformed + "\n" + `{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n"
			conn, out := newTestStdioConn(t, input, maxStdioFrameBytes)
			mustReadPing(t, conn)
			mustReadEOF(t, conn)
			lines := out.Lines()
			if len(lines) != 1 {
				t.Fatalf("got %d output lines %q, want exactly 1 parse error", len(lines), lines)
			}
			assertErrorLine(t, lines[0], -32700)
		})
	}
}

// Valid JSON that is not a JSON-RPC message gets -32600, session continues.
func TestStdioConnRejectsInvalidRequestFrames(t *testing.T) {
	invalidFrames := []string{
		`"just a string"`,
		`42`,
		`{"jsonrpc":"1.0","id":1,"method":"ping"}`,
		`{"jsonrpc":"2.0"}`,
		`[]`,
	}
	for _, invalid := range invalidFrames {
		t.Run(invalid, func(t *testing.T) {
			input := invalid + "\n" + `{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n"
			conn, out := newTestStdioConn(t, input, maxStdioFrameBytes)
			mustReadPing(t, conn)
			mustReadEOF(t, conn)
			lines := out.Lines()
			if len(lines) != 1 {
				t.Fatalf("got %d output lines %q, want exactly 1 invalid-request error", len(lines), lines)
			}
			assertErrorLine(t, lines[0], -32600)
		})
	}
}

// CRLF-terminated frames (issue #668) must decode as if LF-terminated.
func TestStdioConnToleratesCRLFFrames(t *testing.T) {
	input := "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"ping\"}\r\n"
	conn, out := newTestStdioConn(t, input, maxStdioFrameBytes)
	mustReadPing(t, conn)
	mustReadEOF(t, conn)
	if lines := out.Lines(); len(lines) != 0 {
		t.Fatalf("unexpected output %q", lines)
	}
}

// Blank lines are skipped without any response.
func TestStdioConnSkipsBlankLines(t *testing.T) {
	input := "\n\n" + `{"jsonrpc":"2.0","id":1,"method":"ping"}` + "\n\n"
	conn, out := newTestStdioConn(t, input, maxStdioFrameBytes)
	mustReadPing(t, conn)
	mustReadEOF(t, conn)
	if lines := out.Lines(); len(lines) != 0 {
		t.Fatalf("unexpected output %q", lines)
	}
}

// A frame over the size cap is consumed, answered with -32700, and the frame
// after it is still served.  The cap is passed in small to keep the test fast;
// the frame also exceeds the internal read buffer to exercise chunked reads.
func TestStdioConnBoundsFrameSize(t *testing.T) {
	oversized := `{"jsonrpc":"2.0","id":1,"method":"` + strings.Repeat("a", 2*stdioReadBufferBytes) + `"}`
	input := oversized + "\n" + `{"jsonrpc":"2.0","id":2,"method":"ping"}` + "\n"
	conn, out := newTestStdioConn(t, input, 1024)
	mustReadPing(t, conn)
	mustReadEOF(t, conn)
	lines := out.Lines()
	if len(lines) != 1 {
		t.Fatalf("got %d output lines %q, want exactly 1 parse error", len(lines), lines)
	}
	assertErrorLine(t, lines[0], -32700)
}

// A final unterminated frame at EOF is still processed.
func TestStdioConnHandlesUnterminatedFinalFrame(t *testing.T) {
	input := `{"jsonrpc":"2.0","id":1,"method":"ping"}` // no trailing newline
	conn, out := newTestStdioConn(t, input, maxStdioFrameBytes)
	mustReadPing(t, conn)
	mustReadEOF(t, conn)
	if lines := out.Lines(); len(lines) != 0 {
		t.Fatalf("unexpected output %q", lines)
	}
}

// Writes are newline-delimited encodings of the message.
func TestStdioConnWriteFrames(t *testing.T) {
	conn, out := newTestStdioConn(t, "", maxStdioFrameBytes)
	req := &jsonrpc.Request{Method: "notifications/whatever"}
	if err := conn.Write(context.Background(), req); err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	lines := out.Lines()
	if len(lines) != 1 {
		t.Fatalf("got %d output lines %q, want 1", len(lines), lines)
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &decoded); err != nil {
		t.Fatalf("written frame %q is not valid JSON: %v", lines[0], err)
	}
	if decoded["method"] != "notifications/whatever" {
		t.Fatalf("written frame %q lacks expected method", lines[0])
	}
}
