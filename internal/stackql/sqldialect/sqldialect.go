package sqldialect

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/stackql/stackql/internal/stackql/astformat"
	"github.com/stackql/stackql/internal/stackql/astfuncrewrite"
	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/internaldto"
	"github.com/stackql/stackql/internal/stackql/relationaldto"
	"github.com/stackql/stackql/internal/stackql/sqlcontrol"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"vitess.io/vitess/go/vt/sqlparser"
)

type SQLDialect interface {
	ComposeSelectQuery([]relationaldto.RelationalColumn, []string, string, string, string) (string, error)
	DelimitGroupByColumn(term string) string
	DelimitOrderByColumn(term string) string
	// GCAdd() will record a Txn as active
	GCAdd(string, internaldto.TxnControlCounters, internaldto.TxnControlCounters) error
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
	GenerateInsertDML(relationaldto.RelationalTable, internaldto.TxnControlCounters) (string, error)
	GenerateSelectDML(relationaldto.RelationalTable, internaldto.TxnControlCounters, string, string) (string, error)
	GetGCHousekeepingQuery(string, internaldto.TxnControlCounters) string
	//
	GetASTFormatter() sqlparser.NodeFormatter
	GetASTFuncRewriter() astfuncrewrite.ASTFuncRewriter
	GetFullyQualifiedTableName(string) (string, error)
	GetSQLEngine() sqlengine.SQLEngine
	// PurgeAll() drops all data tables, does **not** drop control tables.
	PurgeAll() error
	SanitizeQueryString(queryString string) (string, error)
	// **NOTE**: SanitizeWhereQueryString() is **NOT** idempotent.
	SanitizeWhereQueryString(queryString string) (string, error)
	GetOperatorOr() string
	GetOperatorStringConcat() string
	GetName() string
	GetGolangKind(string) reflect.Kind
	GetGolangValue(string) interface{}
	GetRelationalType(string) string

	QueryNamespaced(string, string, string, string) (*sql.Rows, error)

	IsTablePresent(string, string, string) bool
	TableOldestUpdateUTC(string, string, string, string) (time.Time, internaldto.TxnControlCounters)

	GetCurrentTable(internaldto.HeirarchyIdentifiers) (internaldto.DBTable, error)
	GetTable(internaldto.HeirarchyIdentifiers, int) (internaldto.DBTable, error)

	// Views
	CreateView(viewName string, rawDDL string, translatedDDL string) error
	GetViewByName(viewName string) (internaldto.ViewDTO, bool)
}

func getNodeFormatter(name string) sqlparser.NodeFormatter {
	if name == constants.SQLDialectPostgres {
		return astformat.PostgresSelectExprsFormatter
	}
	return nil
}

func NewSQLDialect(sqlEngine sqlengine.SQLEngine, analyticsNamespaceLikeString string, controlAttributes sqlcontrol.ControlAttributes, sqlCfg dto.SQLBackendCfg) (SQLDialect, error) {
	name := sqlCfg.SQLDialect
	nameLowered := strings.ToLower(name)
	formatter := getNodeFormatter(nameLowered)
	switch nameLowered {
	case constants.SQLDialectSQLite3:
		return newSQLiteDialect(sqlEngine, analyticsNamespaceLikeString, controlAttributes, formatter, sqlCfg)
	case constants.SQLDialectPostgres:
		return newPostgresDialect(sqlEngine, analyticsNamespaceLikeString, controlAttributes, formatter, sqlCfg)
	default:
		return nil, fmt.Errorf("cannot accomodate sql dialect '%s'", name)
	}
}
