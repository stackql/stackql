package mcp_server //nolint:revive // fine for now

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/stackql/stackql/pkg/mcp_server/policy"
	"github.com/stackql/stackql/pkg/sink"
)

// File uploads (io.modelcontextprotocol/files), inline transfer only.  The
// server advertises maxInlineBytes and no upload capability, so a compliant
// client sends files inline in a FileInput's base64 data and the -32021
// upload path never applies.  attach_policy_document accepts one policy or
// compliance document as text and exposes it read-only as a resource.
const (
	attachPolicyToolName   = "attach_policy_document"
	policyURIStem          = "stackql://policies/"
	policyResourceTemplate = policyURIStem + "{id}"
	// maxStoredPolicies bounds process memory; the oldest document is
	// dropped first.
	maxStoredPolicies = 100
	policyIDBytes     = 12
	fileDigestSHA256  = "sha-256"
	octetStreamType   = "application/octet-stream"
)

// policyAccept is the x-mcp-file accept list.
var policyAccept = []string{ //nolint:gochecknoglobals // read-only schema data
	".rego", ".yaml", ".yml", ".json", ".md", ".txt", ".csv",
	"text/plain", "application/json", "application/yaml", "text/markdown",
}

// fileInput is the FileInput wire shape.  Data stays raw so that a null value
// (undecodable inline data) is told apart from an absent one.
type fileInput struct {
	ID       string          `json:"id"`
	Name     string          `json:"name,omitempty"`
	MIMEType string          `json:"mimeType"`
	Size     int64           `json:"size"`
	Digest   *fileDigest     `json:"digest,omitempty"`
	Data     json.RawMessage `json:"data,omitempty"`
}

type fileDigest struct {
	Algorithm string `json:"algorithm"`
	Value     string `json:"value"`
}

type policyDocumentInput struct {
	Document fileInput `json:"document"`
	Label    string    `json:"label,omitempty"`
}

// policyDocumentInputSchema is hand-written: x-mcp-file sits on the file
// argument, never on the schema root.
func policyDocumentInputSchema(maxBytes int64) map[string]any {
	str := map[string]any{"type": "string"}
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"document": map[string]any{
				"type":        "object",
				"description": "The document as an MCP FileInput with its bytes inline in data (base64).",
				"x-mcp-file":  map[string]any{"accept": policyAccept, "maxSize": maxBytes},
				"properties": map[string]any{
					"id":       map[string]any{"type": "string", "minLength": 1, "maxLength": 128},
					"name":     str,
					"mimeType": str,
					"size":     map[string]any{"type": "integer", "minimum": 0},
					"digest": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"algorithm": str,
							"value":     str,
						},
						"required": []string{"algorithm", "value"},
					},
					"data": str,
				},
				"required": []string{"id", "mimeType", "size"},
			},
			"label": map[string]any{"type": "string", "description": "Optional name for the document."},
		},
		"required": []string{"document"},
	}
}

// policyDocument is the data carrier for one stored document.
type policyDocument interface {
	ID() string
	URI() string
	Label() string
	Name() string
	MIMEType() string
	Size() int64
	// Digest is the sha-256 of the bytes, base64url without padding.
	Digest() string
	Text() string
}

type standardPolicyDocument struct {
	id, label, name, mimeType, digest, text string
}

func newPolicyDocument(id, label, name, mimeType string, content []byte) policyDocument {
	sum := sha256.Sum256(content)
	return &standardPolicyDocument{
		id: id, label: label, name: name, mimeType: mimeType,
		digest: base64.RawURLEncoding.EncodeToString(sum[:]), text: string(content),
	}
}

func (d *standardPolicyDocument) ID() string       { return d.id }
func (d *standardPolicyDocument) URI() string      { return policyURIStem + d.id }
func (d *standardPolicyDocument) Label() string    { return d.label }
func (d *standardPolicyDocument) Name() string     { return d.name }
func (d *standardPolicyDocument) MIMEType() string { return d.mimeType }
func (d *standardPolicyDocument) Size() int64      { return int64(len(d.text)) }
func (d *standardPolicyDocument) Digest() string   { return d.digest }
func (d *standardPolicyDocument) Text() string     { return d.text }

// policyStore keeps attached documents in process memory, oldest dropped
// first past maxStoredPolicies.  Documents do not survive a restart.
type policyStore struct {
	mu    sync.Mutex
	byID  map[string]policyDocument
	order []string
}

func newPolicyStore() *policyStore {
	return &policyStore{byID: map[string]policyDocument{}}
}

func (s *policyStore) add(label, name, mimeType string, content []byte) (policyDocument, error) {
	b := make([]byte, policyIDBytes)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("policy id: %w", err)
	}
	doc := newPolicyDocument(base64.RawURLEncoding.EncodeToString(b), label, name, mimeType, content)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byID[doc.ID()] = doc
	s.order = append(s.order, doc.ID())
	for len(s.order) > maxStoredPolicies {
		delete(s.byID, s.order[0])
		s.order = s.order[1:]
	}
	return doc, nil
}

func (s *policyStore) get(id string) (policyDocument, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	doc, ok := s.byID[id]
	return doc, ok
}

// decodePolicyDocument applies the validation order: data present when size
// is positive, valid base64 whose decoded length equals size, the inline
// limit, the digest when supplied, UTF-8 text, the declared type, and a
// non-empty document.  The returned error is the sentence the caller sees.
func decodePolicyDocument(in fileInput, maxBytes int64) ([]byte, error) {
	if in.Size > 0 && len(in.Data) == 0 {
		return nil, fmt.Errorf("document.data is missing: %s accepts files inline only, "+
			"so send the file's bytes base64-encoded in document.data", attachPolicyToolName)
	}
	var content []byte
	if len(in.Data) > 0 {
		var encoded string
		// A null value unmarshals into a string without error; it is undecodable data.
		if err := json.Unmarshal(in.Data, &encoded); err != nil || string(in.Data) == "null" {
			return nil, fmt.Errorf("document.data must be a base64 string, not %s", string(in.Data))
		}
		decoded, err := base64.StdEncoding.Strict().DecodeString(encoded)
		if err != nil {
			return nil, fmt.Errorf("document.data is not valid RFC 4648 base64 (standard alphabet, padded): %w", err)
		}
		content = decoded
	}
	if int64(len(content)) != in.Size {
		return nil, fmt.Errorf("document.size is %d but document.data decodes to %d bytes; send the exact size",
			in.Size, len(content))
	}
	if int64(len(content)) > maxBytes {
		return nil, fmt.Errorf("the document is %d bytes, over this server's %d byte inline limit "+
			"(server.extensions.files.max_inline_bytes)", len(content), maxBytes)
	}
	if in.Digest != nil {
		if !strings.EqualFold(in.Digest.Algorithm, fileDigestSHA256) {
			return nil, fmt.Errorf("document.digest.algorithm %q is not supported; use %q", in.Digest.Algorithm, fileDigestSHA256)
		}
		sum := sha256.Sum256(content)
		if in.Digest.Value != base64.RawURLEncoding.EncodeToString(sum[:]) {
			return nil, fmt.Errorf("document.digest does not match the decoded bytes " +
				"(sha-256, base64url without padding); the file was altered or truncated")
		}
	}
	if !utf8.Valid(content) {
		return nil, fmt.Errorf("the document is binary, not UTF-8 text; %s accepts text documents only", attachPolicyToolName)
	}
	if !policyMIMETypeAccepted(in.MIMEType) {
		return nil, fmt.Errorf("document.mimeType %q is not accepted; use a text type such as text/plain, "+
			"application/json, application/yaml or text/markdown", in.MIMEType)
	}
	if len(content) == 0 {
		return nil, fmt.Errorf("the document is empty; attach a document with content")
	}
	return content, nil
}

// policyMIMETypeAccepted accepts the listed media types, any text/* type and
// application/octet-stream, which is judged by content (UTF-8 above).
func policyMIMETypeAccepted(mimeType string) bool {
	base := strings.ToLower(strings.TrimSpace(strings.SplitN(mimeType, ";", 2)[0]))
	if base == octetStreamType || strings.HasPrefix(base, "text/") {
		return true
	}
	for _, accepted := range policyAccept {
		if base == accepted {
			return true
		}
	}
	return false
}

// servedPolicyMIMEType is the type resources/read reports.
func servedPolicyMIMEType(declared string) string {
	base := strings.ToLower(strings.TrimSpace(strings.SplitN(declared, ";", 2)[0]))
	if base == "" || base == octetStreamType {
		return "text/plain"
	}
	return base
}

// policyAuditArgs keeps only name, mimeType, size and digest: the document's
// data never reaches an audit record.
func policyAuditArgs(args any) map[string]any {
	in, ok := args.(policyDocumentInput)
	if !ok {
		return nil
	}
	out := map[string]any{"mimeType": in.Document.MIMEType, "size": in.Document.Size}
	if in.Document.Name != "" {
		out["name"] = in.Document.Name
	}
	if in.Document.Digest != nil {
		out["digest"] = in.Document.Digest.Algorithm + ":" + in.Document.Digest.Value
	}
	return out
}

// registerPolicyDocumentTool publishes attach_policy_document and the
// stackql://policies/{id} template.  Like pull_provider it is classified as a
// select, so it is allowed in every mode: it touches no cloud resource and
// writes only process memory.
func registerPolicyDocumentTool(
	server *mcp.Server, cfg *Config, auditSink sink.Sink, approvals *approvalBinder, descriptions toolDescriptions,
) error {
	maxBytes := cfg.Server.Extensions.Files.GetMaxInlineBytes()
	store := newPolicyStore()
	err := addToolWithGate(
		server, cfg, auditSink, approvals, descriptions,
		toolGate{toolName: attachPolicyToolName, defaultClass: policy.QueryClassSelect, extractArgs: policyAuditArgs},
		&mcp.Tool{
			Name:        attachPolicyToolName,
			InputSchema: policyDocumentInputSchema(maxBytes),
			Annotations: &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: boolPtr(false)},
		},
		func(_ context.Context, _ *mcp.CallToolRequest, args policyDocumentInput) (*mcp.CallToolResult, map[string]any, error) {
			content, decodeErr := decodePolicyDocument(args.Document, maxBytes)
			if decodeErr != nil {
				return nil, nil, decodeErr
			}
			doc, addErr := newPolicyDocumentFrom(store, args, content)
			if addErr != nil {
				return nil, nil, addErr
			}
			size := doc.Size()
			summary := fmt.Sprintf("Stored %s (%s, %d bytes, sha-256 %s) at %s. Policy documents are held in "+
				"server memory and do not survive a restart.", describePolicy(doc), doc.MIMEType(), size, doc.Digest(), doc.URI())
			out := map[string]any{
				"uri": doc.URI(), "id": doc.ID(), "label": doc.Label(), "name": doc.Name(),
				"mimeType": doc.MIMEType(), "size": size, "digest": fileDigestSHA256 + ":" + doc.Digest(),
			}
			return &mcp.CallToolResult{Content: []mcp.Content{
				&mcp.TextContent{Text: summary},
				&mcp.ResourceLink{URI: doc.URI(), Name: describePolicy(doc), MIMEType: doc.MIMEType(), Size: &size},
			}}, out, nil
		},
	)
	if err != nil || !cfg.IsToolEnabled(attachPolicyToolName) {
		return err
	}
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "stackql_policy_documents",
		Title:       "Attached policy documents",
		Description: "Documents attached with attach_policy_document, read-only; held in memory until restart.",
		URITemplate: policyResourceTemplate,
	}, func(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		doc, ok := store.get(strings.TrimPrefix(req.Params.URI, policyURIStem))
		if !ok || doc.URI() != req.Params.URI {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{
			URI: doc.URI(), MIMEType: doc.MIMEType(), Text: doc.Text(),
		}}}, nil
	})
	return nil
}

func newPolicyDocumentFrom(store *policyStore, args policyDocumentInput, content []byte) (policyDocument, error) {
	return store.add(args.Label, args.Document.Name, servedPolicyMIMEType(args.Document.MIMEType), content)
}

func describePolicy(doc policyDocument) string {
	switch {
	case doc.Label() != "":
		return fmt.Sprintf("policy document %q", doc.Label())
	case doc.Name() != "":
		return fmt.Sprintf("policy document %q", doc.Name())
	default:
		return "policy document " + doc.ID()
	}
}
