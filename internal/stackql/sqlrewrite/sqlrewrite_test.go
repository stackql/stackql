package sqlrewrite //nolint:testpackage

import (
	"errors"
	"reflect"
	"testing"

	"github.com/stackql/any-sdk/pkg/db/sqlcontrol"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/sql_system"
	"github.com/stackql/stackql/internal/stackql/tableinsertioncontainer"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
	"github.com/stackql/stackql/internal/stackql/typing"
)

type fakeControlAttributes struct{ sqlcontrol.ControlAttributes }

func (fakeControlAttributes) GetControlGenIDColumnName() string           { return "gen_id" }
func (fakeControlAttributes) GetControlSsnIDColumnName() string           { return "ssn_id" }
func (fakeControlAttributes) GetControlTxnIDColumnName() string           { return "txn_id" }
func (fakeControlAttributes) GetControlInsIDColumnName() string           { return "ins_id" }
func (fakeControlAttributes) GetControlInsertEncodedIDColumnName() string { return "ins_enc" }

type fakeSQLSystem struct {
	sql_system.SQLSystem
	query, err             string
	aliases, hoisted, rest []string
	offset                 int
}

func (s *fakeSQLSystem) ComposeSelectQuery(
	_ []typing.RelationalColumn, aliases, hoisted []string, from, where, qualifier, suffix string,
	offset int, _ map[string][]internaldto.TxnControlCounters,
) (string, error) {
	s.aliases, s.hoisted, s.rest, s.offset = aliases, hoisted, []string{from, where, qualifier, suffix}, offset
	if s.err != "" {
		return "", errors.New(s.err)
	}
	return s.query, nil
}

type fakeDRMConfig struct {
	drm.Config
	sqlSystem *fakeSQLSystem
}

func (c fakeDRMConfig) GetControlAttributes() sqlcontrol.ControlAttributes {
	return fakeControlAttributes{}
}
func (c fakeDRMConfig) GetSQLSystem() sql_system.SQLSystem                { return c.sqlSystem }
func (c fakeDRMConfig) GetNamespaceCollection() tablenamespace.Collection { return nil }

type fakeTable struct {
	tablemetadata.ExtendedTableMetadata
	alias                             string
	hoistable, physical, materialized bool
}

func (m *fakeTable) GetAlias() string          { return m.alias }
func (m *fakeTable) IsOnClauseHoistable() bool { return m.hoistable }
func (m *fakeTable) IsPhysicalTable() bool     { return m.physical }
func (m *fakeTable) IsMaterializedView() bool  { return m.materialized }

type fakeContainer struct {
	tableinsertioncontainer.TableInsertionContainer
	tm       *fakeTable
	counters internaldto.TxnControlCounters
}

func (c *fakeContainer) GetTableMetadata() tablemetadata.ExtendedTableMetadata { return c.tm }
func (c *fakeContainer) GetTableTxnCounters() (string, internaldto.TxnControlCounters) {
	return c.tm.alias, c.counters
}

func counters(n int) internaldto.TxnControlCounters {
	return internaldto.NewTxnControlCountersFromVals(n, n, n, n)
}

func newInput(
	s *fakeSQLSystem, base internaldto.TxnControlCounters, secondary []internaldto.TxnControlCounters,
	tables taxonomy.TblMap, containers []tableinsertioncontainer.TableInsertionContainer, hoisted []sqlparser.SQLNode,
) SQLRewriteInput {
	return NewStandardSQLRewriteInput(
		fakeDRMConfig{sqlSystem: s}, []typing.RelationalColumn{typing.NewRelationalColumn("name", "text")},
		base, "ORDER BY name", "name = 'x'", secondary, tables, "FROM t", containers, nil, hoisted,
		map[string]interface{}{"p": 1},
	)
}

type want struct {
	err         string
	query       string
	counters    []internaldto.TxnControlCounters
	repeats     int
	indirect    int
	aliases     []string
	hoisted     []string
	rest        []string
	offset      int
	aliasMap    map[string][]internaldto.TxnControlCounters
	orderedTccs []internaldto.TxnControlCounters
}

func TestGenerateRewrittenSelectDML(t *testing.T) {
	tests := []struct {
		name  string
		input func(s *fakeSQLSystem) SQLRewriteInput
		sys   *fakeSQLSystem
		want  want
	}{
		{
			name: "without containers uses base counters",
			sys:  &fakeSQLSystem{query: "SELECT name FROM t"},
			input: func(s *fakeSQLSystem) SQLRewriteInput {
				tables := taxonomy.TblMap{
					&sqlparser.AliasedTableExpr{}: &fakeTable{alias: "t"},
					&sqlparser.AliasedTableExpr{}: &fakeTable{alias: "p", physical: true},
					&sqlparser.AliasedTableExpr{}: &fakeTable{alias: "m", materialized: true},
				}
				indirect := []drm.PreparedStatementCtx{drm.NewQueryOnlyPreparedStatementCtx("SELECT 1", nil)}
				return newInput(s, counters(1), []internaldto.TxnControlCounters{counters(2)}, tables, nil, nil).
					WithSelectQualifier("DISTINCT").WithPrepStmtOffset(2).WithIndirectContexts(indirect)
			},
			want: want{
				query:    "SELECT name FROM t",
				counters: []internaldto.TxnControlCounters{counters(1), counters(2)},
				// Only table t is counted. Physical tables (p) and materialized views (m) are not.
				repeats:  1,
				indirect: 1,
				aliases:  []string{},
				hoisted:  []string{},
				rest:     []string{"FROM t", "name = 'x'", "DISTINCT", "ORDER BY name"},
				offset:   2,
			},
		},
		{
			name: "with containers and a hoisted table",
			sys:  &fakeSQLSystem{query: "SELECT 1"},
			input: func(s *fakeSQLSystem) SQLRewriteInput {
				hoistedNode := &sqlparser.AliasedTableExpr{}
				hoistedTable := &fakeTable{alias: "h", hoistable: true}
				containers := []tableinsertioncontainer.TableInsertionContainer{
					&fakeContainer{tm: &fakeTable{alias: "a"}, counters: counters(2)},
					&fakeContainer{tm: hoistedTable, counters: counters(1)},
					&fakeContainer{tm: &fakeTable{alias: "a"}, counters: counters(3)},
				}
				// When there are containers, the base counters (9) are not used.
				return newInput(s, counters(9), nil, taxonomy.TblMap{hoistedNode: hoistedTable}, containers,
					[]sqlparser.SQLNode{hoistedNode})
			},
			want: want{
				query: "SELECT 1",
				// The hoisted table (h) comes first, then the other tables in order.
				counters: []internaldto.TxnControlCounters{counters(1), counters(2), counters(3)},
				repeats:  1,
				hoisted:  []string{"h"},
				// Alias a appears twice but is sent to ComposeSelectQuery once.
				aliases: []string{"a"},
				rest:    []string{"FROM t", "name = 'x'", "", "ORDER BY name"},
				// Counters for the same table are kept together in one list.
				aliasMap: map[string][]internaldto.TxnControlCounters{
					"a": {counters(2), counters(3)},
					"h": {counters(1)},
				},
				orderedTccs: []internaldto.TxnControlCounters{counters(2), counters(3)},
			},
		},
		{
			name: "hoisted table without a matching alias",
			sys:  &fakeSQLSystem{},
			input: func(s *fakeSQLSystem) SQLRewriteInput {
				hoistedNode := &sqlparser.AliasedTableExpr{}
				tables := taxonomy.TblMap{hoistedNode: &fakeTable{alias: "other", hoistable: true}}
				containers := []tableinsertioncontainer.TableInsertionContainer{
					&fakeContainer{tm: &fakeTable{alias: "h", hoistable: true}},
				}
				return newInput(s, nil, nil, tables, containers, []sqlparser.SQLNode{hoistedNode})
			},
			want: want{err: "could not find alias for hoisted table"},
		},
		{
			name: "compose error is returned",
			sys:  &fakeSQLSystem{err: "compose failed"},
			input: func(s *fakeSQLSystem) SQLRewriteInput {
				return newInput(s, nil, nil, nil, nil, nil)
			},
			want: want{err: "compose failed"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := tc.input(tc.sys)
			ctx, err := GenerateRewrittenSelectDML(input)
			if tc.want.err != "" {
				if err == nil || err.Error() != tc.want.err {
					t.Fatalf("expected error %q, got %v", tc.want.err, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			checkContext(t, ctx, tc.want)
			checkComposeArgs(t, tc.sys, tc.want)
			if tc.want.aliasMap != nil && !reflect.DeepEqual(input.GetAliasToControlCounterMap(), tc.want.aliasMap) {
				t.Errorf("unexpected alias to counter map %v", input.GetAliasToControlCounterMap())
			}
		})
	}
}

func checkContext(t *testing.T, ctx drm.PreparedStatementCtx, w want) {
	t.Helper()
	if ctx.GetQuery() != w.query {
		t.Errorf("unexpected query %q", ctx.GetQuery())
	}
	if !reflect.DeepEqual(ctx.GetAllCtrlCtrs(), w.counters) {
		t.Errorf("unexpected control counters %v", ctx.GetAllCtrlCtrs())
	}
	if ctx.GetCtrlColumnRepeats() != w.repeats {
		t.Errorf("expected %d control column repeats, got %d", w.repeats, ctx.GetCtrlColumnRepeats())
	}
	if len(ctx.GetNonControlColumns()) != 1 || len(ctx.GetIndirectContexts()) != w.indirect {
		t.Error("expected columns and indirect contexts to be carried over")
	}
	if w.orderedTccs != nil && !reflect.DeepEqual(ctx.GetOrderedTccs(), w.orderedTccs) {
		t.Errorf("unexpected ordered control counters %v", ctx.GetOrderedTccs())
	}
}

func checkComposeArgs(t *testing.T, s *fakeSQLSystem, w want) {
	t.Helper()
	if !sameStrings(s.aliases, w.aliases) || !sameStrings(s.hoisted, w.hoisted) {
		t.Errorf("unexpected aliases %v and hoisted aliases %v", s.aliases, s.hoisted)
	}
	if !reflect.DeepEqual(s.rest, w.rest) || s.offset != w.offset {
		t.Errorf("unexpected ComposeSelectQuery arguments %v, offset %d", s.rest, s.offset)
	}
}

func sameStrings(a, b []string) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	return reflect.DeepEqual(a, b)
}
