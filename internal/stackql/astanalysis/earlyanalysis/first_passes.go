package earlyanalysis

import (
	"github.com/stackql/stackql/internal/stackql/astanalysis/annotatedast"
	"github.com/stackql/stackql/internal/stackql/astanalysis/routeanalysis"
	"github.com/stackql/stackql/internal/stackql/astvisit"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/planbuilderinput"
	"github.com/stackql/stackql/internal/stackql/primitivegenerator"
	"vitess.io/vitess/go/vt/sqlparser"
)

type InstructionType int

const (
	StandardInstruction InstructionType = iota
	CachedInstruction
	InternallyRoutableInstruction
	DummiedPGInstruction
	NopInstruction
)

type Analyzer interface {
	Analyze(statement sqlparser.Statement, handlerCtx handler.HandlerContext, tcc internaldto.TxnControlCounters) error
}

type InitialPassesScreener interface {
	GetInstructionType() InstructionType
	GetPlanBuilderInput() planbuilderinput.PlanBuilderInput
	GetStatementType() sqlparser.StatementType
	IsCacheExemptMaterialDetected() bool
}

type InitialPassesScreenerAnalyzer interface {
	Analyzer
	InitialPassesScreener
}

var (
	_ InitialPassesScreenerAnalyzer = &standardInitialPasses{}
)

func NewEarlyScreenerAnalyzer(primitiveGenerator primitivegenerator.PrimitiveGenerator, parentAnnotatedAST annotatedast.AnnotatedAst, parentWhereParams parserutil.ParameterMap) (InitialPassesScreenerAnalyzer, error) {
	return &standardInitialPasses{
		primitiveGenerator: primitiveGenerator,
		parentAnnotatedAST: parentAnnotatedAST,
		parentWhereParams:  parentWhereParams,
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
}

func (sp *standardInitialPasses) GetInstructionType() InstructionType {
	return sp.instructionType
}

func (sp *standardInitialPasses) GetPlanBuilderInput() planbuilderinput.PlanBuilderInput {
	return sp.planBuilderInput
}

func (sp *standardInitialPasses) GetStatementType() sqlparser.StatementType {
	return sqlparser.ASTToStatementType(sp.result.AST)
}

func (sp *standardInitialPasses) IsCacheExemptMaterialDetected() bool {
	return sp.isCacheExemptMaterialDetected
}

func (sp *standardInitialPasses) Analyze(statement sqlparser.Statement, handlerCtx handler.HandlerContext, tcc internaldto.TxnControlCounters) error {
	return sp.initialPasses(statement, handlerCtx, tcc)
}

func (sp *standardInitialPasses) initialPasses(statement sqlparser.Statement, handlerCtx handler.HandlerContext, tcc internaldto.TxnControlCounters) error {

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
		logging.GetLogger().Debugf("%v", opType)
		pbi, err := planbuilderinput.NewPlanBuilderInput(
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
		handlerCtx.GetSQLDialect(),
		handlerCtx.GetASTFormatter(),
		handlerCtx.GetNamespaceCollection(),
		whereParams,
		tcc,
	)
	if err != nil {
		return err
	}
	err = astExpandVisitor.Analyze()
	if err != nil {
		return err
	}
	annotatedAST = astExpandVisitor.GetAnnotatedAST()

	// Second pass AST analysis; extract provider strings for auth.
	provStrSlice, isCacheExemptMaterialDetected := astvisit.ExtractProviderStringsAndDetectCacheExemptMaterial(annotatedAST, annotatedAST.GetAST(), handlerCtx.GetSQLDialect(), handlerCtx.GetASTFormatter(), handlerCtx.GetNamespaceCollection())
	sp.isCacheExemptMaterialDetected = isCacheExemptMaterialDetected
	for _, p := range provStrSlice {
		_, err := handlerCtx.GetProvider(p)
		if err != nil {
			return err
		}
	}

	// Third pass AST analysis; extract parser table objects.
	// Extracts:
	//   - parser objects representing tables.
	//   - mapping of string aliases to tables.
	tVis := astvisit.NewTableExtractAstVisitor(annotatedAST)
	tVis.Visit(ast)

	// Fourth pass AST analysis.
	// Accepts slice of parser table objects
	// extracted from previous analysis.
	// Extracts:
	//   - Col Refs; mapping columnar objects to tables.
	//   - Alias Map; mapping the "TableName" objects
	//     defining aliases to table objects.
	aVis := astvisit.NewTableAliasAstVisitor(annotatedAST, tVis.GetTables())
	aVis.Visit(ast)

	// Fifth pass AST analysis.
	// Extracts:
	//   - Columnar parameters with null values.
	//     Useful for method matching.
	//     Especially for "Insert" queries.
	tpv := astvisit.NewPlaceholderParamAstVisitor(annotatedAST, "", false)
	tpv.Visit(ast)

	pbi, err := planbuilderinput.NewPlanBuilderInput(
		annotatedAST,
		handlerCtx,
		ast,
		tVis.GetTables(),
		aVis.GetAliasedColumns(),
		tVis.GetAliasMap(),
		aVis.GetColRefs(),
		tpv.GetParameters(),
		tcc.Clone(),
	)
	if err != nil {
		return err
	}

	if sel, ok := planbuilderinput.IsPGSetupQuery(pbi); ok {
		if sel != nil {
			sp.instructionType = DummiedPGInstruction
			pbi, err := planbuilderinput.NewPlanBuilderInput(
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
		} else {
			sp.instructionType = NopInstruction
			pbi, err := planbuilderinput.NewPlanBuilderInput(
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
			if err != nil {
				return err
			}
			sp.planBuilderInput = pbi
			return nil
		}
	}
	switch node := ast.(type) {
	case *sqlparser.Select:
		routeAnalyzer := routeanalysis.NewSelectRoutePass(node, pbi, whereParams)
		err = routeAnalyzer.RoutePass()
		if err != nil {
			return err
		}
		pbi = routeAnalyzer.GetPlanBuilderInput()
	case *sqlparser.ParenSelect:
		routeAnalyzer := routeanalysis.NewSelectRoutePass(node.Select, pbi, whereParams)
		err = routeAnalyzer.RoutePass()
		if err != nil {
			return err
		}
		pbi = routeAnalyzer.GetPlanBuilderInput()
	case *sqlparser.Union:
		routeAnalyzer := routeanalysis.NewSelectRoutePass(node, pbi, whereParams)
		err = routeAnalyzer.RoutePass()
		if err != nil {
			return err
		}
		pbi = routeAnalyzer.GetPlanBuilderInput()
	}

	sp.instructionType = StandardInstruction
	sp.planBuilderInput = pbi
	return nil
}
