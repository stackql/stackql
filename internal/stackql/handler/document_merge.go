package handler

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"gopkg.in/yaml.v2"
)

// DocumentRegistry selects this client's effective view without changing the base registry.
func (hc *standardHandlerContext) DocumentRegistry(base string) string {
	state := hc.providerSessionState()
	state.mu.Lock()
	defer state.mu.Unlock()
	if state.documentView != "" {
		return state.documentView
	}
	return base
}

func (hc *standardHandlerContext) setDocumentMerge(value interface{}) error {
	state := hc.providerSessionState()
	state.mu.Lock()
	defer state.mu.Unlock()
	if value == nil {
		state.documentView = ""
		return nil
	}
	if text, ok := value.(string); ok {
		raw := []byte(text)
		if filename, file := strings.CutPrefix(text, "@"); file {
			data, err := os.ReadFile(filename)
			if err != nil {
				return fmt.Errorf("document_merge: %w", err)
			}
			raw = data
		}
		if err := json.Unmarshal(raw, &value); err != nil {
			return fmt.Errorf("document_merge: invalid JSON: %w", err)
		}
	}
	var config struct {
		Patches []omnisdk.DocPatch `json:"patches"`
		Cache   omnisdk.DocCache   `json:"doc_cache"`
	}
	if err := decodeSessionObject(value, &config); err != nil {
		return err
	}
	if len(config.Patches) == 0 {
		return fmt.Errorf("document_merge requires patches; use UNSET to restore the base")
	}
	if config.Cache.Fresh {
		return fmt.Errorf("document_merge: fresh is unsupported; use a new cache directory")
	}
	if config.Cache.Dir == "" {
		config.Cache.Dir = filepath.Join(hc.runtimeContext.ApplicationFilesRootPath, "cache", "document-views")
	}
	var registry omnisdk.RegistryConfig
	if err := yaml.Unmarshal([]byte(hc.runtimeContext.RegistryRaw), &registry); err != nil {
		return err
	}
	base := registry.LocalDocRoot
	if base == "" {
		base = hc.runtimeContext.ApplicationFilesRootPath
		if strings.HasPrefix(registry.RegistryURL, "file:") {
			base = filepath.Clean(filepath.Join(strings.TrimPrefix(registry.RegistryURL, "file:"), ".."))
		}
	}
	prefix := "src"
	if registry.SrcPrefix != nil {
		prefix = *registry.SrcPrefix
	}
	view, err := omnisdk.EffectiveRegistry(filepath.Join(base, prefix), config.Patches, config.Cache)
	if err != nil {
		return fmt.Errorf("document_merge: %w", err)
	}
	state.documentView = view
	return nil
}
