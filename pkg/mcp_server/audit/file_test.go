package audit_test

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stackql/stackql/pkg/mcp_server/audit"
)

func TestFileSink_WritesJSONL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.log")
	sink, err := audit.NewFileSink(audit.FileConfig{Path: path})
	if err != nil {
		t.Fatalf("NewFileSink: %v", err)
	}
	t.Cleanup(func() { _ = sink.Close() })

	ev1 := audit.Event{
		Timestamp:  time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC),
		Tool:       "run_select_query",
		Mode:       "safe",
		Decision:   audit.DecisionAllow,
		QueryClass: "select",
		SQL:        "select 1",
		DurationMs: 12,
	}
	ev2 := audit.Event{
		Timestamp: time.Date(2026, 5, 16, 12, 0, 1, 0, time.UTC),
		Tool:      "run_mutation_query",
		Mode:      "read_only",
		Decision:  audit.DecisionRefuseImmediate,
		Error:     "server is in 'read_only' mode",
	}
	if err := sink.Record(context.Background(), ev1); err != nil {
		t.Fatalf("Record ev1: %v", err)
	}
	if err := sink.Record(context.Background(), ev2); err != nil {
		t.Fatalf("Record ev2: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open audit log: %v", err)
	}
	t.Cleanup(func() { _ = f.Close() })
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %v", len(lines), lines)
	}
	var back1, back2 audit.Event
	if err := json.Unmarshal([]byte(lines[0]), &back1); err != nil {
		t.Fatalf("unmarshal line 1: %v\nline: %s", err, lines[0])
	}
	if err := json.Unmarshal([]byte(lines[1]), &back2); err != nil {
		t.Fatalf("unmarshal line 2: %v\nline: %s", err, lines[1])
	}
	if back1.Tool != "run_select_query" || back1.SQL != "select 1" || back1.Decision != audit.DecisionAllow {
		t.Errorf("line 1 round-trip mismatch: %+v", back1)
	}
	if back2.Decision != audit.DecisionRefuseImmediate || back2.Error == "" {
		t.Errorf("line 2 round-trip mismatch: %+v", back2)
	}
}

func TestFileSink_OpensFailsOnUnwritableDir(t *testing.T) {
	// Point at a path inside a non-existent parent directory tree to force
	// a fail-fast.  On Unix os.MkdirAll succeeds for nested paths so we use
	// an absolute path under a file (not a directory) to provoke the error
	// on both Unix and Windows.
	dir := t.TempDir()
	notADir := filepath.Join(dir, "regular.file")
	if err := os.WriteFile(notADir, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	cfg := audit.FileConfig{Path: filepath.Join(notADir, "child.log")}
	if _, err := audit.NewFileSink(cfg); err == nil {
		t.Fatal("expected NewFileSink to fail when parent is not a directory")
	}
}

func TestFileSink_RecordWithoutPathUsesDefaultNameInCwd(t *testing.T) {
	// Run from t.TempDir so we don't leave a file behind in the repo.
	originalCwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tmp := t.TempDir()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir tmp: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalCwd); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})

	sink, err := audit.NewFileSink(audit.FileConfig{})
	if err != nil {
		t.Fatalf("NewFileSink: %v", err)
	}
	t.Cleanup(func() { _ = sink.Close() })

	entries, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatalf("read tmp: %v", err)
	}
	var found string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "stackql_mcp_server_") && strings.HasSuffix(e.Name(), ".log") {
			found = e.Name()
			break
		}
	}
	if found == "" {
		t.Fatalf("expected a default-named audit file in %s, got %v", tmp, entries)
	}
}

func TestNopSink_RecordIsHarmless(t *testing.T) {
	s := audit.NewNopSink()
	if err := s.Record(context.Background(), audit.Event{Tool: "anything"}); err != nil {
		t.Fatalf("nop sink Record should not error: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("nop sink Close should not error: %v", err)
	}
}
