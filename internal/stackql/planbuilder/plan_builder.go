package planbuilder

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/stackql/any-sdk/anysdk"

	"github.com/stackql/any-sdk/pkg/logging"
	"github.com/stackql/any-sdk/pkg/streaming"
	"github.com/stackql/stackql/internal/stackql/acid/txn_context"
	"github.com/stackql/stackql/internal/stackql/astanalysis/routeanalysis"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/builder_input"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/primitive_context"
	"github.com/stackql/stackql/internal/stackql/iqlerror"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/plan"
	"github.com/stackql/stackql/internal/stackql/planbuilderinput"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivebuilder"
	"github.com/stackql/stackql/internal/stackql/primitivegenerator"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/tableinsertioncontainer"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"github.com/stackql/stackql/internal/stackql/util"
	"golang.org/x/mod/semver"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

const (
	negativePlanCacheEnabled = "false"
	defaultPlanCacheEnabled  = negativePlanCacheEnabled
)

var (
	// only string "false" will disable.
	PlanCacheEnabled string           = defaultPlanCacheEnabled //nolint:revive,gochecknoglobals // acceptable
	_                planGraphBuilder = &standardPlanGraphBuilder{}
)

func isPlanCacheEnabled() bool {
	return strings.ToLower(PlanCacheEnabled) != negativePlanCacheEnabled
}

type planGraphBuilder interface {
	setRootPrimitiveGenerator(primitivegenerator.PrimitiveGenerator)
	pgInternal(planbuilderinput.PlanBuilderInput) error
	createInstructionFor(planbuilderinput.PlanBuilderInput) error
	nop(planbuilderinput.PlanBuilderInput) error
	getPlanGraphHolder() primitivegraph.PrimitiveGraphHolder
	setPrebuiltIndirect([]primitivebuilder.Builder)
	getPrebuiltIndirect() ([]primitivebuilder.Builder, bool)
}

type standardPlanGraphBuilder struct {
	planGraphHolder            primitivegraph.PrimitiveGraphHolder
	rootPrimitiveGenerator     primitivegenerator.PrimitiveGenerator
	transactionContext         txn_context.ITransactionContext
	preBuiltIndirectCollection []primitivebuilder.Builder
}

func (pgb *standardPlanGraphBuilder) getPrebuiltIndirect() ([]primitivebuilder.Builder, bool) {
	return pgb.preBuiltIndirectCollection, pgb.preBuiltIndirectCollection != nil
}

func (pgb *standardPlanGraphBuilder) setPrebuiltIndirect(builderz []primitivebuilder.Builder) {
	pgb.preBuiltIndirectCollection = builderz
}

func (pgb *standardPlanGraphBuilder) setRootPrimitiveGenerator(
	primitiveGenerator primitivegenerator.PrimitiveGenerator) {
	pgb.rootPrimitiveGenerator = primitiveGenerator
}

func (pgb *standardPlanGraphBuilder) getPlanGraphHolder() primitivegraph.PrimitiveGraphHolder {
	return pgb.planGraphHolder
}

func newPlanGraphBuilder(concurrencyLimit int, transactionContext txn_context.ITransactionContext) planGraphBuilder {
	return &standardPlanGraphBuilder{
		planGraphHolder:    primitivegraph.NewPrimitiveGraphHolder(concurrencyLimit),
		transactionContext: transactionContext,
	}
}

//nolint:funlen // no big deal
func (pgb *standardPlanGraphBuilder) createInstructionFor(pbi planbuilderinput.PlanBuilderInput) error {
	stmt := pbi.GetStatement()
	switch stmt := stmt.(type) {
	case *sqlparser.Auth:
		return pgb.handleAuth(pbi)
	case *sqlparser.AuthRevoke:
		return pgb.handleAuthRevoke(pbi)
	case *sqlparser.Begin:
		return pgb.nop(pbi)
	case *sqlparser.Commit:
		return pgb.nop(pbi)
	case *sqlparser.DBDDL:
		return iqlerror.GetStatementNotSupportedError(fmt.Sprintf("unsupported: Database DDL %v", sqlparser.String(stmt)))
	case *sqlparser.DDL:
		return pgb.handleDDL(pbi)
	case *sqlparser.Delete:
		return pgb.handleDelete(pbi)
	case *sqlparser.DescribeTable:
		return pgb.handleDescribe(pbi)
	case *sqlparser.Exec:
		return pgb.handleExec(pbi)
	case *sqlparser.Explain:
		return pgb.handleExplain(pbi)
	case *sqlparser.Insert:
		return pgb.handleInsert(pbi)
	case *sqlparser.NativeQuery:
		return pgb.handleNativeQuery(pbi)
	case *sqlparser.OtherRead, *sqlparser.OtherAdmin:
		return iqlerror.GetStatementNotSupportedError("OTHER")
	case *sqlparser.Purge:
		return pgb.handlePurge(pbi)
	case *sqlparser.RefreshMaterializedView:
		return pgb.handleRefreshMaterializedView(pbi)
	case *sqlparser.Registry:
		return pgb.handleRegistry(pbi)
	case *sqlparser.Rollback:
		return pgb.nop(pbi)
	case *sqlparser.Savepoint:
		return pgb.nop(pbi)
	case *sqlparser.Select:
		_, _, err := pgb.handleSelect(pbi)
		return err
	case *sqlparser.Set:
		return pgb.handleSet(pbi)
	case *sqlparser.SetTransaction:
		return pgb.nop(pbi)
	case *sqlparser.Show:
		return pgb.handleShow(pbi)
	case *sqlparser.Sleep:
		return pgb.handleSleep(pbi)
	case *sqlparser.SRollback:
		return pgb.nop(pbi)
	case *sqlparser.Release:
		return pgb.nop(pbi)
	case *sqlparser.Union:
		_, _, err := pgb.handleUnion(pbi)
		return err
	case *sqlparser.Update:
		return pgb.handleUpdate(pbi)
	case *sqlparser.Use:
		return pgb.handleUse(pbi)
	}
	return iqlerror.GetStatementNotSupportedError(fmt.Sprintf("BUG: unexpected statement type: %T", stmt))
}

func (pgb *standardPlanGraphBuilder) nop(pbi planbuilderinput.PlanBuilderInput) error {
	primitiveGenerator := pgb.rootPrimitiveGenerator
	err := primitiveGenerator.AnalyzeNop(pbi)
	if err != nil {
		return err
	}
	builder := primitiveGenerator.GetPrimitiveComposer().GetBuilder()
	if builder == nil {
		return fmt.Errorf("nil nop builder")
	}
	err = builder.Build()
	return err
}

func (pgb *standardPlanGraphBuilder) handleExplain(pbi planbuilderinput.PlanBuilderInput) error {
	primitiveGenerator := pgb.rootPrimitiveGenerator
	explain, _ok := pbi.GetExplain()
	if !_ok {
		return fmt.Errorf("could not retrieve explain AST node for statement of type '%T'", pbi.GetStatement())
	}
	childPbi, err := planbuilderinput.NewPlanBuilderInput(
		pbi.GetAnnotatedAST(),
		pbi.GetHandlerCtx(),
		explain.Statement,
		pbi.GetTableExprs(),
		pbi.GetAssignedAliasedColumns(),
		pbi.GetAliasedTables(),
		pbi.GetColRefs(),
		pbi.GetPlaceholderParams(),
		pbi.GetTxnCtrlCtrs(),
	)
	childPbi.SetReadOnly(true)
	if err != nil {
		return err
	}
	instructionErr := pgb.createInstructionFor(childPbi)
	explainMessages := []string{}
	if instructionErr == nil {
		explainMessages = append(explainMessages, "Execution plan generated successfully")
	}
	genErr := primitiveGenerator.AnalyzeExplain(pbi, explainMessages, instructionErr)
	if genErr != nil {
		return genErr
	}
	builder := primitiveGenerator.GetPrimitiveComposer().GetBuilder()
	if builder == nil {
		return fmt.Errorf("nil explain builder")
	}
	buildErr := builder.Build()
	return buildErr
}

func setLogic(pbi planbuilderinput.PlanBuilderInput, setExpr *sqlparser.SetExpr) error {
	lhsRaw := setExpr.Name.GetRawVal()
	lhsTrimmed := strings.TrimPrefix(lhsRaw, "$.")
	if lhsTrimmed == lhsRaw {
		return nil
	}
	exprStr := strings.Trim(sqlparser.String(setExpr.Expr), "'")
	exprObj := map[string]interface{}{}
	deserErr := json.Unmarshal([]byte(exprStr), &exprObj)
	if deserErr != nil {
		rawRv := pbi.GetHandlerCtx().SetConfigAtPath(lhsTrimmed, exprStr, setExpr.Scope)
		return rawRv
	}
	rv := pbi.GetHandlerCtx().SetConfigAtPath(lhsTrimmed, exprObj, setExpr.Scope)
	return rv
}

func (pgb *standardPlanGraphBuilder) handleSet(pbi planbuilderinput.PlanBuilderInput) error {
	primitiveGenerator := pgb.rootPrimitiveGenerator
	err := primitiveGenerator.AnalyzeStatement(pbi)
	if err != nil {
		return err
	}
	setStmt, ok := pbi.GetSet()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Set", pbi.GetStatement())
	}
	if len(setStmt.Exprs) < 1 {
		return fmt.Errorf("no set expressions found")
	}
	pr := primitive.NewLocalPrimitive(
		//nolint:revive // acceptable for now
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			err = setLogic(pbi, setStmt.Exprs[0])
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, err)
		})
	pgb.planGraphHolder.GetPrimitiveGraph().CreatePrimitiveNode(pr)
	return nil
}

func (pgb *standardPlanGraphBuilder) pgInternal(pbi planbuilderinput.PlanBuilderInput) error {
	primitiveGenerator := pgb.rootPrimitiveGenerator
	err := primitiveGenerator.AnalyzePGInternal(pbi)
	if err != nil {
		return err
	}
	builder := primitiveGenerator.GetPrimitiveComposer().GetBuilder()
	if builder == nil {
		return fmt.Errorf("nil pg internal builder")
	}
	err = builder.Build()
	return err
}

func (pgb *standardPlanGraphBuilder) handleAuth(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetAuth()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Auth", pbi.GetStatement())
	}
	primitiveGenerator := pgb.rootPrimitiveGenerator
	prov, err := handlerCtx.GetProvider(node.Provider)
	if err != nil {
		return err
	}
	err = primitiveGenerator.AnalyzeStatement(pbi)
	if err != nil {
		logging.GetLogger().Debugln(fmt.Sprintf("err = %s", err.Error()))
		return err
	}
	authCtx, authErr := handlerCtx.GetAuthContext(node.Provider)
	if authErr != nil {
		return authErr
	}
	pr := primitive.NewMetaDataPrimitive(
		prov,
		//nolint:revive // acceptable for now
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			authType := strings.ToLower(node.Type)
			if node.KeyFilePath != "" {
				authCtx.KeyFilePath = node.KeyFilePath
			}
			if node.KeyEnvVar != "" {
				authCtx.KeyEnvVar = node.KeyEnvVar
			}
			_, err = prov.Auth(authCtx, authType, true)
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, err)
		})
	pgb.planGraphHolder.GetPrimitiveGraph().CreatePrimitiveNode(pr)
	return nil
}

func (pgb *standardPlanGraphBuilder) handleAuthRevoke(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	stmt := pbi.GetStatement()
	node, ok := stmt.(*sqlparser.AuthRevoke)
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required AuthRevoke", stmt)
	}
	primitiveGenerator := pgb.rootPrimitiveGenerator
	err := primitiveGenerator.AnalyzeStatement(pbi)
	if err != nil {
		return err
	}
	prov, err := handlerCtx.GetProvider(node.Provider)
	if err != nil {
		return err
	}
	authCtx, authErr := handlerCtx.GetAuthContext(node.Provider)
	if authErr != nil {
		return authErr
	}
	pr := primitive.NewMetaDataPrimitive(
		prov,
		//nolint:revive // acceptable for now
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, prov.AuthRevoke(authCtx))
		})
	pgb.planGraphHolder.CreatePrimitiveNode(pr)
	return nil
}

func (pgb *standardPlanGraphBuilder) handleDescribe(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetDescribeTable()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required DescribeTable", pbi.GetStatement())
	}
	primitiveGenerator := pgb.rootPrimitiveGenerator
	err := primitiveGenerator.AnalyzeStatement(pbi)
	if err != nil {
		return err
	}
	md, err := primitiveGenerator.GetPrimitiveComposer().GetTable(node)
	if err != nil {
		return err
	}
	prov, err := md.GetProvider()
	if err != nil {
		return err
	}
	var extended bool = strings.TrimSpace(strings.ToUpper(node.Extended)) == "EXTENDED"
	var full bool = strings.TrimSpace(strings.ToUpper(node.Full)) == "FULL"
	_, isView := md.GetHeirarchyObjects().GetHeirarchyIDs().GetView()
	if isView {
		stmtCtx, sOk := primitiveGenerator.GetPrimitiveComposer().GetIndirectDescribeSelectCtx()
		if !sOk || stmtCtx == nil {
			return fmt.Errorf("cannot describe view without context")
		}
		nonControlColummns := stmtCtx.GetNonControlColumns()
		if len(nonControlColummns) < 1 {
			return fmt.Errorf("cannot describe view lacking columns")
		}
		pr := primitive.NewMetaDataPrimitive(
			prov,
			//nolint:revive // acceptable for now
			func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
				return primitivebuilder.NewDescribeViewInstructionExecutor(handlerCtx, md, nonControlColummns, extended, full)
			})
		pgb.planGraphHolder.CreatePrimitiveNode(pr)
		return nil
	}
	pr := primitive.NewMetaDataPrimitive(
		prov,
		//nolint:revive // acceptable for now
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			return primitivebuilder.NewDescribeTableInstructionExecutor(handlerCtx, md, extended, full)
		})
	pgb.planGraphHolder.CreatePrimitiveNode(pr)
	return nil
}

func (pgb *standardPlanGraphBuilder) arrangePrebuildIndirects(
	preBuiltIndirectCollection []primitivebuilder.Builder) (primitivegraph.PrimitiveNode, error) {
	var selectPrimitiveNode primitivegraph.PrimitiveNode
	for i, prebuiltIndirect := range preBuiltIndirectCollection {
		buildErr := prebuiltIndirect.Build()
		if buildErr != nil {
			return nil, buildErr
		}
		tailNode := prebuiltIndirect.GetTail()
		localRootNode := prebuiltIndirect.GetRoot()
		if tailNode != nil {
			if i > 0 {
				// TODO: create graph edge from previous tail to current head
				pgb.planGraphHolder.GetPrimitiveGraph().NewDependency(
					selectPrimitiveNode, localRootNode, 1.0,
				)
			}
			selectPrimitiveNode = tailNode
		} else {
			return nil, fmt.Errorf("could not obtain tail node from prebuilt indirect")
		}
	}
	return selectPrimitiveNode, nil
}

func (pgb *standardPlanGraphBuilder) handleDDL(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetDDL()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required DDL", pbi.GetStatement())
	}
	bldrInput := builder_input.NewBuilderInput(
		pgb.planGraphHolder,
		handlerCtx,
		nil,
	)
	bldrInput.SetParserNode(node)
	//nolint:nestif // TODO: refactor
	if node.SelectStatement != nil && parserutil.IsCreateMaterializedView(node) {
		preBuiltIndirectCollection, prebuildIndirectExists := pgb.getPrebuiltIndirect()
		var selectPrimitiveNode primitivegraph.PrimitiveNode
		if prebuildIndirectExists {
			var arrangeErr error
			selectPrimitiveNode, arrangeErr = pgb.arrangePrebuildIndirects(preBuiltIndirectCollection)
			if arrangeErr != nil {
				return arrangeErr
			}
		} else {
			selPbi, selErr := planbuilderinput.NewPlanBuilderInput(
				pbi.GetAnnotatedAST(),
				pbi.GetHandlerCtx(),
				node.SelectStatement,
				pbi.GetTableExprs(),
				pbi.GetAssignedAliasedColumns(),
				pbi.GetAliasedTables(),
				pbi.GetColRefs(),
				pbi.GetPlaceholderParams(),
				pbi.GetTxnCtrlCtrs())
			if selErr != nil {
				return selErr
			}
			var err error
			switch selStmt := node.SelectStatement.(type) {
			case *sqlparser.Select:
				_, selectPrimitiveNode, err = pgb.handleSelect(selPbi)
			case *sqlparser.ParenSelect:
				selPbi, selErr = planbuilderinput.NewPlanBuilderInput(
					pbi.GetAnnotatedAST(),
					pbi.GetHandlerCtx(),
					selStmt.Select,
					pbi.GetTableExprs(),
					pbi.GetAssignedAliasedColumns(),
					pbi.GetAliasedTables(),
					pbi.GetColRefs(),
					pbi.GetPlaceholderParams(),
					pbi.GetTxnCtrlCtrs())
				if selErr != nil {
					return selErr
				}
				_, selectPrimitiveNode, err = pgb.handleSelect(selPbi)
			case *sqlparser.Union:
				_, selectPrimitiveNode, err = pgb.handleUnion(selPbi)

			default:
				return fmt.Errorf("unsupported select statement type '%T'", selStmt)
			}
			if err != nil {
				return err
			}
		}
		bldrInput.SetDependencyNode(selectPrimitiveNode)
	}
	bldrInput.SetAnnotatedAST(pbi.GetAnnotatedAST())
	bldr, bldrErr := primitivebuilder.NewDDL(
		bldrInput,
	)
	if bldrErr != nil {
		return bldrErr
	}
	err := bldr.Build()
	if err != nil {
		return err
	}
	return nil
}

func (pgb *standardPlanGraphBuilder) handleRefreshMaterializedView(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetRefreshedMaterializedView()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required DDL", pbi.GetStatement())
	}
	bldrInput := builder_input.NewBuilderInput(
		pgb.planGraphHolder,
		handlerCtx,
		nil,
	)
	bldrInput.SetParserNode(node)
	//nolint:nestif // acceptable
	if node.ImplicitSelect != nil {
		preBuiltIndirectCollection, prebuildIndirectExists := pgb.getPrebuiltIndirect()
		var selectPrimitiveNode primitivegraph.PrimitiveNode
		if prebuildIndirectExists {
			var arrangeErr error
			selectPrimitiveNode, arrangeErr = pgb.arrangePrebuildIndirects(preBuiltIndirectCollection)
			if arrangeErr != nil {
				return arrangeErr
			}
		} else {
			selPbi, selErr := planbuilderinput.NewPlanBuilderInput(
				pbi.GetAnnotatedAST(),
				pbi.GetHandlerCtx(),
				node.ImplicitSelect,
				pbi.GetTableExprs(),
				pbi.GetAssignedAliasedColumns(),
				pbi.GetAliasedTables(),
				pbi.GetColRefs(),
				pbi.GetPlaceholderParams(),
				pbi.GetTxnCtrlCtrs())
			if selErr != nil {
				return selErr
			}
			var err error
			_, selectPrimitiveNode, err = pgb.handleSelect(selPbi)
			if err != nil {
				return err
			}
		}
		bldrInput.SetDependencyNode(selectPrimitiveNode)
	}
	bldrInput.SetAnnotatedAST(pbi.GetAnnotatedAST())
	bldr, bldrErr := primitivebuilder.NewRefreshMaterializedView(
		bldrInput,
	)
	if bldrErr != nil {
		return bldrErr
	}
	err := bldr.Build()
	if err != nil {
		return err
	}
	return nil
}

//nolint:gocognit,unparam // acceptable
func (pgb *standardPlanGraphBuilder) handleSelect(
	pbi planbuilderinput.PlanBuilderInput,
) (primitivegraph.PrimitiveNode, primitivegraph.PrimitiveNode, error) {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetSelect()
	if !ok {
		return nil, nil, fmt.Errorf("could not cast statement of type '%T' to required Select", pbi.GetStatement())
	}
	if !handlerCtx.GetRuntimeContext().TestWithoutAPICalls { //nolint:nestif // acceptable
		primitiveGenerator := pgb.rootPrimitiveGenerator
		err := primitiveGenerator.AnalyzeStatement(pbi)
		if err != nil {
			logging.GetLogger().Infoln(fmt.Sprintf("select statement analysis error = '%s'", err.Error()))
			return nil, nil, err
		}
		builder := primitiveGenerator.GetPrimitiveComposer().GetBuilder()
		_, isNativeSelect := builder.(*primitivebuilder.NativeSelect)
		_, isRawNativeSelect := builder.(*primitivebuilder.RawNativeSelect)
		_, isRawNativeExec := builder.(*primitivebuilder.RawNativeExec)
		isLocallyExecutable := !isNativeSelect && !isRawNativeSelect && !isRawNativeExec
		// check tables only if not native
		if isLocallyExecutable {
			for _, val := range primitiveGenerator.GetPrimitiveComposer().GetTables() {
				isLocallyExecutable = isLocallyExecutable && val.IsLocallyExecutable()
			}
		}
		if isLocallyExecutable {
			var colz []map[string]interface{}
			for idx := range primitiveGenerator.GetPrimitiveComposer().GetValOnlyColKeys() {
				col := primitiveGenerator.GetPrimitiveComposer().GetValOnlyCol(idx)
				colz = append(colz, col)
			}
			pr, prErr := primitivebuilder.NewLocalSelectExecutor(handlerCtx, node, util.DefaultRowSort, colz)
			if prErr != nil {
				return nil, nil, prErr
			}
			rv := pgb.planGraphHolder.CreatePrimitiveNode(pr)
			return rv, rv, nil
		}
		if primitiveGenerator.GetPrimitiveComposer().GetBuilder() == nil {
			return nil, nil, fmt.Errorf("builder not created for select, cannot proceed")
		}
		err = builder.Build()
		if err != nil {
			return nil, nil, err
		}
		root := builder.GetRoot()
		tail := builder.GetTail()
		return root, tail, nil
	}
	pr := primitive.NewLocalPrimitive(nil)
	rv := pgb.planGraphHolder.CreatePrimitiveNode(pr)
	return rv, rv, nil
}

//nolint:unparam // acceptable
func (pgb *standardPlanGraphBuilder) handleUnion(
	pbi planbuilderinput.PlanBuilderInput) (primitivegraph.PrimitiveNode, primitivegraph.PrimitiveNode, error) {
	// handlerCtx := pbi.GetHandlerCtx()
	_, ok := pbi.GetUnion()
	if !ok {
		return nil, nil, fmt.Errorf("could not cast node of type '%T' to required Delete", pbi.GetStatement())
	}
	primitiveGenerator := pgb.rootPrimitiveGenerator
	err := primitiveGenerator.AnalyzeStatement(pbi)
	if err != nil {
		logging.GetLogger().Infoln(fmt.Sprintf("select statement analysis error = '%s'", err.Error()))
		return nil, nil, err
	}
	isLocallyExecutable := true
	for _, val := range primitiveGenerator.GetPrimitiveComposer().GetTables() {
		isLocallyExecutable = isLocallyExecutable && val.IsLocallyExecutable()
	}
	if primitiveGenerator.GetPrimitiveComposer().GetBuilder() == nil {
		return nil, nil, fmt.Errorf("builder not created for union, cannot proceed")
	}
	builder := primitiveGenerator.GetPrimitiveComposer().GetBuilder()
	err = builder.Build()
	if err != nil {
		return nil, nil, err
	}
	root := builder.GetRoot()
	tail := builder.GetTail()
	return root, tail, nil
}

func (pgb *standardPlanGraphBuilder) handleDelete(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetDelete()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Delete", pbi.GetStatement())
	}
	if !handlerCtx.GetRuntimeContext().TestWithoutAPICalls { //nolint:nestif // tolerable for now
		primitiveGenerator := pgb.rootPrimitiveGenerator
		err := primitiveGenerator.AnalyzeStatement(pbi)
		if err != nil {
			return err
		}
		tbl, err := primitiveGenerator.GetPrimitiveComposer().GetTable(node)
		if err != nil {
			return err
		}
		insertCtx := primitiveGenerator.GetPrimitiveComposer().GetInsertPreparedStatementCtx()
		var selectCtx drm.PreparedStatementCtx
		if len(node.SelectExprs) > 0 {
			selectCtx = primitiveGenerator.GetPrimitiveComposer().GetSelectPreparedStatementCtx()
		}
		isPhysicalTable := tbl.IsPhysicalTable()
		var bldr primitivebuilder.Builder
		if !isPhysicalTable {
			bldr = primitivebuilder.NewDelete(
				pgb.planGraphHolder,
				handlerCtx,
				insertCtx,
				selectCtx,
				node,
				tbl,
				nil,
				primitiveGenerator.GetPrimitiveComposer().IsAwait(),
			)
		} else {
			bi := builder_input.NewBuilderInput(
				pgb.planGraphHolder,
				handlerCtx,
				tbl,
			)
			tcc := pbi.GetTxnCtrlCtrs()
			bldr = primitivebuilder.NewRawNativeExec(
				pgb.planGraphHolder,
				handlerCtx,
				tcc,
				handlerCtx.GetQuery(),
				bi,
			)
		}
		err = bldr.Build()
		if err != nil {
			return err
		}
		return nil
	}
	pr := primitive.NewGenericPrimitive(nil, nil, nil, primitive_context.NewPrimitiveContext())
	pgb.planGraphHolder.CreatePrimitiveNode(pr)
	return nil
}

//nolint:gocognit // acceptable
func (pgb *standardPlanGraphBuilder) handleRegistry(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetRegistry()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Registry", pbi.GetStatement())
	}
	primitiveGenerator := pgb.rootPrimitiveGenerator
	err := primitiveGenerator.AnalyzeRegistry(pbi)
	if err != nil {
		return err
	}
	reg, err := handler.GetRegistry(handlerCtx.GetRuntimeContext())
	if err != nil {
		return err
	}
	pr := primitive.NewLocalPrimitive(
		func(_ primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			switch at := strings.ToLower(node.ActionType); at {
			case "pull":
				return pgb.handleRegistryPull(reg, node, pbi)
			case "list":
				var colz []string
				var provz map[string]anysdk.ProviderDescription
				keys := make(map[string]map[string]interface{})
				if node.ProviderId == "" {
					provz, err = reg.ListAllAvailableProviders()
					if err != nil {
						return internaldto.NewErroneousExecutorOutput(err)
					}
					colz = []string{"provider", "version"}
					var dks []string
					for k := range provz {
						dks = append(dks, k)
					}
					sort.Strings(dks)
					for i, k := range dks {
						v := provz[k]
						for _, ver := range v.Versions {
							keys[strconv.Itoa(i)] = map[string]interface{}{
								"provider": k,
								"version":  ver,
							}
						}
					}
				} else {
					provz, err = reg.ListAllProviderVersions(node.ProviderId)
					if err != nil {
						return internaldto.NewErroneousExecutorOutput(err)
					}
					colz = []string{"provider", "versions"}
					i := 0
					for k, v := range provz {
						keys[strconv.Itoa(i)] = map[string]interface{}{
							"provider": k,
							"versions": strings.Join(v.Versions, ", "),
						}
						i++
					}
				}
				return util.PrepareResultSet(
					internaldto.NewPrepareResultSetPlusRawDTO(
						nil, keys, colz, nil, nil, nil, nil,
						pbi.GetHandlerCtx().GetTypingConfig()))
			default:
				return internaldto.NewErroneousExecutorOutput(fmt.Errorf("registry action '%s' no supported", at))
			}
		},
	)
	pgb.planGraphHolder.CreatePrimitiveNode(pr)
	return nil
}

// handleRegistryPull is a helper function that pulls a provider version from the registry and persists it locally.
func (pgb *standardPlanGraphBuilder) handleRegistryPull(
	reg anysdk.RegistryAPI,
	node *sqlparser.Registry,
	pbi planbuilderinput.PlanBuilderInput,
) internaldto.ExecutorOutput {
	providerVersion := node.ProviderVersion
	var err error
	if providerVersion == "" {
		providerVersion, err = reg.GetLatestPublishedVersion(node.ProviderId)
		if err != nil {
			return internaldto.NewErroneousExecutorOutput(err)
		}
	}
	// Get all existing versions from local filesystem
	localProviders := reg.ListLocallyAvailableProviders()
	// Handle special case for google provider
	providerID := node.ProviderId
	if providerID == "google" {
		providerID = "googleapis.com"
	}
	if providerDesc, exists := localProviders[providerID]; exists {
		// Remove all existing versions
		for _, version := range providerDesc.Versions {
			if err = reg.RemoveProviderVersion(providerID, version); err != nil {
				return internaldto.NewErroneousExecutorOutput(err)
			}
		}
	}
	// Clear provider cache before pulling new version
	if err = reg.ClearProviderCache(providerID); err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	handlerCtx := pbi.GetHandlerCtx()
	_ = handlerCtx.DeleteProvider(providerID)
	// Pull and persist the requested version
	if err = reg.PullAndPersistProviderArchive(node.ProviderId, providerVersion); err != nil {
		return internaldto.NewErroneousExecutorOutput(err)
	}
	newProvider, newProviderErr := reg.LoadProviderByName(node.ProviderId, providerVersion)
	if newProviderErr != nil {
		return internaldto.NewErroneousExecutorOutput(fmt.Errorf("Error parsing new provider: %w", newProviderErr))
	}
	minSupprotedStackqlVersion := newProvider.GetMinStackQLVersion()
	currentStackqlVersion := handlerCtx.GetStacqklSemver()
	messages := []string{
		fmt.Sprintf(
			"%s provider, version '%s' successfully installed",
			node.ProviderId, providerVersion),
	}
	if minSupprotedStackqlVersion != "" {
		isUpToDate := semver.Compare(currentStackqlVersion, minSupprotedStackqlVersion) >= 0
		if !isUpToDate {
			messages = append(messages, fmt.Sprintf(
				//nolint:lll // acceptable
				"warning: installed %s provider version '%s' requires minimum stackql version '%s', current stackql version is '%s'",
				node.ProviderId, providerVersion, minSupprotedStackqlVersion, currentStackqlVersion))
		}
	}
	return util.PrepareResultSet(
		internaldto.NewPrepareResultSetPlusRawDTO(
			nil, nil, nil, nil, nil,
			internaldto.NewBackendMessages(messages),
			nil,
			pbi.GetHandlerCtx().GetTypingConfig()))
}

func (pgb *standardPlanGraphBuilder) handlePurge(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetPurge()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Purge", pbi.GetStatement())
	}
	pr := primitive.NewLocalPrimitive(
		//nolint:revive // acceptable for now
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			if node.IsGlobal {
				err := handlerCtx.GetGarbageCollector().Purge()
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
				return util.PrepareResultSet(
					internaldto.NewPrepareResultSetPlusRawDTO(
						nil,
						map[string]map[string]interface{}{"0": {"message": "purge 'GLOBAL' completed"}},
						[]string{"message"},
						nil,
						nil,
						internaldto.NewBackendMessages(
							[]string{"Global PURGE successfully completed"},
						),
						nil,
						pbi.GetHandlerCtx().GetTypingConfig(),
					),
				)
			}
			targetStr := strings.ToLower(node.Target.GetRawVal())
			switch targetStr {
			case "cache":
				err := handlerCtx.GetGarbageCollector().PurgeCache()
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
			case "conservative":
				err := handlerCtx.GetGarbageCollector().Collect()
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
			case "control_tables":
				err := handlerCtx.GetGarbageCollector().PurgeControlTables()
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
			case "ephemeral":
				err := handlerCtx.GetGarbageCollector().PurgeEphemeral()
				if err != nil {
					return internaldto.NewErroneousExecutorOutput(err)
				}
			default:
				return internaldto.NewErroneousExecutorOutput(fmt.Errorf("purge target '%s' not supported", targetStr))
			}
			// This happens in all cases, provided the ourge is successful.
			handlerCtx.GetLRUCache().Clear()
			purgeMsg := fmt.Sprintf("PURGE of type '%s' successfully completed", targetStr)
			return util.PrepareResultSet(
				internaldto.NewPrepareResultSetPlusRawDTO(
					nil,
					map[string]map[string]interface{}{"0": {"message": purgeMsg}},
					[]string{"message"},
					nil,
					nil,
					nil,
					// &internaldto.BackendMessages{
					// 	WorkingMessages: []string{fmt.Sprintf("PURGE of type '%s' successfully completed", targetStr)}},
					nil,
					pbi.GetHandlerCtx().GetTypingConfig(),
				),
			)
		},
	)
	pgb.planGraphHolder.CreatePrimitiveNode(pr)

	return nil
}

func (pgb *standardPlanGraphBuilder) handleNativeQuery(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetNativeQuery()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Purge", pbi.GetStatement())
	}
	rns := primitivebuilder.NewRawNativeSelect(pgb.planGraphHolder, handlerCtx, pbi.GetTxnCtrlCtrs(), node.QueryString)

	err := rns.Build()

	if err != nil {
		return err
	}

	return nil
}

//nolint:gocognit,funlen // acceptable complexity
func (pgb *standardPlanGraphBuilder) handleInsert(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	annotatedAST := pbi.GetAnnotatedAST()
	node, ok := pbi.GetInsert()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Insert", pbi.GetStatement())
	}
	if !handlerCtx.GetRuntimeContext().TestWithoutAPICalls { //nolint:nestif // acceptable complexity
		primitiveGenerator := primitivegenerator.NewRootPrimitiveGenerator(node, handlerCtx, pgb.planGraphHolder)
		err := primitiveGenerator.AnalyzeInsert(pbi)
		if err != nil {
			return err
		}
		insertValOnlyRows, nonValCols, err := parserutil.ExtractInsertValColumns(node)
		if err != nil {
			return err
		}
		// selectPrimitive here forms the insert data
		var selectPrimitive primitive.IPrimitive
		var selectPrimitiveNode primitivegraph.PrimitiveNode
		if nonValCols > 0 {
			switch rowsNode := node.Rows.(type) {
			case *sqlparser.Select:
				selPbi, selErr := planbuilderinput.NewPlanBuilderInput(
					pbi.GetAnnotatedAST(),
					pbi.GetHandlerCtx(),
					rowsNode,
					pbi.GetTableExprs(),
					pbi.GetAssignedAliasedColumns(),
					pbi.GetAliasedTables(),
					pbi.GetColRefs(),
					pbi.GetPlaceholderParams(),
					pbi.GetTxnCtrlCtrs())
				if selErr != nil {
					return selErr
				}
				sr := routeanalysis.NewSelectRoutePass(rowsNode, selPbi, nil)
				err = sr.RoutePass()
				if err != nil {
					return err
				}
				_, selectPrimitiveNode, err = pgb.handleSelect(selPbi)
				if err != nil {
					return err
				}
				selectIndirect, selectIndirectExists := annotatedAST.GetSelectIndirect(rowsNode)
				if !selectIndirectExists {
					return fmt.Errorf("could not obtain select statement in insert analysis")
				}
				annotatedAST.SetInsertRowsIndirect(node, selectIndirect)
			default:
				return fmt.Errorf("insert with rows of type '%T' not currently supported", rowsNode)
			}
		} else {
			selectPrimitive, err = primitivebuilder.NewInsertableValsPrimitive(handlerCtx, insertValOnlyRows)
			if err != nil {
				return err
			}

			sn := pgb.planGraphHolder.CreatePrimitiveNode(selectPrimitive)
			selectPrimitiveNode = sn
		}
		if selectPrimitiveNode == nil {
			return fmt.Errorf("nil selection for insert -- cannot work")
		}
		tbl, err := primitiveGenerator.GetPrimitiveComposer().GetTable(node)
		if err != nil {
			return err
		}
		bldrInput := builder_input.NewBuilderInput(
			pgb.planGraphHolder,
			handlerCtx,
			tbl,
		)
		bldrInput.SetDependencyNode(selectPrimitiveNode)
		bldrInput.SetCommentDirectives(primitiveGenerator.GetPrimitiveComposer().GetCommentDirectives())
		isAwait := primitiveGenerator.GetPrimitiveComposer().IsAwait()
		bldrInput.SetIsAwait(isAwait)
		bldrInput.SetParserNode(node)
		bldrInput.SetAnnotatedAST(pbi.GetAnnotatedAST())
		bldrInput.SetTxnCtrlCtrs(pbi.GetTxnCtrlCtrs())
		isPhysicalTable := tbl.IsPhysicalTable()
		if isPhysicalTable {
			bldrInput.SetIsTargetPhysicalTable(true)
		}
		return pgb.handleMutationOperation(
			handlerCtx,
			pbi,
			primitiveGenerator,
			node,
			tbl,
			selectPrimitiveNode,
			bldrInput,
			isAwait,
		)
	}
	pr := primitive.NewGenericPrimitive(nil, nil, nil, primitive_context.NewPrimitiveContext())
	pgb.planGraphHolder.CreatePrimitiveNode(pr)
	return nil
}

func (pgb *standardPlanGraphBuilder) handleMutationOperation(
	handlerCtx handler.HandlerContext,
	pbi planbuilderinput.PlanBuilderInput,
	primitiveGenerator primitivegenerator.PrimitiveGenerator,
	node sqlparser.SQLNode,
	tbl tablemetadata.ExtendedTableMetadata,
	selectPrimitiveNode primitivegraph.PrimitiveNode,
	bldrInput builder_input.BuilderInput,
	isAwait bool,
) error {
	var returningExpressions sqlparser.SelectExprs
	var inputAction string
	var bldr primitivebuilder.Builder
	switch n := node.(type) {
	case *sqlparser.Insert:
		returningExpressions = n.SelectExprs
		inputAction = n.Action
	case *sqlparser.Update:
		returningExpressions = n.SelectExprs
		inputAction = n.Action
	default:
		return fmt.Errorf("unsupported mutation operation of type '%T'", node)
	}
	//nolint:nestif // acceptable complexity
	if len(returningExpressions) > 0 {
		// Two cases:
		//   1. Synchronous.  Equivalent to select.
		//   2. Asynchronous.  Whole other story.
		tableMeta, tableMetaExists := bldrInput.GetTableMetadata()
		if !tableMetaExists {
			return fmt.Errorf("could not obtain table metadata for node '%s'", inputAction)
		}
		rc, rcErr := tableinsertioncontainer.NewTableInsertionContainer(
			tableMeta,
			handlerCtx.GetSQLEngine(),
			handlerCtx.GetTxnCounterMgr(),
		)
		if rcErr != nil {
			return rcErr
		}
		bldrInput.SetTableInsertionContainer(rc)
		bldrInput.SetIsReturning(true)
		if !isAwait {
			bldr = primitivebuilder.NewSingleAcquireAndSelect(
				bldrInput,
				primitiveGenerator.GetPrimitiveComposer().GetInsertPreparedStatementCtx(),
				primitiveGenerator.GetPrimitiveComposer().GetSelectPreparedStatementCtx(),
				nil,
			)
		} else {
			bldrInput.SetIsAwait(true)
			bldrInput.SetIsReturning(true)
			bldrInput.SetInsertCtx(primitiveGenerator.GetPrimitiveComposer().GetInsertPreparedStatementCtx())
			lhsBldr := primitivebuilder.NewInsertOrUpdate(
				bldrInput,
			)
			newBldrInput := builder_input.NewBuilderInput(
				pgb.planGraphHolder,
				handlerCtx,
				tbl,
			)
			newBldrInput.SetParserNode(node)
			newBldrInput.SetAnnotatedAST(pbi.GetAnnotatedAST())
			newBldrInput.SetTxnCtrlCtrs(pbi.GetTxnCtrlCtrs())
			newBldrInput.SetTableInsertionContainer(rc)
			newBldrInput.SetDependencyNode(selectPrimitiveNode)
			newBldrInput.SetIsAwait(isAwait)
			rhsBldr := primitivebuilder.NewSingleSelect(
				pgb.planGraphHolder, handlerCtx, primitiveGenerator.GetPrimitiveComposer().GetSelectPreparedStatementCtx(),
				[]tableinsertioncontainer.TableInsertionContainer{rc},
				nil,
				streaming.NewNopMapStream(),
			)
			bldr = primitivebuilder.NewDependencySubDAGBuilder(
				pgb.planGraphHolder,
				[]primitivebuilder.Builder{lhsBldr},
				rhsBldr,
			)
		}
	} else {
		bldr = primitivebuilder.NewInsertOrUpdate(
			bldrInput,
		)
	}
	err := bldr.Build()
	if err != nil {
		return err
	}
	return nil
}

func (pgb *standardPlanGraphBuilder) handleUpdate(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetUpdate()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Insert", pbi.GetStatement())
	}
	if !handlerCtx.GetRuntimeContext().TestWithoutAPICalls { //nolint:nestif // acceptable complexity
		primitiveGenerator := pgb.rootPrimitiveGenerator
		err := primitiveGenerator.AnalyzeUpdate(pbi)
		if err != nil {
			return err
		}
		insertValOnlyRows, nonValCols, err := parserutil.ExtractUpdateValColumns(node)
		if err != nil {
			return err
		}
		// selectPrimitive here forms the insert data
		var selectPrimitive primitive.IPrimitive
		var selectPrimitiveNode primitivegraph.PrimitiveNode
		if len(nonValCols) > 0 {
			// TODO: support dynamic content
			return fmt.Errorf("update does not currently support dynamic content")
		}
		// TODO: elide for physical tables
		selectPrimitive, err = primitivebuilder.NewUpdateableValsPrimitive(handlerCtx, insertValOnlyRows)
		if err != nil {
			return err
		}
		sn := pgb.planGraphHolder.CreatePrimitiveNode(selectPrimitive)
		selectPrimitiveNode = sn
		if selectPrimitiveNode == nil {
			return fmt.Errorf("nil selection for insert -- cannot work")
		}
		tbl, err := primitiveGenerator.GetPrimitiveComposer().GetTable(node)
		if err != nil {
			return err
		}
		bldrInput := builder_input.NewBuilderInput(
			pgb.planGraphHolder,
			handlerCtx,
			tbl,
		)
		bldrInput.SetDependencyNode(selectPrimitiveNode)
		bldrInput.SetCommentDirectives(primitiveGenerator.GetPrimitiveComposer().GetCommentDirectives())
		bldrInput.SetIsAwait(primitiveGenerator.GetPrimitiveComposer().IsAwait())
		bldrInput.SetParserNode(node)
		isPhysicalTable := tbl.IsPhysicalTable()
		if isPhysicalTable {
			bldrInput.SetTxnCtrlCtrs(pbi.GetTxnCtrlCtrs())
			bldrInput.SetIsTargetPhysicalTable(true)
		}
		isAwait := primitiveGenerator.GetPrimitiveComposer().IsAwait()
		return pgb.handleMutationOperation(
			handlerCtx,
			pbi,
			primitiveGenerator,
			node,
			tbl,
			selectPrimitiveNode,
			bldrInput,
			isAwait,
		)
	}
	pr := primitive.NewGenericPrimitive(nil, nil, nil, primitive_context.NewPrimitiveContext())
	pgb.planGraphHolder.CreatePrimitiveNode(pr)
	return nil
}

func (pgb *standardPlanGraphBuilder) handleExec(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetExec()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Exec", pbi.GetStatement())
	}
	if !handlerCtx.GetRuntimeContext().TestWithoutAPICalls { //nolint:nestif // acceptable complexity
		primitiveGenerator := pgb.rootPrimitiveGenerator
		err := primitiveGenerator.AnalyzeStatement(pbi)
		if err != nil {
			return err
		}
		//
		if primitiveGenerator.IsShowResults() && primitiveGenerator.GetPrimitiveComposer().GetBuilder() != nil {
			err = primitiveGenerator.GetPrimitiveComposer().GetBuilder().Build()
			if err != nil {
				return err
			}
			return nil
		}
		tbl, err := primitiveGenerator.GetPrimitiveComposer().GetTable(node)
		if err != nil {
			return err
		}
		tcc := pbi.GetTxnCtrlCtrs()
		bldr := primitivebuilder.NewExec(
			primitiveGenerator.GetPrimitiveComposer().GetGraphHolder(),
			handlerCtx,
			node,
			tbl,
			primitiveGenerator.GetPrimitiveComposer().IsAwait(),
			primitiveGenerator.IsShowResults(),
			tcc,
		)
		err = bldr.Build()
		if err != nil {
			return err
		}
		return nil
	}
	pr := primitive.NewGenericPrimitive(nil, nil, nil, primitive_context.NewPrimitiveContext())
	pgb.planGraphHolder.CreatePrimitiveNode(pr)
	return nil
}

func (pgb *standardPlanGraphBuilder) handleShow(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetShow()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Show", pbi.GetStatement())
	}
	primitiveGenerator := pgb.rootPrimitiveGenerator
	err := primitiveGenerator.AnalyzeStatement(pbi)
	if err != nil {
		return err
	}
	nodeTypeUpper := strings.ToUpper(node.Type)
	var tbl tablemetadata.ExtendedTableMetadata
	switch nodeTypeUpper {
	case "TRANSACTION_ISOLATION_LEVEL":
		builder := primitiveGenerator.GetPrimitiveComposer().GetBuilder()
		_, isNativeSelect := builder.(*primitivebuilder.NativeSelect)
		if isNativeSelect {
			err = builder.Build()
			return err
		}
		return fmt.Errorf("improper usage of 'show transaction isolation level'")
	case "INSERT":
		tbl, err = primitiveGenerator.GetPrimitiveComposer().GetTable(node)
		if err != nil {
			return err
		}
	case "METHODS":
		//nolint:wastedassign // TODO: fix this
		tbl, err = primitiveGenerator.GetPrimitiveComposer().GetTable(node.OnTable) //nolint:ineffassign,staticcheck,lll // TODO: fix this
	}
	prov := primitiveGenerator.GetPrimitiveComposer().GetProvider()
	pr := primitive.NewMetaDataPrimitive(
		prov,
		//nolint:revive // acceptable for now
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			return primitivebuilder.NewShowInstructionExecutor(
				node,
				prov,
				tbl,
				handlerCtx,
				primitiveGenerator.GetPrimitiveComposer().GetCommentDirectives(),
				primitiveGenerator.GetPrimitiveComposer().GetTableFilter(),
			)
		})
	pgb.planGraphHolder.CreatePrimitiveNode(pr)
	return nil
}

func (pgb *standardPlanGraphBuilder) handleSleep(pbi planbuilderinput.PlanBuilderInput) error {
	// handlerCtx := pbi.GetHandlerCtx()
	_, ok := pbi.GetSleep()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Sleep", pbi.GetStatement())
	}
	primitiveGenerator := pgb.rootPrimitiveGenerator
	err := primitiveGenerator.AnalyzeStatement(pbi)
	if err != nil {
		return err
	}
	return nil
}

func (pgb *standardPlanGraphBuilder) handleUse(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetUse()
	if !ok {
		return fmt.Errorf("node type '%T' is incorrect; expected *Use", node)
	}
	primitiveGenerator := pgb.rootPrimitiveGenerator
	err := primitiveGenerator.AnalyzeStatement(pbi)
	if err != nil {
		return err
	}
	pr := primitive.NewMetaDataPrimitive(
		primitiveGenerator.GetPrimitiveComposer().GetProvider(),
		//nolint:revive // acceptable for now
		func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			handlerCtx.SetCurrentProvider(node.DBName.GetRawVal())
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, nil)
		})
	pgb.planGraphHolder.CreatePrimitiveNode(pr)
	return nil
}

//nolint:unparam // TODO: fix this
func createErroneousPlan(
	handlerCtx handler.HandlerContext,
	qPlan plan.Plan,
	rowSort func(map[string]map[string]interface{}) []string,
	err error) (plan.Plan, error) {
	instructions := primitivegraph.NewPrimitiveGraphHolder(
		handlerCtx.GetRuntimeContext().ExecutionConcurrencyLimit,
	)
	instructions.CreatePrimitiveNode(
		//nolint:revive // acceptable for now
		primitive.NewLocalPrimitive(func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
			return util.PrepareResultSet(
				internaldto.PrepareResultSetDTO{
					OutputBody:  nil,
					Msg:         nil,
					RowMap:      nil,
					ColumnOrder: nil,
					RowSort:     rowSort,
					Err:         err,
				},
			)
		},
		),
	)
	qPlan.SetInstructions(
		instructions,
	)
	return qPlan, err
}
