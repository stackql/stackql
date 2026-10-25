package sql_system //nolint:stylecheck,revive // package name is meaningful and readable

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/stackql/stackql/internal/stackql/constants"
	dto "github.com/stackql/stackql/internal/stackql/runtimeconfig"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/astfuncrewrite"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/typing"
	"github.com/stackql/stackql/pkg/astformat"
)

type SQLSystem interface {
	DelimitGroupByColumn(term string) string
	DelimitOrderByColumn(term string) string
	GCPurgeCache() error
	GetASTFormatter() sqlparser.NodeFormatter
	GetASTFuncRewriter() astfuncrewrite.ASTFuncRewriter
	GetFullyQualifiedTableName(string) (string, error)
	GetSQLEngine() sqlengine.SQLEngine
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
	TableOldestUpdateUTC(string, string, string, string) (time.Time, bool)

	GetCurrentTable(internaldto.HeirarchyIdentifiers) (internaldto.DBTable, error)
	GetTable(internaldto.HeirarchyIdentifiers, int) (internaldto.DBTable, error)

	// Views
	CreateView(viewName string, rawDDL string, replaceAllowed bool, requiredParams []string) error
	DropView(viewName string) error
	GetViewByName(viewName string) (internaldto.RelationDTO, bool)
	GetViewByNameAndParameters(viewName string, params map[string]any) (internaldto.RelationDTO, bool)

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
	RegisterExternalTable(connectionName string, tableDetails sqlengine.SQLExternalTable) error
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
