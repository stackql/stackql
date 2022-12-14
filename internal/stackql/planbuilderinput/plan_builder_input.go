package planbuilderinput

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/astanalysis/annotatedast"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internaldto"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"vitess.io/vitess/go/vt/sqlparser"
)

type PlanBuilderInput interface {
	GetAliasedTables() parserutil.TableAliasMap
	GetAnnotatedAST() annotatedast.AnnotatedAst
	GetAuth() (*sqlparser.Auth, bool)
	GetAuthRevoke() (*sqlparser.AuthRevoke, bool)
	GetAssignedAliasedColumns() map[sqlparser.TableName]sqlparser.TableExpr
	GetColRefs() parserutil.ColTableMap
	GetDelete() (*sqlparser.Delete, bool)
	GetDescribeTable() (*sqlparser.DescribeTable, bool)
	GetExec() (*sqlparser.Exec, bool)
	GetHandlerCtx() handler.HandlerContext
	GetInsert() (*sqlparser.Insert, bool)
	GetNativeQuery() (*sqlparser.NativeQuery, bool)
	GetPlaceholderParams() parserutil.ParameterMap
	GetPurge() (*sqlparser.Purge, bool)
	GetRawQuery() string
	GetRegistry() (*sqlparser.Registry, bool)
	GetSelect() (*sqlparser.Select, bool)
	GetShow() (*sqlparser.Show, bool)
	GetSleep() (*sqlparser.Sleep, bool)
	GetStatement() sqlparser.SQLNode
	GetTableExprs() sqlparser.TableExprs
	GetTxnCtrlCtrs() internaldto.TxnControlCounters
	GetUnion() (*sqlparser.Union, bool)
	GetUpdate() (*sqlparser.Update, bool)
	GetUse() (*sqlparser.Use, bool)
}

type StandardPlanBuilderInput struct {
	annotatedAST           annotatedast.AnnotatedAst
	handlerCtx             handler.HandlerContext
	stmt                   sqlparser.SQLNode
	colRefs                parserutil.ColTableMap
	aliasedTables          parserutil.TableAliasMap
	assignedAliasedColumns parserutil.TableExprMap
	tables                 sqlparser.TableExprs
	paramsPlaceheld        parserutil.ParameterMap
	tcc                    internaldto.TxnControlCounters
}

func NewPlanBuilderInput(
	annotatedAST annotatedast.AnnotatedAst,
	handlerCtx handler.HandlerContext,
	stmt sqlparser.SQLNode,
	tables sqlparser.TableExprs,
	assignedAliasedColumns parserutil.TableExprMap,
	aliasedTables parserutil.TableAliasMap,
	colRefs parserutil.ColTableMap,
	paramsPlaceheld parserutil.ParameterMap,
	tcc internaldto.TxnControlCounters,
) (PlanBuilderInput, error) {
	rv := &StandardPlanBuilderInput{
		annotatedAST:           annotatedAST,
		handlerCtx:             handlerCtx,
		stmt:                   stmt,
		tables:                 tables,
		aliasedTables:          aliasedTables,
		assignedAliasedColumns: assignedAliasedColumns,
		colRefs:                colRefs,
		paramsPlaceheld:        paramsPlaceheld,
		tcc:                    tcc,
	}
	if handlerCtx == nil {
		return nil, fmt.Errorf("plan builder input invariant violation: nil handler context")
	}
	if rv.assignedAliasedColumns == nil {
		rv.assignedAliasedColumns = make(map[sqlparser.TableName]sqlparser.TableExpr)
	}
	return rv, nil
}

func (pbi *StandardPlanBuilderInput) GetRawQuery() string {
	return pbi.handlerCtx.GetRawQuery()
}

func (pbi *StandardPlanBuilderInput) GetAnnotatedAST() annotatedast.AnnotatedAst {
	return pbi.annotatedAST
}

func (pbi *StandardPlanBuilderInput) GetStatement() sqlparser.SQLNode {
	return pbi.stmt
}

func (pbi *StandardPlanBuilderInput) GetTxnCtrlCtrs() internaldto.TxnControlCounters {
	return pbi.tcc
}

func (pbi *StandardPlanBuilderInput) GetPlaceholderParams() parserutil.ParameterMap {
	return pbi.paramsPlaceheld
}

func (pbi *StandardPlanBuilderInput) GetAssignedAliasedColumns() map[sqlparser.TableName]sqlparser.TableExpr {
	return pbi.assignedAliasedColumns
}

func (pbi *StandardPlanBuilderInput) GetAliasedTables() parserutil.TableAliasMap {
	return pbi.aliasedTables
}

func (pbi *StandardPlanBuilderInput) GetColRefs() parserutil.ColTableMap {
	return pbi.colRefs
}

func (pbi *StandardPlanBuilderInput) GetTableExprs() sqlparser.TableExprs {
	return pbi.tables
}

func (pbi *StandardPlanBuilderInput) GetAuth() (*sqlparser.Auth, bool) {
	rv, ok := pbi.stmt.(*sqlparser.Auth)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetAuthRevoke() (*sqlparser.AuthRevoke, bool) {
	rv, ok := pbi.stmt.(*sqlparser.AuthRevoke)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetDelete() (*sqlparser.Delete, bool) {
	rv, ok := pbi.stmt.(*sqlparser.Delete)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetDescribeTable() (*sqlparser.DescribeTable, bool) {
	rv, ok := pbi.stmt.(*sqlparser.DescribeTable)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetExec() (*sqlparser.Exec, bool) {
	rv, ok := pbi.stmt.(*sqlparser.Exec)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetInsert() (*sqlparser.Insert, bool) {
	rv, ok := pbi.stmt.(*sqlparser.Insert)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetRegistry() (*sqlparser.Registry, bool) {
	rv, ok := pbi.stmt.(*sqlparser.Registry)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetPurge() (*sqlparser.Purge, bool) {
	rv, ok := pbi.stmt.(*sqlparser.Purge)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetNativeQuery() (*sqlparser.NativeQuery, bool) {
	rv, ok := pbi.stmt.(*sqlparser.NativeQuery)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetSelect() (*sqlparser.Select, bool) {
	rv, ok := pbi.stmt.(*sqlparser.Select)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetShow() (*sqlparser.Show, bool) {
	rv, ok := pbi.stmt.(*sqlparser.Show)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetSleep() (*sqlparser.Sleep, bool) {
	rv, ok := pbi.stmt.(*sqlparser.Sleep)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetUnion() (*sqlparser.Union, bool) {
	rv, ok := pbi.stmt.(*sqlparser.Union)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetUse() (*sqlparser.Use, bool) {
	rv, ok := pbi.stmt.(*sqlparser.Use)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetUpdate() (*sqlparser.Update, bool) {
	rv, ok := pbi.stmt.(*sqlparser.Update)
	return rv, ok
}

func (pbi *StandardPlanBuilderInput) GetHandlerCtx() handler.HandlerContext {
	return pbi.handlerCtx
}
