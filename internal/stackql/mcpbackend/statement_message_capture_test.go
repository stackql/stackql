package mcpbackend //nolint:testpackage // exercise unexported capture

import (
	"fmt"
	"strings"
	"testing"
)

func TestStatementMessageCapture_SplitsLinesAndSkipsBlanks(t *testing.T) {
	c := newStatementMessageCapture()
	fmt.Fprintf(c, "first message\n\nsecond message\r\n")
	got := c.drain(nil)
	if len(got) != 2 || got[0] != "first message" || got[1] != "second message" {
		t.Fatalf("unexpected messages: %#v", got)
	}
}

func TestStatementMessageCapture_PartialWritesJoinIntoOneMessage(t *testing.T) {
	c := newStatementMessageCapture()
	// Two Write calls carrying one logical line, as a diagnostic writer might.
	c.Write([]byte("http response status code: 404, "))       //nolint:errcheck // never fails
	c.Write([]byte("response body is nil\nand a second one")) //nolint:errcheck // never fails
	got := c.drain(nil)
	if len(got) != 2 || got[0] != "http response status code: 404, response body is nil" ||
		got[1] != "and a second one" {
		t.Fatalf("unexpected messages: %#v", got)
	}
}

func TestStatementMessageCapture_TruncatesLongMessages(t *testing.T) {
	c := newStatementMessageCapture()
	long := strings.Repeat("x", maxCapturedMessageLen+100)
	fmt.Fprintf(c, "%s\n", long)
	got := c.drain(nil)
	if len(got) != 1 {
		t.Fatalf("expected one message, got %#v", got)
	}
	if len(got[0]) > maxCapturedMessageLen+len(messageTruncationMarker) ||
		!strings.HasSuffix(got[0], messageTruncationMarker) {
		t.Fatalf("expected bounded message with truncation marker, got %d bytes: %q...", len(got[0]), got[0][:40])
	}
}

func TestStatementMessageCapture_CapsMessageCountWithMarker(t *testing.T) {
	c := newStatementMessageCapture()
	total := maxCapturedMessages + 5
	for i := 0; i < total; i++ {
		fmt.Fprintf(c, "message %d\n", i)
	}
	got := c.drain(nil)
	if len(got) != maxCapturedMessages+1 {
		t.Fatalf("expected %d messages plus suppression marker, got %d", maxCapturedMessages, len(got))
	}
	if !strings.Contains(got[len(got)-1], "5 further engine message(s) suppressed") {
		t.Fatalf("expected suppression marker, got %q", got[len(got)-1])
	}
}

func TestStatementMessageCapture_DrainFoldsInExtraMessages(t *testing.T) {
	c := newStatementMessageCapture()
	fmt.Fprintf(c, "captured diagnostic\n")
	got := c.drain([]string{"executor message"})
	if len(got) != 2 || got[0] != "captured diagnostic" || got[1] != "executor message" {
		t.Fatalf("unexpected messages: %#v", got)
	}
}

func TestStatementMessageCapture_EmptyDrainsNil(t *testing.T) {
	c := newStatementMessageCapture()
	if got := c.drain(nil); got != nil {
		t.Fatalf("expected nil for no messages, got %#v", got)
	}
}
