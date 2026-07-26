package mcp_server //nolint:testpackage,revive // exercise internal wiring

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stackql/stackql/pkg/mcp_server/dto"
)

// libraryFixture is an in-memory implementation of the query library URL
// contract, served over httptest.  This is the mock server for Go testing;
// the flask twin under test/python serves the same contract for black-box use.
type libraryFixture struct {
	buildID  string
	index    libIndex
	docs     map[string]libDoc
	requests atomic.Int64
	fail     atomic.Bool
}

func (f *libraryFixture) handler() http.Handler {
	mux := http.NewServeMux()
	writeJSON := func(w http.ResponseWriter, v any) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(v) //nolint:errcheck // test fixture
	}
	mux.HandleFunc("/docs/query-library/", func(w http.ResponseWriter, r *http.Request) {
		f.requests.Add(1)
		if f.fail.Load() {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/docs/query-library/")
		switch {
		case path == "manifest.json":
			writeJSON(w, libManifest{BuildID: f.buildID, EntryCount: len(f.index.Entries)})
		case path == "index.json":
			idx := f.index
			idx.BuildID = f.buildID
			writeJSON(w, idx)
		case strings.HasPrefix(path, "queries/") && strings.HasSuffix(path, ".json"):
			id := strings.TrimSuffix(strings.TrimPrefix(path, "queries/"), ".json")
			doc, ok := f.docs[id]
			if !ok {
				http.NotFound(w, r)
				return
			}
			writeJSON(w, doc)
		default:
			http.NotFound(w, r)
		}
	})
	return mux
}

func newTestFixture() *libraryFixture {
	return &libraryFixture{
		buildID: "build-001",
		index: libIndex{Entries: []libIndexEntry{
			{
				ID: "aws/ec2/regions-enabled", Title: "Enabled AWS regions",
				Description: "Lists AWS regions with their opt-in status.",
				Providers:   []string{"aws"}, Services: []string{"ec2"},
				Tags:           []string{"aws", "regions"},
				IntentKeywords: []string{"list enabled aws regions", "enumerate aws regions"},
				Status:         "stable",
			},
			{
				ID: "aws/s3/bucket-detail", Title: "S3 bucket security detail",
				Description: "Public access block, encryption, versioning for one bucket.",
				Providers:   []string{"aws"}, Services: []string{"s3"},
				Tags:           []string{"aws", "s3", "security"},
				IntentKeywords: []string{"is my bucket public", "bucket security settings"},
				Status:         "stable", RequiredParams: []string{"region", "bucket_name"},
			},
			{
				ID: "aws/cloud_control/log-group-retention-update", Title: "Set log group retention",
				Description: "Updates RetentionInDays via Cloud Control.",
				Providers:   []string{"aws"}, Services: []string{"cloud_control"},
				IntentKeywords: []string{"set log group retention"},
				Mutation:       true, Status: "stable",
				RequiredParams: []string{"region", "log_group_name", "retention_days"},
			},
		}},
		docs: map[string]libDoc{
			"aws/s3/bucket-detail": {
				ID: "aws/s3/bucket-detail", Title: "S3 bucket security detail",
				Params: []libParam{
					{Name: "region", Type: "identifier", Required: true},
					{Name: "bucket_name", Type: "string", Required: true, Description: "Bucket name",
						Example: "my-bucket"},
				},
				Template: "SELECT bucket_name FROM aws.s3.buckets WHERE region = '{{region}}' " +
					"AND data__Identifier = '{{bucket_name}}';",
				DocURL: "https://stackql.io/docs/query-library/queries/aws/s3/bucket-detail",
			},
			"aws/cloud_control/log-group-retention-update": {
				ID: "aws/cloud_control/log-group-retention-update", Mutation: true,
				Params: []libParam{
					{Name: "region", Type: "identifier", Required: true},
					{Name: "log_group_name", Type: "string", Required: true},
					{Name: "retention_days", Type: "number", Required: true},
				},
				Template: "UPDATE aws.cloud_control.resources SET data__PatchDocument = " +
					"string('[{\"op\":\"replace\",\"path\":\"/RetentionInDays\",\"value\":{{retention_days}}}]') " +
					"WHERE region = '{{region}}' AND data__TypeName = 'AWS::Logs::LogGroup' " +
					"AND data__Identifier = '{{log_group_name}}';",
			},
		},
	}
}

func newClientForFixture(t *testing.T, f *libraryFixture) *queryLibraryClient {
	t.Helper()
	srv := httptest.NewServer(f.handler())
	t.Cleanup(srv.Close)
	return newQueryLibraryClient(QueryLibraryConfig{
		BaseURL:     srv.URL + "/docs/query-library",
		FallbackURL: srv.URL + "/nonexistent",
	})
}

func TestQueryLibrary_SearchRanksIntent(t *testing.T) {
	client := newClientForFixture(t, newTestFixture())
	out, err := client.search(context.Background(), dto.QueryLibrarySearchInput{
		Intent: "list enabled aws regions",
	}, false)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(out.Hits) == 0 || out.Hits[0].ID != "aws/ec2/regions-enabled" {
		t.Fatalf("expected regions entry first, got %+v", out.Hits)
	}
	if out.Miss {
		t.Errorf("expected a confident hit, got miss path")
	}
	if out.SourceTier != querySourceTierPrimary {
		t.Errorf("expected primary tier, got %q", out.SourceTier)
	}
}

func TestQueryLibrary_SearchExcludesMutationsByDefault(t *testing.T) {
	client := newClientForFixture(t, newTestFixture())
	out, err := client.search(context.Background(), dto.QueryLibrarySearchInput{
		Intent: "set log group retention",
	}, false)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	for _, h := range out.Hits {
		if h.Mutation {
			t.Errorf("mutation entry leaked into default search: %s", h.ID)
		}
	}
	outWith, err := client.search(context.Background(), dto.QueryLibrarySearchInput{
		Intent: "set log group retention", IncludeMutations: true,
	}, true)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(outWith.Hits) == 0 || outWith.Hits[0].ID != "aws/cloud_control/log-group-retention-update" {
		t.Fatalf("expected mutation entry first with include_mutations, got %+v", outWith.Hits)
	}
}

func TestQueryLibrary_SearchMissPathCarriesDialectGuide(t *testing.T) {
	client := newClientForFixture(t, newTestFixture())
	out, err := client.search(context.Background(), dto.QueryLibrarySearchInput{
		Intent: "quantum flux capacitor telemetry",
	}, false)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if !out.Miss {
		t.Fatalf("expected miss path")
	}
	if !strings.Contains(out.DialectGuide, "Dialect rules") {
		t.Errorf("miss path should carry the dialect guide, got %q", out.DialectGuide[:min(80, len(out.DialectGuide))])
	}
}

func TestQueryLibrary_SearchIsLocalInSteadyState(t *testing.T) {
	f := newTestFixture()
	client := newClientForFixture(t, f)
	if _, err := client.search(context.Background(), dto.QueryLibrarySearchInput{Intent: "regions"}, false); err != nil {
		t.Fatalf("search: %v", err)
	}
	before := f.requests.Load()
	for i := 0; i < 5; i++ {
		if _, err := client.search(context.Background(), dto.QueryLibrarySearchInput{Intent: "buckets"}, false); err != nil {
			t.Fatalf("search: %v", err)
		}
	}
	if f.requests.Load() != before {
		t.Errorf("steady-state search must not hit the network: %d extra requests", f.requests.Load()-before)
	}
}

func TestQueryLibrary_GetTeachingSurface(t *testing.T) {
	client := newClientForFixture(t, newTestFixture())
	out, err := client.get(context.Background(), dto.QueryLibraryGetInput{ID: "aws/s3/bucket-detail"})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !strings.Contains(out.Template, "{{bucket_name}}") {
		t.Errorf("template placeholders must remain intact, got %q", out.Template)
	}
	if out.Rendered || out.NextTool != nextToolSelect {
		t.Errorf("unexpected teaching surface: %+v", out)
	}
	if len(out.Params) != 2 {
		t.Errorf("expected 2 params, got %+v", out.Params)
	}
}

func TestQueryLibrary_GetRendersAndEscapes(t *testing.T) {
	client := newClientForFixture(t, newTestFixture())
	out, err := client.get(context.Background(), dto.QueryLibraryGetInput{
		ID:     "aws/s3/bucket-detail",
		Params: map[string]any{"region": "us-east-1", "bucket_name": "it's-a-bucket"},
	})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !out.Valid {
		t.Fatalf("expected valid render, got %+v", out)
	}
	if !strings.Contains(out.SQL, "data__Identifier = 'it''s-a-bucket'") {
		t.Errorf("string param must be escaped for its literal position, got %q", out.SQL)
	}
	if strings.Contains(out.SQL, "{{") {
		t.Errorf("rendered SQL must not contain placeholders: %q", out.SQL)
	}
}

func TestQueryLibrary_GetValidationFailures(t *testing.T) {
	client := newClientForFixture(t, newTestFixture())
	ctx := context.Background()

	out, err := client.get(ctx, dto.QueryLibraryGetInput{
		ID:     "aws/s3/bucket-detail",
		Params: map[string]any{"region": "us-east-1", "bucket_name": "b", "bogus": "x"},
	})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if out.Valid || len(out.UnknownParams) != 1 || out.UnknownParams[0] != "bogus" {
		t.Errorf("unknown params must be rejected, got %+v", out)
	}

	out, err = client.get(ctx, dto.QueryLibraryGetInput{
		ID:     "aws/s3/bucket-detail",
		Params: map[string]any{"region": "us-east-1"},
	})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if out.Valid || len(out.MissingParams) != 1 || out.MissingParams[0].Name != "bucket_name" {
		t.Errorf("missing required param must be reported structurally, got %+v", out)
	}
	if out.MissingParams[0].Example == "" {
		t.Errorf("missing param report should carry the example for user prompting")
	}

	out, err = client.get(ctx, dto.QueryLibraryGetInput{
		ID:     "aws/s3/bucket-detail",
		Params: map[string]any{"region": "us-east-1'; DROP TABLE x; --", "bucket_name": "b"},
	})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if out.Valid {
		t.Errorf("identifier param must reject non-identifier values, got %+v", out)
	}

	out, err = client.get(ctx, dto.QueryLibraryGetInput{
		ID: "aws/cloud_control/log-group-retention-update",
		Params: map[string]any{
			"region": "ap-southeast-1", "log_group_name": "lg", "retention_days": "not-a-number",
		},
	})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if out.Valid {
		t.Errorf("number param must reject non-numeric values, got %+v", out)
	}
}

func TestQueryLibrary_GetMutationRoutesToMutationTool(t *testing.T) {
	client := newClientForFixture(t, newTestFixture())
	out, err := client.get(context.Background(), dto.QueryLibraryGetInput{
		ID: "aws/cloud_control/log-group-retention-update",
		Params: map[string]any{
			"region": "ap-southeast-1", "log_group_name": "lg", "retention_days": 180,
		},
	})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if !out.Valid || out.NextTool != nextToolMutation {
		t.Fatalf("mutation entry must route to run_mutation_query, got %+v", out)
	}
	if !strings.Contains(out.SQL, `"value":180}]`) {
		t.Errorf("number param must interpolate into the patch document, got %q", out.SQL)
	}
}

func TestQueryLibrary_BuildChangeDiscardsDocCache(t *testing.T) {
	f := newTestFixture()
	client := newClientForFixture(t, f)
	ctx := context.Background()
	if _, err := client.get(ctx, dto.QueryLibraryGetInput{ID: "aws/s3/bucket-detail"}); err != nil {
		t.Fatalf("get: %v", err)
	}
	client.mu.Lock()
	if len(client.docs) != 1 {
		t.Fatalf("expected 1 cached doc")
	}
	client.fetchedAt = client.fetchedAt.Add(-2 * client.ttl) // force TTL expiry
	client.mu.Unlock()
	f.buildID = "build-002"
	if _, err := client.search(ctx, dto.QueryLibrarySearchInput{Intent: "regions"}, false); err != nil {
		t.Fatalf("search: %v", err)
	}
	client.mu.Lock()
	defer client.mu.Unlock()
	if client.buildID != "build-002" {
		t.Errorf("expected refreshed build id, got %q", client.buildID)
	}
	if len(client.docs) != 0 {
		t.Errorf("doc cache must be discarded on build change")
	}
}

func TestQueryLibrary_FallbackTierAndSnapshot(t *testing.T) {
	f := newTestFixture()
	primary := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "down", http.StatusServiceUnavailable)
	}))
	t.Cleanup(primary.Close)
	fallbackSrv := httptest.NewServer(f.handler())
	t.Cleanup(fallbackSrv.Close)

	client := newQueryLibraryClient(QueryLibraryConfig{
		BaseURL:     primary.URL + "/docs/query-library",
		FallbackURL: fallbackSrv.URL + "/docs/query-library",
	})
	out, err := client.search(context.Background(), dto.QueryLibrarySearchInput{Intent: "regions"}, false)
	if err != nil {
		t.Fatalf("search via fallback: %v", err)
	}
	if out.SourceTier != querySourceTierFallback {
		t.Errorf("expected fallback tier, got %q", out.SourceTier)
	}

	// Both tiers down on a cold client: snapshot serves, marked stale.
	cold := newQueryLibraryClient(QueryLibraryConfig{
		BaseURL:     primary.URL + "/docs/query-library",
		FallbackURL: primary.URL + "/also-down",
	})
	out, err = cold.search(context.Background(), dto.QueryLibrarySearchInput{Intent: "list enabled aws regions"}, false)
	if err != nil {
		t.Fatalf("snapshot search: %v", err)
	}
	if out.SourceTier != querySourceTierSnapshot || !out.Stale {
		t.Errorf("expected stale snapshot serve, got tier=%q stale=%v", out.SourceTier, out.Stale)
	}
}

func TestQueryLibrary_OfflineSnapshotEndToEnd(t *testing.T) {
	client := newQueryLibraryClient(QueryLibraryConfig{Offline: true})
	ctx := context.Background()
	out, err := client.search(ctx, dto.QueryLibrarySearchInput{Intent: "list enabled aws regions"}, false)
	if err != nil {
		t.Fatalf("offline search: %v", err)
	}
	if len(out.Hits) == 0 || out.Hits[0].ID != "aws/ec2/regions-enabled" {
		t.Fatalf("expected embedded seed hit, got %+v", out.Hits)
	}
	if out.SourceTier != querySourceTierSnapshot || out.Stale {
		t.Errorf("offline is intentional snapshot use, not stale: tier=%q stale=%v", out.SourceTier, out.Stale)
	}
	got, err := client.get(ctx, dto.QueryLibraryGetInput{
		ID: "aws/ec2/regions-enabled", Params: map[string]any{"seed_region": "us-west-2"},
	})
	if err != nil {
		t.Fatalf("offline get: %v", err)
	}
	if !got.Valid || !strings.Contains(got.SQL, "region = 'us-west-2'") {
		t.Fatalf("offline render failed: %+v", got)
	}
}

func TestQueryLibrary_GetUnknownAndInvalidIDs(t *testing.T) {
	client := newClientForFixture(t, newTestFixture())
	ctx := context.Background()
	if _, err := client.get(ctx, dto.QueryLibraryGetInput{ID: "no/such/entry"}); err == nil {
		t.Errorf("unknown id must error")
	}
	if _, err := client.get(ctx, dto.QueryLibraryGetInput{ID: "../../etc/passwd"}); err == nil {
		t.Errorf("path-traversal shaped id must be rejected")
	}
}

func TestQueryLibrary_ToolsRegisteredReadOnly(t *testing.T) {
	cs := connectInProcess(t, DefaultConfig(), &testBackend{})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tools, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	found := 0
	for _, tool := range tools.Tools {
		if tool.Name == "query_library_search" || tool.Name == "query_library_get" {
			found++
			if tool.Annotations == nil || !tool.Annotations.ReadOnlyHint {
				t.Errorf("%s must carry readOnlyHint", tool.Name)
			}
		}
	}
	if found != 2 {
		t.Errorf("expected both query library tools registered, found %d", found)
	}
}
