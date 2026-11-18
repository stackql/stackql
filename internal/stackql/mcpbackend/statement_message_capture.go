package mcpbackend

import (
	"fmt"
	"strings"
	"sync"
)

// Bounds on per-statement engine message capture: an upstream response body
// can be arbitrarily large (eg a whole HTML page), so each message is
// truncated and the number of retained messages is capped, with a marker
// reporting anything cut.
const (
	maxCapturedMessages   = 24
	maxCapturedMessageLen = 2048
)

const messageTruncationMarker = " ... [message truncated]"

// statementMessageCapture is a bounded io.Writer installed as the OutErrFile
// of a per-statement handler context clone, so that engine-emitted
// diagnostics (eg "http response status code: 404, response body is nil")
// are attributed to exactly one statement rather than interleaved on a
// process-wide stream.  Writes are line-split: each newline-terminated line
// becomes one message.
type statementMessageCapture struct {
	mu                sync.Mutex
	residual          string
	residualTruncated bool
	messages          []string
	suppressed        int
}

func newStatementMessageCapture() *statementMessageCapture {
	return &statementMessageCapture{}
}

// Write never fails: diagnostics must not alter statement execution.
func (c *statementMessageCapture) Write(p []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	data := c.residual + string(p)
	for {
		idx := strings.IndexByte(data, '\n')
		if idx < 0 {
			break
		}
		c.append(data[:idx], c.residualTruncated)
		c.residualTruncated = false
		data = data[idx+1:]
	}
	// Bound the partial line so an unterminated giant write cannot grow
	// without limit; only the retained prefix can ever become a message.
	if len(data) > maxCapturedMessageLen {
		data = data[:maxCapturedMessageLen]
		c.residualTruncated = true
	}
	c.residual = data
	return len(p), nil
}

// append records one line as a message, applying the length and count caps.
// alreadyTruncated marks lines whose tail was dropped while buffered.
func (c *statementMessageCapture) append(line string, alreadyTruncated bool) {
	line = strings.TrimRight(line, "\r")
	if strings.TrimSpace(line) == "" {
		return
	}
	if len(c.messages) >= maxCapturedMessages {
		c.suppressed++
		return
	}
	if len(line) > maxCapturedMessageLen {
		line = line[:maxCapturedMessageLen]
		alreadyTruncated = true
	}
	if alreadyTruncated {
		line += messageTruncationMarker
	}
	c.messages = append(c.messages, line)
}

// drain flushes any unterminated partial line, folds in the supplied extra
// messages (eg ExecutorOutput.GetMessages()) under the same caps, and returns
// the bounded message list, ending with a suppression marker when the count
// cap was hit.  nil is returned when nothing was captured, so a statement
// with no messages produces exactly the output it does today.
func (c *statementMessageCapture) drain(extra []string) []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.residual != "" {
		c.append(c.residual, c.residualTruncated)
		c.residual = ""
		c.residualTruncated = false
	}
	for _, msg := range extra {
		c.append(msg, false)
	}
	if c.suppressed > 0 {
		c.messages = append(
			c.messages,
			fmt.Sprintf("[%d further engine message(s) suppressed]", c.suppressed),
		)
		c.suppressed = 0
	}
	if len(c.messages) == 0 {
		return nil
	}
	return c.messages
}
