package drm

import (
	"strings"

	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/sql_system"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
	"github.com/stackql/stackql/internal/stackql/typing"
)

var (
	_ PreparedStatementCtx = (*standardPreparedStatementCtx)(nil)
)

type PreparedStatementCtx interface {
	GetAllCtrlCtrs() []internaldto.TxnControlCounters
	GetGCCtrlCtrs() internaldto.TxnControlCounters
	GetIndirectContexts() []PreparedStatementCtx
	GetCtrlColumnRepeats() int
	GetNonControlColumns() []typing.ColumnMetadata
	GetGCHousekeepingQueries() string
	GetQuery() string
	SetGCCtrlCtrs(tcc internaldto.TxnControlCounters)
	SetIndirectContexts(indirectContexts []PreparedStatementCtx)
	SetKind(kind string)
	UpdateQuery(q string)
}

type standardPreparedStatementCtx struct {
	query                   string
	kind                    string // string annotation applicable only in some cases eg UNION [ALL]
	genIDControlColName     string
	sessionIDControlColName string
	TableNames              []string
	txnIDControlColName     string
	insIDControlColName     string
	insEncodedColName       string
	nonControlColumns       []typing.ColumnMetadata
	ctrlColumnRepeats       int
	txnCtrlCtrs             internaldto.TxnControlCounters
	selectTxnCtrlCtrs       []internaldto.TxnControlCounters
	namespaceCollection     tablenamespace.Collection
	sqlSystem               sql_system.SQLSystem
	indirectContexts        []PreparedStatementCtx
	parameters              map[string]interface{}
}

func (ps *standardPreparedStatementCtx) SetIndirectContexts(indirectContexts []PreparedStatementCtx) {
	ps.indirectContexts = indirectContexts
}

func (ps *standardPreparedStatementCtx) GetIndirectContexts() []PreparedStatementCtx {
	return ps.indirectContexts
}

func (ps *standardPreparedStatementCtx) GetCtrlColumnRepeats() int {
	return ps.ctrlColumnRepeats
}

func (ps *standardPreparedStatementCtx) SetKind(kind string) {
	ps.kind = kind
}

func (ps *standardPreparedStatementCtx) GetQuery() string {
	return ps.query
}

func (ps *standardPreparedStatementCtx) UpdateQuery(q string) {
	ps.query = q
}

func (ps *standardPreparedStatementCtx) GetGCCtrlCtrs() internaldto.TxnControlCounters {
	return ps.txnCtrlCtrs
}

func (ps *standardPreparedStatementCtx) SetGCCtrlCtrs(tcc internaldto.TxnControlCounters) {
	ps.txnCtrlCtrs = tcc
}

func (ps *standardPreparedStatementCtx) GetNonControlColumns() []typing.ColumnMetadata {
	return ps.nonControlColumns
}

func (ps *standardPreparedStatementCtx) GetAllCtrlCtrs() []internaldto.TxnControlCounters {
	var rv []internaldto.TxnControlCounters
	rv = append(rv, ps.txnCtrlCtrs)
	rv = append(rv, ps.selectTxnCtrlCtrs...)
	return rv
}

func NewPreparedStatementCtx(
	query string,
	kind string,
	genIDControlColName string,
	sessionIDControlColName string,
	tableNames []string,
	txnIDControlColName string,
	insIDControlColName string,
	insEncodedColName string,
	nonControlColumns []typing.ColumnMetadata,
	ctrlColumnRepeats int,
	txnCtrlCtrs internaldto.TxnControlCounters,
	secondaryCtrs []internaldto.TxnControlCounters,
	namespaceCollection tablenamespace.Collection,
	sqlSystem sql_system.SQLSystem,
	parameters map[string]interface{},
) PreparedStatementCtx {
	return &standardPreparedStatementCtx{
		query:                   query,
		kind:                    kind,
		genIDControlColName:     genIDControlColName,
		sessionIDControlColName: sessionIDControlColName,
		TableNames:              tableNames,
		txnIDControlColName:     txnIDControlColName,
		insIDControlColName:     insIDControlColName,
		insEncodedColName:       insEncodedColName,
		nonControlColumns:       nonControlColumns,
		ctrlColumnRepeats:       ctrlColumnRepeats,
		txnCtrlCtrs:             txnCtrlCtrs,
		selectTxnCtrlCtrs:       secondaryCtrs,
		namespaceCollection:     namespaceCollection,
		sqlSystem:               sqlSystem,
		parameters:              parameters,
	}
}

func NewQueryOnlyPreparedStatementCtx(query string, nonControlCols []typing.ColumnMetadata) PreparedStatementCtx {
	return &standardPreparedStatementCtx{query: query, nonControlColumns: nonControlCols, ctrlColumnRepeats: 0}
}

func (ps *standardPreparedStatementCtx) GetGCHousekeepingQueries() string {
	var housekeepingQueries []string
	for _, table := range ps.TableNames {
		housekeepingQueries = append(housekeepingQueries, ps.sqlSystem.GetGCHousekeepingQuery(table, ps.txnCtrlCtrs))
	}
	return strings.Join(housekeepingQueries, "; ")
}
