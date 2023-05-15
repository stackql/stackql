package primitivecomposer

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/primitivebuilder"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/provider"
	"github.com/stackql/stackql/internal/stackql/sql_system"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/suffix"
	"github.com/stackql/stackql/internal/stackql/symtab"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"github.com/stackql/stackql/internal/stackql/taxonomy"

	"github.com/stackql/go-openapistackql/openapistackql"

	"github.com/stackql/stackql/pkg/txncounter"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

type PrimitiveComposer interface {
	AddChild(val PrimitiveComposer)
	AddIndirect(val PrimitiveComposer)
	AssignParameters() (internaldto.TableParameterCollection, error)
	ContainsSQLDataSource() bool
	GetAssignedParameters() (internaldto.TableParameterCollection, bool)
	GetAst() sqlparser.SQLNode
	GetASTFormatter() sqlparser.NodeFormatter
	GetBuilder() primitivebuilder.Builder
	GetChildren() []PrimitiveComposer
	GetColumnOrder() []string
	GetCommentDirectives() sqlparser.CommentDirectives
	GetCtrlColumnRepeats() int
	GetDRMConfig() drm.Config
	GetGraph() primitivegraph.PrimitiveGraph
	GetInsertPreparedStatementCtx() drm.PreparedStatementCtx
	GetInsertValOnlyRows() map[int]map[int]interface{}
	GetLikeAbleColumns() []string
	GetParent() PrimitiveComposer
	GetProvider() provider.IProvider
	GetRoot() primitivegraph.PrimitiveNode
	GetSelectPreparedStatementCtx() drm.PreparedStatementCtx
	GetIndirectDescribeSelectCtx() (drm.PreparedStatementCtx, bool)
	GetIndirectSelectPreparedStatementCtx() drm.PreparedStatementCtx
	GetSQLEngine() sqlengine.SQLEngine
	GetSQLSystem() sql_system.SQLSystem
	GetSymbol(k interface{}) (symtab.Entry, error)
	GetSymTab() symtab.SymTab
	GetTable(node sqlparser.SQLNode) (tablemetadata.ExtendedTableMetadata, error)
	GetTableFilter() func(openapistackql.ITable) (openapistackql.ITable, error)
	GetTables() taxonomy.TblMap
	GetTxnCounterManager() txncounter.Manager
	GetTxnCtrlCtrs() internaldto.TxnControlCounters
	GetValOnlyCol(key int) map[string]interface{}
	GetValOnlyColKeys() []int
	GetWhere() *sqlparser.Where
	IsAwait() bool
	IsIndirect() bool
	IsTccSetAheadOfTime() bool
	NewChildPrimitiveComposer(ast sqlparser.SQLNode) PrimitiveComposer
	SetAwait(await bool)
	SetBuilder(builder primitivebuilder.Builder)
	SetColumnOrder(co []parserutil.ColumnHandle)
	SetColVisited(colname string, isVisited bool)
	SetCommentDirectives(dirs sqlparser.CommentDirectives)
	SetDataflowDependent(val PrimitiveComposer)
	SetInsertPreparedStatementCtx(ctx drm.PreparedStatementCtx)
	SetInsertValOnlyRows(m map[int]map[int]interface{})
	SetIsIndirect(isIndirect bool)
	SetIsTccSetAheadOfTime(bool)
	SetLikeAbleColumns(cols []string)
	SetProvider(prov provider.IProvider)
	SetRoot(root primitivegraph.PrimitiveNode)
	SetSelectPreparedStatementCtx(ctx drm.PreparedStatementCtx)
	SetSymbol(k interface{}, v symtab.Entry) error
	SetSymTab(symtab.SymTab)
	SetTable(node sqlparser.SQLNode, table tablemetadata.ExtendedTableMetadata)
	SetTableFilter(tableFilter func(openapistackql.ITable) (openapistackql.ITable, error))
	SetTxnCtrlCtrs(tc internaldto.TxnControlCounters)
	SetUnionSelectPreparedStatementCtx(ctx drm.PreparedStatementCtx)
	SetValOnlyCols(m map[int]map[string]interface{})
	SetWhere(where *sqlparser.Where)
	ShouldCollectGarbage() bool
}

type standardPrimitiveComposer struct {
	parent PrimitiveComposer

	children []PrimitiveComposer

	indirects []PrimitiveComposer

	dataflowDependent PrimitiveComposer

	await bool

	ast sqlparser.SQLNode

	builder primitivebuilder.Builder

	graph primitivegraph.PrimitiveGraph

	drmConfig drm.Config

	// needed globally for non-heirarchy queries, such as "SHOW SERVICES FROM google;"
	prov            provider.IProvider
	tableFilter     func(openapistackql.ITable) (openapistackql.ITable, error)
	colsVisited     map[string]bool
	likeAbleColumns []string

	// per table
	tables taxonomy.TblMap

	// per query
	columnOrder       []string
	commentDirectives sqlparser.CommentDirectives
	txnCounterManager txncounter.Manager
	txnCtrlCtrs       internaldto.TxnControlCounters

	// per query -- SELECT only
	insertValOnlyRows               map[int]map[int]interface{}
	valOnlyCols                     map[int]map[string]interface{}
	insertPreparedStatementCtx      drm.PreparedStatementCtx
	selectPreparedStatementCtx      drm.PreparedStatementCtx
	unionSelectPreparedStatementCtx drm.PreparedStatementCtx

	// TODO: universally retire in favour of builder, which returns primitive.IPrimitive
	root primitivegraph.PrimitiveNode

	symTab symtab.SymTab

	where *sqlparser.Where

	sqlEngine sqlengine.SQLEngine

	sqlSystem sql_system.SQLSystem

	formatter sqlparser.NodeFormatter

	unionNonControlColumns []internaldto.ColumnMetadata

	tccSetAheadOfTime bool

	paramCollection internaldto.TableParameterCollection

	isIndirect bool
}

func (pb *standardPrimitiveComposer) GetCtrlColumnRepeats() int {
	return pb.selectPreparedStatementCtx.GetCtrlColumnRepeats()
}

func (pb *standardPrimitiveComposer) ContainsSQLDataSource() bool {
	for _, tb := range pb.tables {
		_, isSQLDataSource := tb.GetSQLDataSource()
		if isSQLDataSource {
			return true
		}
	}
	return false
}

func (pb *standardPrimitiveComposer) IsIndirect() bool {
	return pb.isIndirect
}

func (pb *standardPrimitiveComposer) SetIsIndirect(isIndirect bool) {
	pb.isIndirect = isIndirect
}

func (pb *standardPrimitiveComposer) GetAssignedParameters() (internaldto.TableParameterCollection, bool) {
	return pb.paramCollection, pb.paramCollection != nil
}

func (pb *standardPrimitiveComposer) SetSymTab(symTab symtab.SymTab) {
	pb.symTab = symTab
}

func (pb *standardPrimitiveComposer) GetNonControlColumns() []internaldto.ColumnMetadata {
	if pb.GetSelectPreparedStatementCtx() != nil {
		return pb.GetSelectPreparedStatementCtx().GetNonControlColumns()
	}
	return pb.unionNonControlColumns
}

func (pb *standardPrimitiveComposer) ShouldCollectGarbage() bool {
	return pb.parent == nil
}

func (pb *standardPrimitiveComposer) IsTccSetAheadOfTime() bool {
	return pb.tccSetAheadOfTime
}

func (pb *standardPrimitiveComposer) SetIsTccSetAheadOfTime(tccSetAheadOfTime bool) {
	pb.tccSetAheadOfTime = tccSetAheadOfTime
}

func (pb *standardPrimitiveComposer) SetTxnCtrlCtrs(tc internaldto.TxnControlCounters) {
	pb.txnCtrlCtrs = tc
}

func (pb *standardPrimitiveComposer) GetTxnCtrlCtrs() internaldto.TxnControlCounters {
	return pb.txnCtrlCtrs
}

func (pb *standardPrimitiveComposer) GetGraph() primitivegraph.PrimitiveGraph {
	return pb.graph
}

func (pb *standardPrimitiveComposer) GetASTFormatter() sqlparser.NodeFormatter {
	return pb.formatter
}

func (pb *standardPrimitiveComposer) GetParent() PrimitiveComposer {
	return pb.parent
}

func (pb *standardPrimitiveComposer) GetChildren() []PrimitiveComposer {
	return pb.children
}

func (pb *standardPrimitiveComposer) AssignParameters() (internaldto.TableParameterCollection, error) {
	requiredParameters := suffix.NewParameterSuffixMap()
	remainingRequiredParameters := suffix.NewParameterSuffixMap()
	optionalParameters := suffix.NewParameterSuffixMap()
	tbVisited := map[tablemetadata.ExtendedTableMetadata]struct{}{}
	for _, tb := range pb.GetTables() {
		if _, ok := tbVisited[tb]; ok {
			continue
		}
		tbVisited[tb] = struct{}{}
		tbID := tb.GetUniqueID()
		var reqParams, tblOptParams map[string]openapistackql.Addressable
		if view, isView := tb.GetIndirect(); isView {
			// TODO: fill this out
			assignedParams, ok := view.GetAssignedParameters()
			if !ok {
				continue
			}
			reqParams = assignedParams.GetRequiredParams().GetAll()
			tblOptParams = assignedParams.GetOptionalParams().GetAll()
		} else {
			// These methods need to incorporate request body parameters
			reqParams = tb.GetRequiredParameters()
			tblOptParams = tb.GetOptionalParameters()
		}
		// This method needs to incorporate request body parameters

		for k, v := range reqParams {
			key := fmt.Sprintf("%s.%s", tbID, k)
			_, keyExists := requiredParameters.Get(key)
			if keyExists {
				return nil, fmt.Errorf("key already is required: %s", k)
			}
			requiredParameters.Put(key, v)
		}
		for k, vOpt := range tblOptParams {
			key := fmt.Sprintf("%s.%s", tbID, k)
			_, keyExists := optionalParameters.Get(key)
			if keyExists {
				return nil, fmt.Errorf("key already is optional: %s", k)
			}
			optionalParameters.Put(key, vOpt)
		}
	}
	rv := internaldto.NewTableParameterCollection(requiredParameters, optionalParameters, remainingRequiredParameters)
	pb.paramCollection = rv
	return rv, nil
}

func (pb *standardPrimitiveComposer) AddChild(val PrimitiveComposer) {
	pb.children = append(pb.children, val)
}

func (pb *standardPrimitiveComposer) AddIndirect(val PrimitiveComposer) {
	pb.indirects = append(pb.indirects, val)
}

func (pb *standardPrimitiveComposer) GetIndirectDescribeSelectCtx() (drm.PreparedStatementCtx, bool) {
	if len(pb.indirects) != 1 || pb.indirects[0] == nil {
		return nil, false
	}
	rv := pb.indirects[0].GetIndirectSelectPreparedStatementCtx()
	return rv, rv != nil
}

func (pb *standardPrimitiveComposer) SetDataflowDependent(val PrimitiveComposer) {
	pb.dataflowDependent = val
}

func (pb *standardPrimitiveComposer) GetSymbol(k interface{}) (symtab.Entry, error) {
	return pb.symTab.GetSymbol(k)
}

func (pb *standardPrimitiveComposer) GetSymTab() symtab.SymTab {
	return pb.symTab
}

func (pb *standardPrimitiveComposer) SetSymbol(k interface{}, v symtab.Entry) error {
	return pb.symTab.SetSymbol(k, v)
}

func (pb *standardPrimitiveComposer) GetWhere() *sqlparser.Where {
	return pb.where
}

func (pb *standardPrimitiveComposer) SetWhere(where *sqlparser.Where) {
	pb.where = where
}

func (pb *standardPrimitiveComposer) GetAst() sqlparser.SQLNode {
	return pb.ast
}

func (pb *standardPrimitiveComposer) GetTxnCounterManager() txncounter.Manager {
	return pb.txnCounterManager
}

func (pb *standardPrimitiveComposer) NewChildPrimitiveComposer(ast sqlparser.SQLNode) PrimitiveComposer {
	child := NewPrimitiveComposer(
		pb, ast, pb.drmConfig, pb.txnCounterManager,
		pb.graph, pb.tables, pb.symTab, pb.sqlEngine, pb.sqlSystem, pb.formatter)
	pb.children = append(pb.children, child)
	return child
}

func (pb *standardPrimitiveComposer) GetInsertValOnlyRows() map[int]map[int]interface{} {
	return pb.insertValOnlyRows
}

func (pb *standardPrimitiveComposer) SetInsertValOnlyRows(m map[int]map[int]interface{}) {
	pb.insertValOnlyRows = m
}

func (pb *standardPrimitiveComposer) GetColumnOrder() []string {
	return pb.columnOrder
}

func (pb *standardPrimitiveComposer) SetColumnOrder(co []parserutil.ColumnHandle) {
	var colOrd []string
	for _, v := range co {
		colOrd = append(colOrd, v.Name)
	}
	pb.columnOrder = colOrd
}

func (pb *standardPrimitiveComposer) GetRoot() primitivegraph.PrimitiveNode {
	return pb.root
}

func (pb *standardPrimitiveComposer) SetRoot(root primitivegraph.PrimitiveNode) {
	pb.root = root
}

func (pb *standardPrimitiveComposer) GetCommentDirectives() sqlparser.CommentDirectives {
	return pb.commentDirectives
}

func (pb *standardPrimitiveComposer) SetCommentDirectives(dirs sqlparser.CommentDirectives) {
	pb.commentDirectives = dirs
}

func (pb *standardPrimitiveComposer) GetLikeAbleColumns() []string {
	return pb.likeAbleColumns
}

func (pb *standardPrimitiveComposer) SetLikeAbleColumns(cols []string) {
	pb.likeAbleColumns = cols
}

func (pb *standardPrimitiveComposer) GetValOnlyColKeys() []int {
	keys := make([]int, 0, len(pb.valOnlyCols))
	for k := range pb.valOnlyCols {
		keys = append(keys, k)
	}
	return keys
}

func (pb *standardPrimitiveComposer) GetValOnlyCol(key int) map[string]interface{} {
	return pb.valOnlyCols[key]
}

func (pb *standardPrimitiveComposer) SetValOnlyCols(m map[int]map[string]interface{}) {
	pb.valOnlyCols = m
}

func (pb *standardPrimitiveComposer) SetColVisited(colname string, isVisited bool) {
	pb.colsVisited[colname] = isVisited
}

func (pb *standardPrimitiveComposer) GetTableFilter() func(openapistackql.ITable) (openapistackql.ITable, error) {
	return pb.tableFilter
}

func (pb *standardPrimitiveComposer) SetTableFilter(
	tableFilter func(openapistackql.ITable) (openapistackql.ITable, error)) {
	pb.tableFilter = tableFilter
}

func (pb *standardPrimitiveComposer) SetInsertPreparedStatementCtx(ctx drm.PreparedStatementCtx) {
	pb.insertPreparedStatementCtx = ctx
}

func (pb *standardPrimitiveComposer) GetInsertPreparedStatementCtx() drm.PreparedStatementCtx {
	return pb.insertPreparedStatementCtx
}

func (pb *standardPrimitiveComposer) SetSelectPreparedStatementCtx(ctx drm.PreparedStatementCtx) {
	pb.selectPreparedStatementCtx = ctx
}

func (pb *standardPrimitiveComposer) SetUnionSelectPreparedStatementCtx(ctx drm.PreparedStatementCtx) {
	pb.unionSelectPreparedStatementCtx = ctx
}

func (pb *standardPrimitiveComposer) GetSelectPreparedStatementCtx() drm.PreparedStatementCtx {
	return pb.selectPreparedStatementCtx
}

func (pb *standardPrimitiveComposer) GetIndirectSelectPreparedStatementCtx() drm.PreparedStatementCtx {
	if pb.unionSelectPreparedStatementCtx != nil {
		return pb.unionSelectPreparedStatementCtx
	}
	return pb.selectPreparedStatementCtx
}

func (pb *standardPrimitiveComposer) GetProvider() provider.IProvider {
	return pb.prov
}

func (pb *standardPrimitiveComposer) SetProvider(prov provider.IProvider) {
	pb.prov = prov
}

func (pb *standardPrimitiveComposer) GetBuilder() primitivebuilder.Builder {
	if pb.children == nil || len(pb.children) == 0 {
		return pb.builder
	}
	var builders []primitivebuilder.Builder
	for _, child := range pb.children {
		if bldr := child.GetBuilder(); bldr != nil {
			builders = append(builders, bldr)
		}
	}
	simpleDiamond := primitivebuilder.NewDiamondBuilder(
		pb.builder, builders, pb.graph, pb.sqlSystem, pb.ShouldCollectGarbage())
	if len(pb.indirects) > 0 {
		var indirectBuilders []primitivebuilder.Builder
		for _, ind := range pb.indirects {
			if bldr := ind.GetBuilder(); bldr != nil {
				indirectBuilders = append(indirectBuilders, bldr)
			}
		}
		indirectDiamond := primitivebuilder.NewDiamondBuilder(
			pb.builder, indirectBuilders, pb.graph, pb.sqlSystem,
			pb.ShouldCollectGarbage())
		return primitivebuilder.NewDependencySubDAGBuilder(
			pb.graph,
			[]primitivebuilder.Builder{indirectDiamond}, simpleDiamond)
	}
	return simpleDiamond
}

func (pb *standardPrimitiveComposer) SetBuilder(builder primitivebuilder.Builder) {
	pb.builder = builder
}

func (pb *standardPrimitiveComposer) IsAwait() bool {
	return pb.await
}

func (pb *standardPrimitiveComposer) SetAwait(await bool) {
	pb.await = await
}

func (pb *standardPrimitiveComposer) GetTable(node sqlparser.SQLNode) (tablemetadata.ExtendedTableMetadata, error) {
	return pb.tables.GetTable(node)
}

func (pb *standardPrimitiveComposer) SetTable(node sqlparser.SQLNode, table tablemetadata.ExtendedTableMetadata) {
	pb.tables.SetTable(node, table)
}

func (pb *standardPrimitiveComposer) GetTables() taxonomy.TblMap {
	return pb.tables
}

func (pb *standardPrimitiveComposer) GetDRMConfig() drm.Config {
	return pb.drmConfig
}

func (pb *standardPrimitiveComposer) GetSQLEngine() sqlengine.SQLEngine {
	return pb.sqlEngine
}

func (pb *standardPrimitiveComposer) GetSQLSystem() sql_system.SQLSystem {
	return pb.sqlSystem
}

func NewPrimitiveComposer(
	parent PrimitiveComposer, ast sqlparser.SQLNode,
	drmConfig drm.Config, txnCtrMgr txncounter.Manager,
	graph primitivegraph.PrimitiveGraph,
	tblMap taxonomy.TblMap, symTab symtab.SymTab,
	sqlEngine sqlengine.SQLEngine, sqlSystem sql_system.SQLSystem,
	formatter sqlparser.NodeFormatter) PrimitiveComposer {
	return &standardPrimitiveComposer{
		parent:            parent,
		ast:               ast,
		drmConfig:         drmConfig,
		tables:            tblMap,
		valOnlyCols:       make(map[int]map[string]interface{}),
		insertValOnlyRows: make(map[int]map[int]interface{}),
		colsVisited:       make(map[string]bool),
		txnCounterManager: txnCtrMgr,
		symTab:            symTab,
		graph:             graph,
		sqlEngine:         sqlEngine,
		sqlSystem:         sqlSystem,
		formatter:         formatter,
	}
}
