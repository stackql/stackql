package handler //nolint:testpackage // Exercise private session configuration boundaries.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDocumentMergeSessionBoundary(t *testing.T) {
	hc := sessionTestHandler()
	app := t.TempDir()
	base := filepath.Join(app, "registry", "src")
	service := "widgets/v00.00.00000/services/things.yaml"
	write := func(name, body string) {
		t.Helper()
		if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(base, "widgets/v00.00.00000/provider.yaml"), `{"id":"widgets","name":"widgets","version":"v00.00.00000","providerServices":{"things":{"id":"things","name":"things","service":{"$ref":"widgets/v00.00.00000/services/things.yaml"}}}}`)
	write(filepath.Join(base, service), `{"openapi":"3.0.0","info":{"title":"original","version":"v1"}}`)
	hc.runtimeContext.ApplicationFilesRootPath = app
	raw, _ := json.Marshal(map[string]string{"localDocRoot": filepath.Join(app, "registry")})
	hc.runtimeContext.RegistryRaw = string(raw)
	config := `{"patches":[{"service":"stackql_unstable_widgets.things","merge":{"info":{"title":"patched"}}}]}`
	for name, file := range map[string]bool{"inline": false, "file": true} {
		t.Run(name, func(t *testing.T) {
			value := config
			if file {
				name := filepath.Join(app, "merge.json")
				write(name, config)
				value = "@" + name
			}
			checkDocumentMergeBoundary(t, hc, app, base, service, value)
		})
	}
}

func checkDocumentMergeBoundary(t *testing.T, hc *standardHandlerContext, app, base, service, value string) {
	t.Helper()
	if err := hc.SetConfigAtPath("document_merge", value, "session"); err != nil {
		t.Fatal(err)
	}
	view := hc.DocumentRegistry(base)
	if filepath.Dir(view) != filepath.Join(app, "cache", "document-views") {
		t.Fatalf("unexpected default cache: %s", view)
	}
	original, _ := os.ReadFile(filepath.Join(base, service))
	if string(original) != `{"openapi":"3.0.0","info":{"title":"original","version":"v1"}}` {
		t.Fatal("base changed")
	}
	for _, invalid := range []string{`{`, `{"patches":[]}`, `{"typo":true}`, `{"patches":[{"service":"missing.things","merge":{}}]}`, `{"patches":[{"service":"stackql_unstable_widgets.things","merge":{}}],"doc_cache":{"fresh":true}}`, "@" + filepath.Join(app, "missing.json")} {
		if err := hc.SetConfigAtPath("document_merge", invalid, "session"); err == nil {
			t.Fatalf("accepted %s", invalid)
		}
		if hc.DocumentRegistry(base) != view {
			t.Fatal("failed SET changed view")
		}
	}
	other := sessionTestHandler()
	if other.DocumentRegistry(base) != base {
		t.Fatal("view leaked")
	}
	if err := hc.SetConfigAtPath("document_merge", nil, "session"); err != nil {
		t.Fatal(err)
	}
	if hc.DocumentRegistry(base) != base {
		t.Fatal("UNSET failed")
	}
}
