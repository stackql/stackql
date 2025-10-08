package earlyanalysis //nolint:cyclop // analysers are inherently complex, for now

import (
	"fmt"

	"github.com/stackql/any-sdk/pkg/constants"
	"github.com/stackql/any-sdk/pkg/logging"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/astanalysis/annotatedast"
	"github.com/stackql/stackql/internal/stackql/astanalysis/routeanalysis"
	"github.com/stackql/stackql/internal/stackql/astvisit"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/planbuilderinput"
	"github.com/stackql/stackql/internal/stackql/primitivebuilder"
	"github.com/stackql/stackql/internal/stackql/primitivegenerator"
)

type InstructionType int

const (
	StandardInstruction InstructionType = iota
	CachedInstruction
	InternallyRoutableInstruction
	DummiedPGInstruction
	NopInstruction
)

var (
	errPgOnly error = fmt.Errorf("cannot accomodate PG-only statement when backend is not matched to PG")
)

type Analyzer interface {
	Analyze(statement sqlparser.Statement, handlerCtx handler.HandlerContext, tcc internaldto.TxnControlCounters) error
}

type InitialPassesScreener interface {
	GetInstructionType() InstructionType
	GetPlanBuilderInput() planbuilderinput.PlanBuilderInput
	GetStatementType() sqlparser.StatementType
	GetStatement() sqlparser.Statement
	GetIndirectionDepth() int
	IsCacheExemptMaterialDetected() bool
	IsReadOnly() bool
}

type threeToFivePassAggregate interface {
	GetAliasMap() parserutil.TableAliasMap
	GetTables() sqlparser.TableExprs
	GetColRefs() parserutil.ColTableMap
	GetAliasedColumns() parserutil.TableExprMap
	GetParameters() parserutil.ParameterMap
}

var (
	_ threeToFivePassAggregate = &threeToFivePassAggregateImpl{}
)

func newThreeToFivePassAggregate(
	firstPassVisitor astvisit.ParserTableExtractAstVisitor,
	secondPassVisitor astvisit.ParserTableAliasPairingAstVisitor,
	thirdPassVisitor astvisit.ParserPlaceholderParamAstVisitor,
) threeToFivePassAggregate {
	return &threeToFivePassAggregateImpl{
		firstPassVisitor:  firstPassVisitor,
		secondPassVisitor: secondPassVisitor,
		thirdPassVisitor:  thirdPassVisitor,
	}
}

type threeToFivePassAggregateImpl struct {
	firstPassVisitor  astvisit.ParserTableExtractAstVisitor
	secondPassVisitor astvisit.ParserTableAliasPairingAstVisitor
	thirdPassVisitor  astvisit.ParserPlaceholderParamAstVisitor
}

func (tfa *threeToFivePassAggregateImpl) GetAliasMap() parserutil.TableAliasMap {
	return tfa.firstPassVisitor.GetAliasMap()
}

func (tfa *threeToFivePassAggregateImpl) GetTables() sqlparser.TableExprs {
	return tfa.firstPassVisitor.GetTables()
}

func (tfa *threeToFivePassAggregateImpl) GetColRefs() parserutil.ColTableMap {
	return tfa.secondPassVisitor.GetColRefs()
}

func (tfa *threeToFivePassAggregateImpl) GetAliasedColumns() parserutil.TableExprMap {
	return tfa.secondPassVisitor.GetAliasedColumns()
}

func (tfa *threeToFivePassAggregateImpl) GetParameters() parserutil.ParameterMap {
	return tfa.thirdPassVisitor.GetParameters()
}

type InitialPassesScreenerAnalyzer interface {
	Analyzer
	InitialPassesScreener
	GetIndirectCreateTail() ([]primitivebuilder.Builder, bool)
	SetIndirectCreateTail(preBuiltIndirectCollection []primitivebuilder.Builder)
}

var (
	_ InitialPassesScreenerAnalyzer = &standardInitialPasses{}
)

func NewEarlyScreenerAnalyzer(
	primitiveGenerator primitivegenerator.PrimitiveGenerator,
	parentAnnotatedAST annotatedast.AnnotatedAst,
	parentWhereParams parserutil.ParameterMap,
	indirectDepth int,
) (InitialPassesScreenerAnalyzer, error) {
	return &standardInitialPasses{
		primitiveGenerator: primitiveGenerator,
		parentAnnotatedAST: parentAnnotatedAST,
		parentWhereParams:  parentWhereParams,
		indirectionDepth:   indirectDepth,
	}, nil
}

type standardInitialPasses struct {
	instructionType               InstructionType
	isCacheExemptMaterialDetected bool
	planBuilderInput              planbuilderinput.PlanBuilderInput
	result                        *sqlparser.RewriteASTResult
	primitiveGenerator            primitivegenerator.PrimitiveGenerator
	parentAnnotatedAST            annotatedast.AnnotatedAst
	parentWhereParams             parserutil.ParameterMap
	indirectionDepth              int
	isReadOnly                    bool
	preBuiltIndirectCollection    []primitivebuilder.Builder
}

func (sp *standardInitialPasses) GetIndirectionDepth() int {
	return sp.indirectionDepth
}

func (sp *standardInitialPasses) GetIndirectCreateTail() ([]primitivebuilder.Builder, bool) {
	return sp.preBuiltIndirectCollection, sp.preBuiltIndirectCollection != nil
}

func (sp *standardInitialPasses) SetIndirectCreateTail(preBuiltIndirectCollection []primitivebuilder.Builder) {
	sp.preBuiltIndirectCollection = preBuiltIndirectCollection
}

func (sp *standardInitialPasses) GetInstructionType() InstructionType {
	return sp.instructionType
}

func (sp *standardInitialPasses) IsReadOnly() bool {
	return sp.isReadOnly
}

func (sp *standardInitialPasses) GetPlanBuilderInput() planbuilderinput.PlanBuilderInput {
	return sp.planBuilderInput
}

func (sp *standardInitialPasses) GetStatementType() sqlparser.StatementType {
	return sqlparser.ASTToStatementType(sp.result.AST)
}

func (sp *standardInitialPasses) GetStatement() sqlparser.Statement {
	return sp.result.AST
}

func (sp *standardInitialPasses) IsCacheExemptMaterialDetected() bool {
	return sp.isCacheExemptMaterialDetected
}

func (sp *standardInitialPasses) Analyze(
	statement sqlparser.Statement,
	handlerCtx handler.HandlerContext,
	tcc internaldto.TxnControlCounters,
) error {
	return sp.initialPasses(statement, handlerCtx, tcc)
}

//nolint:unparam // future proofing
func thirdToFifthPasses(
	ast sqlparser.SQLNode, annotatedAST annotatedast.AnnotatedAst) (threeToFivePassAggregate, error) {
	// Third pass AST analysis; extract parser table objects.
	// Extracts:
	//   - parser objects representing tables.
	//   - mapping of string aliases to tables.
	tVis := astvisit.NewTableExtractAstVisitor(annotatedAST)
	tVis.Visit(ast) //nolint:errcheck // TODO: fix this

	// Fourth pass AST analysis.
	// Accepts slice of parser table objects
	// extracted from previous analysis.
	// Extracts:
	//   - Col Refs; mapping columnar objects to tables.
	//   - Alias Map; mapping the "TableName" objects
	//     defining aliases to table objects.
	aVis := astvisit.NewTableAliasAstVisitor(annotatedAST, tVis.GetTables())
	aVis.Visit(ast) //nolint:errcheck // TODO: fix this

	// Fifth pass AST analysis.
	// Extracts:
	//   - Columnar parameters with null values.
	//     Useful for method matching.
	//     Especially for "Insert" queries.
	tpv := astvisit.NewPlaceholderParamAstVisitor(annotatedAST, "", false)
	tpv.Visit(ast) //nolint:errcheck // TODO: fix this
	return newThreeToFivePassAggregate(tVis, aVis, tpv), nil
}

//nolint:funlen,gocognit,gocyclo,cyclop // this is a large function abstracting plenty
func (sp *standardInitialPasses) initialPasses(
	statement sqlparser.Statement,
	handlerCtx handler.HandlerContext,
	tcc internaldto.TxnControlCounters,
) error {
	result, err := sqlparser.RewriteAST(statement)
	sp.result = result
	if err != nil {
		return err
	}
	ast := result.AST
	annotatedAST, err := annotatedast.NewAnnotatedAst(sp.parentAnnotatedAST, ast)
	if err != nil {
		return err
	}

	// Before analysing AST, see if we can pass straight to SQL backend
	opType, ok := handlerCtx.GetDBMSInternalRouter().CanRoute(ast)
	if ok {
		sp.instructionType = InternallyRoutableInstruction
		sp.isReadOnly = true
		logging.GetLogger().Debugf("%v", opType)
		subjectAST := result.AST
		//nolint:gocritic // prefer switch
		switch node := subjectAST.(type) {
		case *sqlparser.DDL:
			subjectAST = node.SelectStatement
		}
		pbi, pbiErr := planbuilderinput.NewPlanBuilderInput(
			annotatedAST,
			handlerCtx,
			subjectAST,
			nil,
			nil,
			nil,
			nil,
			nil,
			tcc.Clone(),
		)
		if pbiErr != nil {
			return pbiErr
		}
		sp.planBuilderInput = pbi
		return nil
	}

	var whereParams parserutil.ParameterMap

	// Where clause paramter extract from top down does not require a deep pass
	switch node := ast.(type) {
	case *sqlparser.Select:
		whereParams = astvisit.ExtractParamsFromWhereClause(annotatedAST, node.Where)
	case *sqlparser.Delete:
		whereParams = astvisit.ExtractParamsFromWhereClause(annotatedAST, node.Where)
	}
	if whereParams == nil {
		whereParams = sp.parentWhereParams
	} else {
		whereParams.Merge(sp.parentWhereParams)
	}

	// First pass AST analysis; annotate and expand AST for indirects (views).
	astExpandVisitor, err := newIndirectExpandAstVisitor(
		handlerCtx,
		annotatedAST,
		sp.primitiveGenerator,
		handlerCtx.GetSQLSystem(),
		nil, // minimal formatting prior to view storage
		handlerCtx.GetNamespaceCollection(),
		whereParams,
		tcc,
		sp.GetIndirectionDepth(),
	)
	if err != nil {
		return err
	}
	err = astExpandVisitor.Analyze()
	if err != nil {
		return err
	}
	// TODO: make this iterative
	bldr, createBldrExists := astExpandVisitor.GetCreateBuilder()
	if createBldrExists {
		sp.SetIndirectCreateTail(bldr)
	}
	sp.isReadOnly = astExpandVisitor.IsReadOnly()
	annotatedAST = astExpandVisitor.GetAnnotatedAST()

	// Second pass AST analysis; extract provider strings for auth.
	provStrSlice, isCacheExemptMaterialDetected := astvisit.ExtractProviderStringsAndDetectCacheExemptMaterial(
		annotatedAST,
		annotatedAST.GetAST(),
		handlerCtx.GetSQLSystem(),
		handlerCtx.GetASTFormatter(),
		handlerCtx.GetNamespaceCollection(),
	)
	sp.isCacheExemptMaterialDetected = isCacheExemptMaterialDetected
	for _, p := range provStrSlice {
		_, isSQLDataSource := handlerCtx.GetSQLDataSource(p)
		if isSQLDataSource {
			continue
		}
		_, err = handlerCtx.GetProvider(p)
		if err != nil {
			return err
		}
	}

	// Third to fifth pass AST analysis; extract parser table objects, col refs, and parameters.
	threeToFiveAgg, err := thirdToFifthPasses(ast, annotatedAST)
	if err != nil {
		return err
	}
	parameters := threeToFiveAgg.GetParameters()

	pbi, err := planbuilderinput.NewPlanBuilderInput(
		annotatedAST,
		handlerCtx,
		ast,
		threeToFiveAgg.GetTables(),
		threeToFiveAgg.GetAliasedColumns(),
		threeToFiveAgg.GetAliasMap(),
		threeToFiveAgg.GetColRefs(),
		parameters,
		tcc.Clone(),
	)
	if err != nil {
		return err
	}
	pbi.SetReadOnly(astExpandVisitor.IsReadOnly())
	isCreateMAterializedView := parserutil.IsCreateMaterializedView(ast)
	pbi.SetCreateMaterializedView(isCreateMAterializedView)

	sel, selOk := planbuilderinput.IsPGSetupQuery(pbi)
	if selOk {
		sp.isReadOnly = true
		if sel != nil {
			sp.instructionType = DummiedPGInstruction
			pbi, err := planbuilderinput.NewPlanBuilderInput( //nolint:govet // defer analyser uplifts
				annotatedAST,
				handlerCtx,
				result.AST,
				nil,
				nil,
				nil,
				nil,
				nil,
				tcc.Clone(),
			)
			if err != nil {
				return err
			}
			sp.planBuilderInput = pbi
			return nil
		}
		sp.instructionType = NopInstruction
		otherPbi, otherPbiErr := planbuilderinput.NewPlanBuilderInput(
			annotatedAST,
			handlerCtx,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			tcc.Clone(),
		)
		if otherPbiErr != nil {
			return otherPbiErr
		}
		sp.planBuilderInput = otherPbi
		return nil
	}
	astToAnalyse := ast
	// If the select node has already been analysed, no need to repeat.
	if isCreateMAterializedView {
		switch node := astToAnalyse.(type) {
		case *sqlparser.DDL:
			// TODO: find a better way and also add support for PG-internal materialized views.
			logging.GetLogger().Debugf("DDL: %v", node)
			pbi.SetCreateMaterializedView(true)
			sp.instructionType = StandardInstruction
			sp.planBuilderInput = pbi
			return nil
		default:
			return fmt.Errorf("expected DDL statement in analysing 'create materialized view' statement")
		}
	}
	switch node := astToAnalyse.(type) {
	case *sqlparser.Select:
		routeAnalyzer := routeanalysis.NewSelectRoutePass(node, pbi, whereParams)
		err = routeAnalyzer.RoutePass()
		if err != nil {
			return err
		}
		if routeAnalyzer.IsPGInternalOnly() {
			if sp.primitiveGenerator.GetPrimitiveComposer().GetSQLSystem().GetName() != constants.SQLDialectPostgres {
				return errPgOnly
			}
			sp.instructionType = InternallyRoutableInstruction
		}
		pbi = routeAnalyzer.GetPlanBuilderInput()
	case *sqlparser.ParenSelect:
		routeAnalyzer := routeanalysis.NewSelectRoutePass(node.Select, pbi, whereParams)
		err = routeAnalyzer.RoutePass()
		if err != nil {
			return err
		}
		if routeAnalyzer.IsPGInternalOnly() {
			if sp.primitiveGenerator.GetPrimitiveComposer().GetSQLSystem().GetName() != constants.SQLDialectPostgres {
				return errPgOnly
			}
			sp.instructionType = InternallyRoutableInstruction
		}
		pbi = routeAnalyzer.GetPlanBuilderInput()
	case *sqlparser.Union:
		lhsThreeToFiveAgg, passErr := thirdToFifthPasses(node.FirstStatement, annotatedAST)
		if passErr != nil {
			return passErr
		}
		lhsPbi, pbiErr := planbuilderinput.NewPlanBuilderInput(
			annotatedAST,
			handlerCtx,
			node,
			lhsThreeToFiveAgg.GetTables(),
			lhsThreeToFiveAgg.GetAliasedColumns(),
			lhsThreeToFiveAgg.GetAliasMap(),
			lhsThreeToFiveAgg.GetColRefs(),
			lhsThreeToFiveAgg.GetParameters(),
			tcc.Clone(),
		)
		if pbiErr != nil {
			return pbiErr
		}
		rhsPbi := lhsPbi
		for _, stmt := range node.UnionSelects {
			var nextPbi planbuilderinput.PlanBuilderInput
			rhsThreeToFiveAgg, nextPassErr := thirdToFifthPasses(stmt, annotatedAST)
			if nextPassErr != nil {
				return nextPassErr
			}
			nextPbi, pbiErr = planbuilderinput.NewPlanBuilderInput(
				annotatedAST,
				handlerCtx,
				stmt.Statement,
				rhsThreeToFiveAgg.GetTables(),
				rhsThreeToFiveAgg.GetAliasedColumns(),
				rhsThreeToFiveAgg.GetAliasMap(),
				rhsThreeToFiveAgg.GetColRefs(),
				rhsThreeToFiveAgg.GetParameters(),
				tcc.CloneAndIncrementInsertID(),
			)
			if pbiErr != nil {
				return pbiErr
			}
			rhsPbi.WithNext(nextPbi)
			rhsPbi = nextPbi
		}
		routeAnalyzer := routeanalysis.NewSelectRoutePass(node, lhsPbi, whereParams)
		err = routeAnalyzer.RoutePass()
		if err != nil {
			return err
		}
		if routeAnalyzer.IsPGInternalOnly() {
			if sp.primitiveGenerator.GetPrimitiveComposer().GetSQLSystem().GetName() != constants.SQLDialectPostgres {
				return errPgOnly
			}
			sp.instructionType = InternallyRoutableInstruction
		}
		pbi = routeAnalyzer.GetPlanBuilderInput()
	}
	pbi.SetCreateMaterializedView(isCreateMAterializedView)

	sp.instructionType = StandardInstruction
	sp.planBuilderInput = pbi
	return nil
}
