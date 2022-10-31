package sqldialect

import (
	"fmt"
	"strings"

	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/relationaldto"
	"github.com/stackql/stackql/internal/stackql/sqlcontrol"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
)

type SQLDialect interface {
	ComposeSelectQuery([]relationaldto.RelationalColumn, []string, string, string, string) (string, error)
	DelimitGroupByColumn(term string) string
	DelimitOrderByColumn(term string) string
	// GCAdd() will record a Txn as active
	GCAdd(string, dto.TxnControlCounters, dto.TxnControlCounters) error
	// GCCollectAll() will remove all records from data tables.
	GCCollectAll() error
	// GCCollectObsoleted() must be mutex-protected.
	GCCollectObsoleted(minTransactionID int) error
	// GCControlTablesPurge() will remove all data from non ring control tables.
	GCControlTablesPurge() error
	// GCPurgeCache() will completely wipe the cache.
	GCPurgeCache() error
	// GCPurgeCache() will completely wipe the cache.
	GCPurgeEphemeral() error
	//
	GenerateDDL(relationaldto.RelationalTable, bool) ([]string, error)
	GenerateInsertDML(relationaldto.RelationalTable, *dto.TxnControlCounters) (string, error)
	GenerateSelectDML(relationaldto.RelationalTable, *dto.TxnControlCounters, string, string) (string, error)
	GetGCHousekeepingQuery(string, dto.TxnControlCounters) string
	//
	GetSQLEngine() sqlengine.SQLEngine
	// PurgeAll() drops all data tables, does **not** drop control tables.
	PurgeAll() error
	SanitizeQueryString(queryString string) (string, error)
	// **NOTE**: SanitizeWhereQueryString() is **NOT** idempotent.
	SanitizeWhereQueryString(queryString string) (string, error)
	GetOperatorOr() string
	GetOperatorStringConcat() string
}

func NewSQLDialect(sqlEngine sqlengine.SQLEngine, namespaces tablenamespace.TableNamespaceCollection, controlAttributes sqlcontrol.ControlAttributes, name string) (SQLDialect, error) {
	switch strings.ToLower(name) {
	case constants.SQLDialectSQLite3:
		return newSQLiteDialect(sqlEngine, namespaces, controlAttributes)
	case constants.SQLDialectPostgres:
		return newPostgresDialect(sqlEngine, namespaces, controlAttributes)
	default:
		return nil, fmt.Errorf("cannot accomodate sql dialect '%s'", name)
	}
}
