package drm

import (
	"strings"

	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/sqldialect"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
)

type PreparedStatementCtx interface {
	GetAllCtrlCtrs() []dto.TxnControlCounters
	GetGCCtrlCtrs() dto.TxnControlCounters
	GetNonControlColumns() []ColumnMetadata
	GetGCHousekeepingQueries() string
	GetQuery() string
	SetGCCtrlCtrs(tcc dto.TxnControlCounters)
	SetKind(kind string)
}

type standardPreparedStatementCtx struct {
	query                   string
	kind                    string // string annotation applicable only in some cases eg UNION [ALL]
	genIdControlColName     string
	sessionIdControlColName string
	TableNames              []string
	txnIdControlColName     string
	insIdControlColName     string
	insEncodedColName       string
	nonControlColumns       []ColumnMetadata
	ctrlColumnRepeats       int
	txnCtrlCtrs             dto.TxnControlCounters
	selectTxnCtrlCtrs       []dto.TxnControlCounters
	namespaceCollection     tablenamespace.TableNamespaceCollection
	sqlDialect              sqldialect.SQLDialect
}

func (ps *standardPreparedStatementCtx) SetKind(kind string) {
	ps.kind = kind
}

func (ps *standardPreparedStatementCtx) GetQuery() string {
	return ps.query
}

func (ps *standardPreparedStatementCtx) GetGCCtrlCtrs() dto.TxnControlCounters {
	return ps.txnCtrlCtrs
}

func (ps *standardPreparedStatementCtx) SetGCCtrlCtrs(tcc dto.TxnControlCounters) {
	ps.txnCtrlCtrs = tcc
}

func (ps *standardPreparedStatementCtx) GetNonControlColumns() []ColumnMetadata {
	return ps.nonControlColumns
}

func (ps *standardPreparedStatementCtx) GetAllCtrlCtrs() []dto.TxnControlCounters {
	var rv []dto.TxnControlCounters
	rv = append(rv, ps.txnCtrlCtrs)
	rv = append(rv, ps.selectTxnCtrlCtrs...)
	return rv
}

func NewPreparedStatementCtx(
	query string,
	kind string,
	genIdControlColName string,
	sessionIdControlColName string,
	tableNames []string,
	txnIdControlColName string,
	insIdControlColName string,
	insEncodedColName string,
	nonControlColumns []ColumnMetadata,
	ctrlColumnRepeats int,
	txnCtrlCtrs dto.TxnControlCounters,
	secondaryCtrs []dto.TxnControlCounters,
	namespaceCollection tablenamespace.TableNamespaceCollection,
	sqlDialect sqldialect.SQLDialect,
) PreparedStatementCtx {
	return &standardPreparedStatementCtx{
		query:                   query,
		kind:                    kind,
		genIdControlColName:     genIdControlColName,
		sessionIdControlColName: sessionIdControlColName,
		TableNames:              tableNames,
		txnIdControlColName:     txnIdControlColName,
		insIdControlColName:     insIdControlColName,
		insEncodedColName:       insEncodedColName,
		nonControlColumns:       nonControlColumns,
		ctrlColumnRepeats:       ctrlColumnRepeats,
		txnCtrlCtrs:             txnCtrlCtrs,
		selectTxnCtrlCtrs:       secondaryCtrs,
		namespaceCollection:     namespaceCollection,
		sqlDialect:              sqlDialect,
	}
}

func NewQueryOnlyPreparedStatementCtx(query string) PreparedStatementCtx {
	return &standardPreparedStatementCtx{query: query, ctrlColumnRepeats: 0}
}

func (ps *standardPreparedStatementCtx) GetGCHousekeepingQueries() string {
	var housekeepingQueries []string
	for _, table := range ps.TableNames {
		housekeepingQueries = append(housekeepingQueries, ps.sqlDialect.GetGCHousekeepingQuery(table, ps.txnCtrlCtrs))
	}
	return strings.Join(housekeepingQueries, "; ")
}
