package primitivebuilder

import (
	"fmt"
	"strings"

	"github.com/stackql/stackql/internal/stackql/astformat"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internaldto"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/util"
	"vitess.io/vitess/go/vt/sqlparser"
)

type ddl struct {
	graph      primitivegraph.PrimitiveGraph
	ddlObject  *sqlparser.DDL
	handlerCtx handler.HandlerContext
	root, tail primitivegraph.PrimitiveNode
}

func (un *ddl) Build() error {
	sqlDialect := un.handlerCtx.GetSQLDialect()
	if sqlDialect == nil {
		return fmt.Errorf("cannot proceed DDL execution with nil sql dialect object")
	}
	unionObj := un.ddlObject
	if sqlDialect == nil {
		return fmt.Errorf("cannot proceed DDL execution with nil ddl object")
	}
	unionEx := func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
		actionLowered := strings.ToLower(unionObj.Action)
		switch actionLowered {
		case "create":
			tableName := strings.Trim(astformat.String(unionObj.Table, sqlDialect.GetASTFormatter()), `"`)
			viewDDL := strings.ReplaceAll(astformat.String(unionObj.SelectStatement, sqlDialect.GetASTFormatter()), `"`, "")
			err := sqlDialect.CreateView(tableName, viewDDL)
			if err != nil {
				return internaldto.NewErroneousExecutorOutput(err)
			}
		case "drop":
			if tl := len(unionObj.FromTables); tl != 1 {
				return internaldto.NewErroneousExecutorOutput(fmt.Errorf("cannot drop table with supplied table count = %d", tl))
			}
			tableName := strings.Trim(astformat.String(unionObj.FromTables[0], sqlDialect.GetASTFormatter()), `"`)
			err := sqlDialect.DropView(tableName)
			if err != nil {
				return internaldto.NewErroneousExecutorOutput(err)
			}
		default:
		}
		return util.PrepareResultSet(
			internaldto.NewPrepareResultSetPlusRawDTO(
				nil,
				map[string]map[string]interface{}{"0": {"message": "DDL execution completed"}},
				[]string{"message"},
				nil,
				nil,
				nil,
				nil,
			),
		)
	}
	graph := un.graph
	unionNode := graph.CreatePrimitiveNode(primitive.NewLocalPrimitive(unionEx))
	un.root = unionNode
	un.tail = unionNode
	return nil
}

func NewDDL(graph primitivegraph.PrimitiveGraph, handlerCtx handler.HandlerContext, ddlObject *sqlparser.DDL) Builder {
	return &ddl{
		graph:      graph,
		handlerCtx: handlerCtx,
		ddlObject:  ddlObject,
	}
}

func (ss *ddl) GetRoot() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *ddl) GetTail() primitivegraph.PrimitiveNode {
	return ss.tail
}
