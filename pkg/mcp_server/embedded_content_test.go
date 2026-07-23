package mcp_server //nolint:testpackage,revive // exercise internal wiring

import (
	"context"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestEmbeddedContent_LoadsAndValidates is the build-time gate for all
// embedded markdown: malformed frontmatter, duplicate names or unresolved
// placeholders fail here rather than at server start.
func TestEmbeddedContent_LoadsAndValidates(t *testing.T) {
	instructions, err := loadEmbeddedInstructions()
	if err != nil {
		t.Fatalf("instructions: %v", err)
	}
	if instructions == "" {
		t.Errorf("instructions should not be empty")
	}
	prompts, err := loadEmbeddedPrompts()
	if err != nil {
		t.Fatalf("prompts: %v", err)
	}
	if len(prompts) == 0 {
		t.Errorf("expected at least one embedded prompt")
	}
	resources, err := loadEmbeddedResources()
	if err != nil {
		t.Fatalf("resources: %v", err)
	}
	if len(resources) == 0 {
		t.Errorf("expected at least one embedded resource")
	}
}

// TestEmbeddedPrompt_WriteSafeSelect_ByteIdentical pins the acceptance
// criterion that the migrated prompt body matches the legacy Go constant.
func TestEmbeddedPrompt_WriteSafeSelect_ByteIdentical(t *testing.T) {
	prompts, err := loadEmbeddedPrompts()
	if err != nil {
		t.Fatalf("prompts: %v", err)
	}
	for _, p := range prompts {
		if p.Name() == "write_safe_select" {
			if p.Body() != ExplainerPromptWriteSafeSelectTool {
				t.Errorf("body mismatch.\nwant: %q\ngot:  %q", ExplainerPromptWriteSafeSelectTool, p.Body())
			}
			return
		}
	}
	t.Fatalf("write_safe_select not found among embedded prompts")
}

func TestSplitFrontmatter_Errors(t *testing.T) {
	if _, _, err := splitFrontmatter([]byte("no frontmatter here")); err == nil {
		t.Errorf("expected error for missing open delimiter")
	}
	if _, _, err := splitFrontmatter([]byte("---\nname: x\nno close")); err == nil {
		t.Errorf("expected error for missing close delimiter")
	}
	meta, body, err := splitFrontmatter([]byte("---\r\nname: x\r\n---\r\nbody line\r\n"))
	if err != nil {
		t.Fatalf("splitFrontmatter: %v", err)
	}
	if meta != "name: x" || body != "body line" {
		t.Errorf("unexpected split: meta=%q body=%q", meta, body)
	}
}

func TestEmbeddedPrompt_PlaceholderValidationAndRender(t *testing.T) {
	arg := &mcp.PromptArgument{Name: "region", Description: "cloud region", Required: true}
	p, err := newEmbeddedPrompt("p1", "desc", []*mcp.PromptArgument{arg}, "list resources in {{region}}")
	if err != nil {
		t.Fatalf("newEmbeddedPrompt: %v", err)
	}
	out, err := p.Render(map[string]string{"region": "us-east1"})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if out != "list resources in us-east1" {
		t.Errorf("unexpected render: %q", out)
	}
	if _, err := p.Render(nil); err == nil {
		t.Errorf("expected error for missing required argument")
	}
	if _, err := newEmbeddedPrompt("p2", "desc", nil, "uses {{undeclared}}"); err == nil {
		t.Errorf("expected error for unresolved placeholder")
	}
	if _, err := newEmbeddedPrompt("", "desc", nil, "body"); err == nil {
		t.Errorf("expected error for missing name")
	}
}

func TestLoadPromptsFrom_DetectsDuplicatesAndBadYAML(t *testing.T) {
	dup := fstest.MapFS{
		"prompts/a.md": {Data: []byte("---\nname: same\ndescription: d\n---\nbody a\n")},
		"prompts/b.md": {Data: []byte("---\nname: same\ndescription: d\n---\nbody b\n")},
	}
	if _, err := loadPromptsFrom(dup, "prompts"); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("expected duplicate name error, got %v", err)
	}
	bad := fstest.MapFS{
		"prompts/a.md": {Data: []byte("---\nname: x\nbogus_key: y\ndescription: d\n---\nbody\n")},
	}
	if _, err := loadPromptsFrom(bad, "prompts"); err == nil {
		t.Errorf("expected strict yaml error for unknown key")
	}
}

func TestLoadResourcesFrom_Defaults(t *testing.T) {
	fsys := fstest.MapFS{
		"resources/notes.md": {Data: []byte("---\nname: notes\ndescription: d\n---\nbody\n")},
	}
	resources, err := loadResourcesFrom(fsys, "resources")
	if err != nil {
		t.Fatalf("loadResourcesFrom: %v", err)
	}
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	r := resources[0]
	if r.URI() != "stackql://docs/notes" {
		t.Errorf("default uri wrong: %q", r.URI())
	}
	if r.MIMEType() != "text/markdown" {
		t.Errorf("default mime wrong: %q", r.MIMEType())
	}
}

func TestInstructions_SurfacedInInitialize(t *testing.T) {
	cs := connectInProcess(t, DefaultConfig(), &testBackend{})
	want, err := loadEmbeddedInstructions()
	if err != nil {
		t.Fatalf("loadEmbeddedInstructions: %v", err)
	}
	got := cs.InitializeResult().Instructions
	if got == "" || got != want {
		t.Errorf("initialize instructions mismatch.\nwant: %q\ngot:  %q", want, got)
	}
}

func TestInstructions_DisableInstructionsSuppresses(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DisableInstructions = true
	cs := connectInProcess(t, cfg, &testBackend{})
	if got := cs.InitializeResult().Instructions; got != "" {
		t.Errorf("instructions should be suppressed, got %q", got)
	}
}

func TestResources_ListAndRead(t *testing.T) {
	cs := connectInProcess(t, DefaultConfig(), &testBackend{})
	if cs.InitializeResult().Capabilities.Resources == nil {
		t.Fatalf("resources capability should be declared")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	list, err := cs.ListResources(ctx, nil)
	if err != nil {
		t.Fatalf("ListResources: %v", err)
	}
	var uri string
	for _, r := range list.Resources {
		if r.Name == "stackql_sql_dialect" {
			uri = r.URI
		}
	}
	if uri == "" {
		t.Fatalf("stackql_sql_dialect not listed: %+v", list.Resources)
	}
	res, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
	if err != nil {
		t.Fatalf("ReadResource: %v", err)
	}
	if len(res.Contents) != 1 {
		t.Fatalf("expected 1 content item, got %d", len(res.Contents))
	}
	c := res.Contents[0]
	if c.URI != uri || c.MIMEType != "text/markdown" || !strings.Contains(c.Text, "dialect") {
		t.Errorf("unexpected contents: uri=%q mime=%q text=%q", c.URI, c.MIMEType, c.Text)
	}
}

func TestResources_EnabledResourcesFilters(t *testing.T) {
	cfg := DefaultConfig()
	cfg.EnabledResources = []string{"some_other_resource"}
	cs := connectInProcess(t, cfg, &testBackend{})
	// With every embedded resource filtered out, the capability is absent.
	if cs.InitializeResult().Capabilities.Resources != nil {
		t.Errorf("resources capability should not be declared when all resources are filtered")
	}
}
