package sink

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

// FileConfig is the on-disk file sink configuration.  All fields are optional.
type FileConfig struct {
	// Path is the absolute or relative path to the log file.  Empty means
	// the sink picks a timestamped name in cwd using DefaultFilename.
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

	// DefaultFilename is used when Path is empty.  It is a function so
	// callers can encode their own naming convention (eg
	// "stackql_mcp_server_<timestamp>.log") without bringing the format
	// string into the sink package.  When nil, a generic
	// "sink_<RFC3339-utc-second>.log" filename is used.
	DefaultFilename func(time.Time) string `json:"-" yaml:"-"`
}

// fileSink writes one JSON object per line and fsyncs after each record.
type fileSink struct {
	mu   sync.Mutex
	w    io.WriteCloser
	path string
}

// NewFileSink constructs a file-backed sink.  When cfg.Path is empty a
// timestamped name is generated in cwd via cfg.DefaultFilename (or the
// package-default fallback).  The resolved path is logged to stderr at
// startup so operators can find the file later.
func NewFileSink(cfg FileConfig) (Sink, error) {
	path := cfg.Path
	if path == "" {
		gen := cfg.DefaultFilename
		if gen == nil {
			gen = defaultFilename
		}
		path = gen(time.Now().UTC())
	}
	abs, absErr := filepath.Abs(path)
	if absErr != nil {
		return nil, fmt.Errorf("resolve sink file path %q: %w", path, absErr)
	}
	// Fail fast if the directory isn't writable, instead of waiting for the
	// first record to surface the error.
	if mkdirErr := os.MkdirAll(filepath.Dir(abs), 0o700); mkdirErr != nil {
		return nil, fmt.Errorf("create sink file dir %q: %w", filepath.Dir(abs), mkdirErr)
	}
	probe, openErr := os.OpenFile(abs, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if openErr != nil {
		return nil, fmt.Errorf("open sink file %q: %w", abs, openErr)
	}
	if closeErr := probe.Close(); closeErr != nil {
		return nil, fmt.Errorf("close sink file probe %q: %w", abs, closeErr)
	}

	lj := &lumberjack.Logger{
		Filename:   abs,
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
	}
	fmt.Fprintf(os.Stderr, "sink file: %s\n", abs)
	return &fileSink{w: lj, path: abs}, nil
}

// Path reports the absolute path of the file currently being written.
// Exposed for tests and operator diagnostics.
func (s *fileSink) Path() string { return s.path }

func (s *fileSink) Record(_ context.Context, payload any) error {
	line, marshalErr := json.Marshal(payload)
	if marshalErr != nil {
		return fmt.Errorf("marshal sink payload: %w", marshalErr)
	}
	line = append(line, '\n')
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, writeErr := s.w.Write(line); writeErr != nil {
		return fmt.Errorf("write sink payload: %w", writeErr)
	}
	return nil
}

func (s *fileSink) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Close()
}

// defaultFilename returns sink_<RFC3339-utc-second>.log in cwd.  Stripped of
// colons so the filename is portable across Windows + Unix.
func defaultFilename(t time.Time) string {
	stamp := t.UTC().Format("20060102T150405Z")
	return fmt.Sprintf("sink_%s.log", stamp)
}
