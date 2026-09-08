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
	got := classifyBackendError(err, "aws")
	for _, fragment := range []string{"provider 'aws'", "fix the configured env file", "reload_credentials", "then retry", "references empty string"} {
		if !strings.Contains(got.Error(), fragment) {
			t.Errorf("expected %q in error, got %q", fragment, got.Error())
		}
	}
}

func TestQueryProviderName(t *testing.T) {
	cases := []struct {
		sql  string
		want string
	}{
		{"select name from aws.s3.buckets where region = 'us-east-1';", "aws"},
		{"exec aws.ec2.instances.instances_Start @region = 'ap-southeast-2', @InstanceId = 'id-001';", "aws"},
		{"delete from google.compute.firewalls where project = 'p' and firewall = 'f';", "google"},
		{"select 1;", ""},
		{"not sql at all", ""},
	}
	for _, tc := range cases {
		if got := queryProviderName(tc.sql); got != tc.want {
			t.Errorf("queryProviderName(%q): expected %q, got %q", tc.sql, tc.want, got)
		}
	}
}

func TestCredentialFingerprint(t *testing.T) {
	const varName = "STACKQL_TEST_FINGERPRINT_VAR"
	ac := &dto.AuthCtx{Type: dto.AuthAPIKeyStr, KeyEnvVar: varName}
	t.Setenv(varName, "key-a")
	before := credentialFingerprint(ac)
	if before != credentialFingerprint(ac) {
		t.Errorf("fingerprint must be stable for an unchanged value")
	}
	t.Setenv(varName, "key-b")
	after := credentialFingerprint(ac)
	if before == after {
		t.Errorf("fingerprint must change with the resolved value")
	}
	for _, fp := range []string{before, after} {
		if strings.Contains(fp, "key-a") || strings.Contains(fp, "key-b") {
			t.Errorf("fingerprint must not embed the value: %q", fp)
		}
	}
	successor := &dto.AuthCtx{Type: dto.AuthCustomStr, KeyEnvVar: varName, Successor: &dto.AuthCtx{KeyEnvVar: varName + "_2"}}
	t.Setenv(varName+"_2", "s-a")
	chained := credentialFingerprint(successor)
	t.Setenv(varName+"_2", "s-b")
	if chained == credentialFingerprint(successor) {
		t.Errorf("fingerprint must cover the successor chain")
	}
}
