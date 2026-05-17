package sink_test

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stackql/stackql/pkg/sink"
)

func TestFileSink_WritesJSONL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.log")
	s, err := sink.NewFileSink(sink.FileConfig{Path: path})
	if err != nil {
		t.Fatalf("NewFileSink: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	payload1 := map[string]any{"k": "v1", "n": 1}
	payload2 := map[string]any{"k": "v2", "n": 2}
	if err := s.Record(context.Background(), payload1); err != nil {
		t.Fatalf("Record p1: %v", err)
	}
	if err := s.Record(context.Background(), payload2); err != nil {
		t.Fatalf("Record p2: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open file: %v", err)
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
	var back1, back2 map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &back1); err != nil {
		t.Fatalf("unmarshal line 1: %v", err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &back2); err != nil {
		t.Fatalf("unmarshal line 2: %v", err)
	}
	if back1["k"] != "v1" || back2["k"] != "v2" {
		t.Errorf("round-trip mismatch: %+v %+v", back1, back2)
	}
}

func TestFileSink_OpenFailsOnNonDirParent(t *testing.T) {
	// Provoke a parent-is-a-file error on both Unix and Windows by pointing
	// the log at a path whose parent is an existing regular file.
	dir := t.TempDir()
	notADir := filepath.Join(dir, "regular.file")
	if err := os.WriteFile(notADir, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed file: %v", err)
	}
	cfg := sink.FileConfig{Path: filepath.Join(notADir, "child.log")}
	if _, err := sink.NewFileSink(cfg); err == nil {
		t.Fatal("expected NewFileSink to fail when parent is not a directory")
	}
}

func TestFileSink_EmptyPathAndEmptyDirIsError(t *testing.T) {
	// The sink refuses to silently pick a directory.  Callers must say where
	// the file lives, even when they want the default filename.
	_, err := sink.NewFileSink(sink.FileConfig{})
	if err == nil {
		t.Fatal("expected NewFileSink to reject empty Path + empty Dir")
	}
	if !strings.Contains(err.Error(), "Path or Dir") {
		t.Errorf("error should mention Path or Dir, got %v", err)
	}
}

func TestFileSink_DirWithDefaultFilename(t *testing.T) {
	tmp := t.TempDir()
	s, err := sink.NewFileSink(sink.FileConfig{Dir: tmp})
	if err != nil {
		t.Fatalf("NewFileSink: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	entries, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatalf("read tmp: %v", err)
	}
	var found string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "sink_") && strings.HasSuffix(e.Name(), ".log") {
			found = e.Name()
			break
		}
	}
	if found == "" {
		t.Fatalf("expected default-named sink file in %s, got %v", tmp, entries)
	}
}

func TestFileSink_CallerSuppliedDefaultFilename(t *testing.T) {
	tmp := t.TempDir()
	cfg := sink.FileConfig{
		Dir:             tmp,
		DefaultFilename: func(time.Time) string { return "my-prefix.log" },
	}
	s, err := sink.NewFileSink(cfg)
	if err != nil {
		t.Fatalf("NewFileSink: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	if _, err := os.Stat(filepath.Join(tmp, "my-prefix.log")); err != nil {
		t.Fatalf("expected my-prefix.log in tmp, got: %v", err)
	}
}

func TestFileSink_PathTakesPrecedenceOverDir(t *testing.T) {
	// When Path is supplied, Dir + DefaultFilename are ignored.
	tmp := t.TempDir()
	explicitPath := filepath.Join(tmp, "explicit.log")
	cfg := sink.FileConfig{
		Path:            explicitPath,
		Dir:             filepath.Join(tmp, "should", "not", "be", "used"),
		DefaultFilename: func(time.Time) string { return "should-not-appear.log" },
	}
	s, err := sink.NewFileSink(cfg)
	if err != nil {
		t.Fatalf("NewFileSink: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	if _, err := os.Stat(explicitPath); err != nil {
		t.Fatalf("expected explicit.log to exist, got: %v", err)
	}
	if _, err := os.Stat(filepath.Join(tmp, "should-not-appear.log")); err == nil {
		t.Fatalf("DefaultFilename should not have been consulted when Path is set")
	}
}

func TestNopSink_RecordIsHarmless(t *testing.T) {
	s := sink.NewNopSink()
	if err := s.Record(context.Background(), map[string]any{"any": "thing"}); err != nil {
		t.Fatalf("nop sink Record should not error: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("nop sink Close should not error: %v", err)
	}
}
