package intrinsic //nolint:testpackage // tests unexported translation

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql-labs/omnisdk/pkg/query"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

func parseSelect(t *testing.T, sql string) *sqlparser.Select {
	t.Helper()
	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		t.Fatalf("parse %q: %v", sql, err)
	}
	sel, isSelect := stmt.(*sqlparser.Select)
	if !isSelect {
		t.Fatalf("%q is not a select", sql)
	}
	return sel
}

func TestTranslateSelectJoins(t *testing.T) {
	withUnstable(t, true)
	sel := parseSelect(t, "select k.name as ring, c.name from stackql_unstable_google.cloudkms.key_rings k "+
		"inner join stackql_unstable_google.cloudkms.crypto_keys c on c.keyRingsId = split_part(k.name, '/', 6) "+
		"left join stackql_unstable_google.cloudkms.crypto_keys c2 on c2.name = c.name "+
		"where k.projectsId = 'p' and k.locationsId = 'global' limit 5")
	dq, err := newDocTranslator("", sqliteDialect{}).selectQuery(sel)
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	type joinShape struct {
		alias, handle string
		form          query.JoinForm
		on            int
	}
	var got []joinShape
	for _, j := range dq.getQuery().From() {
		got = append(got, joinShape{j.Resource().Alias(), j.Resource().Handle(), j.Form(), len(j.On())})
	}
	want := []joinShape{
		{"k", "stackql_unstable_google.cloudkms.key_rings", query.Base, 0},
		{"c", "stackql_unstable_google.cloudkms.crypto_keys", query.Inner, 1},
		{"c2", "stackql_unstable_google.cloudkms.crypto_keys", query.Left, 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("joins: got %+v, want %+v", got, want)
	}
	if n := len(dq.getQuery().Where()); n != 2 {
		t.Fatalf("where conjuncts: got %d, want 2", n)
	}
	if !reflect.DeepEqual(dq.getOutputs(), []string{"ring", "name"}) {
		t.Fatalf("outputs: got %v", dq.getOutputs())
	}
	if dq.getLimit() != 5 {
		t.Fatalf("limit: got %d, want 5", dq.getLimit())
	}
}

func TestTranslateSelectResolvesAgainstRegistry(t *testing.T) {
	withUnstable(t, true)
	sel := parseSelect(t, "select login from stackql_unstable_fixture.orgs.members "+
		"where org = 'dummyorg' and (type = 'User' or not id = 2) and login in ('a', 'b')")
	dq, err := newDocTranslator("", sqliteDialect{}).selectQuery(sel)
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	tables := map[string]omnisdk.Table{}
	for _, j := range dq.getQuery().From() {
		tbl, describeErr := omnisdk.DescribeTable(filepath.Join("testdata", "registry"), j.Resource().Handle())
		if describeErr != nil {
			t.Fatalf("describe: %v", describeErr)
		}
		tables[j.Resource().Alias()] = tbl
	}
	res, err := omnisdk.Resolve(dq.getQuery(), tables)
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	// org is the method's path parameter, so it binds to the node; the OR and
	// the IN list are row filters.
	nodes := res.Graph().Nodes()
	if len(nodes) != 1 || !reflect.DeepEqual(nodes[0].Params(), map[string]string{"org": "dummyorg"}) {
		t.Fatalf("nodes: got %d, first params %v", len(nodes), nodes[0].Params())
	}
	if len(res.Params()) != 0 {
		t.Fatalf("query-wide params: got %v, want none", res.Params())
	}
	if n := len(res.Graph().Filters()); n != 2 {
		t.Fatalf("filters: got %d, want 2", n)
	}
}

func TestTranslateSelectRefusals(t *testing.T) {
	withUnstable(t, true)
	for sql, want := range map[string]string{
		"select login from stackql_unstable_github.orgs.members order by login": "ORDER BY cannot be applied to " +
			"stackql_unstable_* relations; remove it from the query",
		"select distinct login from stackql_unstable_github.orgs.members group by login": "GROUP BY, DISTINCT cannot " +
			"be applied to stackql_unstable_* relations; remove them from the query",
		"select count(*) from stackql_unstable_github.orgs.members": "'count(*)' cannot be applied to " +
			"stackql_unstable_* relations",
		"select login from stackql_unstable_github.orgs.members limit 1, 2": "OFFSET cannot be applied to " +
			"stackql_unstable_* relations",
		"select a.login from stackql_unstable_github.orgs.members a right join " +
			"stackql_unstable_github.orgs.members b on a.login = b.login": "RIGHT JOIN cannot be applied to " +
			"stackql_unstable_* relations",
	} {
		_, err := newDocTranslator("", sqliteDialect{}).selectQuery(parseSelect(t, sql))
		if err == nil || err.Error() != want {
			t.Errorf("%s:\n got %v\nwant %s", sql, err, want)
		}
	}
}

func TestFromDocProvidersRefusesMixing(t *testing.T) {
	withUnstable(t, true)
	sel := parseSelect(t, "select a.login from stackql_unstable_github.orgs.members a "+
		"inner join github.orgs.members b on a.login = b.login")
	isDoc, err := fromDocProviders(sel.From, "")
	if !isDoc {
		t.Fatalf("expected the query to be claimed")
	}
	want := "stackql_unstable_* relations cannot be combined with github.orgs.members in one query"
	if err == nil || err.Error() != want {
		t.Fatalf("got %v, want %s", err, want)
	}
	if isDoc, _ := fromDocProviders(parseSelect(t, "select 1 from github.orgs.members").From, ""); isDoc {
		t.Fatalf("a registry-only query must not be claimed")
	}
}

func TestOutputColumns(t *testing.T) {
	batch := []omnisdk.Row{{"b": 1, "a": 2, "x": 3}}
	cols := outputColumns([]string{"x", starOutput}, batch)
	var got []string
	for _, col := range cols {
		got = append(got, col.name)
	}
	if !reflect.DeepEqual(got, []string{"x", "a", "b"}) {
		t.Fatalf("got %v", got)
	}
	if outputColumns([]string{starOutput}, nil) != nil {
		t.Fatalf("a star waits for a row")
	}
	if n := len(outputColumns([]string{"x", "y"}, nil)); n != 2 {
		t.Fatalf("named outputs are known before any row: got %d", n)
	}
}

func parseStatement(t *testing.T, sql string) sqlparser.Statement {
	t.Helper()
	stmt, err := sqlparser.Parse(sql)
	if err != nil {
		t.Fatalf("parse %q: %v", sql, err)
	}
	return stmt
}

type assignmentShape struct {
	column string
	value  string
}

func assignmentShapes(target query.Target) []assignmentShape {
	var out []assignmentShape
	for _, a := range target.Set() {
		value := "?"
		switch v := a.Value().(type) {
		case query.Literal:
			value = fmt.Sprint(v.Value())
		case query.Column:
			value = v.Qualifier() + "." + v.Name()
		case query.Call:
			value = v.Func() + "()"
		}
		out = append(out, assignmentShape{a.Column(), value})
	}
	return out
}

func TestTranslateMutations(t *testing.T) {
	withUnstable(t, true)
	for _, tc := range []struct {
		sql         string
		verb        query.Verb
		alias       string
		assignments []assignmentShape
		sources     []string
		where       int
		outputs     []string
	}{
		{
			sql: "insert into stackql_unstable_google.cloudkms.key_rings (projectsId, locationsId, keyRingId) " +
				"values ('p', 'global', 'r') returning name, createTime",
			verb:  query.Insert,
			alias: "key_rings",
			assignments: []assignmentShape{
				{"projectsId", "p"}, {"locationsId", "global"}, {"keyRingId", "r"},
			},
			outputs: []string{"name", "createTime"},
		},
		{
			sql: "insert into stackql_unstable_google.cloudkms.crypto_keys (projectsId, locationsId, keyRingsId) " +
				"select 'p', 'global', split_part(k.name, '/', 6) from stackql_unstable_google.cloudkms.key_rings k " +
				"where k.projectsId = 'p' and k.locationsId = 'global'",
			verb:  query.Insert,
			alias: "crypto_keys",
			assignments: []assignmentShape{
				{"projectsId", "p"}, {"locationsId", "global"}, {"keyRingsId", "split_part()"},
			},
			sources: []string{"k"},
			where:   2,
		},
		{
			sql: "update stackql_unstable_github.orgs.orgs o set description = 'd' " +
				"where o.org = 'dummyorg' returning login",
			verb:        query.Update,
			alias:       "o",
			assignments: []assignmentShape{{"description", "d"}},
			where:       1,
			outputs:     []string{"login"},
		},
		{
			sql:   "delete from stackql_unstable_google.compute.firewalls where project = 'p' and firewall in ('a', 'b')",
			verb:  query.Delete,
			alias: "firewalls",
			where: 2,
		},
	} {
		dq, err := newDocTranslator("", sqliteDialect{}).mutation(parseStatement(t, tc.sql))
		if err != nil {
			t.Errorf("%s: %v", tc.sql, err)
			continue
		}
		target := dq.getQuery().Target()
		if target.Verb() != tc.verb || target.Resource().Alias() != tc.alias {
			t.Errorf("%s: target %s %s", tc.sql, target.Verb(), target.Resource().Alias())
		}
		if got := assignmentShapes(target); !reflect.DeepEqual(got, tc.assignments) {
			t.Errorf("%s: assignments %v, want %v", tc.sql, got, tc.assignments)
		}
		var sources []string
		for _, j := range dq.getQuery().From() {
			sources = append(sources, j.Resource().Alias())
		}
		if !reflect.DeepEqual(sources, tc.sources) {
			t.Errorf("%s: sources %v, want %v", tc.sql, sources, tc.sources)
		}
		if n := len(dq.getQuery().Where()); n != tc.where {
			t.Errorf("%s: %d where conjuncts, want %d", tc.sql, n, tc.where)
		}
		if !reflect.DeepEqual(dq.getOutputs(), tc.outputs) {
			t.Errorf("%s: outputs %v, want %v", tc.sql, dq.getOutputs(), tc.outputs)
		}
	}
}

func TestTranslateMutationRefusals(t *testing.T) {
	withUnstable(t, true)
	for sql, want := range map[string]string{
		"insert into stackql_unstable_google.cloudkms.key_rings (projectsId) values ('a'), ('b')": "an INSERT " +
			"into stackql_unstable_* relations takes exactly one VALUES row; got 2",
		"insert into stackql_unstable_google.cloudkms.key_rings (projectsId, locationsId) values ('a')": "INSERT " +
			"names 2 columns but supplies 1 values",
		"update stackql_unstable_github.orgs.orgs set description = 'd' where org = 'x' limit 1": "ORDER BY and " +
			"LIMIT cannot be applied to an UPDATE of stackql_unstable_* relations",
		"delete from stackql_unstable_google.compute.firewalls where project = 'p' limit 1": "ORDER BY and " +
			"LIMIT cannot be applied to a DELETE of stackql_unstable_* relations",
	} {
		_, err := newDocTranslator("", sqliteDialect{}).mutation(parseStatement(t, sql))
		if err == nil || err.Error() != want {
			t.Errorf("%s:\n got %v\nwant %s", sql, err, want)
		}
	}
}

func TestMutationTablesIgnoresImplicitDual(t *testing.T) {
	withUnstable(t, true)
	tables, isMutation := mutationTables(parseStatement(t,
		"update stackql_unstable_github.orgs.orgs set description = 'd' where org = 'x'"))
	if !isMutation || len(tables) != 1 {
		t.Fatalf("got %d tables (mutation %v), want the target alone", len(tables), isMutation)
	}
	isDoc, err := fromDocProviders(tables, "")
	if !isDoc || err != nil {
		t.Fatalf("got isDoc %v, err %v", isDoc, err)
	}
}

func TestDocProviderUnderOmniAll(t *testing.T) {
	previous := previewCfg
	t.Cleanup(func() { previewCfg = previous })
	for _, tc := range []struct {
		cfg        previewCfgDTO
		name       string
		wantBundle string
		wantDoc    bool
	}{
		{cfg: previewCfgDTO{}, name: "aws", wantDoc: false},
		{cfg: previewCfgDTO{Unstable: true}, name: "aws", wantDoc: false},
		{cfg: previewCfgDTO{Unstable: true}, name: "stackql_unstable_aws", wantBundle: "aws", wantDoc: true},
		{cfg: previewCfgDTO{Omni: "all"}, name: "aws", wantBundle: "aws", wantDoc: true},
		{cfg: previewCfgDTO{Omni: "all"}, name: "stackql_unstable_aws", wantBundle: "aws", wantDoc: true},
		{cfg: previewCfgDTO{Omni: "all"}, name: ProviderName, wantDoc: false},
		{cfg: previewCfgDTO{Omni: "all"}, name: "", wantDoc: false},
	} {
		previewCfg = newBackendInput(tc.cfg)
		bundle, isDoc := docProvider(tc.name)
		if bundle != tc.wantBundle || isDoc != tc.wantDoc {
			t.Errorf("%+v %q: got %q %v, want %q %v", tc.cfg, tc.name, bundle, isDoc, tc.wantBundle, tc.wantDoc)
		}
	}
}

// Predicates SQL spells as operators become calls omnisdk evaluates: IS [NOT] NULL, [NOT] LIKE with an
// optional ESCAPE, and [NOT] BETWEEN.
func TestTranslateOperatorPredicates(t *testing.T) {
	withUnstable(t, true)
	for cond, want := range map[string]string{
		"login is null":                   "is_null(login)",
		"login is not null":               "not is_null(login)",
		"login like 'a%'":                 "like(a%, login)",
		"login not like 'a!%' escape '!'": "not like(a!%, login, !)",
		"id between 1 and 10":             "between(id, 1, 10)",
		"id not between 1 and 10":         "not between(id, 1, 10)",
	} {
		sel := parseSelect(t, "select login from stackql_unstable_github.orgs.members where "+cond)
		dq, err := newDocTranslator("", sqliteDialect{}).selectQuery(sel)
		if err != nil {
			t.Fatalf("%s: %v", cond, err)
		}
		where := dq.getQuery().Where()
		if len(where) != 1 {
			t.Fatalf("%s: %d conjuncts", cond, len(where))
		}
		if got := describePredicate(where[0]); got != want {
			t.Errorf("%s: got %s, want %s", cond, got, want)
		}
	}
}

// On a Postgres backend LIKE is Postgres's like(value, pattern), an ESCAPE clause rewriting the
// pattern through like_escape as Postgres's parser does.
func TestTranslateLikePostgres(t *testing.T) {
	withUnstable(t, true)
	for cond, want := range map[string]string{
		"login like 'a%'":                 "like(login, a%)",
		"login not like 'a!%' escape '!'": "not like(login, like_escape(a!%, !))",
	} {
		sel := parseSelect(t, "select login from stackql_unstable_github.orgs.members where "+cond)
		dq, err := newDocTranslator("", postgresDialect{}).selectQuery(sel)
		if err != nil {
			t.Fatalf("%s: %v", cond, err)
		}
		if got := describePredicate(dq.getQuery().Where()[0]); got != want {
			t.Errorf("%s: got %s, want %s", cond, got, want)
		}
	}
}

// A comma-separated FROM, CROSS JOIN and a JOIN with no condition are cross joins; JOIN ... USING is
// an inner join on each named column.
func TestTranslateCrossAndUsing(t *testing.T) {
	withUnstable(t, true)
	for sql, want := range map[string][]query.JoinForm{
		"select a.login from stackql_unstable_github.orgs.members a, stackql_unstable_github.orgs.members b":           {query.Base, query.Cross},
		"select a.login from stackql_unstable_github.orgs.members a cross join stackql_unstable_github.orgs.members b": {query.Base, query.Cross},
		"select a.login from stackql_unstable_github.orgs.members a join stackql_unstable_github.orgs.members b":       {query.Base, query.Cross},
	} {
		dq, err := newDocTranslator("", sqliteDialect{}).selectQuery(parseSelect(t, sql))
		if err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
		var got []query.JoinForm
		for _, j := range dq.getQuery().From() {
			got = append(got, j.Form())
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s: forms %v, want %v", sql, got, want)
		}
	}
	dq, err := newDocTranslator("", sqliteDialect{}).selectQuery(parseSelect(t, "select a.login from stackql_unstable_github.orgs.members a "+
		"join stackql_unstable_github.orgs.members b using (login, id)"))
	if err != nil {
		t.Fatal(err)
	}
	b := dq.getQuery().From()[1]
	var on []string
	for _, p := range b.On() {
		on = append(on, describePredicate(p))
	}
	if b.Form() != query.Inner || strings.Join(on, "; ") != "a.login = b.login; a.id = b.id" {
		t.Errorf("using: form %v, on %v", b.Form(), on)
	}
}

// describePredicate renders the predicate shapes these tests build.
func describePredicate(p query.Predicate) string {
	switch p := p.(type) {
	case query.Not:
		return "not " + describePredicate(p.Negated())
	case query.Test:
		return describeExpr(p.Cond())
	case query.Compare:
		return describeExpr(p.Left()) + " " + string(p.Op()) + " " + describeExpr(p.Right())
	}
	return fmt.Sprintf("%T", p)
}

func describeExpr(e query.Expr) string {
	switch e := e.(type) {
	case query.Column:
		if e.Qualifier() == "" {
			return e.Name()
		}
		return e.Qualifier() + "." + e.Name()
	case query.Literal:
		return fmt.Sprint(e.Value())
	case query.Call:
		var args []string
		for _, a := range e.Args() {
			args = append(args, describeExpr(a))
		}
		return e.Func() + "(" + strings.Join(args, ", ") + ")"
	}
	return fmt.Sprintf("%T", e)
}
