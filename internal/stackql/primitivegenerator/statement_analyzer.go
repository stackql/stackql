package primitivegenerator

import (
	"fmt"
	"strings"

	"github.com/stackql/stackql/internal/stackql/astindirect"
	"github.com/stackql/stackql/internal/stackql/astvisit"
	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/builder_input"
	"github.com/stackql/stackql/internal/stackql/intrinsic"
	"github.com/stackql/stackql/internal/stackql/iqlerror"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/planbuilderinput"
	"github.com/stackql/stackql/internal/stackql/primitivebuilder"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
	"github.com/stackql/stackql/internal/stackql/util"
	"github.com/stackql/stackql/pkg/astformat"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

//nolint:funlen // this is unavoidable
func (pb *standardPrimitiveGenerator) AnalyzeStatement(
	pbi planbuilderinput.PlanBuilderInput,
) error {
	var err error
	statement := pbi.GetStatement()
	switch stmt := statement.(type) {
	case *sqlparser.Auth:
		return nil
	case *sqlparser.AuthRevoke:
		return nil
	case *sqlparser.Begin:
		return iqlerror.GetStatementNotSupportedError("TRANSACTION: BEGIN")
	case *sqlparser.Commit:
		return iqlerror.GetStatementNotSupportedError("TRANSACTION: COMMIT")
	case *sqlparser.DBDDL:
		return iqlerror.GetStatementNotSupportedError(fmt.Sprintf("unsupported: Database DDL %v", sqlparser.String(stmt)))
	case *sqlparser.DDL:
		return iqlerror.GetStatementNotSupportedError("DDL")
	case *sqlparser.Delete:
		return pb.analyzeDelete(pbi)
	case *sqlparser.DescribeTable:
		return pb.analyzeDescribe(pbi)
	case *sqlparser.Explain:
		return fmt.Errorf("direct access to %s analysis not supported", "EXPLAIN")
	case *sqlparser.Insert:
		return pb.AnalyzeInsert(pbi)
	case *sqlparser.OtherRead, *sqlparser.OtherAdmin:
		return iqlerror.GetStatementNotSupportedError("OTHER")
	case *sqlparser.Registry:
		return pb.AnalyzeRegistry(pbi)
	case *sqlparser.Rollback:
		return iqlerror.GetStatementNotSupportedError("TRANSACTION: ROLLBACK")
	case *sqlparser.Savepoint:
		return iqlerror.GetStatementNotSupportedError("TRANSACTION: SAVEPOINT")
	case *sqlparser.Select:
		return pb.analyzeSelect(pbi)
	case *sqlparser.Set:
		return pb.analyzeSet(pbi)
	case *sqlparser.SetTransaction:
		return iqlerror.GetStatementNotSupportedError("SET TRANSACTION")
	case *sqlparser.Show:
		return pb.analyzeShow(pbi)
	case *sqlparser.Sleep:
		return nil
	case *sqlparser.SRollback:
		return iqlerror.GetStatementNotSupportedError("TRANSACTION: SROLLBACK")
	case *sqlparser.Release:
		return iqlerror.GetStatementNotSupportedError("TRANSACTION: RELEASE")
	case *sqlparser.Union:
		return pb.analyzeUnion(pbi)
	case *sqlparser.Update:
		return iqlerror.GetStatementNotSupportedError("UPDATE")
	case *sqlparser.Use:
		return pb.analyzeUse(pbi)
	}
	return err
}

func (pb *standardPrimitiveGenerator) analyzeUse(
	pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetUse()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Use", pbi.GetStatement())
	}
	prov, pErr := handlerCtx.GetProvider(node.DBName.GetRawVal())
	if pErr != nil {
		return pErr
	}
	pb.PrimitiveComposer.SetProvider(prov)
	return nil
}

func (pb *standardPrimitiveGenerator) analyzeSet(
	pbi planbuilderinput.PlanBuilderInput) error {
	_, ok := pbi.GetSet()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Set", pbi.GetStatement())
	}
	return nil
}

//nolint:govet,funlen // this is a beast
func (pb *standardPrimitiveGenerator) analyzeUnion(
	pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetUnion()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Union", pbi.GetStatement())
	}
	unionQuery := astvisit.GenerateUnionTemplateQuery(
		pbi.GetAnnotatedAST(),
		node,
		handlerCtx.GetSQLSystem(),
		handlerCtx.GetASTFormatter(),
		handlerCtx.GetNamespaceCollection())
	i := 0
	leaf, err := pb.PrimitiveComposer.GetSymTab().NewLeaf(i)
	if err != nil {
		return err
	}
	pChild := pb.AddChildPrimitiveGenerator(node.FirstStatement, leaf)
	counters := pbi.GetTxnCtrlCtrs()
	sPbi, err := planbuilderinput.NewPlanBuilderInput(
		pbi.GetAnnotatedAST(),
		handlerCtx,
		node.FirstStatement,
		nil, nil, nil, nil, nil, counters)
	sPbi.SetIsTccSetAheadOfTime(true)
	if err != nil {
		return err
	}
	err = pChild.AnalyzeSelectStatement(sPbi)
	if err != nil {
		return err
	}
	pChild.SetElideRead(true)
	var selectStatementContexts []drm.PreparedStatementCtx

	ctx := pChild.GetPrimitiveComposer().GetSelectPreparedStatementCtx()
	selectStatementContexts = append(selectStatementContexts, ctx)

	unionNonControlColumns := pChild.GetPrimitiveComposer().GetSelectPreparedStatementCtx().GetNonControlColumns()
	unionSelectCtx := drm.NewQueryOnlyPreparedStatementCtx(unionQuery, unionNonControlColumns)

	ctrClone := counters.Clone()

	pb.prepStmtOffset = ctx.GetCtrlColumnRepeats()

	currentPbi := pbi
	var currentPbiExists bool

	for _, rhsStmt := range node.UnionSelects {
		i++
		leaf, err := pb.PrimitiveComposer.GetSymTab().NewLeaf(i)
		if err != nil {
			return err
		}
		currentPbi, currentPbiExists = currentPbi.Next()
		if !currentPbiExists {
			return fmt.Errorf("no more PBIs for union selects in analysis of union statement")
		}
		pChild := pb.AddChildPrimitiveGenerator(rhsStmt.Statement, leaf).WithPrepStmtOffset(pb.prepStmtOffset)
		pChild.SetElideRead(true)
		ctrClone = ctrClone.CloneAndIncrementInsertID()

		if err != nil {
			return err
		}
		currentPbi.SetIsTccSetAheadOfTime(true)
		currentPbi.SetPrepStmtOffset(pb.prepStmtOffset)
		err = pChild.AnalyzeSelectStatement(currentPbi)
		if err != nil {
			return err
		}
		ctx := pChild.GetPrimitiveComposer().GetSelectPreparedStatementCtx()
		ctx.SetKind(rhsStmt.Type)
		// ctx.SetGCCtrlCtrs(ctrClone)
		selectStatementContexts = append(selectStatementContexts, ctx)
		pb.prepStmtOffset += ctx.GetCtrlColumnRepeats()
		// unionSelectCtx
	}
	unionSelectCtx.SetIndirectContexts(selectStatementContexts)

	bldr := primitivebuilder.NewUnion(
		pb.PrimitiveComposer.GetGraphHolder(),
		handlerCtx,
		unionSelectCtx,
	)
	pb.PrimitiveComposer.SetBuilder(bldr)
	pb.PrimitiveComposer.SetSelectPreparedStatementCtx(unionSelectCtx)
	if pb.PrimitiveComposer.IsIndirect() {
		retrievedBldr := pb.PrimitiveComposer.GetBuilder()
		pb.SetIndirectCreateTailBuilder([]primitivebuilder.Builder{retrievedBldr})
	}

	return nil
}

func (pb *standardPrimitiveGenerator) AnalyzeSelectStatement(
	pbi planbuilderinput.PlanBuilderInput) error {
	node := pbi.GetStatement()
	switch node := node.(type) {
	case *sqlparser.Select:
		return pb.analyzeSelect(pbi)
	case *sqlparser.ParenSelect:
		clonedPbi := pbi.Clone()
		clonedPbi.Refocus(node.Select)
		return pb.AnalyzeSelectStatement(clonedPbi)
	case *sqlparser.Union:
		return pb.analyzeUnion(pbi)
	}
	return nil
}

func (pb *standardPrimitiveGenerator) parseComments(comments sqlparser.Comments) {
	if comments != nil {
		commentDirectives := sqlparser.ExtractCommentDirectives(comments)
		pb.PrimitiveComposer.SetCommentDirectives(commentDirectives)
	}
	if pb.hasAwaitComment(comments) {
		pb.PrimitiveComposer.SetAwait(true)
	}
}

func (pb *standardPrimitiveGenerator) hasAwaitComment(comments sqlparser.Comments) bool {
	if comments != nil {
		commentDirectives := sqlparser.ExtractCommentDirectives(comments)
		if commentDirectives.IsSet("AWAIT") {
			return true
		}
	}
	return false
}

func (pb *standardPrimitiveGenerator) persistHerarchyToBuilder(
	heirarchy tablemetadata.HeirarchyObjects,
	node sqlparser.SQLNode) {
	pb.PrimitiveComposer.SetTable(node, tablemetadata.NewExtendedTableMetadata(heirarchy,
		taxonomy.GetTableNameFromStatement(
			node, pb.PrimitiveComposer.GetASTFormatter()), taxonomy.GetAliasFromStatement(node)))
}

//nolint:funlen,gocognit // TODO: refactor

func (pb *standardPrimitiveGenerator) AnalyzeNop(
	pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	pb.PrimitiveComposer.SetBuilder(
		primitivebuilder.NewNopBuilder(
			pb.PrimitiveComposer.GetGraphHolder(),
			pb.PrimitiveComposer.GetTxnCtrlCtrs(),
			handlerCtx,
			handlerCtx.GetSQLEngine(),
			pbi.GetMessages(),
		),
	)
	return nil
}

func (pb *standardPrimitiveGenerator) AnalyzeExplain(
	pbi planbuilderinput.PlanBuilderInput,
	messages []string, // future proofing free form messages
	instructionErr error,
) error {
	handlerCtx := pbi.GetHandlerCtx()
	_ = pb.PrimitiveComposer.GetGraphHolder().Blank()
	pb.PrimitiveComposer.SetBuilder(
		primitivebuilder.NewExplainBuilder(
			pb.PrimitiveComposer.GetGraphHolder(),
			pb.PrimitiveComposer.GetTxnCtrlCtrs(),
			handlerCtx,
			handlerCtx.GetSQLEngine(),
			messages,
			instructionErr,
		),
	)
	return nil
}

// acceptsJSONNumber: JSON numbers unmarshal as float64, so an integral value
// also satisfies an integer property (issue #725).

//nolint:funlen,unparam,gocognit // TODO: refactor

func (pb *standardPrimitiveGenerator) AnalyzePGInternal(
	pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	//nolint:gocritic // prefer switch for now
	switch pbi.GetStatement().(type) {
	case *sqlparser.Explain:
		return pb.AnalyzeExplain(pbi, pbi.GetMessages(), nil)
	}
	// pass through
	if backendQueryType, ok := handlerCtx.GetDBMSInternalRouter().CanRoute(pbi.GetStatement()); ok {
		if backendQueryType == constants.BackendQuery {
			bldr := primitivebuilder.NewRawNativeSelect(
				pb.PrimitiveComposer.GetGraphHolder(), handlerCtx, pbi.GetTxnCtrlCtrs(),
				pbi.GetRawQuery())
			pb.PrimitiveComposer.SetBuilder(bldr)
			return nil
		}
		bldrInput := builder_input.NewBuilderInput(
			pb.PrimitiveComposer.GetGraphHolder(),
			handlerCtx,
			nil,
		)
		if backendQueryType == constants.BackendExec {
			bldr := primitivebuilder.NewRawNativeExec(
				pb.PrimitiveComposer.GetGraphHolder(),
				handlerCtx,
				pbi.GetTxnCtrlCtrs(),
				pbi.GetRawQuery(),
				bldrInput,
			)
			pb.PrimitiveComposer.SetBuilder(bldr)
			return nil
		}
		if backendQueryType == constants.BackendNop {
			return pb.AnalyzeNop(pbi)
		}
	}
	return fmt.Errorf("cannot execute PG internal")
}

//nolint:funlen,gocognit // TODO: refactor

//nolint:gocognit,funlen,gocyclo,cyclop // TODO: review
func (pb *standardPrimitiveGenerator) AnalyzeInsert(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	annotatedAST := pbi.GetAnnotatedAST()
	node, ok := pbi.GetInsert()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Insert", pbi.GetStatement())
	}
	isAwait := pb.hasAwaitComment(node.Comments)
	err := pb.inferHeirarchyAndPersist(handlerCtx, node, pbi.GetPlaceholderParams(), false, isAwait)
	if err != nil {
		return err
	}
	tbl, err := pb.PrimitiveComposer.GetTable(node)
	if err != nil {
		return err
	}
	pb.PrimitiveComposer.SetTable(node, tbl)
	insertValOnlyRows, nonValCols, err := parserutil.ExtractInsertValColumnsPlusPlaceHolders(node)
	if err != nil {
		return err
	}
	pb.PrimitiveComposer.SetInsertValOnlyRows(insertValOnlyRows)
	if nonValCols > 0 {
		switch rowsNode := node.Rows.(type) {
		case *sqlparser.Select:
			for k, v := range insertValOnlyRows {
				row := v
				maxKey := util.MaxMapKey(row)
				for i := 0; i < nonValCols; i++ {
					row[maxKey+i+1] = "placeholder"
				}
				insertValOnlyRows[k] = row
			}
		default:
			return fmt.Errorf("insert with rows of type '%T' not currently supported", rowsNode)
		}
	} else {
		selQuery := strings.ReplaceAll(astformat.String(node.Rows, handlerCtx.GetASTFormatter()), "from \"dual\"", "")
		internallyRoutableVisitor := astvisit.NewInternallyRoutableTypingAstVisitor(
			selQuery,
			annotatedAST,
			handlerCtx,
			nil,
			handlerCtx.GetDrmConfig(),
			handlerCtx.GetNamespaceCollection(),
		)
		vizErr := internallyRoutableVisitor.Visit(node)
		if vizErr != nil {
			return vizErr
		}
		selCtx, selCtxExists := internallyRoutableVisitor.GetSelectContext()
		if !selCtxExists {
			return fmt.Errorf("could not find select context for insert values")
		}
		selectIndirect, indirectError := astindirect.NewInsertRowsIndirect(node, selCtx)
		if indirectError != nil {
			return indirectError
		}
		annotatedAST.SetInsertRowsIndirect(node, selectIndirect)
	}
	return nil
}

//nolint:funlen,gocognit // TODO: refactor
func (pb *standardPrimitiveGenerator) AnalyzeUpdate(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetUpdate()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Update", pbi.GetStatement())
	}
	isAwait := pb.hasAwaitComment(node.Comments)
	err := pb.inferHeirarchyAndPersist(handlerCtx, node, pbi.GetPlaceholderParams(), false, isAwait)
	if err != nil {
		return err
	}
	tbl, err := pb.PrimitiveComposer.GetTable(node)
	if err != nil {
		return err
	}

	pb.PrimitiveComposer.SetTable(node, tbl)

	return nil
}

func (pb *standardPrimitiveGenerator) inferHeirarchyAndPersist(
	handlerCtx handler.HandlerContext,
	node sqlparser.SQLNode,
	parameters parserutil.ColumnKeyedDatastore,
	viewPermissive bool,
	isAwait bool,
) error {
	heirarchy, err := taxonomy.GetHeirarchyFromStatement(handlerCtx, node, parameters, viewPermissive, isAwait)
	if err != nil {
		return err
	}
	pb.persistHerarchyToBuilder(heirarchy, node)
	return err
}

//nolint:funlen,gocognit // acceptable for now
func (pb *standardPrimitiveGenerator) analyzeDelete(
	pbi planbuilderinput.PlanBuilderInput,
) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetDelete()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Delete", pbi.GetStatement())
	}
	pb.parseComments(node.Comments)
	paramMap, _ := pbi.GetAnnotatedAST().GetWhereParamMapsEntry(node.Where)
	isAwait := pb.hasAwaitComment(node.Comments)
	err := pb.inferHeirarchyAndPersist(handlerCtx, node, paramMap, false, isAwait)
	if err != nil {
		return err
	}
	return nil
}

func (pb *standardPrimitiveGenerator) AnalyzeRegistry(pbi planbuilderinput.PlanBuilderInput) error {
	_, ok := pbi.GetRegistry()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Registry", pbi.GetStatement())
	}
	return nil
}

func (pb *standardPrimitiveGenerator) analyzeShow(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetShow()
	if !ok {
		return fmt.Errorf("expected SHOW, got %T", pbi.GetStatement())
	}
	if selected, setup := planbuilderinput.IsPGSetupQuery(pbi); setup {
		if selected == nil {
			return pb.AnalyzeNop(pbi)
		}
		pb.PrimitiveComposer.SetBuilder(primitivebuilder.NewNativeSelect(
			pb.PrimitiveComposer.GetGraphHolder(), handlerCtx, selected))
		return nil
	}
	pb.parseComments(node.Comments)
	switch strings.ToUpper(node.Type) {
	case "AUTH":
		return pb.inferProviderForShow(node, handlerCtx)
	case "DEPENDENCIES":
		if node.HasOnTable() || (node.ShowTablesOpt != nil &&
			node.ShowTablesOpt.Filter != nil && node.ShowTablesOpt.Filter.Filter != nil) {
			return fmt.Errorf("SHOW DEPENDENCIES supports only an optional LIKE clause")
		}
		return nil
	case "VERSION", "CONTRIBUTORS", "PROVIDERS":
		return nil
	default:
		return fmt.Errorf("SHOW %s must be planned through omnisdk", node.Type)
	}
}

func (pb *standardPrimitiveGenerator) analyzeDescribe(pbi planbuilderinput.PlanBuilderInput) error {
	if _, handled := intrinsic.GeneratePrimitiveFunc(pbi.GetHandlerCtx(), pbi.GetStatement()); handled {
		return nil
	}
	return fmt.Errorf("cannot resolve metadata for %s", sqlparser.String(pbi.GetStatement()))
}
