package dbmsinternal

import (
	"regexp"

	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/sql_system"
	"vitess.io/vitess/go/vt/sqlparser"
)

var (
	_                      DBMSInternalRouter = &standardDBMSInternalRouter{}
	internalTableRegexp    *regexp.Regexp     = regexp.MustCompile(`(?i)^(?:public\.)?(?:pg_type|pg_namespace|pg_catalog.*|current_schema)`)
	showHousekeepingRegexp *regexp.Regexp     = regexp.MustCompile(`(?i)(?:\s+transaction\s+isolation\s+level|standard_conforming_strings)`)
	funcNameRegexp         *regexp.Regexp     = regexp.MustCompile(`(?i)(?:pg_.*)`)
	internalSchemaRegexp   *regexp.Regexp     = regexp.MustCompile(`(?i)^(?:stackql_intel|stackql_history)`)
)

type DBMSInternalRouter interface {
	CanRoute(node sqlparser.SQLNode) (constants.BackendQueryType, bool)
	ExprIsRoutable(node sqlparser.SQLNode) bool
}

func GetDBMSInternalRouter(cfg dto.DBMSInternalCfg, sqlSystem sql_system.SQLSystem) (DBMSInternalRouter, error) {
	showRegexp := showHousekeepingRegexp
	tableRegexp := internalTableRegexp
	funcRegexp := funcNameRegexp
	schemaRegexp := internalSchemaRegexp
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
	if cfg.SchemaRegex != "" {
		schemaRegexp, err = regexp.Compile(cfg.SchemaRegex)
		if err != nil {
			return nil, err
		}
	}
	if cfg.FuncRegex != "" {
		funcRegexp, err = regexp.Compile(cfg.FuncRegex)
		if err != nil {
			return nil, err
		}
	}
	return &standardDBMSInternalRouter{
		cfg:            cfg,
		sqlSystem:      sqlSystem,
		showRegexp:     showRegexp,
		tableRegexp:    tableRegexp,
		funcNameRegexp: funcRegexp,
		schemaRegexp:   schemaRegexp,
	}, nil
}

type standardDBMSInternalRouter struct {
	cfg            dto.DBMSInternalCfg
	sqlSystem      sql_system.SQLSystem
	showRegexp     *regexp.Regexp
	schemaRegexp   *regexp.Regexp
	tableRegexp    *regexp.Regexp
	funcNameRegexp *regexp.Regexp
}

func (pgr *standardDBMSInternalRouter) CanRoute(node sqlparser.SQLNode) (constants.BackendQueryType, bool) {
	if pgr.sqlSystem.GetName() != constants.SQLDialectPostgres {
		return pgr.negative()
	}
	switch node := node.(type) {
	case *sqlparser.Select:
		logging.GetLogger().Debugf("node = %v\n", node)
		return pgr.analyzeSelect(node)
	case *sqlparser.Set:
		return pgr.affirmativeExec()
	case *sqlparser.Show:
		return pgr.analyzeShow(node)
	case *sqlparser.Begin, *sqlparser.Commit, *sqlparser.Rollback:
		return pgr.affirmativeExec()
	}
	return pgr.negative()
}

func (pgr *standardDBMSInternalRouter) negative() (constants.BackendQueryType, bool) {
	return constants.BackendNop, false
}

func (pgr *standardDBMSInternalRouter) affirmativeQuery() (constants.BackendQueryType, bool) {
	return constants.BackendQuery, true
}

func (pgr *standardDBMSInternalRouter) affirmativeExec() (constants.BackendQueryType, bool) {
	return constants.BackendExec, true
}

func (pgr *standardDBMSInternalRouter) analyzeSelectExprs(selectExprs sqlparser.SelectExprs) bool {
	for _, n := range selectExprs {
		switch n := n.(type) {
		case *sqlparser.AliasedExpr:
			switch et := n.Expr.(type) {
			case *sqlparser.FuncExpr:
				if pgr.funcNameRegexp.MatchString(et.Name.GetRawVal()) {
					return true
				}
			}
		}
	}
	return false
}

func (pgr *standardDBMSInternalRouter) analyzeSelectStatement(node sqlparser.SelectStatement) (constants.BackendQueryType, bool) {
	switch node := node.(type) {
	case *sqlparser.Select:
		return pgr.analyzeSelect(node)
	default:
		return pgr.negative()
	}
}

func (pgr *standardDBMSInternalRouter) analyzeSelect(node *sqlparser.Select) (constants.BackendQueryType, bool) {
	if pgr.analyzeSelectExprs(node.SelectExprs) {
		return pgr.affirmativeQuery()
	}
	if len(node.From) < 1 {
		return pgr.negative()
	}
	if pgr.analyzeTableExpr(node.From[0]) {
		return pgr.affirmativeQuery()
	}
	return pgr.negative()
}

func (pgr *standardDBMSInternalRouter) analyzeShow(node *sqlparser.Show) (constants.BackendQueryType, bool) {
	if node.Type != "" && pgr.showRegexp.MatchString(node.Type) {
		return pgr.affirmativeQuery()
	}
	if pgr.analyzeTableName(node.OnTable) {
		if pgr.analyzeTableName(node.OnTable) {
			return pgr.affirmativeQuery()
		}
	}
	return pgr.negative()
}

func (pgr *standardDBMSInternalRouter) ExprIsRoutable(node sqlparser.SQLNode) bool {
	switch node := node.(type) {
	case sqlparser.TableExpr:
		return pgr.analyzeTableExpr(node)
	case sqlparser.TableName:
		return pgr.analyzeTableName(node)
	default:
		return false
	}
}

func (pgr *standardDBMSInternalRouter) analyzeTableExpr(node sqlparser.TableExpr) bool {
	switch node := node.(type) {
	case *sqlparser.AliasedTableExpr:
		switch expr := node.Expr.(type) {
		case sqlparser.TableName:
			return pgr.analyzeTableName(expr)
		case *sqlparser.Subquery:
			_, rv := pgr.CanRoute(expr.Select)
			return rv
		}
	case *sqlparser.JoinTableExpr:
		lhs := pgr.analyzeTableExpr(node.LeftExpr)
		if lhs {
			return true
		}
		rhs := pgr.analyzeTableExpr(node.RightExpr)
		if rhs {
			return true
		}
	}
	return false
}

func (pgr *standardDBMSInternalRouter) analyzeTableName(node sqlparser.TableName) bool {
	if node.QualifierSecond.GetRawVal() == "" && node.QualifierThird.GetRawVal() == "" && node.Qualifier.GetRawVal() != "" {
		if pgr.analyzeTableIdentForSchema(node.Qualifier) {
			return true
		}
	}
	rawName := node.GetRawVal()
	return pgr.tableRegexp.MatchString(rawName)
}

func (pgr *standardDBMSInternalRouter) analyzeTableIdentForSchema(node sqlparser.TableIdent) bool {
	rawName := node.GetRawVal()
	return pgr.schemaRegexp.MatchString(rawName)
}
