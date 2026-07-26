package mcp_server //nolint:revive // fine for now

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/stackql/stackql/pkg/mcp_server/dto"
	"github.com/stackql/stackql/pkg/mcp_server/policy"
	"github.com/stackql/stackql/pkg/mcp_server/render"
	"github.com/stackql/stackql/pkg/sink"
)

// Query library tools (query_library_search / query_library_get).  Read-only,
// no credential path, no execution path: retrieval of known-good templates
// replaces the model guessing dialect nuances.  Rendering happens here in the
// server; the model never performs substitution.  Values are interpolated
// (with escaping) rather than bound: the run_select_query / run_mutation_query
// tools carry SQL text only, so there is no bind-parameter transport to the
// engine from this surface.

const (
	queryLibraryDefaultBaseURL     = "https://stackql.io/docs/query-library"
	queryLibraryDefaultFallbackURL = "https://raw.githubusercontent.com/stackql/stackql-query-library/main"
	queryLibraryDefaultTTL         = 300 * time.Second
	queryLibraryHTTPTimeout        = 10 * time.Second

	queryLibraryDefaultLimit = 5
	queryLibraryMaxLimit     = 20
	// queryLibraryMissThreshold is the top score below which the miss path
	// fires and the compact dialect guide is attached.
	queryLibraryMissThreshold = 2.5

	querySourceTierPrimary  = "primary"
	querySourceTierFallback = "fallback"
	querySourceTierSnapshot = "snapshot"

	queryLibrarySnapshotDir = "content/query_library"

	nextToolSelect    = "run_select_query"
	nextToolMutation  = "run_mutation_query"
	nextToolLifecycle = "run_lifecycle_operation"
)

var (
	queryLibraryIDRegexp         = regexp.MustCompile(`^[a-z0-9_-]+(/[a-z0-9_-]+)*$`)
	queryLibraryIdentifierRegexp = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)
	queryLibraryPlaceholderRe    = regexp.MustCompile(`\{\{([A-Za-z0-9_]+)\}\}`)
)

// libManifest mirrors manifest.json in the URL contract.
type libManifest struct {
	BuildID       string `json:"build_id"`
	GeneratedAt   string `json:"generated_at,omitempty"`
	LibraryCommit string `json:"library_commit,omitempty"`
	EntryCount    int    `json:"entry_count,omitempty"`
}

// libIndexEntry is one catalogue row from index.json.
type libIndexEntry struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Description    string   `json:"description,omitempty"`
	Providers      []string `json:"providers,omitempty"`
	Services       []string `json:"services,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	Keywords       []string `json:"keywords,omitempty"`
	IntentKeywords []string `json:"intent_keywords,omitempty"`
	Mutation       bool     `json:"mutation"`
	Status         string   `json:"status,omitempty"`
	RequiredParams []string `json:"required_params,omitempty"`
}

type libIndex struct {
	BuildID string          `json:"build_id,omitempty"`
	Entries []libIndexEntry `json:"entries"`
}

// libParam is one declared template parameter from a query document.
type libParam struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Default     any      `json:"default,omitempty"`
	Description string   `json:"description,omitempty"`
	Example     any      `json:"example,omitempty"`
	Enum        []string `json:"enum,omitempty"`
	Pattern     string   `json:"pattern,omitempty"`
}

// libCost carries the pre-execution cost hints (fan_out lets an agent warn
// the user before running something that iterates every region or project).
type libCost struct {
	FanOut    string `json:"fan_out,omitempty"`
	Expensive bool   `json:"expensive,omitempty"`
	Notes     string `json:"notes,omitempty"`
}

// libOutput describes one column the template returns.
type libOutput struct {
	Name        string `json:"name"`
	Type        string `json:"type,omitempty"`
	Description string `json:"description,omitempty"`
}

// libDoc mirrors queries/<id>.json: parsed front matter plus extracted template.
type libDoc struct {
	ID          string      `json:"id"`
	Title       string      `json:"title,omitempty"`
	Description string      `json:"description,omitempty"`
	Mutation    bool        `json:"mutation"`
	Verb        string      `json:"verb,omitempty"` // select | mutation | lifecycle; wins over Mutation
	Status      string      `json:"status,omitempty"`
	Providers   []string    `json:"providers,omitempty"`
	Services    []string    `json:"services,omitempty"`
	Auth        []string    `json:"auth,omitempty"`
	Params      []libParam  `json:"params,omitempty"`
	Outputs     []libOutput `json:"outputs,omitempty"`
	Cost        *libCost    `json:"cost,omitempty"`
	Related     []string    `json:"related,omitempty"`
	Template    string      `json:"template"`
	Notes       string      `json:"notes,omitempty"`
	DocURL      string      `json:"doc_url,omitempty"`
}

// queryLibraryClient fetches and caches library content.  The manifest
// build_id is the cache key: a changed build discards cached documents.
type queryLibraryClient struct {
	mu       sync.Mutex
	baseURL  string
	fallback string
	offline  bool
	ttl      time.Duration
	httpc    *http.Client

	buildID   string
	fetchedAt time.Time
	index     *libIndex
	docs      map[string]*libDoc
	tier      string
	stale     bool
}

// queryLibrarySettings is the resolved (config > env > default) client setup.
type queryLibrarySettings struct {
	baseURL     string
	fallbackURL string
	offline     bool
	ttl         time.Duration
}

func resolveQueryLibrarySettings(cfg QueryLibraryConfig) queryLibrarySettings {
	s := queryLibrarySettings{
		baseURL:     cfg.BaseURL,
		fallbackURL: cfg.FallbackURL,
		offline:     cfg.Offline,
		ttl:         queryLibraryDefaultTTL,
	}
	if s.baseURL == "" {
		s.baseURL = os.Getenv("STACKQL_QUERY_LIBRARY_BASE_URL")
	}
	if s.baseURL == "" {
		s.baseURL = queryLibraryDefaultBaseURL
	}
	if s.fallbackURL == "" {
		s.fallbackURL = queryLibraryDefaultFallbackURL
	}
	if !s.offline {
		switch strings.ToLower(os.Getenv("STACKQL_QUERY_LIBRARY_OFFLINE")) {
		case "1", "true", "yes":
			s.offline = true
		}
	}
	switch {
	case cfg.TTLSeconds > 0:
		s.ttl = time.Duration(cfg.TTLSeconds) * time.Second
	case os.Getenv("STACKQL_QUERY_LIBRARY_TTL") != "":
		if secs, err := strconv.Atoi(os.Getenv("STACKQL_QUERY_LIBRARY_TTL")); err == nil && secs > 0 {
			s.ttl = time.Duration(secs) * time.Second
		}
	}
	return s
}

func newQueryLibraryClient(cfg QueryLibraryConfig) *queryLibraryClient {
	s := resolveQueryLibrarySettings(cfg)
	return &queryLibraryClient{
		baseURL:  strings.TrimRight(s.baseURL, "/"),
		fallback: strings.TrimRight(s.fallbackURL, "/"),
		offline:  s.offline,
		ttl:      s.ttl,
		httpc:    &http.Client{Timeout: queryLibraryHTTPTimeout},
		docs:     map[string]*libDoc{},
	}
}

func (c *queryLibraryClient) fetchJSON(ctx context.Context, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck // read-only body close
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: http status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return err
	}
	return json.Unmarshal(body, v)
}

func loadSnapshotJSON(path string, v any) error {
	raw, err := fs.ReadFile(embeddedContentFS, path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, v)
}

// loadSnapshotIndex loads the bundled snapshot catalogue.
func (c *queryLibraryClient) loadSnapshotIndex() error {
	var m libManifest
	if err := loadSnapshotJSON(queryLibrarySnapshotDir+"/manifest.json", &m); err != nil {
		return err
	}
	var idx libIndex
	if err := loadSnapshotJSON(queryLibrarySnapshotDir+"/index.json", &idx); err != nil {
		return err
	}
	c.buildID = m.BuildID
	c.index = &idx
	c.docs = map[string]*libDoc{}
	c.fetchedAt = time.Now()
	c.tier = querySourceTierSnapshot
	return nil
}

// refreshFromTier fetches manifest + (when the build changed) index from one
// remote tier.  Returns nil on success.
func (c *queryLibraryClient) refreshFromTier(ctx context.Context, base, tier string) error {
	var m libManifest
	if err := c.fetchJSON(ctx, base+"/manifest.json", &m); err != nil {
		return err
	}
	if m.BuildID != "" && m.BuildID == c.buildID && c.index != nil {
		c.fetchedAt = time.Now()
		c.tier = tier
		return nil
	}
	var idx libIndex
	if err := c.fetchJSON(ctx, base+"/index.json", &idx); err != nil {
		return err
	}
	c.buildID = m.BuildID
	c.index = &idx
	c.docs = map[string]*libDoc{}
	c.fetchedAt = time.Now()
	c.tier = tier
	return nil
}

// ensureIndex makes the catalogue available, applying the fallback ordering:
// primary site, then raw fallback, then bundled snapshot.  Stale content
// beats no content: on total fetch failure the cached or snapshot catalogue
// is served with the stale flag set rather than erroring.
func (c *queryLibraryClient) ensureIndex(ctx context.Context) error {
	if c.offline {
		if c.index == nil {
			return c.loadSnapshotIndex()
		}
		return nil
	}
	if c.index != nil && time.Since(c.fetchedAt) < c.ttl {
		return nil
	}
	if err := c.refreshFromTier(ctx, c.baseURL, querySourceTierPrimary); err == nil {
		c.stale = false
		return nil
	}
	if err := c.refreshFromTier(ctx, c.fallback, querySourceTierFallback); err == nil {
		c.stale = false
		return nil
	}
	if c.index != nil {
		c.stale = true
		return nil
	}
	if err := c.loadSnapshotIndex(); err != nil {
		return fmt.Errorf("query library unavailable: all tiers failed and no snapshot: %w", err)
	}
	c.stale = true
	return nil
}

// getDoc returns one query document, fetched just in time and cached by build.
func (c *queryLibraryClient) getDoc(ctx context.Context, id string) (*libDoc, string, bool, error) {
	if !queryLibraryIDRegexp.MatchString(id) {
		return nil, "", false, fmt.Errorf("invalid query id %q", id)
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.ensureIndex(ctx); err != nil {
		return nil, "", false, err
	}
	if doc, ok := c.docs[id]; ok {
		return doc, c.tier, c.stale, nil
	}
	known := false
	for i := range c.index.Entries {
		if c.index.Entries[i].ID == id {
			known = true
			break
		}
	}
	if !known {
		return nil, "", false, fmt.Errorf("unknown query id %q; use query_library_search to find valid ids", id)
	}
	doc, err := c.fetchDoc(ctx, id)
	if err != nil {
		return nil, "", false, err
	}
	c.docs[id] = doc
	return doc, c.tier, c.stale, nil
}

// fetchDoc retrieves one document from the current tier, degrading to the
// snapshot (marked stale) when both remote tiers fail.
func (c *queryLibraryClient) fetchDoc(ctx context.Context, id string) (*libDoc, error) {
	var doc libDoc
	snapPath := queryLibrarySnapshotDir + "/queries/" + id + ".json"
	if c.tier == querySourceTierSnapshot {
		if err := loadSnapshotJSON(snapPath, &doc); err != nil {
			return nil, err
		}
		return &doc, nil
	}
	err := c.fetchJSON(ctx, c.baseURL+"/queries/"+id+".json", &doc)
	if err != nil {
		err = c.fetchJSON(ctx, c.fallback+"/queries/"+id+".json", &doc)
	}
	if err != nil {
		if snapErr := loadSnapshotJSON(snapPath, &doc); snapErr != nil {
			return nil, err
		}
		c.stale = true
	}
	return &doc, nil
}

// --- search ---

func containsFold(hay []string, needle string) bool {
	for _, h := range hay {
		if strings.EqualFold(h, needle) {
			return true
		}
	}
	return false
}

func tokenizeIntent(intent string) []string {
	fields := strings.FieldsFunc(strings.ToLower(intent), func(r rune) bool {
		return !('a' <= r && r <= 'z') && !('0' <= r && r <= '9')
	})
	stop := map[string]bool{
		"the": true, "a": true, "an": true, "of": true, "in": true, "on": true,
		"for": true, "to": true, "and": true, "or": true, "all": true, "my": true,
		"with": true, "show": true, "me": true, "get": true, "every": true,
	}
	var out []string
	for _, f := range fields {
		if len(f) > 1 && !stop[f] {
			out = append(out, f)
		}
	}
	return out
}

func anyContains(values []string, token string) bool {
	for _, v := range values {
		if strings.Contains(strings.ToLower(v), token) {
			return true
		}
	}
	return false
}

// scoreEntry is the lexical ranker: weighted field matching over title,
// description, intent_keywords, keywords, tags and services, plus a phrase
// bonus when the whole intent aligns with a declared intent keyword.
func scoreEntry(e libIndexEntry, tokens []string, intentLower string) float64 {
	var score float64
	for _, t := range tokens {
		if anyContains(e.IntentKeywords, t) {
			score += 4
		}
		if strings.Contains(strings.ToLower(e.Title), t) {
			score += 3
		}
		if anyContains(e.Keywords, t) {
			score += 2.5
		}
		if anyContains(e.Tags, t) {
			score += 2
		}
		if anyContains(e.Services, t) {
			score += 2
		}
		if anyContains(e.Providers, t) {
			score += 1.5
		}
		if strings.Contains(strings.ToLower(e.Description), t) {
			score++
		}
	}
	for _, k := range e.IntentKeywords {
		kl := strings.ToLower(k)
		if strings.Contains(intentLower, kl) || strings.Contains(kl, intentLower) {
			score += 6
			break
		}
	}
	return score
}

func entryToHit(e libIndexEntry, score float64) dto.QueryLibraryHitDTO {
	return dto.QueryLibraryHitDTO{
		ID:             e.ID,
		Title:          e.Title,
		Description:    e.Description,
		Providers:      e.Providers,
		Services:       e.Services,
		Mutation:       e.Mutation,
		RequiredParams: e.RequiredParams,
		Score:          score,
	}
}

type scoredEntry struct {
	entry libIndexEntry
	score float64
}

// entryPassesFilters applies the hard filters: status, mutation visibility,
// provider/service/tags.
func entryPassesFilters(e libIndexEntry, in dto.QueryLibrarySearchInput, allowMutations bool) bool {
	if strings.EqualFold(e.Status, "deprecated") {
		return false
	}
	if e.Mutation && !allowMutations {
		return false
	}
	if in.Provider != "" && !containsFold(e.Providers, in.Provider) {
		return false
	}
	if in.Service != "" && !containsFold(e.Services, in.Service) {
		return false
	}
	for _, t := range in.Tags {
		if !containsFold(e.Tags, t) {
			return false
		}
	}
	return true
}

// search runs locally against the cached index; no network call in steady
// state.  allowMutations reflects both the caller's include_mutations flag
// and the server mode: in read_only mode mutation entries are excluded
// regardless, because run_mutation_query is gated off for the session.
func (c *queryLibraryClient) search(
	ctx context.Context, in dto.QueryLibrarySearchInput, allowMutations bool,
) (dto.QueryLibrarySearchDTO, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.ensureIndex(ctx); err != nil {
		return dto.QueryLibrarySearchDTO{}, err
	}
	limit := in.Limit
	if limit <= 0 {
		limit = queryLibraryDefaultLimit
	}
	if limit > queryLibraryMaxLimit {
		limit = queryLibraryMaxLimit
	}
	tokens := tokenizeIntent(in.Intent)
	intentLower := strings.ToLower(strings.TrimSpace(in.Intent))
	candidates := make([]scoredEntry, 0, len(c.index.Entries))
	for _, e := range c.index.Entries {
		if !entryPassesFilters(e, in, allowMutations) {
			continue
		}
		candidates = append(candidates, scoredEntry{entry: e, score: scoreEntry(e, tokens, intentLower)})
	}
	sort.SliceStable(candidates, func(i, j int) bool { return candidates[i].score > candidates[j].score })
	out := dto.QueryLibrarySearchDTO{SourceTier: c.tier, Stale: c.stale, Hits: []dto.QueryLibraryHitDTO{}}
	for i, cand := range candidates {
		if i >= limit {
			break
		}
		out.Hits = append(out.Hits, entryToHit(cand.entry, cand.score))
	}
	if len(out.Hits) == 0 || out.Hits[0].Score < queryLibraryMissThreshold {
		// Miss path: nearest neighbours plus the compact dialect guide, so
		// the model degrades to authoring with the right rules, not guessing.
		out.Miss = true
		out.DialectGuide = queryLibraryDialectGuide()
	}
	return out, nil
}

// queryLibraryDialectGuide returns the embedded dialect instruction file,
// which doubles as the compact non-ANSI nuance guide on the miss path.
func queryLibraryDialectGuide() string {
	raw, err := fs.ReadFile(embeddedContentFS, "content/instructions/02-dialect.md")
	if err != nil {
		return ""
	}
	return normalizeContent(raw)
}

// --- get / render ---

func paramToDTO(p libParam) dto.QueryLibraryParamDTO {
	out := dto.QueryLibraryParamDTO{
		Name:        p.Name,
		Type:        p.Type,
		Required:    p.Required,
		Description: p.Description,
		Enum:        p.Enum,
		Pattern:     p.Pattern,
	}
	if p.Default != nil {
		out.Default = fmt.Sprint(p.Default)
	}
	if p.Example != nil {
		out.Example = fmt.Sprint(p.Example)
	}
	return out
}

func nextToolFor(doc *libDoc) string {
	switch strings.ToLower(doc.Verb) {
	case "lifecycle":
		return nextToolLifecycle
	case "mutation":
		return nextToolMutation
	case "select":
		return nextToolSelect
	}
	if doc.Mutation {
		return nextToolMutation
	}
	return nextToolSelect
}

// validateStringParam applies the optional declared pattern and escapes the
// value for the string-literal position it occupies.
func validateStringParam(p libParam, s string) (string, error) {
	if p.Pattern != "" {
		re, err := regexp.Compile(p.Pattern)
		if err != nil {
			return "", fmt.Errorf("param %q has an invalid declared pattern: %w", p.Name, err)
		}
		if !re.MatchString(s) {
			return "", fmt.Errorf("param %q must match pattern %s", p.Name, p.Pattern)
		}
	}
	return strings.ReplaceAll(s, "'", "''"), nil
}

func validateEnumParam(p libParam, s string) (string, error) {
	for _, e := range p.Enum {
		if s == e {
			return strings.ReplaceAll(s, "'", "''"), nil
		}
	}
	return "", fmt.Errorf("param %q must be one of %v, got %q", p.Name, p.Enum, s)
}

// validateParamValue validates one supplied value against its declaration and
// returns the literal text to interpolate.  string/enum values are escaped
// for the literal position they occupy; identifier values are strictly
// validated and inserted verbatim.
func validateParamValue(p libParam, val any) (string, error) {
	s := fmt.Sprint(val)
	switch strings.ToLower(p.Type) {
	case "number":
		if _, err := strconv.ParseFloat(s, 64); err != nil {
			return "", fmt.Errorf("param %q must be a number, got %q", p.Name, s)
		}
		return s, nil
	case "boolean":
		if !strings.EqualFold(s, "true") && !strings.EqualFold(s, "false") {
			return "", fmt.Errorf("param %q must be a boolean, got %q", p.Name, s)
		}
		return strings.ToLower(s), nil
	case "identifier":
		if !queryLibraryIdentifierRegexp.MatchString(s) {
			return "", fmt.Errorf("param %q must match %s", p.Name, queryLibraryIdentifierRegexp.String())
		}
		return s, nil
	case "enum":
		return validateEnumParam(p, s)
	default: // string
		return validateStringParam(p, s)
	}
}

// collectUnknownParams flags supplied values that have no declaration.
func collectUnknownParams(declared map[string]libParam, vals map[string]any, out *dto.QueryLibraryGetDTO) {
	for k := range vals {
		if _, ok := declared[k]; !ok {
			out.UnknownParams = append(out.UnknownParams, k)
		}
	}
	sort.Strings(out.UnknownParams)
	if len(out.UnknownParams) > 0 {
		out.Errors = append(out.Errors,
			fmt.Sprintf("unknown params rejected: %s", strings.Join(out.UnknownParams, ", ")))
	}
}

// resolveParamValues validates each declared param against the supplied
// values (or defaults), recording missing/invalid outcomes on the DTO.
func resolveParamValues(doc *libDoc, vals map[string]any, out *dto.QueryLibraryGetDTO) map[string]string {
	resolved := map[string]string{}
	for _, p := range doc.Params {
		val, supplied := vals[p.Name]
		if !supplied {
			switch {
			case p.Default != nil:
				val = p.Default
			case p.Required:
				out.MissingParams = append(out.MissingParams, paramToDTO(p))
				continue
			default:
				continue
			}
		}
		lit, err := validateParamValue(p, val)
		if err != nil {
			out.Errors = append(out.Errors, err.Error())
			continue
		}
		resolved[p.Name] = lit
	}
	return resolved
}

// renderTemplate applies the rendering rules: unknown params are rejected,
// missing required params are reported structurally, values are validated by
// declared type, and any unresolved placeholder after substitution is an error.
func renderTemplate(doc *libDoc, vals map[string]any, out *dto.QueryLibraryGetDTO) {
	out.Rendered = true
	declared := map[string]libParam{}
	for _, p := range doc.Params {
		declared[p.Name] = p
	}
	collectUnknownParams(declared, vals, out)
	resolved := resolveParamValues(doc, vals, out)
	if len(out.MissingParams) > 0 {
		names := make([]string, 0, len(out.MissingParams))
		for _, m := range out.MissingParams {
			names = append(names, m.Name)
		}
		out.Errors = append(out.Errors,
			fmt.Sprintf("missing required params: %s", strings.Join(names, ", ")))
	}
	if len(out.Errors) > 0 {
		return
	}
	sql := queryLibraryPlaceholderRe.ReplaceAllStringFunc(doc.Template, func(m string) string {
		name := queryLibraryPlaceholderRe.FindStringSubmatch(m)[1]
		if lit, ok := resolved[name]; ok {
			return lit
		}
		return m
	})
	if leftover := queryLibraryPlaceholderRe.FindString(sql); leftover != "" {
		out.Errors = append(out.Errors, fmt.Sprintf("unresolved placeholder %s in template", leftover))
		return
	}
	out.Valid = true
	out.SQL = sql
}

// get returns the teaching surface (no params) or validated rendered SQL.
func (c *queryLibraryClient) get(ctx context.Context, in dto.QueryLibraryGetInput) (dto.QueryLibraryGetDTO, error) {
	doc, tier, stale, err := c.getDoc(ctx, in.ID)
	if err != nil {
		return dto.QueryLibraryGetDTO{}, err
	}
	out := dto.QueryLibraryGetDTO{
		ID:          doc.ID,
		Title:       doc.Title,
		Description: doc.Description,
		Mutation:    doc.Mutation,
		Verb:        doc.Verb,
		NextTool:    nextToolFor(doc),
		Auth:        doc.Auth,
		Related:     doc.Related,
		DocURL:      doc.DocURL,
		SourceTier:  tier,
		Stale:       stale,
	}
	if doc.Cost != nil {
		out.Cost = &dto.QueryLibraryCostDTO{
			FanOut: doc.Cost.FanOut, Expensive: doc.Cost.Expensive, Notes: doc.Cost.Notes,
		}
	}
	for _, o := range doc.Outputs {
		out.Outputs = append(out.Outputs, dto.QueryLibraryOutputDTO{
			Name: o.Name, Type: o.Type, Description: o.Description,
		})
	}
	for _, p := range doc.Params {
		out.Params = append(out.Params, paramToDTO(p))
	}
	if len(in.Params) == 0 {
		out.Template = doc.Template
		out.Notes = doc.Notes
		return out, nil
	}
	renderTemplate(doc, in.Params, &out)
	return out, nil
}

// --- tool registration ---

func queryLibrarySearchGate() toolGate {
	return toolGate{
		toolName:     "query_library_search",
		defaultClass: policy.QueryClassSelect,
		extractArgs: func(args any) map[string]any {
			v, ok := args.(dto.QueryLibrarySearchInput)
			if !ok {
				return nil
			}
			out := map[string]any{"intent": v.Intent}
			if v.Provider != "" {
				out["provider"] = v.Provider
			}
			if v.Service != "" {
				out["service"] = v.Service
			}
			return out
		},
	}
}

func queryLibraryGetGate() toolGate {
	return toolGate{
		toolName:     "query_library_get",
		defaultClass: policy.QueryClassSelect,
		extractArgs: func(args any) map[string]any {
			v, ok := args.(dto.QueryLibraryGetInput)
			if !ok {
				return nil
			}
			return map[string]any{"id": v.ID, "rendered": len(v.Params) > 0}
		},
	}
}

func searchHitsToRows(hits []dto.QueryLibraryHitDTO) []map[string]any {
	rows := make([]map[string]any, 0, len(hits))
	for _, h := range hits {
		rows = append(rows, map[string]any{
			"id":              h.ID,
			"title":           h.Title,
			"mutation":        h.Mutation,
			"required_params": strings.Join(h.RequiredParams, ", "),
			"score":           fmt.Sprintf("%.1f", h.Score),
		})
	}
	return rows
}

// registerQueryLibraryTools publishes the two read-only library tools.
func registerQueryLibraryTools(server *mcp.Server, cfg *Config, auditSink sink.Sink) {
	client := newQueryLibraryClient(cfg.QueryLibrary)
	registerQueryLibrarySearchTool(server, cfg, auditSink, client)
	registerQueryLibraryGetTool(server, cfg, auditSink, client)
}

func registerQueryLibrarySearchTool(
	server *mcp.Server, cfg *Config, auditSink sink.Sink, client *queryLibraryClient,
) {
	addToolWithGate(
		server, cfg, auditSink, queryLibrarySearchGate(),
		&mcp.Tool{
			Name: "query_library_search",
			Description: "Search the curated StackQL query library by natural-language intent. Returns ranked " +
				"template ids with required param names. Consult this before composing SQL from scratch; " +
				"follow up with query_library_get. Read-only, no credentials.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, args dto.QueryLibrarySearchInput,
		) (*mcp.CallToolResult, dto.QueryLibrarySearchDTO, error) {
			format, formatErr := resolveRenderFormat(cfg, args.Format)
			if formatErr != nil {
				return nil, dto.QueryLibrarySearchDTO{}, formatErr
			}
			allowMutations := args.IncludeMutations && cfg.GetMode() != policy.ModeReadOnly
			out, err := client.search(ctx, args, allowMutations)
			if err != nil {
				return nil, dto.QueryLibrarySearchDTO{}, err
			}
			text := textForFormat(format, out, func() string {
				body := render.RenderTable(searchHitsToRows(out.Hits))
				if out.Miss {
					body += "\n\nNo hit cleared the relevance threshold. Author the query using the dialect " +
						"guide below and the discovery tools.\n\n" + out.DialectGuide
				}
				return body
			})
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, out, nil
		},
	)
}

func registerQueryLibraryGetTool(
	server *mcp.Server, cfg *Config, auditSink sink.Sink, client *queryLibraryClient,
) {
	addToolWithGate(
		server, cfg, auditSink, queryLibraryGetGate(),
		&mcp.Tool{
			Name: "query_library_get",
			Description: "Retrieve a query library entry by id. Without params: returns the raw template, " +
				"param declarations and notes (adapt from it when no exact match exists). With params: the " +
				"server validates values and returns rendered SQL plus which tool to execute it with " +
				"(run_select_query or run_mutation_query). Read-only, no credentials.",
		},
		func(ctx context.Context, _ *mcp.CallToolRequest, args dto.QueryLibraryGetInput,
		) (*mcp.CallToolResult, dto.QueryLibraryGetDTO, error) {
			format, formatErr := resolveRenderFormat(cfg, args.Format)
			if formatErr != nil {
				return nil, dto.QueryLibraryGetDTO{}, formatErr
			}
			out, err := client.get(ctx, args)
			if err != nil {
				return nil, dto.QueryLibraryGetDTO{}, err
			}
			text := textForFormat(format, out, func() string {
				rec := map[string]any{
					"id": out.ID, "mutation": out.Mutation, "next_tool": out.NextTool,
					"valid": out.Valid, "doc_url": out.DocURL,
				}
				if out.SQL != "" {
					rec["sql"] = out.SQL
				}
				if out.Template != "" {
					rec["template"] = out.Template
				}
				if len(out.Errors) > 0 {
					rec["errors"] = strings.Join(out.Errors, "; ")
				}
				return render.RenderKV("Query Library Entry", []map[string]any{rec})
			})
			return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}, out, nil
		},
	)
}
