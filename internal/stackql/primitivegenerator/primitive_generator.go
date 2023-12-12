package primitivegenerator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/iqlutil"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/planbuilderinput"
	"github.com/stackql/stackql/internal/stackql/primitivebuilder"
	"github.com/stackql/stackql/internal/stackql/primitivecomposer"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/relational"
	"github.com/stackql/stackql/internal/stackql/symtab"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"github.com/stackql/stackql/internal/stackql/taxonomy"

	"github.com/stackql/stackql/pkg/sqltypeutil"

	"github.com/stackql/stackql-parser/go/sqltypes"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

var (
	_ PrimitiveGenerator = &standardPrimitiveGenerator{}
)

type PrimitiveGenerator interface {
	AddChildPrimitiveGenerator(ast sqlparser.SQLNode, leaf symtab.SymTab) PrimitiveGenerator
	AnalyzeInsert(pbi planbuilderinput.PlanBuilderInput) error
	AnalyzeNop(pbi planbuilderinput.PlanBuilderInput) error
	AnalyzeUpdate(pbi planbuilderinput.PlanBuilderInput) error
	AnalyzePGInternal(pbi planbuilderinput.PlanBuilderInput) error
	AnalyzeRegistry(pbi planbuilderinput.PlanBuilderInput) error
	AnalyzeSelectStatement(pbi planbuilderinput.PlanBuilderInput) error
	AnalyzeStatement(pbi planbuilderinput.PlanBuilderInput) error
	AnalyzeUnaryExec(
		pbi planbuilderinput.PlanBuilderInput,
		handlerCtx handler.HandlerContext,
		node *sqlparser.Exec,
		selectNode *sqlparser.Select,
		cols []parserutil.ColumnHandle) (tablemetadata.ExtendedTableMetadata, error)
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
	GetIndirectCreateTailBuilder() (primitivebuilder.Builder, bool)
	SetIndirectCreateTailBuilder(builder primitivebuilder.Builder)
}

type standardPrimitiveGenerator struct {
	Parent                    PrimitiveGenerator
	dataflowDependent         PrimitiveGenerator
	Children                  []PrimitiveGenerator
	indirects                 []PrimitiveGenerator
	prepStmtOffset            int
	PrimitiveComposer         primitivecomposer.PrimitiveComposer
	isElideRead               bool
	indirectCreateTailBuilder primitivebuilder.Builder
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

func (pb *standardPrimitiveGenerator) GetIndirectCreateTailBuilder() (primitivebuilder.Builder, bool) {
	return pb.indirectCreateTailBuilder, pb.indirectCreateTailBuilder != nil
}

func (pb *standardPrimitiveGenerator) SetIndirectCreateTailBuilder(builder primitivebuilder.Builder) {
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

//nolint:unparam,revive // acceptable
func (pb *standardPrimitiveGenerator) comparisonExprToFilterFunc(
	table anysdk.ITable,
	parentNode *sqlparser.Show,
	expr *sqlparser.ComparisonExpr,
) (func(anysdk.ITable) (anysdk.ITable, error), error) {
	qualifiedName, ok := expr.Left.(*sqlparser.ColName)
	if !ok {
		return nil, fmt.Errorf("unexpected: %v", sqlparser.String(expr))
	}
	if !qualifiedName.Qualifier.IsEmpty() {
		return nil, fmt.Errorf("unsupported qualifier for column: %v", sqlparser.String(qualifiedName))
	}
	colName := qualifiedName.Name.GetRawVal()
	tableContainsKey := table.KeyExists(colName)
	if !tableContainsKey {
		return nil, fmt.Errorf("col name = '%s' not found in table name = '%s'", colName, table.GetName())
	}
	_, lhsValErr := table.GetKeyAsSqlVal(colName)
	if lhsValErr != nil {
		return nil, lhsValErr
	}
	var resolved sqltypes.Value
	var rhsStr string
	switch right := expr.Right.(type) {
	case *sqlparser.SQLVal:
		if right.Type != sqlparser.IntVal && right.Type != sqlparser.StrVal {
			return nil, fmt.Errorf("unexpected: %v", sqlparser.String(expr))
		}
		pv, err := sqlparser.NewPlanValue(right)
		if err != nil {
			return nil, err
		}
		rhsStr = string(right.Val)
		resolved, err = pv.ResolveValue(nil)
		if err != nil {
			return nil, err
		}
	case sqlparser.BoolVal:
		var resErr error
		resolved, resErr = sqltypeutil.InterfaceToSQLType(right == true) //nolint:gosimple // test later
		if resErr != nil {
			return nil, resErr
		}
	default:
		return nil, fmt.Errorf("unexpected: %v", sqlparser.String(right))
	}
	var retVal func(anysdk.ITable) (anysdk.ITable, error)
	if expr.Operator == sqlparser.LikeStr || expr.Operator == sqlparser.NotLikeStr {
		likeRegexp, err := regexp.Compile(iqlutil.TranslateLikeToRegexPattern(rhsStr))
		if err != nil {
			return nil, err
		}
		retVal = relational.ConstructLikePredicateFilter(colName, likeRegexp, expr.Operator == sqlparser.NotLikeStr)
		pb.PrimitiveComposer.SetColVisited(colName, true)
		return retVal, nil
	}
	operatorPredicate, preErr := relational.GetOperatorPredicate(expr.Operator)

	if preErr != nil {
		return nil, preErr
	}

	pb.PrimitiveComposer.SetColVisited(colName, true)
	return relational.ConstructTablePredicateFilter(colName, resolved, operatorPredicate), nil
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
