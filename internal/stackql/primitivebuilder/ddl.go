package primitivebuilder

import (
	"fmt"
	"strings"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/astformat"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/util"
)

type ddl struct {
	graph      primitivegraph.PrimitiveGraph
	ddlObject  *sqlparser.DDL
	handlerCtx handler.HandlerContext
	root, tail primitivegraph.PrimitiveNode
}

func (un *ddl) Build() error {
	sqlSystem := un.handlerCtx.GetSQLSystem()
	if sqlSystem == nil {
		return fmt.Errorf("cannot proceed DDL execution with nil sql system object")
	}
	unionObj := un.ddlObject
	if sqlSystem == nil {
		return fmt.Errorf("cannot proceed DDL execution with nil ddl object")
	}
	unionEx := func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
		actionLowered := strings.ToLower(unionObj.Action)
		switch actionLowered {
		case "create":
			tableName := strings.Trim(astformat.String(unionObj.Table, sqlSystem.GetASTFormatter()), `"`)
			viewDDL := strings.ReplaceAll(astformat.String(unionObj.SelectStatement, sqlSystem.GetASTFormatter()), `"`, "")
			err := sqlSystem.CreateView(tableName, viewDDL)
			if err != nil {
				return internaldto.NewErroneousExecutorOutput(err)
			}
		case "drop":
			if tl := len(unionObj.FromTables); tl != 1 {
				return internaldto.NewErroneousExecutorOutput(fmt.Errorf("cannot drop table with supplied table count = %d", tl))
			}
			tableName := strings.Trim(astformat.String(unionObj.FromTables[0], sqlSystem.GetASTFormatter()), `"`)
			err := sqlSystem.DropView(tableName)
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

func (un *ddl) GetRoot() primitivegraph.PrimitiveNode {
	return un.root
}

func (un *ddl) GetTail() primitivegraph.PrimitiveNode {
	return un.tail
}
