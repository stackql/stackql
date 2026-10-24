package intrinsic //nolint:testpackage // tests unexported iac state resolution

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolveStateDirDefaultsToRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "iac")
	got, err := resolveStateDir(root, "")
	if err != nil {
		t.Fatalf("resolveStateDir(root, \"\") error = %v", err)
	}
	if got != root {
		t.Errorf("resolveStateDir(root, \"\") = %q, want %q", got, root)
	}
}

func TestResolveStateDirAllowsNamesUnderRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "iac")
	for _, name := range []string{"prod", "team/prod", "team/../prod", "./prod"} {
		got, err := resolveStateDir(root, name)
		if err != nil {
			t.Errorf("resolveStateDir(%q) error = %v", name, err)
			continue
		}
		if want := filepath.Join(root, name); got != want {
			t.Errorf("resolveStateDir(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestResolveStateDirRejectsEscapes(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "iac")
	outside := filepath.Join(base, "outside")
	for _, name := range []string{outside, "..", "../outside", "team/../../outside", "/"} {
		if _, err := resolveStateDir(root, name); !errors.Is(err, errStateEscapesRoot) {
			t.Errorf("resolveStateDir(%q) error = %v, want errStateEscapesRoot", name, err)
		}
	}
	// A refusal is decided before anything is written.
	for _, dir := range []string{root, outside} {
		if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%q exists after refused names, stat error = %v", dir, err)
		}
	}
}

func TestResolveStateDirRejectsLinkOutOfRoot(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "iac")
	outside := filepath.Join(base, "outside")
	for _, dir := range []string{root, outside} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	for _, name := range []string{"escape", "escape/prod"} {
		if _, err := resolveStateDir(root, name); !errors.Is(err, errStateEscapesRoot) {
			t.Errorf("resolveStateDir(%q) error = %v, want errStateEscapesRoot", name, err)
		}
	}
}

func TestResolveStateDirFollowsLinkedRoot(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "target")
	if err := os.MkdirAll(filepath.Join(target, "prod"), 0o755); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(base, "iac")
	if err := os.Symlink(target, root); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	// The operator may point the root itself at another disk.
	got, err := resolveStateDir(root, "prod")
	if err != nil {
		t.Fatalf("resolveStateDir under a linked root error = %v", err)
	}
	if want := filepath.Join(root, "prod"); got != want {
		t.Errorf("resolveStateDir under a linked root = %q, want %q", got, want)
	}
}

func TestInitReadsIaCStateRoot(t *testing.T) {
	previous := previewCfg
	t.Cleanup(func() { previewCfg = previous })
	Init(`{"unstable": true, "iacStateRoot": " /srv/stackql/iac "}`)
	if got := previewCfg.getIaCStateRoot(); got != "/srv/stackql/iac" {
		t.Errorf("getIaCStateRoot() = %q, want %q", got, "/srv/stackql/iac")
	}
}
