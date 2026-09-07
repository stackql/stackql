package providerconfig_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stackql/stackql/internal/stackql/providerconfig"
)

func TestReadProviderConfig_BasicYAML(t *testing.T) {
	content := `name: test-provider
version: "1.0"
enabled: true
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	result, err := providerconfig.ReadProviderConfig(configPath)
	if err != nil {
		t.Fatalf("ReadProviderConfig() error = %v", err)
	}
	if result == nil {
		t.Fatal("ReadProviderConfig() returned nil map")
	}
	if result["name"] != "test-provider" {
		t.Errorf("name = %v, want 'test-provider'", result["name"])
	}
	if result["version"] != "1.0" {
		t.Errorf("version = %v, want '1.0'", result["version"])
	}
}

func TestReadProviderConfig_NestedYAML(t *testing.T) {
	content := `provider:
  name: nested
  settings:
    timeout: 30
    retries: 3
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nested.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	result, err := providerconfig.ReadProviderConfig(configPath)
	if err != nil {
		t.Fatalf("ReadProviderConfig() error = %v", err)
	}

	nested, ok := result["provider"].(map[interface{}]interface{})
	if !ok {
		t.Fatalf("provider is not a nested map, got %T", result["provider"])
	}
	if nested["name"] != "nested" {
		t.Errorf("provider.name = %v, want 'nested'", nested["name"])
	}
}

func TestReadProviderConfig_EmptyYAML(t *testing.T) {
	content := `# just a comment
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "empty.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	result, err := providerconfig.ReadProviderConfig(configPath)
	if err != nil {
		t.Fatalf("ReadProviderConfig() error = %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map, got %d keys", len(result))
	}
}

func TestReadProviderConfig_FileNotFound(t *testing.T) {
	_, err := providerconfig.ReadProviderConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("ReadProviderConfig() expected error for non-existent file, got nil")
	}
}

func TestReadProviderConfig_InvalidYAML(t *testing.T) {
	content := `invalid: yaml: content:
  - broken
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	_, err := providerconfig.ReadProviderConfig(configPath)
	if err == nil {
		t.Error("ReadProviderConfig() expected error for invalid YAML, got nil")
	}
}

func TestReadProviderConfig_ListValues(t *testing.T) {
	content := `databases:
  - postgres
  - mysql
  - sqlite
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "list.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	result, err := providerconfig.ReadProviderConfig(configPath)
	if err != nil {
		t.Fatalf("ReadProviderConfig() error = %v", err)
	}

	dbs, ok := result["databases"].([]interface{})
	if !ok {
		t.Fatalf("databases is not a list, got %T", result["databases"])
	}
	if len(dbs) != 3 {
		t.Errorf("databases len = %d, want 3", len(dbs))
	}
}

func TestReadProviderConfig_SpecialCharacters(t *testing.T) {
	content := `query: "SELECT * FROM users WHERE name = 'admin'"
escape: "new\nline\ttab"
`
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "special.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	result, err := providerconfig.ReadProviderConfig(configPath)
	if err != nil {
		t.Fatalf("ReadProviderConfig() error = %v", err)
	}
	if result["query"] != "SELECT * FROM users WHERE name = 'admin'" {
		t.Errorf("query = %v, want SQL string", result["query"])
	}
}
