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
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/util"
)

type ddl struct {
	graph        primitivegraph.PrimitiveGraphHolder
	ddlObject    *sqlparser.DDL
	handlerCtx   handler.HandlerContext
	root, tail   primitivegraph.PrimitiveNode
	annotatedAst annotatedast.AnnotatedAst
	bldrInput    builder_input.BuilderInput
}

//nolint:gocognit,nestif,funlen // acceptable
func (ddo *ddl) Build() error {
	sqlSystem := ddo.handlerCtx.GetSQLSystem()
	if sqlSystem == nil {
		return fmt.Errorf("cannot proceed DDL execution with nil sql system object")
	}
	parserDDLObj := ddo.ddlObject
	if sqlSystem == nil {
		return fmt.Errorf("cannot proceed DDL execution with nil ddl object")
	}
	unionEx := func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
		actionLowered := strings.ToLower(parserDDLObj.Action)
		switch actionLowered {
		case "create":
			tableName := strings.Trim(astformat.String(parserDDLObj.Table, sqlSystem.GetASTFormatter()), `"`)
			isTable := parserutil.IsCreatePhysicalTable(parserDDLObj)
			isTempTable := parserutil.IsCreateTemporaryPhysicalTable(parserDDLObj)
			isMaterializedView := parserutil.IsCreateMaterializedView(parserDDLObj)
			//nolint:gocritic // apathy
			if isTable || isTempTable { // TODO: support for create tables
				if isTempTable {
					return internaldto.NewErroneousExecutorOutput(fmt.Errorf("create temp table is not supported"))
				}
				drmCfg := ddo.handlerCtx.GetDrmConfig()
				createTableErr := drmCfg.CreatePhysicalTable(
					tableName,
					parserutil.RenderDDLStmt(parserDDLObj),
					parserDDLObj.TableSpec,
					parserDDLObj.IfNotExists,
				)
				if createTableErr != nil {
					return internaldto.NewErroneousExecutorOutput(createTableErr)
				}
				// return internaldto.NewErroneousExecutorOutput(fmt.Errorf("create table is not supported"))
			} else if isMaterializedView { // TODO: support for create materialized views
				indirect, indirectExists := ddo.annotatedAst.GetIndirect(ddo.ddlObject)
				if !indirectExists {
					return internaldto.NewErroneousExecutorOutput(fmt.Errorf("cannot find indirect object for materialized view"))
				}
				drmCfg := ddo.handlerCtx.GetDrmConfig()

				selStr := parserutil.RenderDDLSelectStmt(ddo.ddlObject)
				rawDDL := fmt.Sprintf(`CREATE MATERIALIZED VIEW "%s" AS %s`, tableName, selStr)
				if ddo.ddlObject.OrReplace {
					rawDDL = fmt.Sprintf(`CREATE OR REPLACE MATERIALIZED VIEW "%s" AS %s`, tableName, selStr)
				}
				selCtx := indirect.GetSelectContext()
				materializedViewCreateError := drmCfg.CreateMaterializedView(
					tableName,
					rawDDL,
					drm.NewPreparedStatementParameterized(selCtx, nil, true),
					ddo.ddlObject.OrReplace,
				)
				if materializedViewCreateError != nil {
					return internaldto.NewErroneousExecutorOutput(materializedViewCreateError)
				}
			} else {
				relationDDL := parserutil.RenderDDLSelectStmt(parserDDLObj)
				err := sqlSystem.CreateView(tableName, relationDDL, parserDDLObj.OrReplace)
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
			}
		case "drop":
			if tl := len(parserDDLObj.FromTables); tl != 1 {
				return internaldto.NewErroneousExecutorOutput(fmt.Errorf("cannot drop table with supplied table count = %d", tl))
			}
			tableName := strings.Trim(astformat.String(parserDDLObj.FromTables[0], sqlSystem.GetASTFormatter()), `"`)
			if parserutil.IsDropMaterializedView(parserDDLObj) { //nolint:gocritic // apathy
				err := sqlSystem.DropMaterializedView(tableName)
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
			} else if parserutil.IsDropPhysicalTable(parserDDLObj) {
				err := sqlSystem.DropPhysicalTable(tableName, parserDDLObj.IfExists)
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
			} else {
				err := sqlSystem.DropView(tableName)
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
			}
		default:
		}
		return util.PrepareResultSet(
			internaldto.NewPrepareResultSetPlusRawDTO(
				nil,
				map[string]map[string]interface{}{},
				[]string{},
				nil,
				nil,
				internaldto.NewBackendMessages(
					[]string{"DDL Execution Completed"},
				),
				nil,
				ddo.handlerCtx.GetTypingConfig(),
			),
		)
	}
	graph := ddo.graph
	ddlGraphNode := graph.CreatePrimitiveNode(primitive.NewLocalPrimitive(unionEx))

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

func NewDDL(
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
	ddlObject, isDDLObject := node.(*sqlparser.DDL)
	if !isDDLObject {
		return nil, fmt.Errorf("DDL builder cannot accomodate nil or non-DDL object")
	}
	annotatedAst, annotatedASTExists := bldrInput.GetAnnotatedAST()
	if !annotatedASTExists {
		return nil, fmt.Errorf("DDL builder cannot accomodate nil annotated AST")
	}
	return &ddl{
		graph:        graphHolder,
		handlerCtx:   handlerCtx,
		ddlObject:    ddlObject,
		annotatedAst: annotatedAst,
		bldrInput:    bldrInput,
	}, nil
}

func (ddo *ddl) GetRoot() primitivegraph.PrimitiveNode {
	return ddo.root
}

func (ddo *ddl) GetTail() primitivegraph.PrimitiveNode {
	return ddo.tail
}
