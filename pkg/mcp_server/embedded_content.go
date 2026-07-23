package mcp_server //nolint:revive // fine for now

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Embedded content framework (issue #696).  MCP server instructions, prompts
// and resources are authored as markdown under content/ and compiled into the
// binary; adding or changing published content is a markdown-only edit.
//
//go:embed content
var embeddedContentFS embed.FS

const (
	embeddedInstructionsDir = "content/instructions"
	embeddedPromptsDir      = "content/prompts"
	embeddedResourcesDir    = "content/resources"

	defaultResourceMIMEType = "text/markdown"
	defaultResourceURIStem  = "stackql://docs/"

	frontmatterDelimiter = "---"
)

// placeholderRegexp matches {{argument}} substitution slots in prompt bodies.
var placeholderRegexp = regexp.MustCompile(`\{\{([A-Za-z0-9_]+)\}\}`)

// normalizeContent maps CRLF to LF and trims trailing newlines so checkout
// line endings and file-final newlines never alter published bytes.
func normalizeContent(raw []byte) string {
	return strings.TrimRight(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
}

// splitFrontmatter separates the leading YAML frontmatter block (fenced by
// "---" lines) from the markdown body.
func splitFrontmatter(raw []byte) (string, string, error) {
	text := normalizeContent(raw)
	open := frontmatterDelimiter + "\n"
	if !strings.HasPrefix(text, open) {
		return "", "", errors.New("missing frontmatter open delimiter")
	}
	rest := text[len(open):]
	closeMark := "\n" + frontmatterDelimiter + "\n"
	idx := strings.Index(rest, closeMark)
	if idx < 0 {
		return "", "", errors.New("missing frontmatter close delimiter")
	}
	return rest[:idx], rest[idx+len(closeMark):], nil
}

// listMarkdownFiles returns the .md files directly under dir, in the lexical
// order guaranteed by fs.ReadDir.  A missing directory yields no files.
func listMarkdownFiles(fsys fs.FS, dir string) ([]string, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// embeddedPrompt is the parsed carrier for one prompt markdown file.
type embeddedPrompt interface {
	Name() string
	Description() string
	Arguments() []*mcp.PromptArgument
	Body() string
	Render(args map[string]string) (string, error)
}

type standardEmbeddedPrompt struct {
	name        string
	description string
	arguments   []*mcp.PromptArgument
	body        string
}

// newEmbeddedPrompt validates required fields and that every {{placeholder}}
// in the body is a declared argument.
func newEmbeddedPrompt(
	name, description string,
	arguments []*mcp.PromptArgument,
	body string,
) (embeddedPrompt, error) {
	if name == "" || description == "" {
		return nil, errors.New("prompt requires name and description")
	}
	declared := map[string]bool{}
	for _, a := range arguments {
		if a.Name == "" {
			return nil, fmt.Errorf("prompt %q: argument requires a name", name)
		}
		declared[a.Name] = true
	}
	for _, m := range placeholderRegexp.FindAllStringSubmatch(body, -1) {
		if !declared[m[1]] {
			return nil, fmt.Errorf("prompt %q: unresolved placeholder {{%s}}", name, m[1])
		}
	}
	return &standardEmbeddedPrompt{
		name:        name,
		description: description,
		arguments:   arguments,
		body:        body,
	}, nil
}

func (p *standardEmbeddedPrompt) Name() string                     { return p.name }
func (p *standardEmbeddedPrompt) Description() string              { return p.description }
func (p *standardEmbeddedPrompt) Arguments() []*mcp.PromptArgument { return p.arguments }
func (p *standardEmbeddedPrompt) Body() string                     { return p.body }

// Render substitutes declared {{argument}} placeholders with caller-supplied
// values; a missing required argument is an error, a missing optional one
// substitutes the empty string.
func (p *standardEmbeddedPrompt) Render(args map[string]string) (string, error) {
	out := p.body
	for _, a := range p.arguments {
		v, ok := args[a.Name]
		if !ok && a.Required {
			return "", fmt.Errorf("prompt %q: missing required argument %q", p.name, a.Name)
		}
		out = strings.ReplaceAll(out, "{{"+a.Name+"}}", v)
	}
	return out, nil
}

// embeddedResource is the parsed carrier for one resource markdown file.
type embeddedResource interface {
	Name() string
	Description() string
	URI() string
	MIMEType() string
	Body() string
}

type standardEmbeddedResource struct {
	name        string
	description string
	uri         string
	mimeType    string
	body        string
}

func newEmbeddedResource(name, description, uri, mimeType, body string) (embeddedResource, error) {
	if name == "" || description == "" {
		return nil, errors.New("resource requires name and description")
	}
	if uri == "" || mimeType == "" {
		return nil, fmt.Errorf("resource %q: uri and mime type must be resolved", name)
	}
	return &standardEmbeddedResource{
		name:        name,
		description: description,
		uri:         uri,
		mimeType:    mimeType,
		body:        body,
	}, nil
}

func (r *standardEmbeddedResource) Name() string        { return r.name }
func (r *standardEmbeddedResource) Description() string { return r.description }
func (r *standardEmbeddedResource) URI() string         { return r.uri }
func (r *standardEmbeddedResource) MIMEType() string    { return r.mimeType }
func (r *standardEmbeddedResource) Body() string        { return r.body }

// promptFrontmatter is the YAML wire form of a prompt file header.
type promptFrontmatter struct {
	Name        string                      `yaml:"name"`
	Description string                      `yaml:"description"`
	Arguments   []promptArgumentFrontmatter `yaml:"arguments"`
}

type promptArgumentFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
}

// resourceFrontmatter is the YAML wire form of a resource file header.
type resourceFrontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	URI         string `yaml:"uri"`
	MIMEType    string `yaml:"mime_type"`
}

// loadInstructionsFrom concatenates instruction files (no frontmatter) in
// lexical order, separated by blank lines.
func loadInstructionsFrom(fsys fs.FS, dir string) (string, error) {
	names, err := listMarkdownFiles(fsys, dir)
	if err != nil {
		return "", err
	}
	parts := make([]string, 0, len(names))
	for _, name := range names {
		raw, readErr := fs.ReadFile(fsys, dir+"/"+name)
		if readErr != nil {
			return "", readErr
		}
		parts = append(parts, normalizeContent(raw))
	}
	return strings.Join(parts, "\n\n"), nil
}

// loadPromptsFrom parses every prompt file, rejecting malformed frontmatter
// and duplicate names.
func loadPromptsFrom(fsys fs.FS, dir string) ([]embeddedPrompt, error) {
	names, err := listMarkdownFiles(fsys, dir)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	prompts := make([]embeddedPrompt, 0, len(names))
	for _, name := range names {
		raw, readErr := fs.ReadFile(fsys, dir+"/"+name)
		if readErr != nil {
			return nil, readErr
		}
		meta, body, splitErr := splitFrontmatter(raw)
		if splitErr != nil {
			return nil, fmt.Errorf("prompt file %s: %w", name, splitErr)
		}
		var fm promptFrontmatter
		if yamlErr := yaml.UnmarshalStrict([]byte(meta), &fm); yamlErr != nil {
			return nil, fmt.Errorf("prompt file %s: %w", name, yamlErr)
		}
		arguments := make([]*mcp.PromptArgument, 0, len(fm.Arguments))
		for _, a := range fm.Arguments {
			arguments = append(arguments, &mcp.PromptArgument{
				Name:        a.Name,
				Description: a.Description,
				Required:    a.Required,
			})
		}
		p, newErr := newEmbeddedPrompt(fm.Name, fm.Description, arguments, body)
		if newErr != nil {
			return nil, fmt.Errorf("prompt file %s: %w", name, newErr)
		}
		if seen[p.Name()] {
			return nil, fmt.Errorf("prompt file %s: duplicate prompt name %q", name, p.Name())
		}
		seen[p.Name()] = true
		prompts = append(prompts, p)
	}
	return prompts, nil
}

// loadResourcesFrom parses every resource file, defaulting the URI to
// stackql://docs/<filename-sans-extension> and the MIME type to markdown.
func loadResourcesFrom(fsys fs.FS, dir string) ([]embeddedResource, error) {
	names, err := listMarkdownFiles(fsys, dir)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	resources := make([]embeddedResource, 0, len(names))
	for _, name := range names {
		raw, readErr := fs.ReadFile(fsys, dir+"/"+name)
		if readErr != nil {
			return nil, readErr
		}
		meta, body, splitErr := splitFrontmatter(raw)
		if splitErr != nil {
			return nil, fmt.Errorf("resource file %s: %w", name, splitErr)
		}
		var fm resourceFrontmatter
		if yamlErr := yaml.UnmarshalStrict([]byte(meta), &fm); yamlErr != nil {
			return nil, fmt.Errorf("resource file %s: %w", name, yamlErr)
		}
		if fm.URI == "" {
			fm.URI = defaultResourceURIStem + strings.TrimSuffix(name, ".md")
		}
		if fm.MIMEType == "" {
			fm.MIMEType = defaultResourceMIMEType
		}
		r, newErr := newEmbeddedResource(fm.Name, fm.Description, fm.URI, fm.MIMEType, body)
		if newErr != nil {
			return nil, fmt.Errorf("resource file %s: %w", name, newErr)
		}
		if seen[r.Name()] || seen[r.URI()] {
			return nil, fmt.Errorf("resource file %s: duplicate resource name or uri %q", name, r.Name())
		}
		seen[r.Name()] = true
		seen[r.URI()] = true
		resources = append(resources, r)
	}
	return resources, nil
}

func loadEmbeddedInstructions() (string, error) {
	return loadInstructionsFrom(embeddedContentFS, embeddedInstructionsDir)
}

func loadEmbeddedPrompts() ([]embeddedPrompt, error) {
	return loadPromptsFrom(embeddedContentFS, embeddedPromptsDir)
}

func loadEmbeddedResources() ([]embeddedResource, error) {
	return loadResourcesFrom(embeddedContentFS, embeddedResourcesDir)
}

// registerEmbeddedPrompts publishes every embedded prompt, subject to the
// EnabledPrompts allowlist.
func registerEmbeddedPrompts(server *mcp.Server, cfg *Config) error {
	prompts, err := loadEmbeddedPrompts()
	if err != nil {
		return fmt.Errorf("embedded prompts: %w", err)
	}
	for _, p := range prompts {
		addPromptIfEnabled(
			server,
			cfg,
			&mcp.Prompt{Name: p.Name(), Description: p.Description(), Arguments: p.Arguments()},
			func(_ context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
				args := map[string]string{}
				if req != nil && req.Params != nil {
					args = req.Params.Arguments
				}
				text, renderErr := p.Render(args)
				if renderErr != nil {
					return nil, renderErr
				}
				return &mcp.GetPromptResult{
					Description: p.Description(),
					Messages: []*mcp.PromptMessage{{
						Role:    "user",
						Content: &mcp.TextContent{Text: text},
					}},
				}, nil
			},
		)
	}
	return nil
}

// registerEmbeddedResources publishes every embedded resource, subject to the
// EnabledResources allowlist.  The SDK declares the resources capability only
// when at least one resource is registered.
func registerEmbeddedResources(server *mcp.Server, cfg *Config) error {
	resources, err := loadEmbeddedResources()
	if err != nil {
		return fmt.Errorf("embedded resources: %w", err)
	}
	for _, r := range resources {
		if !cfg.IsResourceEnabled(r.Name()) {
			continue
		}
		server.AddResource(
			&mcp.Resource{
				Name:        r.Name(),
				Description: r.Description(),
				URI:         r.URI(),
				MIMEType:    r.MIMEType(),
			},
			func(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
				return &mcp.ReadResourceResult{
					Contents: []*mcp.ResourceContents{{
						URI:      r.URI(),
						MIMEType: r.MIMEType(),
						Text:     r.Body(),
					}},
				}, nil
			},
		)
	}
	return nil
}
