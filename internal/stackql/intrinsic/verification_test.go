package intrinsic //nolint:testpackage // tests the unexported verification configuration

import (
	"crypto/mldsa"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/stackql/stackql-provider-registry/signing/pqsig"
	dto "github.com/stackql/stackql/internal/stackql/runtimeconfig"
)

func TestDocumentVerificationConfig(t *testing.T) {
	v, err := documentVerifier(dto.RuntimeCtx{RegistryRaw: `{"verifyConfig":{"nopVerify":true}}`})
	if err != nil || v != nil {
		t.Fatalf("verification opt-out: %v", err)
	}
	key, err := mldsa.GenerateKey(mldsa.MLDSA65())
	if err != nil {
		t.Fatal(err)
	}
	config, err := json.Marshal(map[string]any{"verifyConfig": pqsig.VerifierConfig{
		PublicKeys: map[string]string{"test": hex.EncodeToString(key.PublicKey().Bytes())},
	}})
	if err != nil {
		t.Fatal(err)
	}
	v, err = documentVerifier(dto.RuntimeCtx{RegistryRaw: string(config)})
	if err != nil {
		t.Fatal(err)
	}
	doc, err := pqsig.Sign([]byte("name: example\n"), "test", key)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.Verify(doc); err != nil {
		t.Fatal(err)
	}
	for _, cfg := range []string{`{`, `{"verifyConfig":{"publicKeys":{}}}`, `{"verifyConfig":{"publicKeys":{"test":"invalid"}}}`} {
		if _, err := documentVerifier(dto.RuntimeCtx{RegistryRaw: cfg}); err == nil {
			t.Fatalf("accepted invalid config %s", cfg)
		}
	}
}
