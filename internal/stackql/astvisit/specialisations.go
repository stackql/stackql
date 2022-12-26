package astvisit

import (
	"fmt"
	"strings"

	"github.com/stackql/stackql/internal/stackql/astanalysis/annotatedast"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/sqldialect"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
	"vitess.io/vitess/go/vt/sqlparser"
)

func GenerateModifiedSelectSuffix(annotatedAST annotatedast.AnnotatedAst, node sqlparser.SQLNode, sqlDialect sqldialect.SQLDialect, formatter sqlparser.NodeFormatter, namespaceCollection tablenamespace.TableNamespaceCollection) string {
	v := NewFramentRewriteAstVisitor(annotatedAST, "", false, sqlDialect, formatter, namespaceCollection)
	switch node := node.(type) {
	case *sqlparser.Select:
		var options string
		addIf := func(b bool, s string) {
			if b {
				options += s
			}
		}
		addIf(node.Distinct, sqlparser.DistinctStr)
		if node.Cache != nil {
			if *node.Cache {
				options += sqlparser.SQLCacheStr
			} else {
				options += sqlparser.SQLNoCacheStr
			}
		}
		addIf(node.StraightJoinHint, sqlparser.StraightJoinHint)
		addIf(node.SQLCalcFoundRows, sqlparser.SQLCalcFoundRowsStr)

		var groupByStr, havingStr, orderByStr, limitStr string
		if node.GroupBy != nil {
			node.GroupBy.Accept(v)
			groupByStr = v.GetRewrittenQuery()
		}
		if node.Having != nil {
			node.Having.Accept(v)
			havingStr = v.GetRewrittenQuery()
		}
		if node.OrderBy != nil {
			node.OrderBy.Accept(v)
			orderByStr = v.GetRewrittenQuery()
		}
		if node.Limit != nil {
			node.Limit.Accept(v)
			orderByStr = v.GetRewrittenQuery()
		}
		rq := fmt.Sprintf("%v%v%v%v%s",
			groupByStr, havingStr, orderByStr,
			limitStr, node.Lock)
		v.SetRewrittenQuery(rq)
	}
	return v.GetRewrittenQuery()
}

func GenerateUnionTemplateQuery(annotatedAST annotatedast.AnnotatedAst, node *sqlparser.Union, sqlDialect sqldialect.SQLDialect, formatter sqlparser.NodeFormatter, namespaceCollection tablenamespace.TableNamespaceCollection) string {
	v := NewFramentRewriteAstVisitor(annotatedAST, "", false, sqlDialect, formatter, namespaceCollection)

	var sb strings.Builder
	sb.WriteString("%s ")
	for _, unionSelect := range node.UnionSelects {
		sb.WriteString(fmt.Sprintf("%s %%s ", unionSelect.Type))
	}

	var orderByStr, limitStr string
	if node.OrderBy != nil {
		node.OrderBy.Accept(v)
		orderByStr = v.GetRewrittenQuery()
	}
	if node.Limit != nil {
		node.Limit.Accept(v)
		orderByStr = v.GetRewrittenQuery()
	}
	sb.WriteString(
		fmt.Sprintf(
			"%v%v%s",
			orderByStr,
			limitStr,
			node.Lock))
	v.SetRewrittenQuery(sb.String())

	return v.GetRewrittenQuery()
}

func GenerateModifiedWhereClause(annotatedAST annotatedast.AnnotatedAst, node *sqlparser.Where, sqlDialect sqldialect.SQLDialect, formatter sqlparser.NodeFormatter, namespaceCollection tablenamespace.TableNamespaceCollection) string {
	v := NewFramentRewriteAstVisitor(annotatedAST, "", false, sqlDialect, formatter, namespaceCollection)
	var whereStr string
	if node != nil && node.Expr != nil {
		node.Expr.Accept(v)
		whereStr = v.GetRewrittenQuery()
	} else {
		return "true"
	}
	v.SetRewrittenQuery(whereStr)
	return v.GetRewrittenQuery()
}

func ExtractParamsFromWhereClause(annotatedAST annotatedast.AnnotatedAst, node *sqlparser.Where) parserutil.ParameterMap {
	v := NewParamAstVisitor(annotatedAST, "", false)
	if node != nil && node.Expr != nil {
		node.Expr.Accept(v)
	} else {
		return parserutil.NewParameterMap()
	}
	rv := v.GetParameters()
	annotatedAST.SetWhereParamMapsEntry(node, rv)
	return rv
}

func ExtractParamsFromExecSubqueryClause(annotatedAST annotatedast.AnnotatedAst, node *sqlparser.ExecSubquery) parserutil.ParameterMap {
	v := NewParamAstVisitor(annotatedAST, "", false)
	if node != nil && node.Exec != nil {
		node.Exec.Accept(v)
	} else {
		return parserutil.NewParameterMap()
	}
	return v.GetParameters()
}

func ExtractParamsFromFromClause(annotatedAST annotatedast.AnnotatedAst, node sqlparser.TableExprs) parserutil.ParameterMap {
	v := NewParamAstVisitor(annotatedAST, "", false)
	for _, expr := range node {
		expr.Accept(v)
	}
	return v.GetParameters()
}

func ExtractProviderStringsAndDetectCacheExemptMaterial(annotatedAST annotatedast.AnnotatedAst, node sqlparser.SQLNode, sqlDialect sqldialect.SQLDialect, formatter sqlparser.NodeFormatter, namespaceCollection tablenamespace.TableNamespaceCollection) ([]string, bool) {
	v := NewProviderStringAstVisitor(annotatedAST, sqlDialect, formatter, namespaceCollection)
	node.Accept(v)
	return v.GetProviderStrings(), v.ContainsCacheExemptMaterial()
}
