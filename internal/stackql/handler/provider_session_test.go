package handler //nolint:testpackage // Private session snapshots and state are tested here.

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/stackql/internal/stackql/provider"
	dto "github.com/stackql/stackql/internal/stackql/runtimeconfig"
)

func sessionTestHandler() *standardHandlerContext {
	return &standardHandlerContext{providers: map[string]provider.IProvider{}, providersMapMutex: &sync.Mutex{}, authMapMutex: &sync.Mutex{}, sessionCtxMutex: &sync.Mutex{}, authContexts: dto.AuthContexts{}, executionContext: context.Background()}
}

func TestProviderSessionBindings(t *testing.T) {
	hc := sessionTestHandler()
	binding := map[string]interface{}{"provider": "widgets", "tenancy": "account-a", "auth": map[string]interface{}{"type": "bearer", "credentials": "first"}, "tuning": map[string]interface{}{"max_concurrent": 1}}
	if err := hc.SetConfigAtPath("providers.production", binding, ""); err != nil {
		t.Fatal(err)
	}
	snapshot, err := hc.ProviderSessionArgs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.ProviderAliases["production"].Auth.Credentials != "first" {
		t.Fatal("wrong credential")
	}
	if snapshot.ProviderAliases["stackql_unstable_production"].Auth.Credentials != "first" {
		t.Fatal("SQL alias missing")
	}
	if got := hc.ResolveProviderAddress("stackql_unstable_production.things.widgets"); got != "stackql_unstable_widgets.things.widgets" {
		t.Fatal(got)
	}
	if err := hc.SetConfigAtPath("providers.production.auth.credentials", "second", "session"); err != nil {
		t.Fatal(err)
	}
	if snapshot.ProviderAliases["production"].Auth.Credentials != "first" {
		t.Fatal("snapshot mutated")
	}
	if err := hc.SetConfigAtPath("providers.production.typo", "value", ""); err == nil {
		t.Fatal("unknown field accepted")
	}
	if err := hc.SetConfigAtPath("providers.production.auth", nil, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := hc.ProviderSessionArgs(context.Background()); err == nil {
		t.Fatal("unset auth borrowed credentials")
	}
	if err := hc.SetConfigAtPath("providers.production", nil, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := hc.ProviderSessionArgs(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestSessionSecretStoreConfiguration(t *testing.T) {
	hc := sessionTestHandler()
	cfg := map[string]interface{}{"backend": "os", "namespace": "stackql-test"}
	if err := hc.SetConfigAtPath("secret_store", cfg, ""); err == nil {
		t.Fatal("persistent secret store enabled without operator opt-in")
	}
	hc.runtimeContext.AllowPersistentSecrets = true
	for _, denied := range []map[string]interface{}{
		{"path": filepath.Join(t.TempDir(), "secrets.json")},
		{"backend": "file", "path": filepath.Join(t.TempDir(), "secrets.json")},
		{"backend": "os", "namespace": "stackql-test", "path": "not-allowed"},
	} {
		if err := hc.SetConfigAtPath("secret_store", denied, ""); err == nil {
			t.Fatal("flat-file secret configuration accepted")
		}
	}
}

func TestAuthSessionResetAndSnapshots(t *testing.T) {
	hc := sessionTestHandler()
	hc.authContexts["aws"] = &dto.AuthCtx{AuthConfig: omnisdk.AuthConfig{Type: "basic", Username: "user", Password: "initial"}}
	before, err := hc.GetAuthContext("aws")
	if err != nil {
		t.Fatal(err)
	}
	if err := hc.SetConfigAtPath("auth.aws.password", "updated", ""); err != nil {
		t.Fatal(err)
	}
	if before.Password != "initial" {
		t.Fatal("auth snapshot mutated")
	}
	if err := hc.SetConfigAtPath("auth.aws.password", nil, ""); err != nil {
		t.Fatal(err)
	}
	after, _ := hc.GetAuthContext("aws")
	if after.Password != "" || after.Username != "user" {
		t.Fatal("incorrect field reset")
	}
	if err := hc.SetConfigAtPath("auth.aws.unknown", "bad", ""); err == nil {
		t.Fatal("unknown auth field accepted")
	}
	if err := hc.SetConfigAtPath("auth.aws", map[string]interface{}{"type": "basic", "username": "replacement"}, ""); err != nil {
		t.Fatal(err)
	}
	after, _ = hc.GetAuthContext("aws")
	if after.Username != "replacement" {
		t.Fatal("replacement not applied")
	}
}

func TestClientSessionsDoNotShareMutations(t *testing.T) {
	parent := sessionTestHandler()
	parent.ReplaceAuthContext("widgets", &dto.AuthCtx{AuthConfig: omnisdk.AuthConfig{Type: "basic", Password: "original"}})
	first := parent.NewClientContext().(*standardHandlerContext)
	second := parent.NewClientContext().(*standardHandlerContext)
	if err := first.SetConfigAtPath("auth.widgets.password", "changed", ""); err != nil {
		t.Fatal(err)
	}
	auth, err := second.GetAuthContext("widgets")
	if err != nil || auth.Password != "original" {
		t.Fatal("auth leaked across client sessions")
	}
	binding := map[string]interface{}{"provider": "widgets", "tenancy": "account", "auth": map[string]interface{}{"type": "bearer", "credentials": "token"}}
	if err := first.SetConfigAtPath("providers.private", binding, ""); err != nil {
		t.Fatal(err)
	}
	args, err := second.ProviderSessionArgs(context.Background())
	if err != nil || len(args.ProviderAliases) != 0 {
		t.Fatal("provider binding leaked across client sessions")
	}
	queryClone := first.Clone().(*standardHandlerContext)
	args, err = queryClone.ProviderSessionArgs(context.Background())
	if err != nil || args.ProviderAliases["private"].Auth == nil {
		t.Fatal("query clone lost its client session")
	}
}

func TestClientProviderCachesAreIsolated(t *testing.T) {
	parent := sessionTestHandler()
	first := parent.NewClientContext().(*standardHandlerContext)
	second := parent.NewClientContext().(*standardHandlerContext)
	first.providers["widgets"] = nil
	if _, exists := second.providers["widgets"]; exists {
		t.Fatal("client provider registration leaked into another client's cache")
	}
	if _, exists := parent.providers["widgets"]; exists {
		t.Fatal("client provider registration leaked into the parent cache")
	}
}

func TestProviderSessionRetainsPageLimit(t *testing.T) {
	hc := sessionTestHandler()
	hc.runtimeContext.HTTPPageLimit = 2
	args, err := hc.ProviderSessionArgs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if args.Tuning.PageLimit != 2 {
		t.Fatalf("session lost configured page limit: %d", args.Tuning.PageLimit)
	}
}
