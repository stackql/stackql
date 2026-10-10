package intrinsic //nolint:testpackage // tests unexported staging

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/any-sdk/public/sqlengine"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/output"
	"github.com/stackql/stackql/internal/stackql/typing"
	"github.com/stackql/stackql/pkg/astformat"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

type stagingTestCtx struct {
	engine  sqlengine.SQLEngine
	runtime dto.RuntimeCtx
	typCfg  typing.Config
}

func (c *stagingTestCtx) GetCurrentProvider() string        { return "" }
func (c *stagingTestCtx) SetCurrentProvider(string)         {}
func (c *stagingTestCtx) GetTypingConfig() typing.Config    { return c.typCfg }
func (c *stagingTestCtx) GetRuntimeContext() dto.RuntimeCtx { return c.runtime }
func (c *stagingTestCtx) GetSQLEngine() sqlengine.SQLEngine { return c.engine }
func (c *stagingTestCtx) GetASTFormatter() sqlparser.NodeFormatter {
	return astformat.SQLiteSelectExprsFormatter
}
func (c *stagingTestCtx) GetAuthContext(string) (*dto.AuthCtx, error) {
	return nil, errors.New("no auth configured")
}

func TestNeedsStaging(t *testing.T) {
	for sql, want := range map[string]bool{
		"select login from stackql_unstable_github.orgs.members":                    false,
		"select login from stackql_unstable_github.orgs.members limit 3":            false,
		"select login from stackql_unstable_github.orgs.members order by login":     true,
		"select distinct login from stackql_unstable_github.orgs.members":           true,
		"select count(*) from stackql_unstable_github.orgs.members":                 true,
		"select login from stackql_unstable_github.orgs.members limit 3 offset 1":   true,
		"select type from stackql_unstable_github.orgs.members group by type":       true,
		"select upper(login) from stackql_unstable_github.orgs.members":             false,
		"select login from stackql_unstable_github.orgs.members having login = 'x'": true,
	} {
		if got := needsStaging(parseSelect(t, sql)); got != want {
			t.Errorf("%s: got %v want %v", sql, got, want)
		}
	}
}

func TestPlanStagedSelectOuterStatement(t *testing.T) {
	withUnstable(t, true)
	sel := parseSelect(t, "select k.name as ring, upper(c.name), count(*) as n "+
		"from stackql_unstable_google.cloudkms.key_rings k "+
		"inner join stackql_unstable_google.cloudkms.crypto_keys c on c.keyRingsId = k.name "+
		"where k.projectsId = 'p' group by k.name, c.name having count(c.name) > 1 "+
		"order by n desc, ring limit 5 offset 2")
	cases := []struct {
		formatter sqlparser.NodeFormatter
		table     string
		want      string
	}{
		{
			formatter: astformat.SQLiteSelectExprsFormatter,
			table:     `"__iql__.queries.7"`,
			want: `select _omni_0 as ring, upper(_omni_1) as _omni_out_1, count(*) as n ` +
				`from "__iql__.queries.7" group by _omni_0, _omni_1 having count(_omni_1) > 1 ` +
				`order by n desc, ring asc LIMIT 5 OFFSET 2`,
		},
		{
			formatter: astformat.PostgresSelectExprsFormatter,
			table:     `"stackql_queries"."7"`,
			want: `select "_omni_0" as "ring", upper("_omni_1") as "_omni_out_1", count(*) as "n" ` +
				`from "stackql_queries"."7" group by "_omni_0", "_omni_1" having count("_omni_1") > 1 ` +
				`order by "n" desc, "ring" asc LIMIT 5 OFFSET 2`,
		},
	}
	for _, tc := range cases {
		staged, err := planStagedSelect(sel, newDocTranslator("", sqliteDialect{}), tc.formatter)
		if err != nil {
			t.Fatal(err)
		}
		if got := staged.renderOuter(tc.table); got != tc.want {
			t.Errorf("outer:\n got %s\nwant %s", got, tc.want)
		}
		if got, want := staged.getOutputNames(), []string{"ring", `upper("c".name)`, "n"}; !equalStrings(got, want) {
			t.Errorf("output names: got %v want %v", got, want)
		}
		if got, want := staged.getColumns(), []string{"_omni_0", "_omni_1"}; !equalStrings(got, want) {
			t.Errorf("staged columns: got %v want %v", got, want)
		}
		if staged.getSource().getLimit() != 0 {
			t.Errorf("limit must not be pushed below the staging boundary")
		}
	}
	if _, err := planStagedSelect(parseSelect(t,
		"select * from stackql_unstable_github.orgs.members order by login"), newDocTranslator("", sqliteDialect{}),
		astformat.SQLiteSelectExprsFormatter); err == nil ||
		err.Error() != "'*' cannot be staged for stackql_unstable_* relations; name the columns" {
		t.Fatalf("star refusal: got %v", err)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func newStagingTestCtx(t *testing.T, endpoint string) *stagingTestCtx {
	t.Helper()
	dir := t.TempDir()
	registry, err := filepath.Abs(filepath.Join("testdata", "registry"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.Symlink(registry, filepath.Join(dir, "src")); err != nil {
		t.Fatal(err)
	}
	sqlBackendRaw := `{"dsn":"file:` + filepath.ToSlash(filepath.Join(dir, "stackql.db")) + `"}`
	sqlCfg, err := dto.GetSQLBackendCfg(sqlBackendRaw)
	if err != nil {
		t.Fatal(err)
	}
	engine, err := sqlengine.NewSQLEngine(sqlCfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	db, err := engine.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	typCfg, err := typing.NewTypingConfig(sqlCfg.GetSQLDialect())
	if err != nil {
		t.Fatal(err)
	}
	endpointJSON, err := json.Marshal(endpoint)
	if err != nil {
		t.Fatal(err)
	}
	previous := previewCfg
	previewCfg = newBackendInput(previewCfgDTO{
		Unstable: true, Staging: true, Endpoint: endpointJSON, InsecureSkipTLSVerify: true,
	})
	t.Cleanup(func() { previewCfg = previous })
	return &stagingTestCtx{
		engine: engine,
		runtime: dto.RuntimeCtx{
			RegistryRaw:      `{"url":"file:` + filepath.ToSlash(filepath.Join(dir, "registry")) + `"}`,
			SQLBackendCfgRaw: sqlBackendRaw,
		},
		typCfg: typCfg,
	}
}

func renderCSV(t *testing.T, sql string, out internaldto.ExecutorOutput) string {
	t.Helper()
	if err := out.GetError(); err != nil {
		t.Fatalf("%s: %v", sql, err)
	}
	var buf, errBuf bytes.Buffer
	writer, err := output.GetOutputWriter(&buf, &errBuf, internaldto.OutputContext{
		RuntimeContext: dto.RuntimeCtx{OutputFormat: "csv", Delimiter: ","},
		Result:         out.GetSQLResult(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err = writer.Write(out.GetSQLResult()); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// The staging manager is created once per process, so every end-to-end case
// shares one backend and runs in this one test.
func TestStagedSelectEndToEnd(t *testing.T) {
	members := []map[string]any{
		{"login": "a", "id": 1, "type": "User"},
		{"login": "b", "id": 10, "type": "User"},
		{"login": "c", "id": 2, "type": "Bot"},
		{"login": "d", "id": 3, "type": "User"},
	}
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/orgs/dummyorg/members" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(members)
	}))
	defer srv.Close()
	ctx := newStagingTestCtx(t, srv.URL)
	t.Setenv("FIXTURE_TOKEN", "test-token")

	const from = " from stackql_unstable_fixture.orgs.members where org = 'dummyorg'"
	for _, tc := range []struct {
		sql  string
		want string
	}{
		{
			// Numeric ordering: as text, 10 would sort before 2.
			sql:  "select login, id" + from + " order by id desc limit 2 offset 1",
			want: "login,id\nd,3\nc,2\n",
		},
		{
			sql:  "select type, count(*) as n" + from + " group by type having count(*) > 1 order by n desc",
			want: "type,n\nUser,3\n",
		},
		{
			sql:  "select count(*)" + from,
			want: "count(*)\n4\n",
		},
		{
			sql:  "select distinct type" + from + " order by type",
			want: "type\nBot\nUser\n",
		},
		{
			sql: "select upper(login), id" + from + " order by login desc",
			// Named as the streamed path names an unaliased expression.
			want: "upper(`login`),id\nD,3\nC,2\nB,10\nA,1\n",
		},
	} {
		fn, claimed := selectFunc(ctx, parseSelect(t, tc.sql), "")
		if !claimed {
			t.Fatalf("%s: not claimed", tc.sql)
		}
		if got := renderCSV(t, tc.sql, fn()); got != tc.want {
			t.Errorf("%s:\n got %q\nwant %q", tc.sql, got, tc.want)
		}
	}

	db, err := ctx.engine.GetDB()
	if err != nil {
		t.Fatal(err)
	}
	var remaining int
	if err = db.QueryRow(
		`SELECT count(*) FROM sqlite_master WHERE type = 'table' AND name LIKE '\_\_iql\_\_.queries.%' ESCAPE '\'`,
	).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Fatalf("staging tables left behind: %d", remaining)
	}
}
