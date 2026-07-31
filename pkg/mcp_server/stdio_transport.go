package mcp_server //nolint:revive // package name is established

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sirupsen/logrus"
)

const (
	// maxStdioFrameBytes bounds a single inbound newline-delimited frame so a
	// peer cannot exhaust memory with one unterminated line (issue #701).
	maxStdioFrameBytes = 8 << 20 // 8 MiB
	// loggedFrameSampleBytes caps how much of an offending frame reaches the
	// logs; client input is untrusted and must never be logged unbounded.
	loggedFrameSampleBytes = 256
	stdioReadBufferBytes   = 64 << 10

	// Error responses for undeliverable frames, per JSON-RPC 2.0.  The id is
	// null because the request id cannot be trusted from a frame that failed
	// to decode.  These are pre-marshalled so the write path cannot fail.
	jsonRPCParseErrorLine     = `{"jsonrpc":"2.0","id":null,"error":{"code":-32700,"message":"Parse error"}}`
	jsonRPCInvalidRequestLine = `{"jsonrpc":"2.0","id":null,"error":{"code":-32600,"message":"Invalid Request"}}`
)

// resilientStdioTransport is a drop-in replacement for the SDK's
// mcp.StdioTransport.  The SDK transport treats any JSON decode failure on
// stdin as fatal: its read goroutine exits and the session terminates with a
// non-zero status, so a single garbled frame from a client is a
// denial-of-service primitive (issue #701; still the behaviour at SDK
// v1.7.0).  This transport reads newline-delimited frames itself, answers
// undecodable frames with JSON-RPC error responses (-32700 parse error,
// -32600 invalid request) and keeps serving the session; only frames the SDK
// is guaranteed to accept are handed on.
//
// Deliberate divergence from the SDK transport: inbound JSON-RPC batch
// arrays are answered with -32600 rather than fanned out.  Batching was
// removed in MCP protocol 2025-06-18 (the SDK kills the session when a batch
// arrives on a modern-protocol session, so this is strictly more tolerant).
type resilientStdioTransport struct {
	logger *logrus.Logger
}

func newResilientStdioTransport(logger *logrus.Logger) *resilientStdioTransport {
	return &resilientStdioTransport{logger: logger}
}

func (t *resilientStdioTransport) Connect(_ context.Context) (mcp.Connection, error) {
	return newResilientStdioConn(os.Stdin, os.Stdout, t.logger, maxStdioFrameBytes), nil
}

type stdioMsgOrErr struct {
	msg jsonrpc.Message
	err error
}

// resilientStdioConn implements mcp.Connection over a newline-delimited
// JSON-RPC stream.  All writes (server responses and the guard's own error
// responses) are serialised through one mutex, so stdout stays protocol-pure
// with no interleaved frames.
type resilientStdioConn struct {
	out      io.Writer
	logger   *logrus.Logger
	incoming <-chan stdioMsgOrErr

	writeMu sync.Mutex

	closers   []io.Closer
	closeOnce sync.Once
	closed    chan struct{}
	closeErr  error
}

func newResilientStdioConn(
	in io.ReadCloser,
	out io.WriteCloser,
	logger *logrus.Logger,
	maxFrameBytes int,
) *resilientStdioConn {
	incoming := make(chan stdioMsgOrErr)
	conn := &resilientStdioConn{
		out:      out,
		logger:   logger,
		incoming: incoming,
		closers:  []io.Closer{in, out},
		closed:   make(chan struct{}),
	}
	// A dedicated read goroutine lets Read select against the closed channel
	// (mirroring the SDK transport); it leaks if the underlying stdin read
	// never unblocks after close, which is unavoidable for os.Stdin.
	go conn.readLoop(bufio.NewReaderSize(in, stdioReadBufferBytes), maxFrameBytes, incoming)
	return conn
}

func (c *resilientStdioConn) readLoop(
	reader *bufio.Reader,
	maxFrameBytes int,
	incoming chan<- stdioMsgOrErr,
) {
	for {
		frame, overflowed, err := readBoundedFrame(reader, maxFrameBytes)
		if msg, ok := c.admitFrame(frame, overflowed); ok {
			select {
			case incoming <- stdioMsgOrErr{msg: msg}:
			case <-c.closed:
				return
			}
		}
		if err != nil {
			select {
			case incoming <- stdioMsgOrErr{err: err}:
			case <-c.closed:
			}
			return
		}
	}
}

// admitFrame vets one inbound frame.  Frames that cannot be delivered to the
// SDK are answered on the spot with a JSON-RPC error response and dropped;
// the session continues.  Returns the decoded message and true only for
// frames the SDK's message layer is guaranteed to accept.
func (c *resilientStdioConn) admitFrame(frame []byte, overflowed bool) (jsonrpc.Message, bool) {
	if overflowed {
		c.rejectFrame(jsonRPCParseErrorLine, "oversized", frame)
		return nil, false
	}
	// Windows text-mode pipes CRLF-terminate JSON-RPC lines (issue #668).  A
	// raw CR is illegal inside a JSON string (control characters must be
	// escaped), so stripping every CR from the frame is lossless for
	// spec-compliant traffic.
	frame = bytes.TrimSpace(bytes.ReplaceAll(frame, []byte{'\r'}, nil))
	if len(frame) == 0 {
		return nil, false
	}
	if !json.Valid(frame) {
		c.rejectFrame(jsonRPCParseErrorLine, "malformed JSON", frame)
		return nil, false
	}
	msg, decErr := jsonrpc.DecodeMessage(frame)
	if decErr != nil {
		c.rejectFrame(jsonRPCInvalidRequestLine, "non-JSON-RPC", frame)
		return nil, false
	}
	return msg, true
}

func (c *resilientStdioConn) rejectFrame(response string, reason string, frame []byte) {
	if c.logger != nil {
		sample := frame
		if len(sample) > loggedFrameSampleBytes {
			sample = sample[:loggedFrameSampleBytes]
		}
		c.logger.Warnf(
			"stdio transport: rejecting %s frame (%d bytes, sample %q)",
			reason, len(frame), sample,
		)
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if _, err := io.WriteString(c.out, response+"\n"); err != nil && c.logger != nil {
		c.logger.Warnf("stdio transport: failed to write error response: %v", err)
	}
}

// readBoundedFrame reads one newline-terminated frame, discarding (but fully
// consuming) any frame longer than maxFrameBytes.  The returned frame
// excludes the terminator; overflowed reports whether the cap was exceeded.
func readBoundedFrame(reader *bufio.Reader, maxFrameBytes int) ([]byte, bool, error) {
	var frame []byte
	overflowed := false
	for {
		chunk, err := reader.ReadSlice('\n')
		if !overflowed {
			if len(frame)+len(chunk) > maxFrameBytes {
				overflowed = true
				frame = nil
			} else {
				frame = append(frame, chunk...)
			}
		}
		if errors.Is(err, bufio.ErrBufferFull) {
			continue
		}
		frame = bytes.TrimSuffix(frame, []byte{'\n'})
		return frame, overflowed, err
	}
}

func (c *resilientStdioConn) SessionID() string { return "" }

func (c *resilientStdioConn) Read(ctx context.Context) (jsonrpc.Message, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case v := <-c.incoming:
		if v.err != nil {
			return nil, v.err
		}
		return v.msg, nil
	case <-c.closed:
		return nil, io.EOF
	}
}

func (c *resilientStdioConn) Write(ctx context.Context, msg jsonrpc.Message) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	data, err := jsonrpc.EncodeMessage(msg)
	if err != nil {
		return fmt.Errorf("marshaling message: %w", err)
	}
	data = append(data, '\n') // newline delimited
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_, err = c.out.Write(data)
	return err
}

func (c *resilientStdioConn) Close() error {
	c.closeOnce.Do(func() {
		var errs []error
		for _, closer := range c.closers {
			if err := closer.Close(); err != nil {
				errs = append(errs, err)
			}
		}
		c.closeErr = errors.Join(errs...)
		close(c.closed)
	})
	return c.closeErr
}
