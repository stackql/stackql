package intrinsic //nolint:testpackage // tests unexported iac state resolution

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
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

func TestIaCBindingsPreserveNumbersAndLiteralStrings(t *testing.T) {
	params := map[string]string{
		"inputs":      `{"large":9007199254740993,"literal":"${not_a_template}"}`,
		"environment": `{"flag":true}`, "region": "east",
	}
	inputs, environment, err := iacBindings(params)
	if err != nil {
		t.Fatal(err)
	}
	if number, ok := inputs["large"].(json.Number); !ok || number.String() != "9007199254740993" {
		t.Fatal(inputs)
	}
	if inputs["literal"] != "${not_a_template}" || environment["flag"] != true || len(params) != 1 {
		t.Fatal("binding conversion changed values or scope")
	}
	for _, invalid := range []string{`null`, `[]`, `{} {}`, `{"x":`} {
		if _, _, err := iacBindings(map[string]string{"inputs": invalid}); err == nil {
			t.Fatalf("accepted %s", invalid)
		}
	}
}

func TestIaCResourcesRetainSDKBindingsAndRejectRetiredFields(t *testing.T) {
	resources, err := specResources(`[{"key":"one","provider":"demo","address":"service.items","write_once":{"name":{"from":"env","attribute":"NAME"}},"desired":{}}]`)
	if err != nil {
		t.Fatal(err)
	}
	names, err := omnisdk.ResourceEnvironmentVariables(resources)
	if err != nil || len(names) != 1 || names[0] != "NAME" {
		t.Fatalf("bindings lost: %v, %v", names, err)
	}
	for _, invalid := range []string{
		`[{"key":"one","params":{}}]`, `[{"key":"one","correlationParam":"tag"}]`,
		`[{"key":"one"},{"key":"one"}]`,
	} {
		if _, err := specResources(invalid); err == nil {
			t.Fatalf("accepted retired or duplicate specification: %s", invalid)
		}
	}
}

func TestIaCReportedOutcomesAreApplicationFailures(t *testing.T) {
	for _, row := range []omnisdk.Row{
		{"key": "one", "status": "failed", "error": "upstream failure"},
		{"key": "one", "status": "outstanding"},
		{"key": "one", "status": "applied", "outstanding": []string{"two"}},
	} {
		cursor := &iacRows{Rows: &fakeRows{rows: []omnisdk.Row{row}}}
		if !cursor.Next() || cursor.Row()["key"] != "one" {
			t.Fatal("diagnostic row lost")
		}
		for cursor.Next() {
		}
		if cursor.Err() == nil {
			t.Fatalf("unsuccessful outcome accepted: %v", row)
		}
	}
	cursor := &iacRows{Rows: &fakeRows{rows: []omnisdk.Row{{"status": "applied"}}}}
	for cursor.Next() {
	}
	if err := cursor.Err(); err != nil {
		t.Fatal(err)
	}
}
