package astvisit

import (
	"fmt"
	"strings"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/astanalysis/annotatedast"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/sql_system"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
	"github.com/stackql/stackql/pkg/textutil"
)

//nolint:errcheck // defer analyser uplifts
func GenerateUnionTemplateQuery(
	annotatedAST annotatedast.AnnotatedAst,
	node *sqlparser.Union,
	sqlSystem sql_system.SQLSystem,
	formatter sqlparser.NodeFormatter,
	namespaceCollection tablenamespace.Collection,
) string {
	v := NewFragmentRewriteAstVisitor(annotatedAST, "", false, sqlSystem, formatter, namespaceCollection)

	var sb strings.Builder
	sb.WriteString(textutil.IndirectQueryPlaceholder + " ")
	for _, unionSelect := range node.UnionSelects {
		sb.WriteString(fmt.Sprintf("%s %s ", unionSelect.Type, textutil.IndirectQueryPlaceholder))
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

func ExtractParamsFromWhereClause(
	annotatedAST annotatedast.AnnotatedAst,
	node *sqlparser.Where,
) parserutil.ParameterMap {
	v := NewParamAstVisitor(annotatedAST, "", false)
	if node != nil && node.Expr != nil {
		node.Expr.Accept(v) //nolint:errcheck // defer analyser uplifts
	} else {
		return parserutil.NewParameterMap()
	}
	rv := v.GetParameters()
	annotatedAST.SetWhereParamMapsEntry(node, rv)
	return rv
}

func ExtractProviderStringsAndDetectCacheExemptMaterial(
	annotatedAST annotatedast.AnnotatedAst,
	node sqlparser.SQLNode,
	sqlSystem sql_system.SQLSystem,
	formatter sqlparser.NodeFormatter,
	namespaceCollection tablenamespace.Collection,
) ([]string, bool) {
	v := NewProviderStringAstVisitor(annotatedAST, sqlSystem, formatter, namespaceCollection)
	node.Accept(v) //nolint:errcheck // defer analyser uplifts
	return v.GetProviderStrings(), v.ContainsCacheExemptMaterial()
}
