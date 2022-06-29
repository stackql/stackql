package primitivecomposer

import (
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/primitivebuilder"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/provider"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/symtab"
	"github.com/stackql/stackql/internal/stackql/taxonomy"

	"github.com/stackql/go-openapistackql/openapistackql"

	"github.com/stackql/stackql/pkg/txncounter"

	"vitess.io/vitess/go/vt/sqlparser"
)

type PrimitiveComposer interface {
	AddChild(val PrimitiveComposer)
	GetAst() sqlparser.SQLNode
	GetBuilder() primitivebuilder.Builder
	GetChildren() []PrimitiveComposer
	GetColumnOrder() []string
	GetCommentDirectives() sqlparser.CommentDirectives
	GetDRMConfig() drm.DRMConfig
	GetGraph() *primitivegraph.PrimitiveGraph
	GetInsertPreparedStatementCtx() *drm.PreparedStatementCtx
	GetInsertValOnlyRows() map[int]map[int]interface{}
	GetLikeAbleColumns() []string
	GetParent() PrimitiveComposer
	GetProvider() provider.IProvider
	GetRoot() primitivegraph.PrimitiveNode
	GetSelectPreparedStatementCtx() *drm.PreparedStatementCtx
	GetSQLEngine() sqlengine.SQLEngine
	GetSymbol(k interface{}) (symtab.SymTabEntry, error)
	GetSymTab() symtab.SymTab
	GetTable(node sqlparser.SQLNode) (*taxonomy.ExtendedTableMetadata, error)
	GetTableFilter() func(openapistackql.ITable) (openapistackql.ITable, error)
	GetTables() taxonomy.TblMap
	GetTxnCounterManager() *txncounter.TxnCounterManager
	GetTxnCtrlCtrs() *dto.TxnControlCounters
	GetValOnlyCol(key int) map[string]interface{}
	GetValOnlyColKeys() []int
	GetWhere() *sqlparser.Where
	IsAwait() bool
	NewChildPrimitiveBuilder(ast sqlparser.SQLNode) PrimitiveComposer
	SetAwait(await bool)
	SetBuilder(builder primitivebuilder.Builder)
	SetColumnOrder(co []parserutil.ColumnHandle)
	SetColVisited(colname string, isVisited bool)
	SetCommentDirectives(dirs sqlparser.CommentDirectives)
	SetInsertPreparedStatementCtx(ctx *drm.PreparedStatementCtx)
	SetInsertValOnlyRows(m map[int]map[int]interface{})
	SetLikeAbleColumns(cols []string)
	SetProvider(prov provider.IProvider)
	SetRoot(root primitivegraph.PrimitiveNode)
	SetSelectPreparedStatementCtx(ctx *drm.PreparedStatementCtx)
	SetSymbol(k interface{}, v symtab.SymTabEntry) error
	SetTable(node sqlparser.SQLNode, table *taxonomy.ExtendedTableMetadata)
	SetTableFilter(tableFilter func(openapistackql.ITable) (openapistackql.ITable, error))
	SetTxnCtrlCtrs(tc *dto.TxnControlCounters)
	SetValOnlyCols(m map[int]map[string]interface{})
	SetWhere(where *sqlparser.Where)
	ShouldCollectGarbage() bool
}

type StandardPrimitiveComposer struct {
	parent PrimitiveComposer

	children []PrimitiveComposer

	await bool

	ast sqlparser.SQLNode

	builder primitivebuilder.Builder

	graph *primitivegraph.PrimitiveGraph

	drmConfig drm.DRMConfig

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
	txnCounterManager *txncounter.TxnCounterManager
	txnCtrlCtrs       *dto.TxnControlCounters

	// per query -- SELECT only
	insertValOnlyRows          map[int]map[int]interface{}
	valOnlyCols                map[int]map[string]interface{}
	insertPreparedStatementCtx *drm.PreparedStatementCtx
	selectPreparedStatementCtx *drm.PreparedStatementCtx

	// TODO: universally retire in favour of builder, which returns primitive.IPrimitive
	root primitivegraph.PrimitiveNode

	symTab symtab.SymTab

	where *sqlparser.Where

	sqlEngine sqlengine.SQLEngine
}

func (pb *StandardPrimitiveComposer) ShouldCollectGarbage() bool {
	return pb.parent == nil
}

func (pb *StandardPrimitiveComposer) SetTxnCtrlCtrs(tc *dto.TxnControlCounters) {
	pb.txnCtrlCtrs = tc
}

func (pb *StandardPrimitiveComposer) GetTxnCtrlCtrs() *dto.TxnControlCounters {
	return pb.txnCtrlCtrs
}

func (pb *StandardPrimitiveComposer) GetGraph() *primitivegraph.PrimitiveGraph {
	return pb.graph
}

func (pb *StandardPrimitiveComposer) GetParent() PrimitiveComposer {
	return pb.parent
}

func (pb *StandardPrimitiveComposer) GetChildren() []PrimitiveComposer {
	return pb.children
}

func (pb *StandardPrimitiveComposer) AddChild(val PrimitiveComposer) {
	pb.children = append(pb.children, val)
}

func (pb *StandardPrimitiveComposer) GetSymbol(k interface{}) (symtab.SymTabEntry, error) {
	return pb.symTab.GetSymbol(k)
}

func (pb *StandardPrimitiveComposer) GetSymTab() symtab.SymTab {
	return pb.symTab
}

func (pb *StandardPrimitiveComposer) SetSymbol(k interface{}, v symtab.SymTabEntry) error {
	return pb.symTab.SetSymbol(k, v)
}

func (pb *StandardPrimitiveComposer) GetWhere() *sqlparser.Where {
	return pb.where
}

func (pb *StandardPrimitiveComposer) SetWhere(where *sqlparser.Where) {
	pb.where = where
}

func (pb *StandardPrimitiveComposer) GetAst() sqlparser.SQLNode {
	return pb.ast
}

func (pb *StandardPrimitiveComposer) GetTxnCounterManager() *txncounter.TxnCounterManager {
	return pb.txnCounterManager
}

func (pb *StandardPrimitiveComposer) NewChildPrimitiveBuilder(ast sqlparser.SQLNode) PrimitiveComposer {
	child := NewPrimitiveComposer(pb, ast, pb.drmConfig, pb.txnCounterManager, pb.graph, pb.tables, pb.symTab, pb.sqlEngine)
	pb.children = append(pb.children, child)
	return child
}

func (pb *StandardPrimitiveComposer) GetInsertValOnlyRows() map[int]map[int]interface{} {
	return pb.insertValOnlyRows
}

func (pb *StandardPrimitiveComposer) SetInsertValOnlyRows(m map[int]map[int]interface{}) {
	pb.insertValOnlyRows = m
}

func (pb *StandardPrimitiveComposer) GetColumnOrder() []string {
	return pb.columnOrder
}

func (pb *StandardPrimitiveComposer) SetColumnOrder(co []parserutil.ColumnHandle) {
	var colOrd []string
	for _, v := range co {
		colOrd = append(colOrd, v.Name)
	}
	pb.columnOrder = colOrd
}

func (pb *StandardPrimitiveComposer) GetRoot() primitivegraph.PrimitiveNode {
	return pb.root
}

func (pb *StandardPrimitiveComposer) SetRoot(root primitivegraph.PrimitiveNode) {
	pb.root = root
}

func (pb *StandardPrimitiveComposer) GetCommentDirectives() sqlparser.CommentDirectives {
	return pb.commentDirectives
}

func (pb *StandardPrimitiveComposer) SetCommentDirectives(dirs sqlparser.CommentDirectives) {
	pb.commentDirectives = dirs
}

func (pb *StandardPrimitiveComposer) GetLikeAbleColumns() []string {
	return pb.likeAbleColumns
}

func (pb *StandardPrimitiveComposer) SetLikeAbleColumns(cols []string) {
	pb.likeAbleColumns = cols
}

func (pb *StandardPrimitiveComposer) GetValOnlyColKeys() []int {
	keys := make([]int, 0, len(pb.valOnlyCols))
	for k := range pb.valOnlyCols {
		keys = append(keys, k)
	}
	return keys
}

func (pb *StandardPrimitiveComposer) GetValOnlyCol(key int) map[string]interface{} {
	return pb.valOnlyCols[key]
}

func (pb *StandardPrimitiveComposer) SetValOnlyCols(m map[int]map[string]interface{}) {
	pb.valOnlyCols = m
}

func (pb *StandardPrimitiveComposer) SetColVisited(colname string, isVisited bool) {
	pb.colsVisited[colname] = isVisited
}

func (pb *StandardPrimitiveComposer) GetTableFilter() func(openapistackql.ITable) (openapistackql.ITable, error) {
	return pb.tableFilter
}

func (pb *StandardPrimitiveComposer) SetTableFilter(tableFilter func(openapistackql.ITable) (openapistackql.ITable, error)) {
	pb.tableFilter = tableFilter
}

func (pb *StandardPrimitiveComposer) SetInsertPreparedStatementCtx(ctx *drm.PreparedStatementCtx) {
	pb.insertPreparedStatementCtx = ctx
}

func (pb *StandardPrimitiveComposer) GetInsertPreparedStatementCtx() *drm.PreparedStatementCtx {
	return pb.insertPreparedStatementCtx
}

func (pb *StandardPrimitiveComposer) SetSelectPreparedStatementCtx(ctx *drm.PreparedStatementCtx) {
	pb.selectPreparedStatementCtx = ctx
}

func (pb *StandardPrimitiveComposer) GetSelectPreparedStatementCtx() *drm.PreparedStatementCtx {
	return pb.selectPreparedStatementCtx
}

func (pb *StandardPrimitiveComposer) GetProvider() provider.IProvider {
	return pb.prov
}

func (pb *StandardPrimitiveComposer) SetProvider(prov provider.IProvider) {
	pb.prov = prov
}

func (pb *StandardPrimitiveComposer) GetBuilder() primitivebuilder.Builder {
	if pb.children == nil || len(pb.children) == 0 {
		return pb.builder
	}
	var builders []primitivebuilder.Builder
	for _, child := range pb.children {
		if bldr := child.GetBuilder(); bldr != nil {
			builders = append(builders, bldr)
		}
	}
	if true {
		return primitivebuilder.NewDiamondBuilder(pb.builder, builders, pb.graph, pb.sqlEngine, pb.ShouldCollectGarbage())
	}
	return primitivebuilder.NewSubTreeBuilder(builders)
}

func (pb *StandardPrimitiveComposer) SetBuilder(builder primitivebuilder.Builder) {
	pb.builder = builder
}

func (pb *StandardPrimitiveComposer) IsAwait() bool {
	return pb.await
}

func (pb *StandardPrimitiveComposer) SetAwait(await bool) {
	pb.await = await
}

func (pb *StandardPrimitiveComposer) GetTable(node sqlparser.SQLNode) (*taxonomy.ExtendedTableMetadata, error) {
	return pb.tables.GetTable(node)
}

func (pb *StandardPrimitiveComposer) SetTable(node sqlparser.SQLNode, table *taxonomy.ExtendedTableMetadata) {
	pb.tables.SetTable(node, table)
}

func (pb *StandardPrimitiveComposer) GetTables() taxonomy.TblMap {
	return pb.tables
}

func (pb *StandardPrimitiveComposer) GetDRMConfig() drm.DRMConfig {
	return pb.drmConfig
}

func (pb *StandardPrimitiveComposer) GetSQLEngine() sqlengine.SQLEngine {
	return pb.sqlEngine
}

func NewPrimitiveComposer(parent PrimitiveComposer, ast sqlparser.SQLNode, drmConfig drm.DRMConfig, txnCtrMgr *txncounter.TxnCounterManager, graph *primitivegraph.PrimitiveGraph, tblMap taxonomy.TblMap, symTab symtab.SymTab, sqlEngine sqlengine.SQLEngine) PrimitiveComposer {
	return &StandardPrimitiveComposer{
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
	}
}
