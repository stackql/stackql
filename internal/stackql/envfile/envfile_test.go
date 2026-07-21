package envfile //nolint:testpackage // exercise unexported parse

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	path := filepath.Join(t.TempDir(), "creds.env")
	content := strings.Join([]string{
		"# comment line",
		"",
		"PLAIN_KEY=plain-value",
		"export EXPORTED_KEY=exported-value",
		`DOUBLE_QUOTED="double quoted value"`,
		"SINGLE_QUOTED='single quoted value'",
		"CRLF_KEY=crlf-value\r",
		"EMPTY_KEY=",
		"NOT_A_PAIR",
		"SPACED_KEY = spaced-value",
	}, "\n")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	got, err := parse(path)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	want := map[string]string{
		"PLAIN_KEY":     "plain-value",
		"EXPORTED_KEY":  "exported-value",
		"DOUBLE_QUOTED": "double quoted value",
		"SINGLE_QUOTED": "single quoted value",
		"CRLF_KEY":      "crlf-value",
		"SPACED_KEY":    "spaced-value",
	}
	if len(got) != len(want) {
		t.Errorf("expected %d vars, got %d: %v", len(want), len(got), got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("expected %s=%q, got %q", k, v, got[k])
		}
	}
	if _, present := got["EMPTY_KEY"]; present {
		t.Errorf("empty-valued key must be dropped, got %v", got)
	}
	if _, present := got["NOT_A_PAIR"]; present {
		t.Errorf("non-pair line must be dropped, got %v", got)
	}
}

func TestEnsureExists_CreatesCommentedFileWithParentDirs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dir", "creds.env")
	created, err := EnsureExists(path)
	if err != nil {
		t.Fatalf("EnsureExists: %v", err)
	}
	if !created {
		t.Errorf("expected created=true for absent file")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read created file: %v", err)
	}
	if !strings.HasPrefix(string(b), "# StackQL credential store") {
		t.Errorf("expected comment header, got %q", string(b))
	}
	vars, err := parse(path)
	if err != nil {
		t.Fatalf("parse created file: %v", err)
	}
	if len(vars) != 0 {
		t.Errorf("created file must source no vars, got %v", vars)
	}
	if runtime.GOOS != "windows" {
		info, statErr := os.Stat(path)
		if statErr != nil {
			t.Fatalf("stat created file: %v", statErr)
		}
		if info.Mode().Perm() != 0o600 {
			t.Errorf("expected 0600 permissions, got %v", info.Mode().Perm())
		}
	}
}

func TestEnsureExists_NeverTouchesExistingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "creds.env")
	const populated = "POPULATED_KEY=populated-value\n"
	if err := os.WriteFile(path, []byte(populated), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	created, err := EnsureExists(path)
	if err != nil {
		t.Fatalf("EnsureExists: %v", err)
	}
	if created {
		t.Errorf("expected created=false for existing file")
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(b) != populated {
		t.Errorf("existing file must be untouched, got %q", string(b))
	}
}

func TestEnsureExists_EmptyPathIsNoop(t *testing.T) {
	created, err := EnsureExists("")
	if err != nil || created {
		t.Errorf("empty path must be a silent no-op, got created=%t err=%v", created, err)
	}
}

func TestSource_MissingFileIsNotAnError(t *testing.T) {
	keys, sourced, err := Source(filepath.Join(t.TempDir(), "absent.env"))
	if err != nil {
		t.Fatalf("missing file must not error, got %v", err)
	}
	if sourced {
		t.Errorf("missing file must report sourced=false")
	}
	if len(keys) != 0 {
		t.Errorf("missing file must source no keys, got %v", keys)
	}
}

func TestSource_EmptyPathIsNoop(t *testing.T) {
	keys, sourced, err := Source("")
	if err != nil || sourced || len(keys) != 0 {
		t.Errorf("empty path must be a silent no-op, got keys=%v sourced=%t err=%v", keys, sourced, err)
	}
}

func TestSource_SetsProcessEnv(t *testing.T) {
	path := filepath.Join(t.TempDir(), "creds.env")
	const varName = "STACKQL_TEST_RELOAD_VAR"
	t.Setenv(varName, "stale-value")
	if err := os.WriteFile(path, []byte(varName+"=fresh-value\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	keys, sourced, err := Source(path)
	if err != nil {
		t.Fatalf("Source: %v", err)
	}
	if !sourced {
		t.Errorf("expected sourced=true")
	}
	if len(keys) != 1 || keys[0] != varName {
		t.Errorf("expected sourced keys [%s], got %v", varName, keys)
	}
	if got := os.Getenv(varName); got != "fresh-value" {
		t.Errorf("expected env var overwritten with fresh value, got %q", got)
	}
}
