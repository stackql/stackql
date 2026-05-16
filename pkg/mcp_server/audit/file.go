package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// FileConfig is the on-disk audit sink configuration.  All fields are optional.
type FileConfig struct {
	// Path is the absolute or relative path to the log file.  Empty means
	// the sink picks `stackql_mcp_server_<RFC3339-utc-second>.log` in cwd.
	Path string `json:"path,omitempty" yaml:"path,omitempty"`

	// MaxSizeMB triggers rotation when the file grows past this size.
	// Zero means lumberjack's default (100 MB).
	MaxSizeMB int `json:"max_size_mb,omitempty" yaml:"max_size_mb,omitempty"`

	// MaxBackups is the number of rotated files to keep.
	// Zero means keep all (lumberjack default).
	MaxBackups int `json:"max_backups,omitempty" yaml:"max_backups,omitempty"`

	// MaxAgeDays is the maximum age in days for rotated files.
	// Zero means no age-based deletion (lumberjack default).
	MaxAgeDays int `json:"max_age_days,omitempty" yaml:"max_age_days,omitempty"`
}

// fileSink writes one JSON object per line and fsyncs after each record.
type fileSink struct {
	mu   sync.Mutex
	w    io.WriteCloser
	path string
}

// NewFileSink constructs a file-backed audit sink.  If cfg.Path is empty a
// timestamped name is generated in cwd.  The resolved path is logged to
// stderr at startup so operators can find the file later.
func NewFileSink(cfg FileConfig) (Sink, error) {
	path := cfg.Path
	if path == "" {
		path = defaultLogPath(time.Now().UTC())
	}
	abs, absErr := filepath.Abs(path)
	if absErr != nil {
		return nil, fmt.Errorf("resolve audit log path %q: %w", path, absErr)
	}
	// Fail fast if the directory isn't writable, instead of waiting for the
	// first record to surface the error.
	if mkdirErr := os.MkdirAll(filepath.Dir(abs), 0o700); mkdirErr != nil {
		return nil, fmt.Errorf("create audit log dir %q: %w", filepath.Dir(abs), mkdirErr)
	}
	probe, openErr := os.OpenFile(abs, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if openErr != nil {
		return nil, fmt.Errorf("open audit log %q: %w", abs, openErr)
	}
	if closeErr := probe.Close(); closeErr != nil {
		return nil, fmt.Errorf("close audit log probe %q: %w", abs, closeErr)
	}

	lj := &lumberjack.Logger{
		Filename:   abs,
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
	}
	fmt.Fprintf(os.Stderr, "audit log: %s\n", abs)
	return &fileSink{w: lj, path: abs}, nil
}

// Path reports the absolute path of the file currently being written.
func (s *fileSink) Path() string { return s.path }

func (s *fileSink) Record(_ context.Context, event Event) error {
	line, marshalErr := json.Marshal(event)
	if marshalErr != nil {
		return fmt.Errorf("marshal audit event: %w", marshalErr)
	}
	line = append(line, '\n')
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, writeErr := s.w.Write(line); writeErr != nil {
		return fmt.Errorf("write audit event: %w", writeErr)
	}
	return nil
}

func (s *fileSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Close()
}

// defaultLogPath returns stackql_mcp_server_<RFC3339-utc-second>.log in cwd,
// stripped of colons so the filename is portable across Windows + Unix.
func defaultLogPath(t time.Time) string {
	stamp := t.UTC().Format("20060102T150405Z")
	return fmt.Sprintf("stackql_mcp_server_%s.log", stamp)
}
