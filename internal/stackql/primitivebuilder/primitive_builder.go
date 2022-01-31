package primitivebuilder

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/provider"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/symtab"
	"github.com/stackql/stackql/internal/stackql/taxonomy"

	"github.com/stackql/go-openapistackql/openapistackql"

	"github.com/stackql/stackql/internal/pkg/txncounter"

	"vitess.io/vitess/go/vt/sqlparser"
)

type PrimitiveBuilder struct {
	parent *PrimitiveBuilder

	children []*PrimitiveBuilder

	await bool

	ast sqlparser.SQLNode

	builder Builder

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

func (pb *PrimitiveBuilder) ShouldCollectGarbage() bool {
	return pb.parent == nil
}

func (pb *PrimitiveBuilder) SetTxnCtrlCtrs(tc *dto.TxnControlCounters) {
	pb.txnCtrlCtrs = tc
}

func (pb *PrimitiveBuilder) GetGraph() *primitivegraph.PrimitiveGraph {
	return pb.graph
}

func (pb *PrimitiveBuilder) GetParent() *PrimitiveBuilder {
	return pb.parent
}

func (pb *PrimitiveBuilder) GetChildren() []*PrimitiveBuilder {
	return pb.children
}

func (pb *PrimitiveBuilder) AddChild(val *PrimitiveBuilder) {
	pb.children = append(pb.children, val)
}

func (pb *PrimitiveBuilder) GetSymbol(k interface{}) (symtab.SymTabEntry, error) {
	return pb.symTab.GetSymbol(k)
}

func (pb *PrimitiveBuilder) GetSymTab() symtab.SymTab {
	return pb.symTab
}

func (pb *PrimitiveBuilder) SetSymbol(k interface{}, v symtab.SymTabEntry) error {
	return pb.symTab.SetSymbol(k, v)
}

func (pb *PrimitiveBuilder) GetWhere() *sqlparser.Where {
	return pb.where
}

func (pb *PrimitiveBuilder) SetWhere(where *sqlparser.Where) {
	pb.where = where
}

func (pb *PrimitiveBuilder) GetAst() sqlparser.SQLNode {
	return pb.ast
}

func (pb *PrimitiveBuilder) GetTxnCounterManager() *txncounter.TxnCounterManager {
	return pb.txnCounterManager
}

func (pb *PrimitiveBuilder) NewChildPrimitiveBuilder(ast sqlparser.SQLNode) *PrimitiveBuilder {
	child := NewPrimitiveBuilder(pb, ast, pb.drmConfig, pb.txnCounterManager, pb.graph, pb.tables, pb.symTab, pb.sqlEngine)
	pb.children = append(pb.children, child)
	return child
}

func (pb *PrimitiveBuilder) GetInsertValOnlyRows() map[int]map[int]interface{} {
	return pb.insertValOnlyRows
}

func (pb *PrimitiveBuilder) SetInsertValOnlyRows(m map[int]map[int]interface{}) {
	pb.insertValOnlyRows = m
}

func (pb *PrimitiveBuilder) GetColumnOrder() []string {
	return pb.columnOrder
}

func (pb *PrimitiveBuilder) SetColumnOrder(co []parserutil.ColumnHandle) {
	var colOrd []string
	for _, v := range co {
		colOrd = append(colOrd, v.Name)
	}
	pb.columnOrder = colOrd
}

func (pb *PrimitiveBuilder) GetRoot() primitivegraph.PrimitiveNode {
	return pb.root
}

func (pb *PrimitiveBuilder) SetRoot(root primitivegraph.PrimitiveNode) {
	pb.root = root
}

func (pb *PrimitiveBuilder) GetCommentDirectives() sqlparser.CommentDirectives {
	return pb.commentDirectives
}

func (pb *PrimitiveBuilder) SetCommentDirectives(dirs sqlparser.CommentDirectives) {
	pb.commentDirectives = dirs
}

func (pb *PrimitiveBuilder) GetLikeAbleColumns() []string {
	return pb.likeAbleColumns
}

func (pb *PrimitiveBuilder) SetLikeAbleColumns(cols []string) {
	pb.likeAbleColumns = cols
}

func (pb *PrimitiveBuilder) GetValOnlyColKeys() []int {
	keys := make([]int, 0, len(pb.valOnlyCols))
	for k := range pb.valOnlyCols {
		keys = append(keys, k)
	}
	return keys
}

func (pb *PrimitiveBuilder) GetValOnlyCol(key int) map[string]interface{} {
	return pb.valOnlyCols[key]
}

func (pb *PrimitiveBuilder) SetValOnlyCols(m map[int]map[string]interface{}) {
	pb.valOnlyCols = m
}

func (pb *PrimitiveBuilder) SetColVisited(colname string, isVisited bool) {
	pb.colsVisited[colname] = isVisited
}

func (pb *PrimitiveBuilder) GetTableFilter() func(openapistackql.ITable) (openapistackql.ITable, error) {
	return pb.tableFilter
}

func (pb *PrimitiveBuilder) SetTableFilter(tableFilter func(openapistackql.ITable) (openapistackql.ITable, error)) {
	pb.tableFilter = tableFilter
}

func (pb *PrimitiveBuilder) SetInsertPreparedStatementCtx(ctx *drm.PreparedStatementCtx) {
	pb.insertPreparedStatementCtx = ctx
}

func (pb *PrimitiveBuilder) GetInsertPreparedStatementCtx() *drm.PreparedStatementCtx {
	return pb.insertPreparedStatementCtx
}

func (pb *PrimitiveBuilder) SetSelectPreparedStatementCtx(ctx *drm.PreparedStatementCtx) {
	pb.selectPreparedStatementCtx = ctx
}

func (pb *PrimitiveBuilder) GetSelectPreparedStatementCtx() *drm.PreparedStatementCtx {
	return pb.selectPreparedStatementCtx
}

func (pb *PrimitiveBuilder) GetProvider() provider.IProvider {
	return pb.prov
}

func (pb *PrimitiveBuilder) SetProvider(prov provider.IProvider) {
	pb.prov = prov
}

func (pb *PrimitiveBuilder) GetBuilder() Builder {
	if pb.children == nil || len(pb.children) == 0 {
		return pb.builder
	}
	var builders []Builder
	for _, child := range pb.children {
		if bldr := child.GetBuilder(); bldr != nil {
			builders = append(builders, bldr)
		}
	}
	if true {
		return NewDiamondBuilder(pb.builder, builders, pb.graph, pb.sqlEngine, pb.ShouldCollectGarbage())
	}
	return NewSubTreeBuilder(builders)
}

func (pb *PrimitiveBuilder) SetBuilder(builder Builder) {
	pb.builder = builder
}

func (pb *PrimitiveBuilder) IsAwait() bool {
	return pb.await
}

func (pb *PrimitiveBuilder) SetAwait(await bool) {
	pb.await = await
}

func (pb PrimitiveBuilder) GetTable(node sqlparser.SQLNode) (taxonomy.ExtendedTableMetadata, error) {
	return pb.tables.GetTable(node)
}

func (pb PrimitiveBuilder) SetTable(node sqlparser.SQLNode, table taxonomy.ExtendedTableMetadata) {
	pb.tables.SetTable(node, table)
}

func (pb PrimitiveBuilder) GetTables() taxonomy.TblMap {
	return pb.tables
}

func (pb PrimitiveBuilder) GetDRMConfig() drm.DRMConfig {
	return pb.drmConfig
}

func (pb PrimitiveBuilder) GetSQLEngine() sqlengine.SQLEngine {
	return pb.sqlEngine
}

type HTTPRestPrimitive struct {
	Provider      provider.IProvider
	Executor      func(pc primitive.IPrimitiveCtx) dto.ExecutorOutput
	Preparator    func() *drm.PreparedStatementCtx
	TxnControlCtr *dto.TxnControlCounters
	Inputs        map[int64]dto.ExecutorOutput
	InputAliases  map[string]int64
	id            int64
}

type MetaDataPrimitive struct {
	Provider   provider.IProvider
	Executor   func(pc primitive.IPrimitiveCtx) dto.ExecutorOutput
	Preparator func() *drm.PreparedStatementCtx
	id         int64
}

type LocalPrimitive struct {
	Executor   func(pc primitive.IPrimitiveCtx) dto.ExecutorOutput
	Preparator func() *drm.PreparedStatementCtx
	Inputs     map[int64]dto.ExecutorOutput
	id         int64
}

type PassThroughPrimitive struct {
	Inputs                 map[int64]dto.ExecutorOutput
	id                     int64
	sqlEngine              sqlengine.SQLEngine
	shouldCollectGarbage   bool
	txnControlCounterSlice []dto.TxnControlCounters
}

func (pt *PassThroughPrimitive) collectGarbage() {
	if pt.shouldCollectGarbage {
		for _, gc := range pt.txnControlCounterSlice {
			pt.sqlEngine.GCCollectObsolete(&gc)
		}
	}
}

func (pt *PassThroughPrimitive) Execute(pc primitive.IPrimitiveCtx) dto.ExecutorOutput {
	defer pt.collectGarbage()
	for _, input := range pt.Inputs {
		return input
	}
	return dto.ExecutorOutput{}
}

func (pr *HTTPRestPrimitive) SetTxnId(id int) {
	if pr.TxnControlCtr != nil {
		pr.TxnControlCtr.TxnId = id
	}
}

func (pr *MetaDataPrimitive) SetTxnId(id int) {
}

func (pr *LocalPrimitive) SetTxnId(id int) {
}

func (pr *PassThroughPrimitive) SetTxnId(id int) {
}

func (pr *HTTPRestPrimitive) IncidentData(fromId int64, input dto.ExecutorOutput) error {
	pr.Inputs[fromId] = input
	return nil
}

func (pr *PassThroughPrimitive) IncidentData(fromId int64, input dto.ExecutorOutput) error {
	pr.Inputs[fromId] = input
	return nil
}

func (pr *MetaDataPrimitive) IncidentData(fromId int64, input dto.ExecutorOutput) error {
	return fmt.Errorf("MetaDataPrimitive cannot handle IncidentData")
}

func (pr *LocalPrimitive) IncidentData(fromId int64, input dto.ExecutorOutput) error {
	pr.Inputs[fromId] = input
	return nil
}

func (pr *HTTPRestPrimitive) SetInputAlias(alias string, id int64) error {
	pr.InputAliases[alias] = id
	return nil
}

func (pr *MetaDataPrimitive) SetInputAlias(alias string, id int64) error {
	return nil
}

func (pr *LocalPrimitive) SetInputAlias(alias string, id int64) error {
	return nil
}

func (pr *PassThroughPrimitive) SetInputAlias(alias string, id int64) error {
	return nil
}

func (pr *HTTPRestPrimitive) Optimise() error {
	return nil
}

func (pr *MetaDataPrimitive) Optimise() error {
	return nil
}

func (pr *LocalPrimitive) Optimise() error {
	return nil
}

func (pr *PassThroughPrimitive) Optimise() error {
	return nil
}

func (pr *HTTPRestPrimitive) Execute(pc primitive.IPrimitiveCtx) dto.ExecutorOutput {
	if pr.Executor != nil {
		op := pr.Executor(pc)
		return op
	}
	return dto.NewExecutorOutput(nil, nil, nil, nil, nil)
}

func (pr *HTTPRestPrimitive) ID() int64 {
	return pr.id
}

func (pr *MetaDataPrimitive) ID() int64 {
	return pr.id
}

func (pr *LocalPrimitive) ID() int64 {
	return pr.id
}

func (pr *MetaDataPrimitive) Execute(pc primitive.IPrimitiveCtx) dto.ExecutorOutput {
	if pr.Executor != nil {
		return pr.Executor(pc)
	}
	return dto.NewExecutorOutput(nil, nil, nil, nil, nil)
}

func (pr *LocalPrimitive) Execute(pc primitive.IPrimitiveCtx) dto.ExecutorOutput {
	if pr.Executor != nil {
		return pr.Executor(pc)
	}
	return dto.NewExecutorOutput(nil, nil, nil, nil, nil)
}

func NewMetaDataPrimitive(provider provider.IProvider, executor func(pc primitive.IPrimitiveCtx) dto.ExecutorOutput) *MetaDataPrimitive {
	return &MetaDataPrimitive{
		Provider: provider,
		Executor: executor,
	}
}

func NewHTTPRestPrimitive(provider provider.IProvider, executor func(pc primitive.IPrimitiveCtx) dto.ExecutorOutput, preparator func() *drm.PreparedStatementCtx, txnCtrlCtr *dto.TxnControlCounters) *HTTPRestPrimitive {
	return &HTTPRestPrimitive{
		Provider:      provider,
		Executor:      executor,
		Preparator:    preparator,
		TxnControlCtr: txnCtrlCtr,
		Inputs:        make(map[int64]dto.ExecutorOutput),
		InputAliases:  make(map[string]int64),
	}
}

func NewLocalPrimitive(executor func(pc primitive.IPrimitiveCtx) dto.ExecutorOutput) *LocalPrimitive {
	return &LocalPrimitive{
		Executor: executor,
		Inputs:   make(map[int64]dto.ExecutorOutput),
	}
}

func NewPassThroughPrimitive(sqlEngine sqlengine.SQLEngine, txnControlCounterSlice []dto.TxnControlCounters, shouldCollectGarbage bool) *PassThroughPrimitive {
	return &PassThroughPrimitive{
		Inputs:                 make(map[int64]dto.ExecutorOutput),
		sqlEngine:              sqlEngine,
		txnControlCounterSlice: txnControlCounterSlice,
		shouldCollectGarbage:   shouldCollectGarbage,
	}
}

func NewPrimitiveBuilder(parent *PrimitiveBuilder, ast sqlparser.SQLNode, drmConfig drm.DRMConfig, txnCtrMgr *txncounter.TxnCounterManager, graph *primitivegraph.PrimitiveGraph, tblMap map[sqlparser.SQLNode]taxonomy.ExtendedTableMetadata, symTab symtab.SymTab, sqlEngine sqlengine.SQLEngine) *PrimitiveBuilder {
	return &PrimitiveBuilder{
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
