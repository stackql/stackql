package intrinsic //nolint:testpackage // tests unexported staging

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/output"
	dto "github.com/stackql/stackql/internal/stackql/runtimeconfig"
	"github.com/stackql/stackql/internal/stackql/sql_system"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	sqlcontrol "github.com/stackql/stackql/internal/stackql/sqlengine/control"
	"github.com/stackql/stackql/internal/stackql/typing"
	"github.com/stackql/stackql/pkg/astformat"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

type stagingTestCtx struct {
	engine  sqlengine.SQLEngine
	runtime dto.RuntimeCtx
	typCfg  typing.Config
	system  sql_system.SQLSystem
}

func (c *stagingTestCtx) GetSQLSystem() sql_system.SQLSystem { return c.system }

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
	engine, err := sqlengine.NewSQLEngine(sqlengine.Config{DBEngine: sqlCfg.DBEngine, DSN: sqlCfg.GetDSN()}, nil)
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
		Unstable: true, Endpoint: endpointJSON, InsecureSkipTLSVerify: true,
	})
	t.Cleanup(func() { previewCfg = previous })
	return &stagingTestCtx{
		engine: engine,
		runtime: dto.RuntimeCtx{
			RegistryRaw:      `{"url":"file:` + filepath.ToSlash(filepath.Join(dir, "registry")) + `","verifyConfig":{"nopVerify":true}}`,
			IndirectDepthMax: 5,
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

func TestDescribeDocumentWithoutExecutingProvider(t *testing.T) {
	ctx := newStagingTestCtx(t, "https://unused.invalid")
	for _, sql := range []string{
		"describe fixture.orgs.members",
		"describe method fixture.orgs.members.list_members",
	} {
		statement, err := sqlparser.Parse(sql)
		if err != nil {
			t.Fatal(err)
		}
		execute, handled := GeneratePrimitiveFunc(ctx, statement)
		if !handled {
			t.Fatalf("metadata statement was not handled: %s", sql)
		}
		result := renderCSV(t, sql, execute())
		if !strings.Contains(result, "login,string") || !strings.Contains(result, "id,integer") {
			t.Fatalf("missing declared response columns: %s", result)
		}
	}
}

// These end-to-end cases share a backend to verify staging cleanup across queries.
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

	sqlCfg, sysErr := dto.GetSQLBackendCfg(ctx.runtime.SQLBackendCfgRaw)
	if sysErr != nil {
		t.Fatal(sysErr)
	}
	ctx.system, sysErr = sql_system.NewSQLSystem(ctx.engine, "", sqlcontrol.GetControlAttributes("standard"), sqlCfg, nil, ctx.typCfg, "")
	if sysErr != nil {
		t.Fatal(sysErr)
	}
	if sysErr = ctx.system.CreateView("scoped_members", "select login,id from stackql_unstable_fixture.orgs.members where org = 'dummyorg'", false, nil); sysErr != nil {
		t.Fatal(sysErr)
	}
	columns := []typing.RelationalColumn{typing.NewRelationalColumn("id", "integer"), typing.NewRelationalColumn("note", "text")}
	if sysErr = ctx.system.CreatePhysicalTable("local_people", columns, `CREATE TABLE "local_people" (id INTEGER,note TEXT)`, false); sysErr != nil {
		t.Fatal(sysErr)
	}
	if _, sysErr = ctx.engine.Exec(`INSERT INTO "local_people" (id,note) VALUES (10,'chosen')`); sysErr != nil {
		t.Fatal(sysErr)
	}
	if sysErr = ctx.system.CreateMaterializedView("local_cache", []typing.RelationalColumn{typing.NewRelationalColumn("id", "integer")}, "select 10 as id", false, "select 10 as id"); sysErr != nil {
		t.Fatal(sysErr)
	}

	if sysErr = ctx.system.CreateView("stored_people", "select id,note from local_people", false, nil); sysErr != nil {
		t.Fatal(sysErr)
	}

	const from = " from stackql_unstable_fixture.orgs.members where org = 'dummyorg'"
	for _, tc := range []struct {
		sql  string
		want string
	}{

		{
			sql: "select m.login from stackql_unstable_fixture.orgs.members m join local_people p " +
				"on p.id=e.value, json_each('[10]') e where m.org='dummyorg' and m.id=p.id",
			want: "login\nb\n",
		},

		{
			sql:  "select login, case when id=10 then 'chosen' else json('invalid-json') end as choice" + from + " and id=10",
			want: "login,choice\nb,chosen\n",
		},
		{
			sql: "select login from stackql_unstable_fixture.orgs.members m join local_people p " +
				"on p.id=case m.id when 10 then m.id else -1 end where m.org='dummyorg'",
			want: "login\nb\n",
		},
		{
			sql:  "select substr(login,1,instr(login,'b')-1) as prefix" + from + " and id=10",
			want: "prefix\n\n",
		},

		{
			sql:  "select 'member:' || login as label" + from + " and id=10",
			want: "label\nmember:b\n",
		},

		{
			sql: "with local_people as (select 2 as id) " +
				"select m.login, v.note from stackql_unstable_fixture.orgs.members m " +
				"join stored_people v on m.id=v.id where m.org='dummyorg'",
			want: "login,note\nb,chosen\n",
		},

		{
			sql: "with picked as (select login,id" + from + " order by id desc limit 2) " +
				"select login from picked order by login",
			want: "login\nb\nd\n",
		},
		{
			sql: "with picked as (select login,id" + from + "), " +
				"tallied(user_count) as (select count(*) from picked) select user_count from tallied",
			want: "user_count\n4\n",
		},
		{
			sql: "with picked as (select login" + from + " and id=10), " +
				"nested as (with picked as (select login" + from + " and id=2) select login from picked) " +
				"select p.login, n.login as nested from picked p cross join nested n",
			want: "login,nested\nb,c\n",
		},
		{
			sql: "with local_people as (select id" + from + " and id=2) " +
				"select m.login, p.id from stackql_unstable_fixture.orgs.members m " +
				"join local_people p on m.id=p.id where m.org='dummyorg'",
			want: "login,id\nc,2\n",
		},

		{
			sql: "select count(*) as n from stackql_unstable_fixture.orgs.members m " +
				"join stackql_unstable_fixture.orgs.members n on m.org=n.org and m.id=n.id " +
				"where m.org='dummyorg' and n.org='dummyorg'",
			want: "n\n4\n",
		},

		{
			sql: "select m.login, e.value from stackql_unstable_fixture.orgs.members m, json_each('[1,2]') e " +
				"where m.org='dummyorg' and m.id=10 order by e.value",
			want: "login,value\nb,1\nb,2\n",
		},
		{
			sql: "select count(*) as n from stackql_unstable_fixture.orgs.members m " +
				"join stackql_unstable_fixture.orgs.members n on m.id=n.id where m.org='dummyorg' and n.org='dummyorg'",
			want: "n\n4\n",
		},

		{
			sql:  "select s.login from (select login, id" + from + ") s where s.id > 2 order by s.id desc",
			want: "login\nb\nd\n",
		},
		{
			sql:  "select s.login from (select login, id" + from + " order by id desc limit 2) s order by s.login",
			want: "login\nb\nd\n",
		},
		{
			sql:  "select s.type, s.n from (select type, count(*) as n" + from + " group by type) s where s.n > 1 order by s.n",
			want: "type,n\nUser,3\n",
		},
		{
			sql:  "select outer_rows.login from (select inner_rows.login from (select login, id" + from + ") inner_rows where inner_rows.id = 2) outer_rows",
			want: "login\nc\n",
		},

		{
			sql: "select m.login, s.login as selected from stackql_unstable_fixture.orgs.members m left join (select login, id" + from + " order by id desc limit 2) s on m.id = s.id where m.org = 'dummyorg' order by m.login",
			// CSV presentation spells SQL NULL as null.
			want: "login,selected\na,null\nb,b\nc,null\nd,d\n",
		},

		{
			sql:  "select s.login from (select *" + from + " order by id desc limit 2) s order by s.login",
			want: "login\nb\nd\n",
		},

		{
			sql:  "select m.login, x.label from (select 'b' as login, 'constant' as label) x join (select login" + from + ") m on x.login = m.login",
			want: "login,label\nb,constant\n",
		},
		{
			sql:  "select s.login from (select login, lag(id) over (order by id) as previous_id" + from + ") s where s.previous_id is null",
			want: "login\na\n",
		},

		{
			sql:  "select v.login,p.note from scoped_members v join local_people p on v.id=p.id order by v.login",
			want: "login,note\nb,chosen\n",
		},
		{
			sql:  "select v.login from local_cache m join scoped_members v on m.id=v.id",
			want: "login\nb\n",
		},
		{
			sql:  "select v.login from (select * from scoped_members order by id desc limit 2) v join local_people p on v.id=p.id",
			want: "login\nb\n",
		},
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
			sql:  "select upper(login) as loud" + from + " and loud = 'A' order by loud",
			want: "loud\nA\n",
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

	testMaterializedReads(t, ctx)
	testStatementReads(t, ctx)

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

func TestExpandedCTEDepthHonorsHostLimit(t *testing.T) {
	query := parseSelect(t, "select x.id from (select y.id from (select 1 as id) y) x")
	if err := validateRelationDepth(query.From, 1, 0); err == nil {
		t.Fatal("nested relations exceeded the configured depth without an error")
	}
	if err := validateRelationDepth(query.From, 2, 0); err != nil {
		t.Fatal(err)
	}
}

func testMaterializedReads(t *testing.T, ctx *stagingTestCtx) {
	t.Helper()
	for _, query := range []string{
		"create materialized view omni_members as select login, id from fixture.orgs.members where org = 'dummyorg' order by id",
		"create or replace materialized view omni_members as select login, id from fixture.orgs.members where org = 'dummyorg' order by id",
	} {
		execute, claimed := materializedFunc(ctx, parseStatement(t, query))
		if !claimed {
			t.Fatalf("materialized query not claimed: %s", query)
		}
		if err := execute().GetError(); err != nil {
			t.Fatal(err)
		}
	}
	read, claimed := selectFunc(ctx, parseSelect(t, "select count(*) as n from omni_members"), "")
	if !claimed {
		t.Fatal("stored materialized view not claimed")
	}
	if got := renderCSV(t, "select count(*) as n from omni_members", read()); got != "n\n4\n" {
		t.Fatalf("materialized rows: %q", got)
	}

}

func testStatementReads(t *testing.T, ctx *stagingTestCtx) {
	t.Helper()
	for _, tc := range []struct{ sql, want string }{
		{"select 1 as n union all select 1 as n", "n\n1\n1\n"},
		{"select 1 as n union select 1 as n", "n\n1\n"},
		{"select login, id from fixture.orgs.members where org = 'dummyorg' union all select login, id from fixture.orgs.members where org = 'dummyorg' order by id desc limit 2", "login,id\nb,10\nb,10\n"},
		{"with combined as (select login from fixture.orgs.members where org = 'dummyorg' union all select login from fixture.orgs.members where org = 'dummyorg') select count(*) as n from combined", "n\n8\n"},
	} {
		execute, handled := GenerateStreamFunc(ctx, parseStatement(t, tc.sql))
		if !handled {
			t.Fatalf("statement not claimed: %s", tc.sql)
		}
		if got := renderCSV(t, tc.sql, execute()); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.sql, got, tc.want)
		}
	}
}

func TestValidateStatementDoesNotDispatchRequests(t *testing.T) {
	t.Setenv("FIXTURE_TOKEN", "test-token")
	ctx := newStagingTestCtx(t, "https://unused.invalid")
	for _, test := range []struct {
		sql   string
		valid bool
	}{
		{"select login from fixture.orgs.members where org = 'dummyorg'", true},
		{"select count(*) from fixture.orgs.members where org = 'dummyorg'", true},
		{"select login from fixture.orgs.missing", false},
	} {
		statement, err := sqlparser.Parse(test.sql)
		if err != nil {
			t.Fatal(err)
		}
		handled, err := ValidateStatement(ctx, statement)
		if !handled || (err == nil) != test.valid {
			t.Fatalf("%s: handled=%v err=%v", test.sql, handled, err)
		}
	}
}
