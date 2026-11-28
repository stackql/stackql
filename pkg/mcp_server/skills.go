package mcp_server //nolint:revive // fine for now

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"gopkg.in/yaml.v3"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Skills over MCP (io.modelcontextprotocol/skills, SEP-2640).  Each directory
// under content/skills is one skill: SKILL.md with YAML frontmatter plus any
// supporting files.  Skills are listed with skills/list, fetched with
// skills/get and their files read with resources/read at
// skill://<name>/<relative path>.  Hand-rolled on go-sdk v1.8.0 public API;
// swap to go-sdk extensions/skills once it ships in a tagged release.
const (
	embeddedSkillsDir = "content/skills"
	skillURIScheme    = "skill://"
	skillEntryFile    = "SKILL.md"
	// skillFilesTemplate serves every skill file through one resource
	// template, so resources/list is the same with skills on or off.
	skillFilesTemplate = skillURIScheme + "{+path}"

	maxSkillNameLength        = 64
	maxSkillDescriptionLength = 1024
	maxSkillFiles             = 512
	maxSkillBytes             = 16 << 20
)

// skillNameRegexp: lowercase a-z and 0-9 in hyphen-separated runs, so no
// leading, trailing or doubled hyphen.
var skillNameRegexp = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// skillMIMETypes maps the file extensions a skill may carry to the MIME type
// served by resources/read.  Any other extension fails the startup check.
var skillMIMETypes = map[string]string{ //nolint:gochecknoglobals // read-only lookup table
	".md":   "text/markdown",
	".yml":  "application/yaml",
	".yaml": "application/yaml",
	".iql":  "text/plain",
}

// skillFile is the data carrier for one served skill file.
type skillFile interface {
	URI() string
	MIMEType() string
	// Digest is sha256:<lowercase hex> over the exact bytes served.
	Digest() string
	Size() int64
	Text() string
}

type standardSkillFile struct {
	uri      string
	mimeType string
	digest   string
	text     string
}

func newSkillFile(uri, mimeType string, content []byte) skillFile {
	sum := sha256.Sum256(content)
	return &standardSkillFile{
		uri:      uri,
		mimeType: mimeType,
		digest:   "sha256:" + hex.EncodeToString(sum[:]),
		text:     string(content),
	}
}

func (f *standardSkillFile) URI() string      { return f.uri }
func (f *standardSkillFile) MIMEType() string { return f.mimeType }
func (f *standardSkillFile) Digest() string   { return f.digest }
func (f *standardSkillFile) Size() int64      { return int64(len(f.text)) }
func (f *standardSkillFile) Text() string     { return f.text }

// embeddedSkill is the data carrier for one skill: its frontmatter, passed
// through unchanged, and its complete manifest, SKILL.md included.
type embeddedSkill interface {
	Name() string
	URI() string
	Frontmatter() map[string]any
	Files() []skillFile
}

type standardEmbeddedSkill struct {
	name        string
	frontmatter map[string]any
	files       []skillFile
}

// newEmbeddedSkill enforces the Agent Skills naming, description and size
// rules; dir is the skill's directory name, which the name must match.
func newEmbeddedSkill(dir string, frontmatter map[string]any, files []skillFile) (embeddedSkill, error) {
	name, _ := frontmatter["name"].(string)
	if len(name) == 0 || len(name) > maxSkillNameLength || !skillNameRegexp.MatchString(name) {
		return nil, fmt.Errorf("skill %q: name %q must be 1 to %d characters of a-z, 0-9 and '-', "+
			"with no leading, trailing or doubled hyphen", dir, name, maxSkillNameLength)
	}
	if name != dir {
		return nil, fmt.Errorf("skill %q: name %q must match its directory", dir, name)
	}
	description, _ := frontmatter["description"].(string)
	if n := utf8.RuneCountInString(strings.TrimSpace(description)); n == 0 || n > maxSkillDescriptionLength {
		return nil, fmt.Errorf("skill %q: description must be 1 to %d characters", dir, maxSkillDescriptionLength)
	}
	if len(files) > maxSkillFiles {
		return nil, fmt.Errorf("skill %q: %d files exceeds the %d file limit", dir, len(files), maxSkillFiles)
	}
	var total int64
	entry := skillURIScheme + name + "/" + skillEntryFile
	hasEntry := false
	for _, f := range files {
		total += f.Size()
		hasEntry = hasEntry || f.URI() == entry
	}
	if total > maxSkillBytes {
		return nil, fmt.Errorf("skill %q: %d bytes exceeds the %d byte limit", dir, total, maxSkillBytes)
	}
	if !hasEntry {
		return nil, fmt.Errorf("skill %q: manifest has no %s", dir, skillEntryFile)
	}
	sorted := append([]skillFile(nil), files...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].URI() < sorted[j].URI() })
	return &standardEmbeddedSkill{name: name, frontmatter: frontmatter, files: sorted}, nil
}

func (s *standardEmbeddedSkill) Name() string                { return s.name }
func (s *standardEmbeddedSkill) URI() string                 { return skillURIScheme + s.name + "/" + skillEntryFile }
func (s *standardEmbeddedSkill) Frontmatter() map[string]any { return s.frontmatter }
func (s *standardEmbeddedSkill) Files() []skillFile          { return s.files }

// loadSkillsFrom loads every skill directory under dir.  Plain files directly
// under dir (the maintainer README) are not skills and are not served.
func loadSkillsFrom(fsys fs.FS, dir string) ([]embeddedSkill, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var skills []embeddedSkill
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		skill, loadErr := loadSkill(fsys, dir, e.Name())
		if loadErr != nil {
			return nil, loadErr
		}
		skills = append(skills, skill)
	}
	return skills, nil
}

func loadSkill(fsys fs.FS, root, dir string) (embeddedSkill, error) {
	skillRoot := path.Join(root, dir)
	var files []skillFile
	var frontmatter map[string]any
	walkErr := fs.WalkDir(fsys, skillRoot, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel := strings.TrimPrefix(p, skillRoot+"/")
		mimeType, known := skillMIMETypes[strings.ToLower(path.Ext(rel))]
		if !known {
			return fmt.Errorf("skill %q: file %s has no known MIME type (add one to skillMIMETypes)", dir, rel)
		}
		raw, readErr := fs.ReadFile(fsys, p)
		if readErr != nil {
			return readErr
		}
		// LF only, so published bytes and digests do not depend on checkout line endings.
		content := []byte(strings.ReplaceAll(string(raw), "\r\n", "\n"))
		if !utf8.Valid(content) {
			return fmt.Errorf("skill %q: file %s is not UTF-8 text", dir, rel)
		}
		if rel == skillEntryFile {
			meta, _, splitErr := splitFrontmatter(content)
			if splitErr != nil {
				return fmt.Errorf("skill %q: %s: %w", dir, skillEntryFile, splitErr)
			}
			if yamlErr := yaml.Unmarshal([]byte(meta), &frontmatter); yamlErr != nil {
				return fmt.Errorf("skill %q: %s frontmatter: %w", dir, skillEntryFile, yamlErr)
			}
		}
		files = append(files, newSkillFile(skillURIScheme+dir+"/"+rel, mimeType, content))
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	if frontmatter == nil {
		return nil, fmt.Errorf("skill %q: missing %s frontmatter", dir, skillEntryFile)
	}
	return newEmbeddedSkill(dir, frontmatter, files)
}

func loadEmbeddedSkills() ([]embeddedSkill, error) {
	return loadSkillsFrom(embeddedContentFS, embeddedSkillsDir)
}

// Wire shapes for skills/list and skills/get.

type skillResourceWire struct {
	URI    string `json:"uri"`
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}

type skillWire struct {
	URI         string              `json:"uri"`
	Frontmatter map[string]any      `json:"frontmatter"`
	Resources   []skillResourceWire `json:"resources"`
}

func toSkillWire(s embeddedSkill) skillWire {
	resources := make([]skillResourceWire, 0, len(s.Files()))
	for _, f := range s.Files() {
		resources = append(resources, skillResourceWire{URI: f.URI(), Digest: f.Digest(), Size: f.Size()})
	}
	return skillWire{URI: s.URI(), Frontmatter: s.Frontmatter(), Resources: resources}
}

// skillsCacheFields carries resultType, ttlMs and cacheScope.  They are set
// only on 2026-07-28, where results carry them, and left out on older
// revisions.  TTL and scope follow the list policy in cache_policy.go.
type skillsCacheFields struct {
	ResultType string `json:"resultType,omitempty"`
	TTLMs      *int64 `json:"ttlMs,omitempty"`
	CacheScope string `json:"cacheScope,omitempty"`
}

func newSkillsCacheFields(protocolVersion string) skillsCacheFields {
	if protocolVersion < protocolVersionSessionless {
		return skillsCacheFields{}
	}
	ttl := listCacheTTL.Milliseconds()
	return skillsCacheFields{ResultType: "complete", TTLMs: &ttl, CacheScope: listCacheScope}
}

type skillsListParams struct {
	mcp.ParamsBase
	Cursor string `json:"cursor,omitempty"`
}

type skillsListResult struct {
	mcp.ResultBase
	skillsCacheFields
	Skills []skillWire `json:"skills"`
}

type skillsGetParams struct {
	mcp.ParamsBase
	URI string `json:"uri"`
}

type skillsGetResult struct {
	mcp.ResultBase
	skillsCacheFields
	Skill skillWire `json:"skill"`
}

// customRequestProtocolVersion mirrors ServerRequest.ProtocolVersion for
// custom-method handlers, which receive only the session and params.
func customRequestProtocolVersion(ss *mcp.ServerSession, meta mcp.Meta) string {
	if v, ok := meta[mcp.MetaKeyProtocolVersion].(string); ok {
		return v
	}
	if ss != nil {
		if p := ss.InitializeParams(); p != nil {
			return p.ProtocolVersion
		}
	}
	return ""
}

func invalidParams(format string, args ...any) error {
	return &jsonrpc.Error{Code: jsonrpc.CodeInvalidParams, Message: fmt.Sprintf(format, args...)}
}

// registerSkills serves the enabled skills: skills/list, skills/get and one
// resources/read template for their files.  Every skill is validated before
// anything is registered, so a bad skill stops the server at startup.
func registerSkills(server *mcp.Server, cfg *Config) error {
	all, err := loadEmbeddedSkills()
	if err != nil {
		return fmt.Errorf("embedded skills: %w", err)
	}
	var served []skillWire
	byURI := map[string]skillWire{}
	files := map[string]skillFile{}
	for _, s := range all {
		if !cfg.Server.Extensions.Skills.IsSkillEnabled(s.Name()) {
			continue
		}
		wire := toSkillWire(s)
		served = append(served, wire)
		byURI[s.URI()] = wire
		for _, f := range s.Files() {
			files[f.URI()] = f
		}
	}
	if served == nil {
		served = []skillWire{}
	}
	if regErr := mcp.AddReceivingCustomMethod(server, "skills/list",
		func(_ context.Context, ss *mcp.ServerSession, p *skillsListParams) (*skillsListResult, error) {
			var meta mcp.Meta
			if p != nil {
				// Every skill fits one page, so no cursor is ever issued.
				if p.Cursor != "" {
					return nil, invalidParams("invalid cursor %q", p.Cursor)
				}
				meta = p.Meta
			}
			return &skillsListResult{
				skillsCacheFields: newSkillsCacheFields(customRequestProtocolVersion(ss, meta)),
				Skills:            served,
			}, nil
		}); regErr != nil {
		return regErr
	}
	if regErr := mcp.AddReceivingCustomMethod(server, "skills/get",
		func(_ context.Context, ss *mcp.ServerSession, p *skillsGetParams) (*skillsGetResult, error) {
			var uri string
			var meta mcp.Meta
			if p != nil {
				uri, meta = p.URI, p.Meta
			}
			skill, ok := byURI[uri]
			if !ok {
				return nil, invalidParams("no skill is served at %q", uri)
			}
			return &skillsGetResult{
				skillsCacheFields: newSkillsCacheFields(customRequestProtocolVersion(ss, meta)),
				Skill:             skill,
			}, nil
		}); regErr != nil {
		return regErr
	}
	server.AddResourceTemplate(&mcp.ResourceTemplate{
		Name:        "stackql_skill_files",
		Title:       "StackQL skill files",
		Description: "Files of the skills listed by skills/list, at skill://<skill>/<path>.",
		URITemplate: skillFilesTemplate,
	}, func(_ context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		f, ok := files[req.Params.URI]
		if !ok {
			return nil, mcp.ResourceNotFoundError(req.Params.URI)
		}
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{
			URI: f.URI(), MIMEType: f.MIMEType(), Text: f.Text(),
		}}}, nil
	})
	return nil
}
