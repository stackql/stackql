package astfuncrewrite

import (
	"fmt"
	"strings"

	"github.com/stackql/stackql/internal/stackql/constants"
	"vitess.io/vitess/go/vt/sqlparser"
)

type ASTFuncRewriter interface {
	RewriteFunc(funcExpr *sqlparser.FuncExpr) (*sqlparser.FuncExpr, error)
}

func GetPostgresASTFuncRewriter() ASTFuncRewriter {
	return &postgresFuncRewriter{}
}

func GetNopFuncRewriter() ASTFuncRewriter {
	return &nopFuncRewriter{}
}

type nopFuncRewriter struct{}

func (fr *nopFuncRewriter) RewriteFunc(funcExpr *sqlparser.FuncExpr) (*sqlparser.FuncExpr, error) {
	return funcExpr, nil
}

type postgresFuncRewriter struct{}

func (fr *postgresFuncRewriter) applyJSONConvIdempotent(funcArg sqlparser.SelectExpr) (sqlparser.SelectExpr, error) {
	return funcArg, nil
	// pathExpr, ok := funcArg.(*sqlparser.AliasedExpr)
	// if !ok {
	// 	return nil, fmt.Errorf("cannot accomodate 'json_extract' path expression of type = '%T'", pathExpr)
	// }
	// funcNode, ok := pathExpr.Expr.(*sqlparser.FuncExpr)
	// if ok {
	// 	if strings.ToLower(funcNode.Name.GetRawVal()) == constants.SQLFuncJSONExtractPostgres {
	// 		return funcArg, nil
	// 	}
	// 	return nil, fmt.Errorf("cannot handle postgres json_extract with first arg: '%s'", funcNode.Name.GetRawVal())
	// }
	// newJsonConvExpr := &sqlparser.FuncExpr{
	// 	Name: sqlparser.NewColIdent("to_json"),
	// }
	// newJsonConvExpr.Exprs = append(newJsonConvExpr.Exprs, funcArg)
	// return &sqlparser.AliasedExpr{Expr: newJsonConvExpr}, nil
}

func (fr *postgresFuncRewriter) rewriteJSONExtract(funcExpr *sqlparser.FuncExpr) (*sqlparser.FuncExpr, error) {
	funcExpr.Name = sqlparser.NewColIdent(constants.SQLFuncJSONExtractPostgres)
	if len(funcExpr.Exprs) != 2 {
		return nil, fmt.Errorf("cannot translate 'json_extract' function with arg count = %d", len(funcExpr.Exprs))
	}

	pathExpr, ok := funcExpr.Exprs[1].(*sqlparser.AliasedExpr)
	if !ok {
		return nil, fmt.Errorf("cannot accomodate 'json_extract' path expression of type = '%T'", pathExpr)
	}
	pathVal, ok := pathExpr.Expr.(*sqlparser.SQLVal)
	if !ok {
		return nil, fmt.Errorf("cannot accomodate 'json_extract' path val of type = '%T'", pathVal)
	}
	if pathVal.Type != sqlparser.StrVal {
		return nil, fmt.Errorf("cannot accomodate 'json_extract' path val with value type = '%d'", pathVal.Type)
	}
	pathStr := string(pathVal.Val)
	pathSplit := strings.Split(pathStr, ".")
	var newExprs sqlparser.SelectExprs

	firstArg, err := fr.applyJSONConvIdempotent(funcExpr.Exprs[0])
	if err != nil {
		return nil, err
	}

	newExprs = append(newExprs, firstArg)
	for i, j := range pathSplit {
		if i == 0 && j == "$" {
			continue
		}
		newVal := sqlparser.NewStrVal([]byte(j))
		newExpr := &sqlparser.AliasedExpr{Expr: newVal}
		newExprs = append(newExprs, newExpr)
	}
	funcExpr.Exprs = newExprs
	return funcExpr, nil
}

func (fr *postgresFuncRewriter) RewriteFunc(funcExpr *sqlparser.FuncExpr) (*sqlparser.FuncExpr, error) {
	if funcExpr == nil {
		return nil, nil
	}
	funcNameLowered := strings.ToLower(funcExpr.Name.GetRawVal())
	if funcNameLowered == constants.SQLFuncJSONExtractConformed {
		return fr.rewriteJSONExtract(funcExpr)
	}
	return funcExpr, nil
}
