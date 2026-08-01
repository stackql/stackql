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
	// loggedFrameSampleBytes bounds how much untrusted client input is logged.
	loggedFrameSampleBytes = 256
	stdioReadBufferBytes   = 64 << 10

	// The id is null because the request id cannot be trusted from a frame
	// that failed to decode.
	jsonRPCParseErrorLine     = `{"jsonrpc":"2.0","id":null,"error":{"code":-32700,"message":"Parse error"}}`
	jsonRPCInvalidRequestLine = `{"jsonrpc":"2.0","id":null,"error":{"code":-32600,"message":"Invalid Request"}}`
)

// resilientStdioTransport replaces mcp.StdioTransport, whose read loop
// treats any JSON decode failure on stdin as fatal to the session (issue
// #701; unchanged as of SDK v1.7.0).  Frames are read here instead and
// undecodable ones answered per JSON-RPC 2.0 (-32700 / -32600) without
// terminating the session.  Inbound batch arrays get -32600 rather than
// fan-out: batching was removed in MCP 2025-06-18 and the SDK kills
// modern-protocol sessions on batch frames anyway.
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
// JSON-RPC stream.  All writes (server responses and this type's own error
// responses) share writeMu so frames cannot interleave on stdout.
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
	// A dedicated read goroutine lets Read select against the closed channel,
	// mirroring the SDK transport (including its unavoidable leak if a
	// blocked os.Stdin read never unblocks).
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

// admitFrame vets one inbound frame, answering undeliverable ones with a
// JSON-RPC error response; it returns true only for frames the SDK's
// message layer is guaranteed to accept.
func (c *resilientStdioConn) admitFrame(frame []byte, overflowed bool) (jsonrpc.Message, bool) {
	if overflowed {
		c.rejectFrame(jsonRPCParseErrorLine, "oversized", frame)
		return nil, false
	}
	// Windows text-mode pipes CRLF-terminate JSON-RPC lines (issue #668); a
	// raw CR is illegal inside a JSON string, so stripping every CR is
	// lossless for spec-compliant traffic.
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
