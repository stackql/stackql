package mcp_server //nolint:revive // fine for now

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// approvalStateTTL bounds how long an approval prompt stays answerable.  It
// also bounds replay across instances that share a key but not nonce state.
const approvalStateTTL = 10 * time.Minute

// minApprovalKeyBytes is the shortest key accepted from approval_key_env_var.
const minApprovalKeyBytes = 32

// approvalStateVersion tags the RequestState payload layout.
const approvalStateVersion = 1

// approvalBinding is what an approval answer must match on the retry: the
// tool, a digest over its canonical arguments and SQL, and the session the
// prompt was issued on (empty on stdio and on sessionless HTTP, where the
// per-process or shared key carries the binding instead).
type approvalBinding struct {
	tool    string
	digest  string
	session string
}

// approvalClaims is the signed RequestState payload.
type approvalClaims struct {
	V       int    `json:"v"`
	Tool    string `json:"tool"`
	Digest  string `json:"digest"`
	Session string `json:"sid"`
	Nonce   string `json:"nonce"`
	Exp     int64  `json:"exp"` // unix milliseconds
}

// approvalBinder issues and verifies the RequestState that ties an approval
// answer to the prompt that asked for it (issue #827).  The state is
// authenticated with HMAC-SHA256 under a server key, so sessionless
// transports need no server-side pending table.  A consumed-nonce set blocks
// replay inside one process; across instances that share the key, replay is
// bounded only by approvalStateTTL.
//
// This is workflow integrity against stale, altered or unsolicited answers.
// It cannot prove a human saw the prompt: a client that lies about consent
// can still accept its own prompt.
type approvalBinder struct {
	key []byte
	ttl time.Duration
	now func() time.Time

	mu       sync.Mutex
	consumed map[string]time.Time // nonce -> expiry
}

// newApprovalBinder returns a binder keyed from server.approval_key_env_var
// when set, otherwise from a random per-process key.  The key value is never
// included in errors or logs.
func newApprovalBinder(cfg *Config) (*approvalBinder, error) {
	var key []byte
	envVar := ""
	if cfg != nil {
		envVar = cfg.Server.ApprovalKeyEnvVar
	}
	if envVar != "" {
		v := os.Getenv(envVar)
		if v == "" {
			return nil, fmt.Errorf("server.approval_key_env_var names an unset or empty env var %q", envVar)
		}
		if len(v) < minApprovalKeyBytes {
			return nil, fmt.Errorf("server.approval_key_env_var %q must hold at least %d bytes", envVar, minApprovalKeyBytes)
		}
		key = []byte(v)
	} else {
		key = make([]byte, sha256.Size)
		if _, err := rand.Read(key); err != nil {
			return nil, fmt.Errorf("approval key: %w", err)
		}
	}
	return &approvalBinder{
		key:      key,
		ttl:      approvalStateTTL,
		now:      time.Now,
		consumed: map[string]time.Time{},
	}, nil
}

// approvalDigest is the SHA-256 over the tool name, the canonical JSON of
// the typed arguments the handler will run with, and the SQL.  Typed
// arguments marshal in struct field order, so the encoding is stable.
func approvalDigest(tool string, args any, sql string) (string, error) {
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return "", fmt.Errorf("canonical arguments: %w", err)
	}
	canonical, err := json.Marshal(struct {
		Tool string          `json:"tool"`
		Args json.RawMessage `json:"args"`
		SQL  string          `json:"sql"`
	}{Tool: tool, Args: argsJSON, SQL: sql})
	if err != nil {
		return "", fmt.Errorf("canonical arguments: %w", err)
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

// issue returns a fresh RequestState for the prompt bound to b.
func (a *approvalBinder) issue(b approvalBinding) (string, error) {
	nonce := make([]byte, 16) //nolint:mnd // 128-bit nonce
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("approval nonce: %w", err)
	}
	payload, err := json.Marshal(approvalClaims{
		V:       approvalStateVersion,
		Tool:    b.tool,
		Digest:  b.digest,
		Session: b.session,
		Nonce:   hex.EncodeToString(nonce),
		Exp:     a.now().Add(a.ttl).UnixMilli(),
	})
	if err != nil {
		return "", fmt.Errorf("approval state: %w", err)
	}
	enc := base64.RawURLEncoding
	return enc.EncodeToString(payload) + "." + enc.EncodeToString(a.mac(payload)), nil
}

func (a *approvalBinder) mac(payload []byte) []byte {
	h := hmac.New(sha256.New, a.key)
	h.Write(payload)
	return h.Sum(nil)
}

var (
	errApprovalStateMissing  = errors.New("no approval state was echoed with the answer")
	errApprovalStateInvalid  = errors.New("approval state is malformed or was not issued by this server")
	errApprovalStateMismatch = errors.New("approval state does not match this call")
	errApprovalStateExpired  = errors.New("approval state has expired")
	errApprovalStateReplayed = errors.New("approval state was already used")
)

// verify checks state against the current call and consumes its nonce.  It
// runs before the answer is read, so a failed check never executes the tool.
// A failed check consumes nothing, so the legitimate retry still succeeds.
func (a *approvalBinder) verify(state string, b approvalBinding) error {
	if state == "" {
		return errApprovalStateMissing
	}
	payloadPart, sigPart, found := strings.Cut(state, ".")
	if !found {
		return errApprovalStateInvalid
	}
	enc := base64.RawURLEncoding
	payload, err := enc.DecodeString(payloadPart)
	if err != nil {
		return errApprovalStateInvalid
	}
	sig, err := enc.DecodeString(sigPart)
	if err != nil || !hmac.Equal(sig, a.mac(payload)) {
		return errApprovalStateInvalid
	}
	var c approvalClaims
	if uerr := json.Unmarshal(payload, &c); uerr != nil || c.V != approvalStateVersion || c.Nonce == "" {
		return errApprovalStateInvalid
	}
	if c.Tool != b.tool || c.Session != b.session ||
		subtle.ConstantTimeCompare([]byte(c.Digest), []byte(b.digest)) != 1 {
		return errApprovalStateMismatch
	}
	now := a.now()
	exp := time.UnixMilli(c.Exp)
	if !now.Before(exp) {
		return errApprovalStateExpired
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for n, e := range a.consumed {
		if !now.Before(e) {
			delete(a.consumed, n)
		}
	}
	if _, used := a.consumed[c.Nonce]; used {
		return errApprovalStateReplayed
	}
	a.consumed[c.Nonce] = exp
	return nil
}
