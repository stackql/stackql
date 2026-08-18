package embed

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testBinary(t *testing.T, data []byte) Binary {
	t.Helper()
	sum := sha256.Sum256(data)
	return Binary{
		Data:        data,
		Version:     "0.0.1",
		PlatformKey: PlatformLinuxX64,
		SHA256:      hex.EncodeToString(sum[:]),
	}
}

func TestPlatformKey(t *testing.T) {
	cases := []struct {
		goos, goarch, want string
		wantErr            bool
	}{
		{"linux", "amd64", PlatformLinuxX64, false},
		{"linux", "arm64", PlatformLinuxARM64, false},
		{"windows", "amd64", PlatformWindowsX64, false},
		{"darwin", "amd64", PlatformDarwinUniversal, false},
		{"darwin", "arm64", PlatformDarwinUniversal, false},
		{"windows", "arm64", "", true},
		{"plan9", "amd64", "", true},
	}
	for _, c := range cases {
		got, err := PlatformKey(c.goos, c.goarch)
		if c.wantErr {
			if err == nil {
				t.Errorf("PlatformKey(%s,%s): want error, got %q", c.goos, c.goarch, got)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("PlatformKey(%s,%s) = %q, %v; want %q", c.goos, c.goarch, got, err, c.want)
		}
	}
}

func TestBuildArgsCanonical(t *testing.T) {
	approot := filepath.Join(t.TempDir(), ".stackql")
	args, err := BuildArgs("", approot, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"mcp",
		"--mcp.server.type=stdio",
		"--approot", approot,
		"--mcp.config", `{"server":{"mode":"read_only","audit":{"disabled":true}}}`,
	}
	if len(args) != len(want) {
		t.Fatalf("args = %q, want %q", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Errorf("args[%d] = %q, want %q", i, args[i], want[i])
		}
	}
}

func TestBuildArgsModeAndAuth(t *testing.T) {
	approot := filepath.Join(t.TempDir(), ".stackql")
	args, err := BuildArgs(ModeSafe, approot, map[string]any{
		"github": map[string]any{"type": "null_auth"},
	})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, `"mode":"safe"`) {
		t.Errorf("mode not in config: %s", joined)
	}
	last := args[len(args)-1]
	if !strings.HasPrefix(last, "--auth=") {
		t.Fatalf("last arg = %q, want --auth=...", last)
	}
	var auth map[string]map[string]string
	if err := json.Unmarshal([]byte(strings.TrimPrefix(last, "--auth=")), &auth); err != nil {
		t.Fatalf("auth flag is not valid JSON: %v", err)
	}
	if auth["github"]["type"] != "null_auth" {
		t.Errorf("auth = %v", auth)
	}
}

func TestBuildArgsRejectsBadInput(t *testing.T) {
	if _, err := BuildArgs("yolo", "", nil); err == nil {
		t.Error("unknown mode accepted")
	}
	if _, err := BuildArgs(ModeReadOnly, "relative/path", nil); err == nil {
		t.Error("relative approot accepted")
	}
}

func TestEnsureExtracted(t *testing.T) {
	cache := t.TempDir()
	bin := testBinary(t, []byte("#!/bin/sh\necho fake-stackql\n"))

	path, err := EnsureExtracted(cache, bin)
	if err != nil {
		t.Fatal(err)
	}
	wantPath := filepath.Join(cache, "0.0.1", PlatformLinuxX64, "stackql")
	if path != wantPath {
		t.Errorf("path = %q, want %q", path, wantPath)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != string(bin.Data) {
		t.Fatalf("extracted content mismatch: %v", err)
	}

	// Second call reuses without rewriting: make the file read-only to
	// detect an unwanted rewrite attempt failing.
	info1, _ := os.Stat(path)
	path2, err := EnsureExtracted(cache, bin)
	if err != nil || path2 != path {
		t.Fatalf("re-extract: %q, %v", path2, err)
	}
	info2, _ := os.Stat(path)
	if !info1.ModTime().Equal(info2.ModTime()) {
		t.Error("cached binary was rewritten on second call")
	}
}

func TestEnsureExtractedReplacesCorruptCache(t *testing.T) {
	cache := t.TempDir()
	bin := testBinary(t, []byte("good"))
	target := filepath.Join(cache, "0.0.1", PlatformLinuxX64, "stackql")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("corrupt"), 0o755); err != nil {
		t.Fatal(err)
	}
	path, err := EnsureExtracted(cache, bin)
	if err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if string(got) != "good" {
		t.Errorf("corrupt cache not replaced: %q", got)
	}
}

func TestEnsureExtractedRejectsTamperedData(t *testing.T) {
	bin := testBinary(t, []byte("data"))
	bin.Data = []byte("tampered")
	if _, err := EnsureExtracted(t.TempDir(), bin); err == nil {
		t.Fatal("sha mismatch accepted")
	}
}

func TestBinaryValidate(t *testing.T) {
	if err := (Binary{}).validate(); err == nil {
		t.Error("empty Binary accepted")
	}
	bin := testBinary(t, []byte("x"))
	bin.PlatformKey = "freebsd-x64"
	if err := bin.validate(); err == nil {
		t.Error("unknown platform key accepted")
	}
}
