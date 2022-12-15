package annotatedast

import (
	"github.com/stackql/stackql/internal/stackql/astindirect"
	"vitess.io/vitess/go/vt/sqlparser"
)

type AnnotatedAst interface {
	GetAST() sqlparser.Statement
	GetIndirect(sqlparser.SQLNode) (astindirect.Indirect, bool)
	GetIndirects() map[string]astindirect.Indirect
	SetIndirect(node sqlparser.SQLNode, indirect astindirect.Indirect)
}

type standardAnnotatedAst struct {
	ast            sqlparser.Statement
	tableIndirects map[string]astindirect.Indirect
}

func (aa *standardAnnotatedAst) GetAST() sqlparser.Statement {
	return aa.ast
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

func NewAnnotatedAst(ast sqlparser.Statement) (AnnotatedAst, error) {
	rv := &standardAnnotatedAst{
		ast:            ast,
		tableIndirects: make(map[string]astindirect.Indirect),
	}
	return rv, nil
}
