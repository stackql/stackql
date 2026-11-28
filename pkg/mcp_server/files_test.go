package mcp_server //nolint:testpackage,revive // exercise internal wiring

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/sirupsen/logrus"

	"github.com/stackql/stackql/pkg/mcp_server/audit"
)

const regoPolicy = "package stackql.guardrails\n\ndeny[msg] { input.public; msg := \"public bucket\" }\n"

func fileArg(content []byte, mimeType string) map[string]any {
	sum := sha256.Sum256(content)
	return map[string]any{
		"id": "f1", "name": "guardrails.rego", "mimeType": mimeType, "size": len(content),
		"digest": map[string]any{"algorithm": "sha-256", "value": base64.RawURLEncoding.EncodeToString(sum[:])},
		"data":   base64.StdEncoding.EncodeToString(content),
	}
}

func decodeArg(t *testing.T, raw map[string]any) fileInput {
	t.Helper()
	b, _ := json.Marshal(raw)
	var in fileInput
	if err := json.Unmarshal(b, &in); err != nil {
		t.Fatalf("decode FileInput: %v", err)
	}
	return in
}

func TestDecodePolicyDocumentValidationOrder(t *testing.T) {
	valid := []byte(regoPolicy)
	with := func(mutate func(map[string]any)) map[string]any {
		arg := fileArg(valid, "text/plain")
		mutate(arg)
		return arg
	}
	cases := []struct {
		name string
		arg  map[string]any
		want string
	}{
		{"valid", fileArg(valid, "text/plain"), ""},
		{"octet-stream judged by content", fileArg(valid, "application/octet-stream"), ""},
		{"missing data", with(func(a map[string]any) { delete(a, "data") }), "document.data is missing"},
		{"null data", with(func(a map[string]any) { a["data"] = nil }), "must be a base64 string"},
		{"bad base64", with(func(a map[string]any) { a["data"] = "not base64!" }), "not valid RFC 4648 base64"},
		{"size mismatch", with(func(a map[string]any) { a["size"] = len(valid) + 1 }), "decodes to"},
		{"over inline limit", fileArg(bytes.Repeat([]byte("a"), 129), "text/plain"), "inline limit"},
		{"digest mismatch", with(func(a map[string]any) {
			a["digest"] = map[string]any{"algorithm": "sha-256", "value": "AAAA"}
		}), "digest does not match"},
		{"digest algorithm", with(func(a map[string]any) {
			a["digest"] = map[string]any{"algorithm": "md5", "value": "AAAA"}
		}), "not supported"},
		{"binary", fileArg([]byte{0xff, 0xfe, 0x00, 0x01}, "application/octet-stream"), "binary"},
		{"declared type", fileArg(valid, "image/png"), "is not accepted"},
		{"empty", fileArg(nil, "text/plain"), "empty"},
		{"empty without data", with(func(a map[string]any) { delete(a, "data"); delete(a, "digest"); a["size"] = 0 }), "empty"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			content, err := decodePolicyDocument(decodeArg(t, tc.arg), 128)
			if tc.want == "" {
				if err != nil || !bytes.Equal(content, valid) {
					t.Fatalf("want the decoded document, got %q, %v", content, err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want it to contain %q", err, tc.want)
			}
		})
	}
}

func filesConfig() *Config {
	cfg := readOnlyConfig()
	cfg.Server.Extensions.Files.Enabled = true
	return cfg
}

// The tool is allowed in read_only mode, round-trips the bytes through the
// resource template and never needs the client to declare the extension.
// policyToolSchema returns the published input schema of the policy tool and
// checks its annotations.
func policyToolSchema(t *testing.T, cs *mcp.ClientSession) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	list, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, tool := range list.Tools {
		if tool.Name != attachPolicyToolName {
			continue
		}
		hints := tool.Annotations
		if hints == nil || hints.ReadOnlyHint || hints.DestructiveHint == nil || *hints.DestructiveHint {
			t.Fatalf("annotations = %+v", hints)
		}
		var schema map[string]any
		raw, _ := json.Marshal(tool.InputSchema)
		_ = json.Unmarshal(raw, &schema)
		return schema
	}
	t.Fatalf("%s is not published", attachPolicyToolName)
	return nil
}

func TestAttachPolicyDocumentRoundTrip(t *testing.T) {
	cs := connectInProcess(t, filesConfig(), &testBackend{})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	schema := policyToolSchema(t, cs)
	document := schema["properties"].(map[string]any)["document"].(map[string]any)
	if _, onRoot := schema["x-mcp-file"]; onRoot || document["x-mcp-file"] == nil {
		t.Fatalf("x-mcp-file must sit on the document property only: %v", schema)
	}
	if document["x-mcp-file"].(map[string]any)["maxSize"].(float64) != 1<<20 {
		t.Fatalf("maxSize = %v", document["x-mcp-file"])
	}

	res := callTool(t, cs, attachPolicyToolName, map[string]any{
		"document": fileArg([]byte(regoPolicy), "text/plain"), "label": "guardrails"})
	if res.IsError || len(res.Content) != 2 {
		t.Fatalf("attach failed: %+v", res)
	}
	link, ok := res.Content[1].(*mcp.ResourceLink)
	if !ok || !strings.HasPrefix(link.URI, policyURIStem) || link.Size == nil || *link.Size != int64(len(regoPolicy)) {
		t.Fatalf("content[1] = %#v, want a resource_link", res.Content[1])
	}
	read, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: link.URI})
	if err != nil {
		t.Fatal(err)
	}
	if read.Contents[0].Text != regoPolicy || read.Contents[0].MIMEType != "text/plain" {
		t.Fatalf("read back %+v", read.Contents[0])
	}
	if _, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: policyURIStem + "unknown"}); err == nil {
		t.Fatal("an unknown policy document was readable")
	}
	rejected := callTool(t, cs, attachPolicyToolName, map[string]any{"document": fileArg(nil, "text/plain")})
	if !rejected.IsError || !strings.Contains(firstText(t, rejected), "empty") {
		t.Fatalf("empty document not rejected: %+v", rejected)
	}
}

// recordingAuditSink keeps every event it is given.
type recordingAuditSink struct{ events []audit.Event }

func (s *recordingAuditSink) Record(_ context.Context, payload any) error {
	if event, ok := payload.(audit.Event); ok {
		s.events = append(s.events, event)
	}
	return nil
}

func (s *recordingAuditSink) Close() error { return nil }

// Fails if a policy document's data, encoded or decoded, reaches an audit record.
func TestPolicyDocumentDataNeverAudited(t *testing.T) {
	cfg := filesConfig()
	cfg.Server.Audit.Disabled = false
	recorder := &recordingAuditSink{}
	cs := connectWithSink(t, cfg, &testBackend{}, recorder)
	content := []byte(regoPolicy + "# marker-" + strings.Repeat("z", 32) + "\n")
	arg := fileArg(content, "text/plain")
	if res := callTool(t, cs, attachPolicyToolName, map[string]any{"document": arg}); res.IsError {
		t.Fatalf("attach failed: %+v", res)
	}
	bad := fileArg(content, "text/plain")
	bad["size"] = len(content) + 5
	if res := callTool(t, cs, attachPolicyToolName, map[string]any{"document": bad}); !res.IsError {
		t.Fatal("size mismatch accepted")
	}
	if len(recorder.events) != 2 {
		t.Fatalf("recorded %d events, want 2", len(recorder.events))
	}
	for _, event := range recorder.events {
		raw, _ := json.Marshal(event)
		for _, secret := range []string{arg["data"].(string), "marker-", "stackql.guardrails"} {
			if strings.Contains(string(raw), secret) {
				t.Fatalf("audit record carries document data %q: %s", secret, raw)
			}
		}
		for key := range event.Args {
			if key != "name" && key != "mimeType" && key != "size" && key != "digest" {
				t.Fatalf("audit args carry %q: %v", key, event.Args)
			}
		}
	}
}

// The HTTP request log records method, path and status only, never bodies.
func TestLoggingHandlerDoesNotLogRequestBodies(t *testing.T) {
	buffer := &bytes.Buffer{}
	logger := logrus.New()
	logger.SetOutput(buffer)
	handler := loggingHandler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}), logger)
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"data":"c2VjcmV0LXBvbGljeQ=="}`))
	handler.ServeHTTP(httptest.NewRecorder(), request)
	if strings.Contains(buffer.String(), "c2VjcmV0LXBvbGljeQ") || !strings.Contains(buffer.String(), "POST") {
		t.Fatalf("log = %q", buffer.String())
	}
}

func TestPolicyStoreDropsOldestPastLimit(t *testing.T) {
	store := newPolicyStore()
	first, _ := store.add("", "", "text/plain", []byte("first"))
	for i := 0; i < maxStoredPolicies; i++ {
		if _, err := store.add("", "", "text/plain", []byte("x")); err != nil {
			t.Fatal(err)
		}
	}
	if _, ok := store.get(first.ID()); ok {
		t.Fatal("oldest document kept past the limit")
	}
	if len(store.byID) != maxStoredPolicies {
		t.Fatalf("store holds %d documents", len(store.byID))
	}
}
