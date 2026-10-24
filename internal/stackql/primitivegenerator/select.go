package primitivegenerator

import (
	"fmt"
	"strings"

	"github.com/stackql/stackql/internal/stackql/astindirect"
	"github.com/stackql/stackql/internal/stackql/astvisit"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/planbuilderinput"
	"github.com/stackql/stackql/internal/stackql/primitivebuilder"
	"github.com/stackql/stackql/pkg/astformat"
)

//nolint:errcheck,funlen,gocognit,govet,gocyclo,cyclop // TODO: review
func (pb *standardPrimitiveGenerator) analyzeSelect(pbi planbuilderinput.PlanBuilderInput) error {
	annotatedAST := pbi.GetAnnotatedAST()

	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetSelect()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Select", pbi.GetStatement())
	}

	// Before analysing AST, see if we can pass straight to SQL backend
	_, isInternallyRoutable := handlerCtx.GetDBMSInternalRouter().CanRoute(annotatedAST.GetAST())
	if isInternallyRoutable { //nolint:nestif // acceptable
		selQuery := strings.ReplaceAll(astformat.String(node, handlerCtx.GetASTFormatter()), "from \"dual\"", "")
		v := astvisit.NewInternallyRoutableTypingAstVisitor(
			selQuery,
			annotatedAST,
			handlerCtx,
			nil,
			handlerCtx.GetDrmConfig(),
			handlerCtx.GetNamespaceCollection(),
		)
		visitErr := v.Visit(annotatedAST.GetAST())
		if visitErr != nil {
			return visitErr
		}
		selCtx, selCtxExists := v.GetSelectContext()
		if !selCtxExists {
			return fmt.Errorf("internal routing error: could not obtain select context")
		}
		pb.PrimitiveComposer.SetSelectPreparedStatementCtx(selCtx)
		selectIndirect, indirectError := astindirect.NewParserSelectIndirect(node, selCtx)
		if indirectError != nil {
			return indirectError
		}
		annotatedAST.SetSelectIndirect(node, selectIndirect)
		primitiveGenerator := pb
		clonedPbi := pbi.Clone()
		clonedPbi.SetRawQuery(selQuery)
		err := primitiveGenerator.AnalyzePGInternal(clonedPbi)
		if err != nil {
			return err
		}
		builder := primitiveGenerator.GetPrimitiveComposer().GetBuilder()
		if builder == nil {
			return fmt.Errorf("nil pg internal builder")
		}
		if pb.PrimitiveComposer.IsIndirect() {
			pb.SetIndirectCreateTailBuilder([]primitivebuilder.Builder{builder})
		}
		return nil
	}

	// TODO: get rid of this and dependent tests.
	// We need not emulate postgres for other backends at this stage.
	if sel, ok := planbuilderinput.IsPGSetupQuery(pbi); ok {
		if sel != nil {
			bldr := primitivebuilder.NewNativeSelect(pb.PrimitiveComposer.GetGraphHolder(), handlerCtx, sel)
			pb.PrimitiveComposer.SetBuilder(bldr)
			return nil
		}
		return pb.AnalyzeNop(pbi)
	}

	if table, err := parserutil.ExtractSingleTableFromTableExprs(node.From); err == nil {
		return fmt.Errorf("could not locate table '%s'", table.Name.GetRawVal())
	}
	return fmt.Errorf("SELECT must be planned through omnisdk or the SQL backend")
}
