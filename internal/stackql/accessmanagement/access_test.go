package accessmanagement //nolint:testpackage // Exercises the private directory implementation.

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/hex"
	"encoding/pem"
	"testing"
	"testing/synctest"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stackql-labs/omnisdk/public/authentication"
	"github.com/stackql-labs/omnisdk/public/authorization"
	"github.com/stackql-labs/omnisdk/public/principal"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

//nolint:gocognit // Sequential session lifecycle assertions share an isolated directory fixture.
func TestCallerSessionsRefreshAndFailClosed(t *testing.T) {
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, _ := x509.MarshalPKIXPublicKey(pub)
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := initializeSchema(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	for _, stmt := range []string{
		`INSERT INTO stackql_access_principals VALUES ('alice','{}')`,
		`INSERT INTO stackql_access_groups VALUES ('readers')`,
		`INSERT INTO stackql_access_identities VALUES ('issuer','alice','alice')`,
		`INSERT INTO stackql_access_memberships VALUES ('alice','readers')`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	directory, _ := newDirectory(db)
	m, err := newManager(Config{Issuer: "issuer", Audience: "stackql", Keys: []VerificationKey{{ID: "test", Algorithm: "EdDSA", PEM: string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))}}, Rules: []authorization.Rule{{Subject: "group:readers", Resource: "operation/demo/list/listItems", Action: "execute"}}}, directory)
	if err != nil {
		t.Fatal(err)
	}
	sign := func(modify func(jwt.MapClaims), kid string, signingKey any) string {
		claims := jwt.MapClaims{"iss": "issuer", "sub": "alice", "aud": "stackql", "exp": time.Now().Add(time.Minute).Unix(), "iat": time.Now().Unix(), "groups": []string{"admins"}}
		if modify != nil {
			modify(claims)
		}
		token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
		token.Header["kid"] = kid
		value, err := token.SignedString(signingKey)
		if err != nil {
			t.Fatal(err)
		}
		return value
	}
	establish := func(token string) (authorization.Session, error) {
		caller, _, cancel, err := m.Establish(WithEvidence(context.Background(), authentication.Evidence{Token: token}))
		if cancel != nil {
			t.Cleanup(cancel)
		}
		return caller, err
	}
	valid := sign(nil, "test", key)
	caller, err := establish(valid)
	if err != nil {
		t.Fatal(err)
	}
	if len(caller.Principal().Groups()) != 1 || caller.Principal().Groups()[0] != "readers" {
		t.Fatal("JWT groups replaced directory membership")
	}
	for _, query := range []string{"SELECT 1", "SELECT count(*) FROM demo.service.items"} {
		stmt, _ := sqlparser.Parse(query)
		if err := CheckStatement(caller.Context(context.Background()), caller, stmt); err != nil {
			t.Fatal(err)
		}
	}
	for _, query := range []string{"DELETE FROM demo.service.items", "SELECT * FROM stackql_access_principals", "SELECT pg_read_file('/tmp/x')", "SELECT * FROM stackql_preview.iac.run"} {
		stmt, err := sqlparser.Parse(query)
		if err != nil {
			t.Fatal(err)
		}
		if CheckStatement(caller.Context(context.Background()), caller, stmt) == nil {
			t.Fatalf("allowed %s", query)
		}
	}
	_, other, _ := ed25519.GenerateKey(rand.Reader)
	for name, token := range map[string]string{
		"missing": "", "signature": sign(nil, "test", other), "unknown key": sign(nil, "other", key),
		"issuer":     sign(func(c jwt.MapClaims) { c["iss"] = "other" }, "test", key),
		"audience":   sign(func(c jwt.MapClaims) { c["aud"] = "other" }, "test", key),
		"expiry":     sign(func(c jwt.MapClaims) { delete(c, "exp") }, "test", key),
		"expired":    sign(func(c jwt.MapClaims) { c["exp"] = time.Now().Add(-time.Minute).Unix() }, "test", key),
		"future":     sign(func(c jwt.MapClaims) { c["iat"] = time.Now().Add(time.Hour).Unix() }, "test", key),
		"not before": sign(func(c jwt.MapClaims) { c["nbf"] = time.Now().Add(time.Hour).Unix() }, "test", key),
	} {
		if _, err := establish(token); err == nil {
			t.Fatalf("accepted %s", name)
		}
	}
	if _, err := db.Exec(`DELETE FROM stackql_access_memberships`); err != nil {
		t.Fatal(err)
	}
	next, err := establish(valid)
	if err != nil {
		t.Fatal(err)
	}
	stmt, _ := sqlparser.Parse("SELECT 1")
	if CheckStatement(next.Context(context.Background()), next, stmt) == nil {
		t.Fatal("revoked membership retained")
	}
	if len(caller.Principal().Groups()) != 1 {
		t.Fatal("existing snapshot mutated")
	}
	if _, err := db.Exec(`UPDATE stackql_access_principals SET attributes = 'null'`); err != nil {
		t.Fatal(err)
	}
	if _, err := establish(valid); err == nil {
		t.Fatal("accepted malformed attributes")
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := establish(valid); err == nil {
		t.Fatal("directory outage allowed access")
	}
}

func TestConfigurationSource(t *testing.T) {
	for _, raw := range []string{`{"path":"access.json"}`, "access.json"} {
		path, err := configurationPath(raw)
		if err != nil || path != "access.json" {
			t.Fatalf("%q: path=%q error=%v", raw, path, err)
		}
	}
	for _, raw := range []string{`{}`, `{"path":" "}`, `{"path":null}`, `{"path":3}`, `{"path":"access.json","extra":true}`, `{"path":"access.json"}{}`} {
		if _, err := configurationPath(raw); err == nil {
			t.Fatalf("accepted %q", raw)
		}
	}
}

func TestBearerDeadlineAndRemoteBoundary(t *testing.T) {
	directory, err := principal.NewMemoryDirectoryFromBytes([]byte(`{"groups":["readers"],"principals":[{"id":"alice","identities":[{"issuer":"tokens","subject":"alice"}],"groups":["readers"]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256([]byte("test-token"))
	manager, err := newManager(Config{Type: "bearer", Issuer: "tokens", MaxSessionSeconds: 1, EnforceSessionDeadline: true,
		Tokens: []BearerToken{{Subject: "alice", SHA256: hex.EncodeToString(digest[:])}},
		Rules:  []authorization.Rule{{Subject: "group:readers", Resource: "host/sql/read", Action: "execute"}},
	}, directory)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("STACKQL_CALLER_TOKEN", "test-token")
	caller, ctx, cancel, err := manager.Establish(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer cancel()
	deadline, bounded := ctx.Deadline()
	if !bounded || time.Until(deadline) > time.Second || caller.Principal().ID() != "alice" {
		t.Fatal("unbounded or incorrect bearer session")
	}
	if _, _, _, err := manager.Establish(WithEvidence(context.Background(), authentication.Evidence{})); err == nil {
		t.Fatal("remote request inherited the process token")
	}
	cancel()
	if ctx.Err() == nil {
		t.Fatal("cancelled caller context remained live")
	}
}

func TestModeConfigurationFailsClosed(t *testing.T) {
	for _, cfg := range []Config{
		{Type: "unknown"},
		{Type: "bearer", Issuer: "tokens", Tokens: []BearerToken{{Subject: "alice", SHA256: "00"}}},
		{Type: "local_os", Issuer: "host", Audience: "unexpected"},
		{Type: "jwt", Issuer: "issuer", ClientCAPEM: "unexpected"},
		{Type: "mtls", Issuer: "certs", ClientCAPEM: "not a CA", CertificateIdentity: "uri_san"},
	} {
		if _, _, err := configuredAuthenticator(cfg); err == nil {
			t.Fatalf("accepted mode %q", cfg.Type)
		}
	}
	manager, err := newManager(Config{Type: "local_os", Issuer: "host", Rules: []authorization.Rule{{Subject: "group:readers", Resource: "host/sql/read", Action: "execute"}}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := manager.Establish(WithEvidence(context.Background(), authentication.Evidence{})); err == nil {
		t.Fatal("remote request inherited the server OS identity")
	}
}

// Expiry prevents subsequent dispatch, rather than cancelling work already authorized.
//
//nolint:gocognit // Compare default, opted-in and caller deadlines against one authenticated session clock.
func TestQueryContinuesAfterTokenExpiryByDefault(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		pub, key, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		der, err := x509.MarshalPKIXPublicKey(pub)
		if err != nil {
			t.Fatal(err)
		}
		directory, err := principal.NewMemoryDirectoryFromBytes([]byte(`{"groups":["readers"],"principals":[{"id":"alice","identities":[{"issuer":"issuer","subject":"alice"}],"groups":["readers"]}]}`))
		if err != nil {
			t.Fatal(err)
		}
		cfg := Config{Issuer: "issuer", Audience: "stackql", MaxSessionSeconds: 1, Keys: []VerificationKey{{ID: "test", Algorithm: "EdDSA", PEM: string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der}))}}, Rules: []authorization.Rule{{Subject: "group:readers", Resource: "host/sql/read", Action: "execute"}}}
		token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, jwt.MapClaims{"iss": "issuer", "sub": "alice", "aud": "stackql", "exp": time.Now().Add(time.Minute).Unix()})
		token.Header["kid"] = "test"
		signed, err := token.SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		m, err := newManager(cfg, directory)
		if err != nil {
			t.Fatal(err)
		}
		parent, stop := context.WithCancel(WithEvidence(context.Background(), authentication.Evidence{Token: signed}))
		defer stop()
		_, execution, cancel, err := m.Establish(parent)
		if err != nil {
			t.Fatal(err)
		}
		defer cancel()
		if _, bounded := execution.Deadline(); bounded {
			t.Fatal("default execution acquired an authentication deadline")
		}
		explicitParent, stopExplicit := context.WithTimeout(parent, time.Minute)
		defer stopExplicit()
		_, explicitExecution, cancelExplicit, err := m.Establish(explicitParent)
		if err != nil {
			t.Fatal(err)
		}
		defer cancelExplicit()
		parentDeadline, _ := explicitParent.Deadline()
		deadline, bounded := explicitExecution.Deadline()
		if !bounded || !deadline.Equal(parentDeadline) {
			t.Fatal("explicit caller deadline was lost")
		}
		cfg.EnforceSessionDeadline = true
		opted, err := newManager(cfg, directory)
		if err != nil {
			t.Fatal(err)
		}
		_, limited, stopLimited, err := opted.Establish(parent)
		if err != nil {
			t.Fatal(err)
		}
		defer stopLimited()
		time.Sleep(8 * time.Hour)
		if limited.Err() != context.DeadlineExceeded {
			t.Fatal("opt-in session limit did not cancel execution")
		}
		if explicitExecution.Err() != context.DeadlineExceeded {
			t.Fatal("explicit caller timeout did not cancel execution")
		}

		if err := authorization.Check(execution, "host/sql/read", "execute"); err != nil {
			t.Fatalf("authorized query did not survive expiry: %v", err)
		}
		if _, _, cancel, err := m.Establish(parent); err == nil {
			cancel()
			t.Fatal("expired token dispatched a new query")
		}
		stop()
		if execution.Err() != context.Canceled {
			t.Fatal("explicit cancellation was lost")
		}
	})
}
