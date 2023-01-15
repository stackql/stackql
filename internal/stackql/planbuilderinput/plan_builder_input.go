package planbuilderinput

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/astanalysis/annotatedast"
	"github.com/stackql/stackql/internal/stackql/dataflow"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/router"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
	"vitess.io/vitess/go/vt/sqlparser"
)

type PlanBuilderInput interface {
	Clone() PlanBuilderInput
	GetAliasedTables() parserutil.TableAliasMap
	GetAnnotatedAST() annotatedast.AnnotatedAst
	GetAnnotations() (taxonomy.AnnotationCtxMap, bool)
	GetAuth() (*sqlparser.Auth, bool)
	GetAuthRevoke() (*sqlparser.AuthRevoke, bool)
	GetAssignedAliasedColumns() map[sqlparser.TableName]sqlparser.TableExpr
	GetColRefs() parserutil.ColTableMap
	GetDelete() (*sqlparser.Delete, bool)
	GetDescribeTable() (*sqlparser.DescribeTable, bool)
	GetDDL() (*sqlparser.DDL, bool)
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
	IsTccSetAheadOfTime() bool
	SetIsTccSetAheadOfTime(bool)

	WithParameterRouter(router.ParameterRouter) PlanBuilderInput
	WithTableRouteVisitor(tableRouteVisitor router.TableRouteAstVisitor) PlanBuilderInput
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
	paramRouter            router.ParameterRouter
	tableRouteVisitor      router.TableRouteAstVisitor
	onConditionDataFlows   dataflow.DataFlowCollection
	onConditionsToRewrite  map[*sqlparser.ComparisonExpr]struct{}
	tccSetAheadOfTime      bool
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
	if handlerCtx == nil {
		return nil, fmt.Errorf("plan builder input invariant violation: nil handler context")
	}
	return newPlanBuilderInput(
		annotatedAST,
		handlerCtx,
		stmt,
		tables,
		assignedAliasedColumns,
		aliasedTables,
		colRefs,
		paramsPlaceheld,
		tcc,
	), nil
}

func newPlanBuilderInput(
	annotatedAST annotatedast.AnnotatedAst,
	handlerCtx handler.HandlerContext,
	stmt sqlparser.SQLNode,
	tables sqlparser.TableExprs,
	assignedAliasedColumns parserutil.TableExprMap,
	aliasedTables parserutil.TableAliasMap,
	colRefs parserutil.ColTableMap,
	paramsPlaceheld parserutil.ParameterMap,
	tcc internaldto.TxnControlCounters,
) PlanBuilderInput {
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
	if rv.assignedAliasedColumns == nil {
		rv.assignedAliasedColumns = make(map[sqlparser.TableName]sqlparser.TableExpr)
	}
	return rv
}

func (pbi *StandardPlanBuilderInput) Clone() PlanBuilderInput {
	clonedPbi := newPlanBuilderInput(
		pbi.annotatedAST,
		pbi.handlerCtx,
		pbi.stmt,
		pbi.tables,
		pbi.assignedAliasedColumns,
		pbi.aliasedTables,
		pbi.colRefs,
		pbi.paramsPlaceheld,
		pbi.tcc,
	)
	return clonedPbi
}

func (pbi *StandardPlanBuilderInput) GetOnConditionsToRewrite() map[*sqlparser.ComparisonExpr]struct{} {
	return pbi.onConditionsToRewrite
}

func (pbi *StandardPlanBuilderInput) IsTccSetAheadOfTime() bool {
	return pbi.tccSetAheadOfTime
}

func (pbi *StandardPlanBuilderInput) SetIsTccSetAheadOfTime(tccSetAheadOfTime bool) {
	pbi.tccSetAheadOfTime = tccSetAheadOfTime
}

func (pbi *StandardPlanBuilderInput) GetOnConditionDataFlows() (dataflow.DataFlowCollection, bool) {
	return pbi.onConditionDataFlows, pbi.onConditionDataFlows != nil
}

func (pbi *StandardPlanBuilderInput) SetOnConditionsToRewrite(onConditionsToRewrite map[*sqlparser.ComparisonExpr]struct{}) {
	pbi.onConditionsToRewrite = onConditionsToRewrite
}

func (pbi *StandardPlanBuilderInput) SetOnConditionDataFlows(onConditionDataFlows dataflow.DataFlowCollection) {
	pbi.onConditionDataFlows = onConditionDataFlows
}

func (pbi *StandardPlanBuilderInput) GetTableMap() (taxonomy.TblMap, bool) {
	if pbi.tableRouteVisitor != nil {
		return pbi.tableRouteVisitor.GetTableMap(), true
	}
	return nil, false
}

func (pbi *StandardPlanBuilderInput) GetAnnotations() (taxonomy.AnnotationCtxMap, bool) {
	if pbi.tableRouteVisitor != nil {
		return pbi.tableRouteVisitor.GetAnnotations(), true
	}
	return nil, false
}

func (pbi *StandardPlanBuilderInput) WithTableRouteVisitor(tableRouteVisitor router.TableRouteAstVisitor) PlanBuilderInput {
	pbi.tableRouteVisitor = tableRouteVisitor
	return pbi
}

func (pbi *StandardPlanBuilderInput) GetRawQuery() string {
	return pbi.handlerCtx.GetRawQuery()
}

// router.ParameterRouter
func (pbi *StandardPlanBuilderInput) GetParameterRouter() (router.ParameterRouter, bool) {
	return pbi.paramRouter, true
}

func (pbi *StandardPlanBuilderInput) WithParameterRouter(paramRouter router.ParameterRouter) PlanBuilderInput {
	pbi.paramRouter = paramRouter
	return pbi
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

func (pbi *StandardPlanBuilderInput) GetDDL() (*sqlparser.DDL, bool) {
	rv, ok := pbi.stmt.(*sqlparser.DDL)
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
