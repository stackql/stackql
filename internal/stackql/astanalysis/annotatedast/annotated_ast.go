package annotatedast

import (
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/astanalysis/selectmetadata"
	"github.com/stackql/stackql/internal/stackql/astindirect"
	"github.com/stackql/stackql/internal/stackql/datasource/sql_datasource"
	"github.com/stackql/stackql/internal/stackql/parserutil"
)

func NewAnnotatedAst(parent AnnotatedAst, ast sqlparser.Statement) (AnnotatedAst, error) {
	rv := &standardAnnotatedAst{
		parent:               parent,
		ast:                  ast,
		tableIndirects:       make(map[string]astindirect.Indirect),
		materializedViewRefs: make(map[string]astindirect.Indirect),
		tableSQLDataSources:  make(map[string]sql_datasource.SQLDataSource),
		selectMetadataMap:    make(map[*sqlparser.Select]selectmetadata.SelectMetadata),
		whereParamMaps:       make(map[*sqlparser.Where]parserutil.ParameterMap),
		insertRowsIndirect:   make(map[*sqlparser.Insert]astindirect.Indirect),
		selectIndirectCache:  make(map[*sqlparser.Select]astindirect.Indirect),
		execIndirectCache:    make(map[*sqlparser.Exec]astindirect.Indirect),
		unionIndirectCache:   make(map[*sqlparser.Union]astindirect.Indirect),
	}
	return rv, nil
}

type AnnotatedAst interface {
	GetAST() sqlparser.Statement
	GetIndirect(sqlparser.SQLNode) (astindirect.Indirect, bool)
	GetIndirects() map[string]astindirect.Indirect
	GetMaterializedView(sqlparser.SQLNode) (astindirect.Indirect, bool)
	GetPhysicalTable(sqlparser.SQLNode) (astindirect.Indirect, bool)
	GetSelectMetadata(*sqlparser.Select) (selectmetadata.SelectMetadata, bool)
	GetSQLDataSource(node sqlparser.SQLNode) (sql_datasource.SQLDataSource, bool)
	SetIndirect(node sqlparser.SQLNode, indirect astindirect.Indirect)
	SetMaterializedView(node sqlparser.SQLNode, indirect astindirect.Indirect)
	SetPhysicalTable(node sqlparser.SQLNode, indirect astindirect.Indirect)
	SetSelectMetadata(*sqlparser.Select, selectmetadata.SelectMetadata)
	SetSQLDataSource(node sqlparser.SQLNode, sqlDataSource sql_datasource.SQLDataSource)
	SetWhereParamMapsEntry(*sqlparser.Where, parserutil.ParameterMap)
	GetWhereParamMapsEntry(*sqlparser.Where) (parserutil.ParameterMap, bool)
	IsReadOnly() bool
	SetInsertRowsIndirect(node *sqlparser.Insert, indirect astindirect.Indirect)
	GetInsertRowsIndirect(*sqlparser.Insert) (astindirect.Indirect, bool)
	GetSelectIndirect(selNode *sqlparser.Select) (astindirect.Indirect, bool)
	SetSelectIndirect(selNode *sqlparser.Select, indirect astindirect.Indirect)
	GetExecIndirect(selNode *sqlparser.Exec) (astindirect.Indirect, bool)
	SetExecIndirect(selNode *sqlparser.Exec, indirect astindirect.Indirect)
}

type standardAnnotatedAst struct {
	parent               AnnotatedAst
	ast                  sqlparser.Statement
	tableIndirects       map[string]astindirect.Indirect
	materializedViewRefs map[string]astindirect.Indirect
	physicalTableRefs    map[string]astindirect.Indirect
	insertRowsIndirect   map[*sqlparser.Insert]astindirect.Indirect
	tableSQLDataSources  map[string]sql_datasource.SQLDataSource
	selectMetadataMap    map[*sqlparser.Select]selectmetadata.SelectMetadata
	whereParamMaps       map[*sqlparser.Where]parserutil.ParameterMap
	selectIndirectCache  map[*sqlparser.Select]astindirect.Indirect
	unionIndirectCache   map[*sqlparser.Union]astindirect.Indirect
	execIndirectCache    map[*sqlparser.Exec]astindirect.Indirect
}

func (aa *standardAnnotatedAst) GetExecIndirect(selNode *sqlparser.Exec) (astindirect.Indirect, bool) {
	rv, ok := aa.execIndirectCache[selNode]
	return rv, ok
}

func (aa *standardAnnotatedAst) SetExecIndirect(selNode *sqlparser.Exec, indirect astindirect.Indirect) {
	aa.execIndirectCache[selNode] = indirect
}

func (aa *standardAnnotatedAst) GetSelectIndirect(selNode *sqlparser.Select) (astindirect.Indirect, bool) {
	rv, ok := aa.selectIndirectCache[selNode]
	return rv, ok
}

func (aa *standardAnnotatedAst) SetSelectIndirect(selNode *sqlparser.Select, indirect astindirect.Indirect) {
	aa.selectIndirectCache[selNode] = indirect
}

func (aa *standardAnnotatedAst) SetInsertRowsIndirect(node *sqlparser.Insert, indirect astindirect.Indirect) {
	aa.insertRowsIndirect[node] = indirect
}

func (aa *standardAnnotatedAst) GetInsertRowsIndirect(node *sqlparser.Insert) (astindirect.Indirect, bool) {
	rv, ok := aa.insertRowsIndirect[node]
	if ok {
		return rv, true
	}
	switch n := node.Rows.(type) {
	case *sqlparser.Select:
		return aa.GetSelectIndirect(n)
	case *sqlparser.Union:
		rv, ok = aa.unionIndirectCache[n]
		return rv, ok
	case *sqlparser.ParenSelect:
		return nil, false
	default:
		return nil, false
	}
}

func (aa *standardAnnotatedAst) SetWhereParamMapsEntry(node *sqlparser.Where, paramMap parserutil.ParameterMap) {
	aa.whereParamMaps[node] = paramMap
}

func (aa *standardAnnotatedAst) GetWhereParamMapsEntry(node *sqlparser.Where) (parserutil.ParameterMap, bool) {
	rv, ok := aa.whereParamMaps[node]
	return rv, ok
}

func (aa *standardAnnotatedAst) GetAST() sqlparser.Statement {
	return aa.ast
}

func (aa *standardAnnotatedAst) IsReadOnly() bool {
	return false
}

func (aa *standardAnnotatedAst) GetSelectMetadata(selNode *sqlparser.Select) (selectmetadata.SelectMetadata, bool) {
	sm, ok := aa.selectMetadataMap[selNode]
	return sm, ok
}

func (aa *standardAnnotatedAst) SetSelectMetadata(selNode *sqlparser.Select, meta selectmetadata.SelectMetadata) {
	aa.selectMetadataMap[selNode] = meta
}

func (aa *standardAnnotatedAst) GetIndirect(node sqlparser.SQLNode) (astindirect.Indirect, bool) {
	switch n := node.(type) {
	case *sqlparser.AliasedTableExpr:
		rv, ok := aa.tableIndirects[n.As.GetRawVal()]
		if ok {
			return rv, true
		}
		return aa.GetIndirect(n.Expr)
	case sqlparser.TableName:
		rv, ok := aa.tableIndirects[n.GetRawVal()]
		return rv, ok
	case *sqlparser.DDL:
		rv, ok := aa.tableIndirects[n.Table.GetRawVal()]
		return rv, ok
	case *sqlparser.RefreshMaterializedView:
		rv, ok := aa.tableIndirects[n.ViewName.GetRawVal()]
		return rv, ok
	default:
		return nil, false
	}
}

func (aa *standardAnnotatedAst) GetMaterializedView(node sqlparser.SQLNode) (astindirect.Indirect, bool) {
	switch n := node.(type) {
	case *sqlparser.AliasedTableExpr:
		rv, ok := aa.materializedViewRefs[n.As.GetRawVal()]
		if ok {
			return rv, true
		}
		return aa.GetIndirect(n.Expr)
	case sqlparser.TableName:
		rv, ok := aa.materializedViewRefs[n.GetRawVal()]
		return rv, ok
	default:
		return nil, false
	}
}

func (aa *standardAnnotatedAst) GetPhysicalTable(node sqlparser.SQLNode) (astindirect.Indirect, bool) {
	switch n := node.(type) {
	case *sqlparser.AliasedTableExpr:
		rv, ok := aa.physicalTableRefs[n.As.GetRawVal()]
		if ok {
			return rv, true
		}
		return aa.GetIndirect(n.Expr)
	case sqlparser.TableName:
		rv, ok := aa.physicalTableRefs[n.GetRawVal()]
		return rv, ok
	default:
		return nil, false
	}
}

func (aa *standardAnnotatedAst) GetIndirects() map[string]astindirect.Indirect {
	return aa.tableIndirects
}

func (aa *standardAnnotatedAst) SetIndirect(node sqlparser.SQLNode, indirect astindirect.Indirect) {
	switch n := node.(type) {
	case sqlparser.TableName:
		aa.tableIndirects[n.GetRawVal()] = indirect
	case *sqlparser.AliasedTableExpr:
		// this is for subqueries
		aa.tableIndirects[n.As.GetRawVal()] = indirect
	case *sqlparser.DDL:
		aa.tableIndirects[n.Table.GetRawVal()] = indirect
	case *sqlparser.RefreshMaterializedView:
		aa.tableIndirects[n.ViewName.GetRawVal()] = indirect
	default:
	}
}

func (aa *standardAnnotatedAst) SetMaterializedView(node sqlparser.SQLNode, indirect astindirect.Indirect) {
	switch n := node.(type) {
	case sqlparser.TableName:
		aa.materializedViewRefs[n.GetRawVal()] = indirect
	case *sqlparser.AliasedTableExpr:
		// this is for subqueries
		aa.materializedViewRefs[n.As.GetRawVal()] = indirect
	default:
	}
}

func (aa *standardAnnotatedAst) SetPhysicalTable(node sqlparser.SQLNode, indirect astindirect.Indirect) {
	switch n := node.(type) {
	case sqlparser.TableName:
		aa.physicalTableRefs[n.GetRawVal()] = indirect
	case *sqlparser.AliasedTableExpr:
		// this is for subqueries
		aa.physicalTableRefs[n.As.GetRawVal()] = indirect
	default:
	}
}

func (aa *standardAnnotatedAst) SetSQLDataSource(node sqlparser.SQLNode, sqlDataSource sql_datasource.SQLDataSource) {
	switch n := node.(type) {
	case sqlparser.TableName:
		aa.tableSQLDataSources[n.GetRawVal()] = sqlDataSource
	default:
	}
}

func (aa *standardAnnotatedAst) GetSQLDataSource(node sqlparser.SQLNode) (sql_datasource.SQLDataSource, bool) {
	switch n := node.(type) {
	case *sqlparser.AliasedTableExpr:
		return aa.GetSQLDataSource(n.Expr)
	case sqlparser.TableName:
		rv, ok := aa.tableSQLDataSources[n.GetRawVal()]
		return rv, ok
	default:
		return nil, false
	}
}
