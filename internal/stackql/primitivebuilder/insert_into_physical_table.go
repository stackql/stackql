package primitivebuilder

import (
	"fmt"
	"strings"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/astanalysis/annotatedast"
	"github.com/stackql/stackql/internal/stackql/astformat"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/builder_input"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/util"
)

type insertIntoPhysicalTable struct {
	graph        primitivegraph.PrimitiveGraphHolder
	insertObject *sqlparser.Insert
	handlerCtx   handler.HandlerContext
	root, tail   primitivegraph.PrimitiveNode
	annotatedAst annotatedast.AnnotatedAst
	bldrInput    builder_input.BuilderInput
}

func (ddo *insertIntoPhysicalTable) Build() error {
	sqlSystem := ddo.handlerCtx.GetSQLSystem()
	if sqlSystem == nil {
		return fmt.Errorf("cannot proceed DDL execution with nil sql system object")
	}
	parserInsertObj := ddo.insertObject
	if sqlSystem == nil {
		return fmt.Errorf("cannot proceed DDL execution with nil insertIntoPhysicalTable object")
	}
	refreshEx := func(_ primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
		tableName := strings.Trim(astformat.String(parserInsertObj.Table, sqlSystem.GetASTFormatter()), `"`)
		indirect, indirectExists := ddo.annotatedAst.GetInsertRowsIndirect(ddo.insertObject)
		if !indirectExists {
			return internaldto.NewErroneousExecutorOutput(fmt.Errorf("cannot find indirect object for materialized view"))
		}
		insertColumnsString := astformat.String(parserInsertObj.Columns, sqlSystem.GetASTFormatter())
		drmCfg := ddo.handlerCtx.GetDrmConfig()
		selCtx := indirect.GetSelectContext()
		materializedViewRefreshError := drmCfg.InsertIntoPhysicalTable(
			tableName,
			insertColumnsString,
			drm.NewPreparedStatementParameterized(selCtx, nil, true),
		)
		if materializedViewRefreshError != nil {
			return internaldto.NewErroneousExecutorOutput(materializedViewRefreshError)
		}

		return util.PrepareResultSet(
			internaldto.NewPrepareResultSetPlusRawDTO(
				nil,
				map[string]map[string]interface{}{},
				[]string{},
				nil,
				nil,
				internaldto.NewBackendMessages(
					[]string{"insert into table completed"},
				),
				nil,
				ddo.handlerCtx.GetTypingConfig(),
			),
		)
	}
	graph := ddo.graph
	ddlGraphNode := graph.CreatePrimitiveNode(primitive.NewLocalPrimitive(refreshEx))

	ddo.root = ddlGraphNode
	ddo.tail = ddlGraphNode
	dependencyNode, dependencyNodeExists := ddo.bldrInput.GetDependencyNode()
	if dependencyNodeExists {
		//nolint:errcheck // TODO: fix this
		ddlGraphNode.SetInputAlias("", dependencyNode.ID())
		ddo.graph.NewDependency(dependencyNode, ddlGraphNode, 1.0)
		ddo.root = dependencyNode
	}
	return nil
}

func newInsertIntoPhysicalTable(
	bldrInput builder_input.BuilderInput,
) (Builder, error) {
	graphHolder, graphHolderExists := bldrInput.GetGraphHolder()
	if !graphHolderExists {
		return nil, fmt.Errorf("DDL builder cannot accomodate nil graph holder")
	}
	handlerCtx, handlerCtxExists := bldrInput.GetHandlerContext()
	if !handlerCtxExists {
		return nil, fmt.Errorf("DDL builder cannot accomodate nil handler context")
	}
	node, nodeExists := bldrInput.GetParserNode()
	if !nodeExists {
		return nil, fmt.Errorf("DDL builder cannot accomodate nil node")
	}
	insertObject, isDDLObject := node.(*sqlparser.Insert)
	if !isDDLObject {
		return nil, fmt.Errorf("DDL builder cannot accomodate nil or non-DDL object")
	}
	annotatedAst, annotatedASTExists := bldrInput.GetAnnotatedAST()
	if !annotatedASTExists {
		return nil, fmt.Errorf("DDL builder cannot accomodate nil annotated AST")
	}
	return &insertIntoPhysicalTable{
		graph:        graphHolder,
		handlerCtx:   handlerCtx,
		insertObject: insertObject,
		annotatedAst: annotatedAst,
		bldrInput:    bldrInput,
	}, nil
}

func (ddo *insertIntoPhysicalTable) GetRoot() primitivegraph.PrimitiveNode {
	return ddo.root
}

func (ddo *insertIntoPhysicalTable) GetTail() primitivegraph.PrimitiveNode {
	return ddo.tail
}
