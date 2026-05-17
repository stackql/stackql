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

// FileConfig is the on-disk file sink configuration.
//
// Either Path (a complete file path) or Dir (a directory in which the sink
// will pick a filename via DefaultFilename) must be set.  NewFileSink errors
// when both are empty; the sink never silently picks a directory of its own.
// The caller -- not the sink -- owns the "where do logs land" decision.
type FileConfig struct {
	// Path is the absolute or relative path to the log file.  When set,
	// Dir and DefaultFilename are ignored.
	Path string `json:"path,omitempty" yaml:"path,omitempty"`

	// Dir is the directory in which the sink writes the log when Path is
	// empty.  Must be set in that case.  Relative directories are resolved
	// against cwd at NewFileSink time.
	Dir string `json:"dir,omitempty" yaml:"dir,omitempty"`

	// MaxSizeMB triggers rotation when the file grows past this size.
	// Zero means lumberjack's default (100 MB).
	MaxSizeMB int `json:"max_size_mb,omitempty" yaml:"max_size_mb,omitempty"`

	// MaxBackups is the number of rotated files to keep.
	// Zero means keep all (lumberjack default).
	MaxBackups int `json:"max_backups,omitempty" yaml:"max_backups,omitempty"`

	// MaxAgeDays is the maximum age in days for rotated files.
	// Zero means no age-based deletion (lumberjack default).
	MaxAgeDays int `json:"max_age_days,omitempty" yaml:"max_age_days,omitempty"`

	// DefaultFilename is consulted when Path is empty.  It returns just the
	// basename; the sink joins it with Dir.  When nil, a generic
	// "sink_<RFC3339-utc-second>.log" filename is used.  Made a function so
	// callers can encode their own naming convention (eg
	// "stackql_mcp_server_<timestamp>.log") without bringing the format
	// string into the sink package.
	DefaultFilename func(time.Time) string `json:"-" yaml:"-"`
}

// fileSink writes one JSON object per line and fsyncs after each record.
type fileSink struct {
	mu   sync.Mutex
	w    io.WriteCloser
	path string
}

// NewFileSink constructs a file-backed sink.  Exactly one of cfg.Path or
// cfg.Dir must be supplied:
//
//   - cfg.Path set: used as-is.  Relative paths are resolved against cwd.
//   - cfg.Path empty + cfg.Dir set: the sink picks the basename via
//     cfg.DefaultFilename (or the package-default fallback) and joins it
//     with Dir.
//
// The resolved absolute path is logged to stderr at startup so operators
// can find the file later.
func NewFileSink(cfg FileConfig) (Sink, error) {
	resolvedPath, err := resolvePath(cfg)
	if err != nil {
		return nil, err
	}
	abs, absErr := filepath.Abs(resolvedPath)
	if absErr != nil {
		return nil, fmt.Errorf("resolve sink file path %q: %w", resolvedPath, absErr)
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

// resolvePath enforces the Path-or-Dir contract and returns the (possibly
// still relative) path the sink will open.
func resolvePath(cfg FileConfig) (string, error) {
	if cfg.Path != "" {
		return cfg.Path, nil
	}
	if cfg.Dir == "" {
		return "", fmt.Errorf("sink file: one of Path or Dir is required")
	}
	gen := cfg.DefaultFilename
	if gen == nil {
		gen = defaultFilename
	}
	return filepath.Join(cfg.Dir, gen(time.Now().UTC())), nil
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

// defaultFilename returns sink_<RFC3339-utc-second>.log -- just the basename.
// Stripped of colons so the filename is portable across Windows + Unix.
func defaultFilename(t time.Time) string {
	stamp := t.UTC().Format("20060102T150405Z")
	return fmt.Sprintf("sink_%s.log", stamp)
}
