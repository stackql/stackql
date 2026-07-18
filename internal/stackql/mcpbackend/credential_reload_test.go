package mcpbackend //nolint:testpackage // exercise unexported helpers

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/stackql/any-sdk/pkg/dto"
)

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
