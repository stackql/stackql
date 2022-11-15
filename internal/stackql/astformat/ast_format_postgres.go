package astformat

import (
	"fmt"
	"strings"

	"github.com/stackql/stackql/internal/stackql/constants"
	"vitess.io/vitess/go/vt/sqlparser"
)

var (
	_ sqlparser.NodeFormatter = PostgresSelectExprsFormatter
)

func PostgresSelectExprsFormatter(buf *sqlparser.TrackedBuffer, node sqlparser.SQLNode) {
	switch node := node.(type) {
	case sqlparser.ColIdent:
		formatColIdent(node, buf)
		return
	case *sqlparser.FuncExpr:
		if strings.ToLower(node.Name.GetRawVal()) == constants.SQLFuncJSONExtractPostgres {
			sb := sqlparser.NewTrackedBuffer(PostgresSelectExprsFormatter)
			sb.AstPrintf(node, "%s(%v::json, %v)", constants.SQLFuncJSONExtractPostgres, node.Exprs[0], node.Exprs[1])
			buf.WriteString(sb.String())
			return
		}
		// if strings.ToLower(node.Name.GetRawVal()) == constants.SQLFuncGroupConcatConformed {
		// 	sb := sqlparser.NewTrackedBuffer(PostgresSelectExprsFormatter)
		// 	sb.AstPrintf(node, "%s(%v, %s)", constants.SQLFuncGroupConcatPostgres, node.Exprs[0], "','")
		// 	buf.WriteString(sb.String())
		// 	return
		// }
		node.Format(buf)
		return
	case *sqlparser.GroupConcatExpr:
		sb := sqlparser.NewTrackedBuffer(PostgresSelectExprsFormatter)
		sb.AstPrintf(node, "%s(%v, %s)", constants.SQLFuncGroupConcatPostgres, node.Exprs[0], `','`)
		buf.WriteString(sb.String())
		return

	default:
		node.Format(buf)
		return
	}
}

func formatColIdent(node sqlparser.ColIdent, buf *sqlparser.TrackedBuffer) {
	if node.AtCount() > 0 {
		sqlparser.FormatID(buf, node.String(), node.Lowered(), node.AtCount())
		return
	}
	buf.WriteString(fmt.Sprintf(`"%s"`, node.String()))
}

func String(node sqlparser.SQLNode, formatter sqlparser.NodeFormatter) string {
	return formattedString(node, formatter)
}

func formattedString(node sqlparser.SQLNode, formatter sqlparser.NodeFormatter) string {
	buf := sqlparser.NewTrackedBuffer(formatter)
	buf.Myprintf("%v", node)
	return buf.String()
}
