package sql_system //nolint:revive,stylecheck // package name is meaningful and readable

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/astformat"
	"github.com/stackql/stackql/internal/stackql/astfuncrewrite"
	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/relationaldto"
	"github.com/stackql/stackql/internal/stackql/sqlcontrol"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/typing"
)

type SQLSystem interface {
	ComposeSelectQuery([]typing.RelationalColumn, []string, []string, string, string, string, int) (string, error)
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
	CreateView(viewName string, rawDDL string, replaceAllowed bool) error
	DropView(viewName string) error
	GetViewByName(viewName string) (internaldto.RelationDTO, bool)

	// Materialized Views
	CreateMaterializedView(
		relationName string,
		colz []typing.RelationalColumn,
		rawDDL string,
		replaceAllowed bool,
		selectQuery string,
		varargs ...any,
	) error
	RefreshMaterializedView(viewName string,
		colz []typing.RelationalColumn,
		selectQuery string,
		varargs ...any) error
	DropMaterializedView(viewName string) error
	GetMaterializedViewByName(viewName string) (internaldto.RelationDTO, bool)
	QueryMaterializedView(colzString, actualRelationName, whereClause string) (*sql.Rows, error)

	// Tables, both permanent and temp
	CreatePhysicalTable(
		relationName string,
		colz []typing.RelationalColumn,
		rawDDL string,
		ifNotExists bool,
	) error
	DropPhysicalTable(
		tableName string,
		ifExists bool,
	) error
	GetPhysicalTableByName(
		tableName string,
	) (internaldto.RelationDTO, bool)
	InsertIntoPhysicalTable(tableName string,
		insertClause string,
		selectQuery string,
		varargs ...any) error

	// External SQL data sources
	RegisterExternalTable(connectionName string, tableDetails anysdk.SQLExternalTable) error
	ObtainRelationalColumnFromExternalSQLtable(
		hierarchyIDs internaldto.HeirarchyIdentifiers,
		colName string,
	) (typing.RelationalColumn, error)
	ObtainRelationalColumnsFromExternalSQLtable(
		hierarchyIDs internaldto.HeirarchyIdentifiers,
	) ([]typing.RelationalColumn, error)

	GetFullyQualifiedRelationName(tableName string) string
	DelimitFullyQualifiedRelationName(string) string
	IsRelationExported(relationName string) bool
}

func getNodeFormatter(name string) sqlparser.NodeFormatter {
	if name == constants.SQLDialectPostgres {
		return astformat.PostgresSelectExprsFormatter
	}
	if name == constants.SQLDialectSQLite3 {
		return astformat.SQLiteSelectExprsFormatter
	}
	return astformat.DefaultSelectExprsFormatter
}

func NewSQLSystem(
	sqlEngine sqlengine.SQLEngine,
	analyticsNamespaceLikeString string,
	controlAttributes sqlcontrol.ControlAttributes,
	sqlCfg dto.SQLBackendCfg,
	authCfg map[string]*dto.AuthCtx,
	typCfg typing.Config,
	exportNamepsace string,
) (SQLSystem, error) {
	name := sqlCfg.SQLSystem
	nameLowered := strings.ToLower(name)
	formatter := getNodeFormatter(nameLowered)
	switch nameLowered {
	case constants.SQLDialectSQLite3:
		return newSQLiteSystem(
			sqlEngine,
			analyticsNamespaceLikeString,
			controlAttributes,
			formatter,
			sqlCfg,
			authCfg,
			typCfg,
			exportNamepsace,
		)
	case constants.SQLDialectPostgres:
		return newPostgresSystem(
			sqlEngine,
			analyticsNamespaceLikeString,
			controlAttributes,
			formatter,
			sqlCfg,
			authCfg,
			typCfg,
			exportNamepsace,
		)
	default:
		return nil, fmt.Errorf("cannot initialise sql system: cannot accomodate sql dialect '%s'", name)
	}
}
