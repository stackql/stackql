package annotatedast

import (
	"github.com/stackql/stackql/internal/stackql/astanalysis/selectmetadata"
	"github.com/stackql/stackql/internal/stackql/astindirect"
	"vitess.io/vitess/go/vt/sqlparser"
)

func NewAnnotatedAst(ast sqlparser.Statement) (AnnotatedAst, error) {
	rv := &standardAnnotatedAst{
		ast:               ast,
		tableIndirects:    make(map[string]astindirect.Indirect),
		selectMetadataMap: make(map[*sqlparser.Select]selectmetadata.SelectMetadata),
	}
	return rv, nil
}

type AnnotatedAst interface {
	GetAST() sqlparser.Statement
	GetIndirect(sqlparser.SQLNode) (astindirect.Indirect, bool)
	GetIndirects() map[string]astindirect.Indirect
	GetSelectMetadata(*sqlparser.Select) (selectmetadata.SelectMetadata, bool)
	SetIndirect(node sqlparser.SQLNode, indirect astindirect.Indirect)
	SetSelectMetadata(*sqlparser.Select, selectmetadata.SelectMetadata)
}

type standardAnnotatedAst struct {
	ast               sqlparser.Statement
	tableIndirects    map[string]astindirect.Indirect
	selectMetadataMap map[*sqlparser.Select]selectmetadata.SelectMetadata
}

func (aa *standardAnnotatedAst) GetAST() sqlparser.Statement {
	return aa.ast
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
		return aa.GetIndirect(n.Expr)
	case sqlparser.TableName:
		rv, ok := aa.tableIndirects[n.GetRawVal()]
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
	default:
	}
}
