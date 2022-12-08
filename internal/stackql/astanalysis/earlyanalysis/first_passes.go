package earlyanalysis

import (
	"github.com/stackql/stackql/internal/stackql/astvisit"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/parse"
	"github.com/stackql/stackql/internal/stackql/planbuilderinput"
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
	Analyze(handlerCtx handler.HandlerContext, tcc internaldto.TxnControlCounters) error
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

func NewEarlyScreenerAnalyzer() (InitialPassesScreenerAnalyzer, error) {
	return &standardInitialPasses{}, nil
}

type standardInitialPasses struct {
	instructionType               InstructionType
	isCacheExemptMaterialDetected bool
	planBuilderInput              planbuilderinput.PlanBuilderInput
	result                        *sqlparser.RewriteASTResult
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

func (sp *standardInitialPasses) Analyze(handlerCtx handler.HandlerContext, tcc internaldto.TxnControlCounters) error {
	return sp.initialPasses(handlerCtx, tcc)
}

func (sp *standardInitialPasses) initialPasses(handlerCtx handler.HandlerContext, tcc internaldto.TxnControlCounters) error {
	statement, err := parse.ParseQuery(handlerCtx.GetQuery())
	if err != nil {
		return err
	}
	result, err := sqlparser.RewriteAST(statement)
	sp.result = result
	if err != nil {
		return err
	}

	// Before analysing AST, see if we can pass stright to SQL backend
	opType, ok := handlerCtx.GetDBMSInternalRouter().CanRoute(result.AST)
	if ok {
		sp.instructionType = InternallyRoutableInstruction
		logging.GetLogger().Debugf("%v", opType)
		pbi, err := planbuilderinput.NewPlanBuilderInput(handlerCtx, result.AST, nil, nil, nil, nil, nil, tcc.Clone())
		if err != nil {
			return err
		}
		sp.planBuilderInput = pbi
		return nil
	}

	// First pass AST analysis; extract provider strings for auth.
	provStrSlice, isCacheExemptMaterialDetected := astvisit.ExtractProviderStringsAndDetectCacheExceptMaterial(result.AST, handlerCtx.GetSQLDialect(), handlerCtx.GetASTFormatter(), handlerCtx.GetNamespaceCollection())
	sp.isCacheExemptMaterialDetected = isCacheExemptMaterialDetected
	for _, p := range provStrSlice {
		_, err := handlerCtx.GetProvider(p)
		if err != nil {
			return err
		}
	}

	ast := result.AST

	// Second pass AST analysis; extract provider strings for auth.
	// Extracts:
	//   - parser objects representing tables.
	//   - mapping of string aliases to tables.
	tVis := astvisit.NewTableExtractAstVisitor()
	tVis.Visit(ast)

	// Third pass AST analysis.
	// Accepts slice of parser table objects
	// extracted from previous analysis.
	// Extracts:
	//   - Col Refs; mapping columnar objects to tables.
	//   - Alias Map; mapping the "TableName" objects
	//     defining aliases to table objects.
	aVis := astvisit.NewTableAliasAstVisitor(tVis.GetTables())
	aVis.Visit(ast)

	// Fourth pass AST analysis.
	// Extracts:
	//   - Columnar parameters with null values.
	//     Useful for method matching.
	//     Especially for "Insert" queries.
	tpv := astvisit.NewPlaceholderParamAstVisitor("", false)
	tpv.Visit(ast)

	pbi, err := planbuilderinput.NewPlanBuilderInput(handlerCtx, ast, tVis.GetTables(), aVis.GetAliasedColumns(), tVis.GetAliasMap(), aVis.GetColRefs(), tpv.GetParameters(), tcc.Clone())
	if err != nil {
		return err
	}

	if sel, ok := planbuilderinput.IsPGSetupQuery(pbi); ok {
		if sel != nil {
			sp.instructionType = DummiedPGInstruction
			pbi, err := planbuilderinput.NewPlanBuilderInput(handlerCtx, result.AST, nil, nil, nil, nil, nil, tcc.Clone())
			if err != nil {
				return err
			}
			sp.planBuilderInput = pbi
			return nil
		} else {
			sp.instructionType = NopInstruction
			pbi, err := planbuilderinput.NewPlanBuilderInput(handlerCtx, nil, nil, nil, nil, nil, nil, tcc.Clone())
			if err != nil {
				return err
			}
			sp.planBuilderInput = pbi
			return nil
		}
	}
	sp.instructionType = StandardInstruction
	sp.planBuilderInput = pbi
	return nil
}
