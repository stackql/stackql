package envfile //nolint:testpackage // exercise unexported parse

import (
	"os"
	"path/filepath"
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
