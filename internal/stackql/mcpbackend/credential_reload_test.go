package mcpbackend //nolint:testpackage // exercise unexported helpers

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stackql/any-sdk/pkg/dto"
)

func TestParseEnvFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "creds.env")
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
	got, err := parseEnvFile(path)
	if err != nil {
		t.Fatalf("parseEnvFile: %v", err)
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

func TestSourceEnvFile_MissingFileIsNotAnError(t *testing.T) {
	keys, sourced, err := sourceEnvFile(filepath.Join(t.TempDir(), "absent.env"))
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

func TestSourceEnvFile_EmptyPathIsNoop(t *testing.T) {
	keys, sourced, err := sourceEnvFile("")
	if err != nil || sourced || len(keys) != 0 {
		t.Errorf("empty path must be a silent no-op, got keys=%v sourced=%t err=%v", keys, sourced, err)
	}
}

func TestSourceEnvFile_SetsProcessEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "creds.env")
	const varName = "STACKQL_TEST_RELOAD_VAR"
	t.Setenv(varName, "stale-value")
	if err := os.WriteFile(path, []byte(varName+"=fresh-value\n"), 0o600); err != nil {
		t.Fatalf("write env file: %v", err)
	}
	keys, sourced, err := sourceEnvFile(path)
	if err != nil {
		t.Fatalf("sourceEnvFile: %v", err)
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

func TestCredentialSource(t *testing.T) {
	cases := []struct {
		name string
		ac   *dto.AuthCtx
		want string
	}{
		{"env var key", &dto.AuthCtx{KeyEnvVar: "MY_SECRET"}, "env:MY_SECRET"},
		{"file path", &dto.AuthCtx{KeyFilePath: "/path/key.json"}, "file:/path/key.json"},
		{"file path env var", &dto.AuthCtx{KeyFilePathEnvVar: "KEY_PATH"}, "env:KEY_PATH"},
		{"basic env pair", &dto.AuthCtx{EnvVarUsername: "U", EnvVarPassword: "P"}, "env:U,env:P"},
		{"inline basic", &dto.AuthCtx{Username: "u", Password: "p"}, "inline"},
		{"nothing", &dto.AuthCtx{}, "none"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := credentialSource(tc.ac); got != tc.want {
				t.Errorf("expected %q, got %q", tc.want, got)
			}
		})
	}
}

func TestProviderCredentialStatus(t *testing.T) {
	const varName = "STACKQL_TEST_STATUS_VAR"
	t.Run("unresolved env var", func(t *testing.T) {
		os.Unsetenv(varName) //nolint:errcheck // test hygiene
		got := providerCredentialStatus("okta", &dto.AuthCtx{Type: dto.AuthAPIKeyStr, KeyEnvVar: varName})
		if got.Status != credentialStatusUnresolved {
			t.Errorf("expected unresolved, got %q (detail %q)", got.Status, got.Detail)
		}
		if !strings.Contains(got.Detail, "references empty string") {
			t.Errorf("expected resolution detail, got %q", got.Detail)
		}
	})
	t.Run("resolved env var", func(t *testing.T) {
		t.Setenv(varName, "some-secret")
		got := providerCredentialStatus("okta", &dto.AuthCtx{Type: dto.AuthAPIKeyStr, KeyEnvVar: varName})
		if got.Status != credentialStatusOK {
			t.Errorf("expected ok, got %q (detail %q)", got.Status, got.Detail)
		}
		if strings.Contains(got.Detail, "some-secret") || strings.Contains(got.SourcedFrom, "some-secret") {
			t.Errorf("secret value must never appear in the report: %+v", got)
		}
	})
	t.Run("uncheckable auth type", func(t *testing.T) {
		got := providerCredentialStatus("azure", &dto.AuthCtx{Type: dto.AuthAzureDefaultStr})
		if got.Status != credentialStatusNotChecked {
			t.Errorf("expected not_checked, got %q", got.Status)
		}
	})
}

func TestClassifyBackendError_CredentialResolutionHint(t *testing.T) {
	err := fmt.Errorf("credentials error: credentialsenvvar references empty string")
	got := classifyBackendError(err)
	if !strings.Contains(got.Error(), "reload_credentials") {
		t.Errorf("expected reload_credentials hint, got %q", got.Error())
	}
	if !strings.Contains(got.Error(), "references empty string") {
		t.Errorf("expected underlying detail preserved, got %q", got.Error())
	}
}
