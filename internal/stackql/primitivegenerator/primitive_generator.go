package primitivegenerator

import (
	"fmt"
	"strings"

	"github.com/stackql/stackql/internal/stackql/astindirect"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/planbuilderinput"
	"github.com/stackql/stackql/internal/stackql/primitivebuilder"
	"github.com/stackql/stackql/internal/stackql/primitivecomposer"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/symtab"
	"github.com/stackql/stackql/internal/stackql/taxonomy"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

var (
	_ PrimitiveGenerator = &standardPrimitiveGenerator{}
)

type PrimitiveGenerator interface {
	AddChildPrimitiveGenerator(ast sqlparser.SQLNode, leaf symtab.SymTab) PrimitiveGenerator
	AnalyzeInsert(pbi planbuilderinput.PlanBuilderInput) error
	AnalyzeNop(pbi planbuilderinput.PlanBuilderInput) error
	AnalyzeExplain(pbi planbuilderinput.PlanBuilderInput, messages []string, instructionErr error) error
	AnalyzeUpdate(pbi planbuilderinput.PlanBuilderInput) error
	AnalyzePGInternal(pbi planbuilderinput.PlanBuilderInput) error
	AnalyzeRegistry(pbi planbuilderinput.PlanBuilderInput) error
	AnalyzeSelectStatement(pbi planbuilderinput.PlanBuilderInput) error
	AnalyzeStatement(pbi planbuilderinput.PlanBuilderInput) error
	CreateIndirectPrimitiveGenerator(ast sqlparser.SQLNode, handlerCtx handler.HandlerContext) PrimitiveGenerator
	GetPrimitiveComposer() primitivecomposer.PrimitiveComposer
	SetIsIndirect(isIndirect bool)
	SetContainsUserManagedRelation(bool)
	WithDataFlowDependentPrimitiveGenerator(PrimitiveGenerator) PrimitiveGenerator
	WithPrepStmtOffset(offset int) PrimitiveGenerator
	GetPrepStmtOffset() int
	SetElideRead(bool)
	IsElideRead() bool
	IsShowResults() bool
	GetIndirectCreateTailBuilder() ([]primitivebuilder.Builder, bool)
	SetIndirectCreateTailBuilder(builder []primitivebuilder.Builder)
	WithAstIndirect(astindirect.Indirect) PrimitiveGenerator
}

type standardPrimitiveGenerator struct {
	Parent                    PrimitiveGenerator
	dataflowDependent         PrimitiveGenerator
	Children                  []PrimitiveGenerator
	indirects                 []PrimitiveGenerator
	prepStmtOffset            int
	PrimitiveComposer         primitivecomposer.PrimitiveComposer
	isElideRead               bool
	indirectCreateTailBuilder []primitivebuilder.Builder
}

func NewRootPrimitiveGenerator(
	ast sqlparser.SQLNode,
	handlerCtx handler.HandlerContext, graph primitivegraph.PrimitiveGraphHolder) PrimitiveGenerator {
	tblMap := make(taxonomy.TblMap)
	symTab := symtab.NewHashMapTreeSymTab()
	return &standardPrimitiveGenerator{
		PrimitiveComposer: primitivecomposer.NewPrimitiveComposer(
			nil,
			ast,
			handlerCtx.GetDrmConfig(),
			handlerCtx.GetTxnCounterMgr(),
			graph, tblMap, symTab,
			handlerCtx.GetSQLEngine(), handlerCtx.GetSQLSystem(),
			handlerCtx.GetASTFormatter()),
	}
}

func (pb *standardPrimitiveGenerator) WithAstIndirect(astIndirect astindirect.Indirect) PrimitiveGenerator {
	pb.PrimitiveComposer.WithAstIndirect(astIndirect)
	return pb
}

func (pb *standardPrimitiveGenerator) GetIndirectCreateTailBuilder() ([]primitivebuilder.Builder, bool) {
	return pb.indirectCreateTailBuilder, pb.indirectCreateTailBuilder != nil
}

func (pb *standardPrimitiveGenerator) SetIndirectCreateTailBuilder(builder []primitivebuilder.Builder) {
	pb.indirectCreateTailBuilder = builder
}

func (pb *standardPrimitiveGenerator) WithDataFlowDependentPrimitiveGenerator(
	dependent PrimitiveGenerator) PrimitiveGenerator {
	pb.dataflowDependent = dependent
	return pb
}

func (pb *standardPrimitiveGenerator) SetElideRead(isElideRead bool) {
	pb.isElideRead = isElideRead
}

func (pb *standardPrimitiveGenerator) IsElideRead() bool {
	return pb.isElideRead
}

func (pb *standardPrimitiveGenerator) GetPrepStmtOffset() int {
	return pb.prepStmtOffset
}

func (pb *standardPrimitiveGenerator) WithPrepStmtOffset(offset int) PrimitiveGenerator {
	pb.prepStmtOffset = offset
	return pb
}

func (pb *standardPrimitiveGenerator) GetPrimitiveComposer() primitivecomposer.PrimitiveComposer {
	return pb.PrimitiveComposer
}

func (pb *standardPrimitiveGenerator) SetIsIndirect(isIndirect bool) {
	pb.PrimitiveComposer.SetIsIndirect(isIndirect)
}

func (pb *standardPrimitiveGenerator) SetContainsUserManagedRelation(containsUserRelation bool) {
	pb.PrimitiveComposer.SetContainsUserRelation(containsUserRelation)
	pb.PrimitiveComposer.GetGraphHolder().SetContainsUserManagedRelation(containsUserRelation)
}

func (pb *standardPrimitiveGenerator) CreateIndirectPrimitiveGenerator(
	ast sqlparser.SQLNode,
	handlerCtx handler.HandlerContext) PrimitiveGenerator {
	rv := NewRootPrimitiveGenerator(
		ast, handlerCtx, pb.PrimitiveComposer.GetGraphHolder(),
	).WithDataFlowDependentPrimitiveGenerator(pb)
	pb.indirects = append(pb.indirects, rv)
	pb.PrimitiveComposer.GetGraphHolder().SetContainsIndirect(true)
	pb.PrimitiveComposer.AddIndirect(rv.GetPrimitiveComposer())
	rv.SetIsIndirect(true)
	return rv
}

func (pb *standardPrimitiveGenerator) AddChildPrimitiveGenerator(
	ast sqlparser.SQLNode, leaf symtab.SymTab) PrimitiveGenerator {
	tables := pb.PrimitiveComposer.GetTables()
	switch node := ast.(type) { //nolint:gocritic // acceptable
	case sqlparser.Statement:
		logging.GetLogger().Infoln(fmt.Sprintf("creating new table map for node = %v", node))
		tables = make(taxonomy.TblMap)
	}
	retVal := &standardPrimitiveGenerator{
		Parent: pb,
		PrimitiveComposer: primitivecomposer.NewPrimitiveComposer(
			pb.PrimitiveComposer,
			ast,
			pb.PrimitiveComposer.GetDRMConfig(),
			pb.PrimitiveComposer.GetTxnCounterManager(),
			pb.PrimitiveComposer.GetGraphHolder(),
			tables,
			leaf,
			pb.PrimitiveComposer.GetSQLEngine(),
			pb.PrimitiveComposer.GetSQLSystem(),
			pb.PrimitiveComposer.GetASTFormatter(),
		),
	}
	pb.Children = append(pb.Children, retVal)
	pb.PrimitiveComposer.AddChild(retVal.PrimitiveComposer)
	return retVal
}

func (pb *standardPrimitiveGenerator) inferProviderForShow(
	node *sqlparser.Show,
	handlerCtx handler.HandlerContext) error {
	nodeTypeUpperCase := strings.ToUpper(node.Type)
	switch nodeTypeUpperCase {
	case "AUTH":
		prov, err := handlerCtx.GetProvider(node.OnTable.Name.GetRawVal())
		if err != nil {
			return err
		}
		pb.PrimitiveComposer.SetProvider(prov)
	case "INSERT":
		prov, err := handlerCtx.GetProvider(node.OnTable.QualifierSecond.GetRawVal())
		if err != nil {
			return err
		}
		pb.PrimitiveComposer.SetProvider(prov)

	case "METHODS":
		prov, err := handlerCtx.GetProvider(node.OnTable.QualifierSecond.GetRawVal())
		if err != nil {
			return err
		}
		pb.PrimitiveComposer.SetProvider(prov)
	case "PROVIDERS":
		// no provider, might create some dummy object dunno
	case "VERSION", "CONTRIBUTORS", "DEPENDENCIES":
		// no provider needed
	case "RESOURCES":
		prov, err := handlerCtx.GetProvider(node.OnTable.Qualifier.GetRawVal())
		if err != nil {
			return err
		}
		pb.PrimitiveComposer.SetProvider(prov)
	case "SERVICES":
		prov, err := handlerCtx.GetProvider(node.OnTable.Name.GetRawVal())
		if err != nil {
			return err
		}
		pb.PrimitiveComposer.SetProvider(prov)
	default:
		return fmt.Errorf("unsuported node type: '%s'", node.Type)
	}
	return nil
}

func (pb *standardPrimitiveGenerator) IsShowResults() bool {
	return pb.isShowResults()
}

func (pb *standardPrimitiveGenerator) isShowResults() bool {
	//nolint:lll // acceptable
	return pb.PrimitiveComposer.GetCommentDirectives() != nil && pb.PrimitiveComposer.GetCommentDirectives().IsSet("SHOWRESULTS")
}
