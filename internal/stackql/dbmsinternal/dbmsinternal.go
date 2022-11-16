package dbmsinternal

import (
	"regexp"

	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/sqldialect"
	"vitess.io/vitess/go/vt/sqlparser"
)

var (
	_                      DBMSInternalRouter = &standardDBMSInternalRouter{}
	internalTableRegexp    *regexp.Regexp     = regexp.MustCompile(`(?i)^(?:public\.)?(?:pg_type|pg_namespace|pg_catalog.*|current_schema)`)
	showHousekeepingRegexp *regexp.Regexp     = regexp.MustCompile(`(?i)(?:\s+transaction\s+isolation\s+level|standard_conforming_strings)`)
)

type DBMSInternalRouter interface {
	CanRoute(node sqlparser.SQLNode) (constants.BackendQueryType, bool)
}

func GetDBMSInternalRouter(cfg dto.DBMSInternalCfg, sqlDialect sqldialect.SQLDialect) (DBMSInternalRouter, error) {
	showRegexp := showHousekeepingRegexp
	tableRegexp := internalTableRegexp
	var err error
	if cfg.ShowRegex != "" {
		showRegexp, err = regexp.Compile(cfg.ShowRegex)
		if err != nil {
			return nil, err
		}
	}
	if cfg.TableRegex != "" {
		tableRegexp, err = regexp.Compile(cfg.TableRegex)
		if err != nil {
			return nil, err
		}
	}
	return &standardDBMSInternalRouter{
		cfg:         cfg,
		sqlDialect:  sqlDialect,
		showRegexp:  showRegexp,
		tableRegexp: tableRegexp,
	}, nil
}

type standardDBMSInternalRouter struct {
	cfg         dto.DBMSInternalCfg
	sqlDialect  sqldialect.SQLDialect
	showRegexp  *regexp.Regexp
	tableRegexp *regexp.Regexp
}

func (pgr *standardDBMSInternalRouter) CanRoute(node sqlparser.SQLNode) (constants.BackendQueryType, bool) {
	if pgr.sqlDialect.GetName() != constants.SQLDialectPostgres {
		return pgr.negative()
	}
	switch node := node.(type) {
	case *sqlparser.Select:
		logging.GetLogger().Debugf("node = %v\n", node)
		return pgr.analyzeSelect(node)
	case *sqlparser.Set:
		return constants.BackendExec, true
	case *sqlparser.Show:
		return pgr.analyzeShow(node)
	}
	return pgr.negative()
}

func (pgr *standardDBMSInternalRouter) negative() (constants.BackendQueryType, bool) {
	return constants.BackendNop, false
}

func (pgr *standardDBMSInternalRouter) analyzeSelect(node *sqlparser.Select) (constants.BackendQueryType, bool) {
	if len(node.From) < 1 {
		return pgr.negative()
	}
	if pgr.analyzeTableExpr(node.From[0]) {
		return constants.BackendQuery, true
	}
	return pgr.negative()
}

func (pgr *standardDBMSInternalRouter) analyzeShow(node *sqlparser.Show) (constants.BackendQueryType, bool) {
	if node.Type != "" && pgr.showRegexp.MatchString(node.Type) {
		return constants.BackendQuery, true
	}
	if pgr.analyzeTableName(node.OnTable) {
		if pgr.analyzeTableName(node.OnTable) {
			return constants.BackendQuery, true
		}
	}
	return pgr.negative()
}

func (pgr *standardDBMSInternalRouter) analyzeTableExpr(node sqlparser.TableExpr) bool {
	switch node := node.(type) {
	case *sqlparser.AliasedTableExpr:
		switch expr := node.Expr.(type) {
		case sqlparser.TableName:
			return pgr.analyzeTableName(expr)
		}
	}
	return false
}

func (pgr *standardDBMSInternalRouter) analyzeTableName(node sqlparser.TableName) bool {
	rawName := node.GetRawVal()
	return pgr.tableRegexp.MatchString(rawName)
}
