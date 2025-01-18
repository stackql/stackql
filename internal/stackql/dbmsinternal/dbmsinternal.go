package dbmsinternal

import (
	"regexp"
	"strings"

	"github.com/stackql/any-sdk/pkg/constants"
	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/any-sdk/pkg/logging"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/sql_system"
)

//nolint:lll,revive // complex regex
var (
	_                      Router         = &standardDBMSInternalRouter{}
	internalTableRegexp    *regexp.Regexp = regexp.MustCompile(`(?i)^(?:public\.)?(?:pg_type|pg_namespace|pg_catalog.*|current_schema|pg_.*|information_schema.*)`)
	showHousekeepingRegexp *regexp.Regexp = regexp.MustCompile(`(?i)(?:\s+transaction\s+isolation\s+level|standard_conforming_strings)`)
	funcNameRegexp         *regexp.Regexp = regexp.MustCompile(`(?i)(?:pg_.*|information_schema.*)`)
	internalSchemaRegexp   *regexp.Regexp = regexp.MustCompile(`(?i)^(?:stackql_intel|stackql_history)`)
)

type Router interface {
	CanRoute(node sqlparser.SQLNode) (constants.BackendQueryType, bool)
	ExprIsRoutable(node sqlparser.SQLNode) bool
}

func GetDBMSInternalRouter(cfg dto.DBMSInternalCfg, sqlSystem sql_system.SQLSystem) (Router, error) {
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
		switch node := node.(type) {
		case *sqlparser.Select:
			logging.GetLogger().Debugf("node = %v\n", node)
			if pgr.analyzeTableExprAllRDBMS(node.From[0]) {
				return pgr.affirmativeQuery()
			}
			return pgr.negative()
		default:
			return pgr.negative()
		}
	}
	switch node := node.(type) {
	case *sqlparser.Select:
		logging.GetLogger().Debugf("node = %v\n", node)
		if pgr.analyzeTableExprAllRDBMS(node.From[0]) {
			return pgr.affirmativeQuery()
		}
		return pgr.analyzeSelect(node)
	case *sqlparser.Set:
		return pgr.analyzeSet(node)
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
		//nolint:gocritic // TODO: investigate
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

func (pgr *standardDBMSInternalRouter) analyzeSelect(node *sqlparser.Select) (constants.BackendQueryType, bool) {
	// TODO: need to add check for subqueries
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

func (pgr *standardDBMSInternalRouter) analyzeSet(node *sqlparser.Set) (constants.BackendQueryType, bool) {
	for _, n := range node.Exprs {
		if strings.HasPrefix(n.Name.GetRawVal(), "$") {
			return pgr.negative()
		}
	}
	return pgr.affirmativeExec()
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

func (pgr *standardDBMSInternalRouter) analyzeTableExprAllRDBMS(node sqlparser.TableExpr) bool {
	switch node := node.(type) {
	case *sqlparser.AliasedTableExpr:
		switch expr := node.Expr.(type) {
		case sqlparser.TableName:
			return pgr.analyzeTableNameAllRDBMS(expr)
		case *sqlparser.Subquery:
			_, rv := pgr.CanRoute(expr.Select)
			return rv
		}
	case *sqlparser.JoinTableExpr:
		lhs := pgr.analyzeTableExprAllRDBMS(node.LeftExpr)
		if lhs {
			return true
		}
		rhs := pgr.analyzeTableExprAllRDBMS(node.RightExpr)
		if rhs {
			return true
		}
	}
	return false
}

func (pgr *standardDBMSInternalRouter) analyzeTableName(node sqlparser.TableName) bool {
	//nolint:lll // long conditional
	if node.QualifierSecond.GetRawVal() == "" && node.QualifierThird.GetRawVal() == "" && node.Qualifier.GetRawVal() != "" {
		if pgr.analyzeTableIdentForSchema(node.Qualifier) {
			return true
		}
	}
	rawName := node.GetRawVal()
	return pgr.tableRegexp.MatchString(rawName)
}

func (pgr *standardDBMSInternalRouter) analyzeTableNameAllRDBMS(node sqlparser.TableName) bool {
	//nolint:lll // long conditional
	if node.QualifierSecond.GetRawVal() == "" && node.QualifierThird.GetRawVal() == "" && node.Qualifier.GetRawVal() == "" {
		if node.Name.GetRawVal() == "dual" {
			return true
		}
	}
	return false
}

func (pgr *standardDBMSInternalRouter) analyzeTableIdentForSchema(node sqlparser.TableIdent) bool {
	rawName := node.GetRawVal()
	return pgr.schemaRegexp.MatchString(rawName)
}
