package registry_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/stackql-provider-registry/signing/pqsig"
	"github.com/stackql/stackql/internal/stackql/registry"
)

func TestLocalRegistrationKeepsHostSQLAndCredentialDefaults(t *testing.T) {
	root := t.TempDir()
	providerDir := filepath.Join(root, "src", "demo", "v1.2.3")
	if err := os.MkdirAll(providerDir, 0755); err != nil {
		t.Fatal(err)
	}
	const doc = `name: demo
config:
  minStackQLVersion: v1.0.0
  auth:
    type: bearer
    credentialsenvvar: DEMO_TOKEN
    successor:
      type: null
  sqlExternalTables:
    widgets:
      catalogName: catalog
      schemaName: public
      name: widgets
      columns:
        - name: id
          type: integer
          oid: 23
`
	if err := os.WriteFile(filepath.Join(providerDir, "provider.yaml"), []byte(doc), 0600); err != nil {
		t.Fatal(err)
	}
	reg, err := registry.NewRegistry(omnisdk.RegistryConfig{LocalDocRoot: root, VerifyConfig: &pqsig.VerifierConfig{NopVerify: true}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	available := reg.ListLocallyAvailableProviders()
	if len(available["demo"].Versions) != 1 {
		t.Fatalf("missing local provider: %v", available)
	}
	loaded, err := reg.LoadProviderByName("demo", "")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.GetName() != "demo" || loaded.GetMinStackQLVersion() != "v1.0.0" {
		t.Fatal("lost registration metadata")
	}
	auth, ok := loaded.GetAuth()
	if !ok || auth.KeyEnvVar != "DEMO_TOKEN" || auth.Successor == nil {
		t.Fatal("lost credential defaults")
	}
	tables, err := loaded.ExternalTables()
	if err != nil {
		t.Fatal(err)
	}
	table := tables["widgets"]
	if table.GetName() != "widgets" || table.GetSchemaName() != "public" || len(table.GetColumns()) != 1 || table.GetColumns()[0].GetOid() != 23 {
		t.Fatal("lost external SQL metadata")
	}
}

func TestLatestVersionUsesSemanticOrdering(t *testing.T) {
	latest, err := registry.LatestVersion([]string{"v0.13.9", "v0.13.100", "v0.13.10"})
	if err != nil || latest != "v0.13.100" {
		t.Fatalf("latest=%q err=%v", latest, err)
	}
}

func newErrorStatusRegistry(t *testing.T, status int) registry.RegistryAPI {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = w.Write([]byte("<html>error page</html>"))
	}))
	t.Cleanup(srv.Close)
	reg, err := registry.NewRegistry(omnisdk.RegistryConfig{
		RegistryURL:      srv.URL,
		LocalDocRoot:     t.TempDir(),
		AllowSrcDownload: true,
		VerifyConfig:     &pqsig.VerifierConfig{NopVerify: true},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

func TestRemoteErrorStatusIsReported(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusInternalServerError} {
		want := fmt.Sprintf("HTTP %d", status)
		t.Run(want, func(t *testing.T) {
			reg := newErrorStatusRegistry(t, status)

			if _, err := reg.ListAllAvailableProviders(); err == nil || !strings.Contains(err.Error(), want) {
				t.Errorf("ListAllAvailableProviders error = %v, want it to mention %q", err, want)
			}
			if _, err := reg.ListAllProviderVersions("demo"); err == nil || !strings.Contains(err.Error(), want) {
				t.Errorf("ListAllProviderVersions error = %v, want it to mention %q", err, want)
			}
			if err := reg.PullAndPersistProviderArchive("demo", "v1.0.0"); err == nil || !strings.Contains(err.Error(), want) {
				t.Errorf("PullAndPersistProviderArchive error = %v, want it to mention %q", err, want)
			}
			if _, err := reg.GetDocBytes("demo/v1.0.0/provider.yaml"); err == nil || !strings.Contains(err.Error(), want) {
				t.Errorf("GetDocBytes error = %v, want it to mention %q", err, want)
			}
		})
	}
}
