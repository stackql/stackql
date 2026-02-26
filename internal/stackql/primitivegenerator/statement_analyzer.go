package primitivegenerator

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stackql/any-sdk/pkg/constants"
	"github.com/stackql/any-sdk/pkg/dto"
	anysdk_internaldto "github.com/stackql/any-sdk/pkg/internaldto"
	"github.com/stackql/any-sdk/pkg/logging"
	"github.com/stackql/any-sdk/public/formulation"
	"github.com/stackql/stackql/internal/stackql/astindirect"
	"github.com/stackql/stackql/internal/stackql/astvisit"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/builder_input"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/iqlerror"
	"github.com/stackql/stackql/internal/stackql/iqlutil"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/planbuilderinput"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivebuilder"
	"github.com/stackql/stackql/internal/stackql/provider"
	"github.com/stackql/stackql/internal/stackql/relational"
	"github.com/stackql/stackql/internal/stackql/suffix"
	"github.com/stackql/stackql/internal/stackql/symtab"
	"github.com/stackql/stackql/internal/stackql/tableinsertioncontainer"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
	"github.com/stackql/stackql/internal/stackql/util"
	"github.com/stackql/stackql/pkg/astformat"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

var (
	synonymJSONRegexp *regexp.Regexp = regexp.MustCompile(`^application/[\S]*json[\S]*$`)
	synonymXMLRegexp  *regexp.Regexp = regexp.MustCompile(`^(?:application|text)/[\S]*xml[\S]*$`)
)

//nolint:funlen // this is unavoidable
func (pb *standardPrimitiveGenerator) AnalyzeStatement(
	pbi planbuilderinput.PlanBuilderInput,
) error {
	var err error
	statement := pbi.GetStatement()
	switch stmt := statement.(type) {
	case *sqlparser.Auth:
		return pb.analyzeAuth(pbi)
	case *sqlparser.AuthRevoke:
		return pb.analyzeAuthRevoke(pbi)
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
	case *sqlparser.Exec:
		return pb.analyzeExec(pbi)
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
		return pb.analyzeSleep(pbi)
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

func (pb *standardPrimitiveGenerator) analyzeAuth(
	pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	authNode, ok := pbi.GetAuth()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Auth", pbi.GetStatement())
	}
	provider, pErr := handlerCtx.GetProvider(authNode.Provider)
	if pErr != nil {
		return pErr
	}
	pb.PrimitiveComposer.SetProvider(provider)
	return nil
}

func (pb *standardPrimitiveGenerator) analyzeAuthRevoke(
	pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	authNode, ok := pbi.GetAuthRevoke()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required AuthRevoke", pbi.GetStatement())
	}
	authCtx, authErr := handlerCtx.GetAuthContext(authNode.Provider)
	if authErr != nil {
		return authErr
	}
	switch strings.ToLower(authCtx.Type) {
	case dto.AuthServiceAccountStr, dto.AuthInteractiveStr:
		return nil
	}
	//nolint:stylecheck // prescribed
	return fmt.Errorf(`Auth revoke for Google Failed; improper auth method: "%s" specified`, authCtx.Type)
}

func checkResource(
	handlerCtx handler.HandlerContext,
	prov provider.IProvider,
	service string,
	resource string,
) (formulation.Resource, error) {
	return prov.GetResource(service, resource, handlerCtx.GetRuntimeContext())
}

func (pb *standardPrimitiveGenerator) assembleResources(
	handlerCtx handler.HandlerContext,
	prov provider.IProvider,
	service string,
) (map[string]formulation.Resource, error) {
	rm, err := prov.GetResourcesMap(service, handlerCtx.GetRuntimeContext())
	if err != nil {
		return nil, err
	}
	return rm, err
}

func (pb *standardPrimitiveGenerator) analyzeShowFilter(node *sqlparser.Show, table formulation.ITable) error {
	showFilter := node.ShowTablesOpt.Filter
	if showFilter == nil {
		return nil
	}
	if showFilter.Like != "" {
		likeRegexp, err := regexp.Compile(iqlutil.TranslateLikeToRegexPattern(showFilter.Like))
		if err != nil {
			return fmt.Errorf("cannot compile like string '%s': %w", showFilter.Like, err)
		}
		tableFilter := pb.PrimitiveComposer.GetTableFilter()
		for _, col := range pb.PrimitiveComposer.GetLikeAbleColumns() {
			tableFilter = relational.OrTableFilters(tableFilter, relational.ConstructLikePredicateFilter(col, likeRegexp, false))
		}
		pb.PrimitiveComposer.SetTableFilter(relational.OrTableFilters(pb.PrimitiveComposer.GetTableFilter(), tableFilter))
	} else if showFilter.Filter != nil {
		tableFilter, err := pb.traverseShowFilter(table, node, showFilter.Filter)
		if err != nil {
			return err
		}
		pb.PrimitiveComposer.SetTableFilter(tableFilter)
	}
	return nil
}

func (pb *standardPrimitiveGenerator) traverseShowFilter(
	table formulation.ITable,
	node *sqlparser.Show,
	filter sqlparser.Expr,
) (func(formulation.ITable) (formulation.ITable, error), error) {
	switch filter := filter.(type) {
	case *sqlparser.ComparisonExpr:
		return pb.comparisonExprToFilterFunc(table, node, filter)
	case *sqlparser.AndExpr:
		logging.GetLogger().Infoln("complex AND expr detected")
		lhs, lhErr := pb.traverseShowFilter(table, node, filter.Left)
		rhs, rhErr := pb.traverseShowFilter(table, node, filter.Right)
		if lhErr != nil {
			return nil, lhErr
		}
		if rhErr != nil {
			return nil, rhErr
		}
		return relational.AndTableFilters(lhs, rhs), nil
	case *sqlparser.OrExpr:
		logging.GetLogger().Infoln("complex OR expr detected")
		lhs, lhErr := pb.traverseShowFilter(table, node, filter.Left)
		rhs, rhErr := pb.traverseShowFilter(table, node, filter.Right)
		if lhErr != nil {
			return nil, lhErr
		}
		if rhErr != nil {
			return nil, rhErr
		}
		return relational.OrTableFilters(lhs, rhs), nil
	case *sqlparser.FuncExpr:
		return nil, fmt.Errorf("unsupported constraint in anysdk filter: %v", sqlparser.String(filter))
	default:
		return nil, fmt.Errorf("unsupported constraint in anysdk filter: %v", sqlparser.String(filter))
	}
}

func (pb *standardPrimitiveGenerator) traverseWhereFilter(
	node sqlparser.SQLNode,
	requiredParameters,
	optionalParameters suffix.ParameterSuffixMap,
) (sqlparser.Expr, []string, error) {
	switch node := node.(type) {
	case *sqlparser.ComparisonExpr:
		exp, cn, err := pb.whereComparisonExprCopyAndReWrite(node, requiredParameters, optionalParameters)
		return exp, []string{cn}, err
	case *sqlparser.AndExpr:
		logging.GetLogger().Infoln("complex AND expr detected")
		lhs, lParams, lhErr := pb.traverseWhereFilter(node.Left, requiredParameters, optionalParameters)
		rhs, rParams, rhErr := pb.traverseWhereFilter(node.Right, requiredParameters, optionalParameters)
		if lhErr != nil {
			return nil, nil, lhErr
		}
		if rhErr != nil {
			return nil, nil, rhErr
		}
		lParams = append(lParams, rParams...)
		return &sqlparser.AndExpr{Left: lhs, Right: rhs}, lParams, nil
	case *sqlparser.OrExpr:
		logging.GetLogger().Infoln("complex OR expr detected")
		lhs, lParams, lhErr := pb.traverseWhereFilter(node.Left, requiredParameters, optionalParameters)
		rhs, rParams, rhErr := pb.traverseWhereFilter(node.Right, requiredParameters, optionalParameters)
		if lhErr != nil {
			return nil, nil, lhErr
		}
		if rhErr != nil {
			return nil, nil, rhErr
		}
		lParams = append(lParams, rParams...)
		return &sqlparser.OrExpr{Left: lhs, Right: rhs}, lParams, nil
	case *sqlparser.FuncExpr:
		return nil, nil, fmt.Errorf("unsupported constraint in anysdk filter: %v", sqlparser.String(node))
	case *sqlparser.IsExpr:
		return &sqlparser.IsExpr{
			Operator: node.Operator,
			Expr:     node.Expr,
		}, nil, nil
	default:
		return nil, nil, fmt.Errorf("unsupported constraint in anysdk filter: %v", sqlparser.String(node))
	}
}

func (pb *standardPrimitiveGenerator) whereComparisonExprCopyAndReWrite(
	expr *sqlparser.ComparisonExpr,
	requiredParameters,
	optionalParameters suffix.ParameterSuffixMap,
) (sqlparser.Expr, string, error) {
	if parserutil.IsScalarComparison(expr) {
		return expr, "", nil
	}
	qualifiedName, ok := expr.Left.(*sqlparser.ColName)
	if !ok {
		if qualifiedLHSName, lhsOk := expr.Left.(*sqlparser.FuncExpr); lhsOk {
			funcName := strings.ToLower(qualifiedLHSName.Name.GetRawVal())
			logging.GetLogger().Infoln(fmt.Sprintf("funcName = %s", funcName))
			return expr, "", nil
		} else if qualifiedRHSName, rhsOk := expr.Right.(*sqlparser.FuncExpr); rhsOk {
			funcName := strings.ToLower(qualifiedRHSName.Name.GetRawVal())
			logging.GetLogger().Infoln(fmt.Sprintf("funcName = %s", funcName))
			return expr, "", nil
		}
		return nil, "", fmt.Errorf("unexpected: %v", sqlparser.String(expr))
	}
	colName := internaldto.GeneratePutativelyUniqueColumnID(qualifiedName.Qualifier, qualifiedName.Name.GetRawVal())
	symTabEntry, symTabErr := pb.PrimitiveComposer.GetSymbol(colName)
	_, requiredParamPresent := requiredParameters.Get(colName)
	_, optionalParamPresent := optionalParameters.Get(colName)
	logging.GetLogger().Infoln(fmt.Sprintf("symTabEntry = %v", symTabEntry))
	containsSQLDataSource := pb.GetPrimitiveComposer().ContainsSQLDataSource()
	if !containsSQLDataSource && symTabErr != nil && !(requiredParamPresent || optionalParamPresent) {
		return nil, colName, symTabErr
	}
	if requiredParamPresent {
		requiredParameters.Delete(colName)
	}
	if optionalParamPresent {
		optionalParameters.Delete(colName)
	}
	if containsSQLDataSource {
		return expr, colName, nil
	}
	if symTabErr == nil && symTabEntry.In != "server" {
		if !(requiredParamPresent || optionalParamPresent) {
			return &sqlparser.ComparisonExpr{
				Left:     expr.Left,
				Right:    expr.Right,
				Operator: expr.Operator,
				Escape:   expr.Escape,
			}, colName, nil
		}
	}
	rewrittenOperator := expr.Operator
	if strings.ToUpper(expr.Operator) == "IN" {
		rewrittenOperator = "="
	}
	return &sqlparser.ComparisonExpr{
		Left:     &sqlparser.SQLVal{Type: sqlparser.IntVal, Val: []byte("1")},
		Right:    &sqlparser.SQLVal{Type: sqlparser.IntVal, Val: []byte("1")},
		Operator: rewrittenOperator,
		Escape:   expr.Escape,
	}, colName, nil
}

func (pb *standardPrimitiveGenerator) analyzeWhere(
	where *sqlparser.Where,
	existingParams map[string]interface{},
) (*sqlparser.Where, []string, error) {
	var retVal sqlparser.Expr
	var paramsSupplied []string
	tableParameterCollection, err := pb.PrimitiveComposer.AssignParameters()
	if err != nil {
		return nil, paramsSupplied, err
	}
	optionalParameters := tableParameterCollection.GetOptionalParams()
	requiredParameters := tableParameterCollection.GetRequiredParams()
	remainingRequiredParameters := tableParameterCollection.GetRemainingRequiredParams()
	if where != nil {
		retVal, paramsSupplied, err = pb.traverseWhereFilter(where.Expr, requiredParameters, optionalParameters)
		if err != nil {
			return nil, paramsSupplied, err
		}
	}

	for k := range existingParams {
		remainingRequiredParameters.Delete(k)
	}

	// TODO: consume parent parameters for any shortfall in required params
	// TODO: same, for optional params

	isIndirect := pb.PrimitiveComposer.IsIndirect()
	if remainingRequiredParameters.Size() > 0 && !isIndirect {
		if where == nil {
			return nil, paramsSupplied,
				fmt.Errorf("WHERE clause not supplied, run DESCRIBE EXTENDED for the resource to see required parameters")
		}
		var keys []string
		for k := range remainingRequiredParameters.GetAll() {
			keys = append(keys, k)
		}
		return nil, paramsSupplied,
			fmt.Errorf("query cannot be executed, missing required parameters: { %s }", strings.Join(keys, ", "))
	}
	if where == nil {
		return nil, paramsSupplied, nil
	}
	return &sqlparser.Where{Type: where.Type, Expr: retVal}, paramsSupplied, nil
}

func extractVarDefFromExec(node *sqlparser.Exec, argName string) (*sqlparser.ExecVarDef, error) {
	for _, varDef := range node.ExecVarDefs {
		if varDef.ColIdent.GetRawVal() == argName {
			return &varDef, nil
		}
	}
	return nil, fmt.Errorf("could not find variable '%s'", argName)
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
func (pb *standardPrimitiveGenerator) AnalyzeUnaryExec(
	pbi planbuilderinput.PlanBuilderInput,
	handlerCtx handler.HandlerContext,
	node *sqlparser.Exec,
	selectNode *sqlparser.Select,
	cols []parserutil.ColumnHandle,
) (tablemetadata.ExtendedTableMetadata, error) {
	isAwait := pb.hasAwaitComment(node.Comments)
	err := pb.inferHeirarchyAndPersist(handlerCtx, node, nil, false, isAwait)
	if err != nil {
		return nil, err
	}
	pb.parseComments(node.Comments)

	meta, err := pb.PrimitiveComposer.GetTable(node)
	if err != nil {
		return nil, err
	}

	method, err := meta.GetMethod()
	if err != nil {
		return nil, err
	}

	if pb.PrimitiveComposer.IsAwait() && !method.IsAwaitable() {
		return nil, fmt.Errorf("method %s is not awaitable", method.GetName())
	}

	requiredParams := method.GetRequiredParameters()

	colz, err := parserutil.GetColumnUsageTypesForExec(node)
	if err != nil {
		return nil, err
	}
	usageErr := parserutil.CheckColUsagesAgainstTable(colz, method)
	if usageErr != nil {
		return nil, usageErr
	}
	for k, param := range requiredParams {
		logging.GetLogger().Debugln(fmt.Sprintf("param = %v", param))
		_, err = extractVarDefFromExec(node, k)
		if err != nil {
			return nil, fmt.Errorf("required param not supplied for exec: %w", err)
		}
	}
	prov, err := meta.GetProvider()
	if err != nil {
		return nil, err
	}
	svcStr, err := meta.GetServiceStr()
	if err != nil {
		return nil, err
	}
	svc, err := meta.GetService()
	if err != nil {
		return nil, err
	}
	rStr, err := meta.GetResourceStr()
	if err != nil {
		return nil, err
	}
	logging.GetLogger().Infoln(
		fmt.Sprintf("provider = '%s', service = '%s', resource = '%s'",
			prov.GetProviderString(), svcStr, rStr))
	requestSchema, err := method.GetRequestBodySchema()
	// requestSchema, err := prov.GetObjectSchema(svcStr, rStr, method.Request.BodyMediaType)
	req, reqExists := method.GetRequest()
	if err != nil && reqExists {
		return nil, err
	}
	var execPayload anysdk_internaldto.ExecPayload
	if node.OptExecPayload != nil {
		mediaType := "application/json" //nolint:goconst // acceptable for now
		if reqExists && req.GetBodyMediaType() != "" {
			mediaType = req.GetBodyMediaType()
		}
		execPayload, err = pb.parseExecPayload(node.OptExecPayload, mediaType)
		if err != nil {
			return nil, err
		}
		err = pb.analyzeSchemaVsMap(handlerCtx, requestSchema, execPayload.GetPayloadMap(), method)
		if err != nil {
			return nil, err
		}
	}
	rsc, err := meta.GetResource()
	if err != nil {
		return nil, err
	}
	err = pb.buildRequestContext(node, meta, formulation.NewExecContext(execPayload, rsc), nil)
	if err != nil {
		return nil, err
	}
	pb.PrimitiveComposer.SetTable(node, meta)

	// parse response with SQL
	if method.IsNullary() && !pb.PrimitiveComposer.IsAwait() {
		return meta, nil
	}
	// TODO: columns in and replace hadnrolled analysis
	analysisInput := formulation.NewMethodAnalysisInput(
		method,
		svc,
		true,
		[]formulation.ColumnDescriptor{},
		false,
	)
	analyser := formulation.NewMethodAnalyzer()
	methodAnalysisOutput, analysisErr := analyser.AnalyzeUnaryAction(analysisInput)
	if analysisErr != nil {
		return meta, analysisErr
	}
	if selectNode != nil {
		return meta, pb.analyzeUnarySelection(pbi, handlerCtx, selectNode, selectNode.Where, meta, cols, methodAnalysisOutput)
	}
	return meta, pb.analyzeUnarySelection(pbi, handlerCtx, node, nil, meta, cols, methodAnalysisOutput)
}

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

func (pb *standardPrimitiveGenerator) analyzeExec(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	annotatedAST := pbi.GetAnnotatedAST()
	node, ok := pbi.GetExec()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Exec", pbi.GetStatement())
	}
	tbl, err := pb.AnalyzeUnaryExec(pbi, handlerCtx, node, nil, nil)
	if err != nil {
		return err
	}
	insertionContainer, err := tableinsertioncontainer.NewTableInsertionContainer(
		tbl,
		handlerCtx.GetSQLEngine(),
		handlerCtx.GetTxnCounterMgr())
	if err != nil {
		return err
	}
	m, err := tbl.GetMethod()
	if err != nil {
		return err
	}
	if m.IsNullary() && !pb.PrimitiveComposer.IsAwait() {
		bldrInput := builder_input.NewBuilderInput(
			pb.PrimitiveComposer.GetGraphHolder(),
			handlerCtx,
			tbl,
		)
		bldrInput.SetIsAwait(false) // returning hardcoded to false for now
		pb.PrimitiveComposer.SetBuilder(
			primitivebuilder.NewSingleSelectAcquire(
				pb.PrimitiveComposer.GetGraphHolder(),
				handlerCtx,
				insertionContainer,
				pb.PrimitiveComposer.GetInsertPreparedStatementCtx(),
				nil,
				nil,
				bldrInput,
			),
		)
		return nil
	}
	selIndirect, indirectErr := astindirect.NewParserExecIndirect(
		node, pb.PrimitiveComposer.GetSelectPreparedStatementCtx())
	if indirectErr != nil {
		return indirectErr
	}
	annotatedAST.SetExecIndirect(node, selIndirect)
	bldrInput := builder_input.NewBuilderInput(
		pb.PrimitiveComposer.GetGraphHolder(),
		handlerCtx,
		tbl,
	)
	bldrInput.SetTxnCtrlCtrs(pb.PrimitiveComposer.GetTxnCtrlCtrs())
	bldrInput.SetTableInsertionContainer(insertionContainer)
	pb.PrimitiveComposer.SetBuilder(
		primitivebuilder.NewSingleAcquireAndSelect(
			bldrInput,
			pb.PrimitiveComposer.GetInsertPreparedStatementCtx(),
			pb.PrimitiveComposer.GetSelectPreparedStatementCtx(),
			nil,
		))
	return nil
}

func (pb *standardPrimitiveGenerator) sanitiseMediaType(mediaType string) string {
	if synonymJSONRegexp.MatchString(mediaType) {
		return "application/json"
	}
	if synonymXMLRegexp.MatchString(mediaType) {
		return "application/xml"
	}
	return mediaType
}

func (pb *standardPrimitiveGenerator) parseExecPayload(
	node *sqlparser.ExecVarDef,
	payloadType string,
) (anysdk_internaldto.ExecPayload, error) {
	var b []byte
	m := make(map[string][]string)
	var pm map[string]interface{}
	payloadType = pb.sanitiseMediaType(payloadType)
	switch val := node.Val.(type) {
	case *sqlparser.SQLVal:
		b = val.Val
	default:
		return nil, fmt.Errorf("payload map of SQL type = '%T' not allowed", val)
	}
	switch payloadType {
	case constants.JSONStr, "application/json":
		m["Content-Type"] = []string{"application/json"}
		err := json.Unmarshal(b, &pm)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("payload map of declared type = '%T' not allowed", payloadType)
	}
	return anysdk_internaldto.NewExecPayload(
		b,
		m,
		pm,
	), nil
}

//nolint:funlen,unparam,gocognit // TODO: refactor
func (pb *standardPrimitiveGenerator) analyzeSchemaVsMap(
	handlerCtx handler.HandlerContext,
	schema formulation.Schema,
	payload map[string]interface{},
	method formulation.OperationStore,
) error {
	requiredElements := make(map[string]bool)
	schemas, err := schema.GetProperties()
	if err != nil {
		return err
	}
	for k := range schemas {
		if schema.IsRequired(k) {
			requiredElements[k] = true
		}
	}
	for k, v := range payload {
		ss, propertyExists := schema.GetProperty(k)
		if !propertyExists {
			return fmt.Errorf("schema does not possess payload key '%s'", k)
		}
		switch val := v.(type) {
		case map[string]interface{}:
			delete(requiredElements, k)
			err = pb.analyzeSchemaVsMap(handlerCtx, ss, val, method)
			if err != nil {
				return err
			}
		case []interface{}:
			subSchema, sErr := schema.GetPropertySchema(k)
			if sErr != nil {
				return sErr
			}
			arraySchema, itemsErr := subSchema.GetItemsSchema()
			if itemsErr != nil {
				return itemsErr
			}
			delete(requiredElements, k)
			if len(val) > 0 && val[0] != nil {
				switch item := val[0].(type) {
				case map[string]interface{}:
					err = pb.analyzeSchemaVsMap(handlerCtx, arraySchema, item, method)
					if err != nil {
						return err
					}
				case string:
					if arraySchema.GetType() != "string" {
						return fmt.Errorf(
							"array at key '%s' expected to contain elemenst of type 'string' but instead they are type '%T'",
							k, item)
					}
				default:
					return fmt.Errorf("array at key '%s' does not contain recognisable type '%T'", k, item)
				}
			}
		case string:
			if ss.GetType() != "string" {
				return fmt.Errorf("key '%s' expected to contain element of type 'string' but instead it is type '%T'", k, val)
			}
			delete(requiredElements, k)
		case int:
			if ss.IsIntegral() {
				delete(requiredElements, k)
				continue
			}
			return fmt.Errorf("key '%s' expected to contain element of type 'int' but instead it is type '%T'", k, val)
		case bool:
			if ss.IsBoolean() {
				delete(requiredElements, k)
				continue
			}
			return fmt.Errorf("key '%s' expected to contain element of type 'bool' but instead it is type '%T'", k, val)
		case float64:
			if ss.IsFloat() {
				delete(requiredElements, k)
				continue
			}
			return fmt.Errorf("key '%s' expected to contain element of type 'float64' but instead it is type '%T'", k, val)
		default:
			return fmt.Errorf("key '%s' of type '%T' not currently supported", k, val)
		}
	}
	if len(requiredElements) != 0 {
		var missingKeys []string
		for k := range requiredElements {
			missingKeys = append(missingKeys, k)
		}
		return fmt.Errorf(
			"required elements not included in suplied object; the following keys are missing: %s",
			strings.Join(missingKeys, ", "))
	}
	return nil
}

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
func (pb *standardPrimitiveGenerator) expandTable(
	tbl tablemetadata.ExtendedTableMetadata) error {
	if viewIndirect, isView := tbl.GetIndirect(); isView {
		viewAST := viewIndirect.GetSelectAST()

		mergeErr := pb.PrimitiveComposer.MergeSymTab(viewIndirect.GetUnderlyingSymTab(), tbl.GetAlias())
		if mergeErr != nil {
			return mergeErr
		}

		logging.GetLogger().Debugf("viewAST = %v\n", viewAST)
		return nil
	}
	if sqlDataSource, isSQLDataSource := tbl.GetSQLDataSource(); isSQLDataSource {
		logging.GetLogger().Debugf("sqlDataSource = %v\n", sqlDataSource)
		return nil
	}
	// TODO: encapsulate the mapping of openapi schemas to symbol table entries.
	//   - This operates atop DRM.
	svc, err := tbl.GetService()
	if err != nil {
		return err
	}
	var availableServers openapi3.Servers
	var availableServersDoExist bool
	method, _ := tbl.GetMethod()
	if method != nil {
		srvs, ok := method.GetServers()
		if ok && len(srvs) > 0 {
			availableServers = srvs
			availableServersDoExist = true
		}
	}
	if len(availableServers) == 0 {
		availableServers, availableServersDoExist = svc.GetServers()
	}
	if availableServersDoExist {
		for _, sv := range availableServers {
			for k := range sv.Variables {
				colEntry := symtab.NewSymTabEntry(
					pb.PrimitiveComposer.GetDRMConfig().GetRelationalType("string"),
					"",
					"server",
				)
				uid := fmt.Sprintf("%s.%s", tbl.GetUniqueID(), k)
				pb.PrimitiveComposer.SetSymbol(uid, colEntry) //nolint:errcheck // TODO: review
			}
			break //nolint:staticcheck // TODO: review
		}
	}
	responseSchema, err := tbl.GetSelectableObjectSchema()
	if err != nil {
		return err
	}
	cols, err := responseSchema.GetProperties()
	if err != nil {
		return err
	}
	if len(cols) == 0 {
		cols = formulation.Schemas{formulation.AnonymousColumnName: responseSchema}
	}
	for colName, colSchema := range cols {
		if colSchema == nil {
			return fmt.Errorf("could not infer column information")
		}
		colEntry := symtab.NewSymTabEntry(
			pb.PrimitiveComposer.GetDRMConfig().GetRelationalType(colSchema.GetType()),
			colSchema,
			"",
		)
		uid := fmt.Sprintf("%s.%s", tbl.GetUniqueID(), colName)
		pb.PrimitiveComposer.SetSymbol(uid, colEntry) //nolint:errcheck // TODO: review
	}
	return nil
}

func (pb *standardPrimitiveGenerator) buildRequestContext(
	node sqlparser.SQLNode,
	meta tablemetadata.ExtendedTableMetadata,
	execContext formulation.ExecContext,
	rowsToInsert map[int]map[int]interface{},
) error {
	paramMapArray, paramErr := util.ExtractSQLNodeParams(node, rowsToInsert)
	if paramErr != nil {
		return paramErr
	}
	return pb.buildRequestContextFromMapArray(
		node,
		meta,
		execContext,
		paramMapArray,
	)
}

func (pb *standardPrimitiveGenerator) buildRequestContextFromMapArray(
	node sqlparser.SQLNode, //nolint:revive,unparam // TODO: review
	meta tablemetadata.ExtendedTableMetadata,
	execContext formulation.ExecContext,
	paramMapArray map[int]map[string]interface{},
) error {
	m, err := meta.GetMethod()
	if err != nil {
		return err
	}
	prov, err := meta.GetProvider()
	if err != nil {
		return err
	}
	svc, err := meta.GetService()
	if err != nil {
		return err
	}
	pr, prErr := prov.GetProvider()
	if prErr != nil {
		return prErr
	}
	meta.WithGetHTTPArmoury(
		func() (formulation.HTTPArmoury, error) {
			httpPreparator := formulation.NewHTTPPreparator(
				pr,
				svc,
				m,
				paramMapArray,
				nil,
				execContext,
				logging.GetLogger(),
			)
			httpArmoury, httpErr := httpPreparator.BuildHTTPRequestCtx(
				formulation.NewHTTPPreparatorConfig(false),
			)
			if httpErr != nil {
				return nil, httpErr
			}
			return httpArmoury, nil
		},
	)
	return err
}

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
	isPhysicalTable := tbl.IsPhysicalTable()
	if isPhysicalTable {
		return nil
	}
	prov, err := tbl.GetProvider()
	if err != nil {
		return err
	}
	currentService, err := tbl.GetServiceStr()
	if err != nil {
		return err
	}
	currentResource, err := tbl.GetResourceStr()
	if err != nil {
		return err
	}

	pb.parseComments(node.Comments)

	method, err := tbl.GetMethod()
	if err != nil {
		return err
	}

	if pb.PrimitiveComposer.IsAwait() && !method.IsAwaitable() {
		return fmt.Errorf("method %s is not awaitable", method.GetName())
	}

	_, err = checkResource(handlerCtx, prov, currentService, currentResource)
	if err != nil {
		return err
	}
	pb.parseComments(node.Comments)

	if tbl.IsPhysicalTable() {
		return nil
	}
	svc, err := tbl.GetService()
	if err != nil {
		return err
	}
	isOpenapi := svc.IsOpenapi()
	if !isOpenapi {
		err = pb.buildRequestContext(node, tbl, nil, insertValOnlyRows)
		if err != nil {
			return err
		}
		return nil
	}

	if pb.PrimitiveComposer.IsAwait() && !method.IsAwaitable() {
		return fmt.Errorf("method %s is not awaitable", method.GetName())
	}
	if pb.PrimitiveComposer.IsAwait() && !method.IsAwaitable() {
		return fmt.Errorf("method %s is not awaitable", method.GetName())
	}
	analysisInput := formulation.NewMethodAnalysisInput(
		method,
		svc,
		true,
		[]formulation.ColumnDescriptor{},
		pb.PrimitiveComposer.IsAwait(),
	)
	analyser := formulation.NewMethodAnalyzer()
	// TODO: this ought to cater for async
	methodAnalysisOutput, analysisErr := analyser.AnalyzeUnaryAction(analysisInput)
	if analysisErr != nil {
		return analysisErr
	}
	err = pb.buildRequestContext(node, tbl, nil, insertValOnlyRows)
	if err != nil {
		return err
	}
	columnHandles := []parserutil.ColumnHandle{}
	if len(node.SelectExprs) > 0 {
		starColumns, _ := methodAnalysisOutput.GetOrderedStarColumnsNames()
		columnHandles, err = parserutil.ExtractInsertReturningColumnNames(node, starColumns, handlerCtx.GetASTFormatter())
		if err != nil {
			return err
		}
	}
	err = pb.analyzeUnaryAction(
		pbi,
		handlerCtx,
		node,
		nil,
		tbl,
		columnHandles,
		methodAnalysisOutput,
	)
	if err != nil {
		return err
	}
	pb.PrimitiveComposer.SetTable(node, tbl)
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

	isPhysicalTable := tbl.IsPhysicalTable()
	if isPhysicalTable {
		return nil
	}

	pb.parseComments(node.Comments)

	method, err := tbl.GetMethod()
	if err != nil {
		return err
	}

	if pb.PrimitiveComposer.IsAwait() && !method.IsAwaitable() {
		return fmt.Errorf("method %s is not awaitable", method.GetName())
	}

	if tbl.IsPhysicalTable() {
		return nil
	}
	svc, err := tbl.GetService()
	if err != nil {
		return err
	}
	updateValOnlyRows, _, err := parserutil.ExtractUpdateValColumns(node)
	if err != nil {
		return err
	}
	firstRow := make(map[string]interface{})
	for k, v := range updateValOnlyRows {
		firstRow[k.GetRawVal()] = v
	}
	updateValOnlyRowsMap := map[int]map[string]interface{}{0: firstRow}
	isOpenapi := svc.IsOpenapi()
	if !isOpenapi {
		err = pb.buildRequestContextFromMapArray(
			node,
			tbl,
			nil,
			updateValOnlyRowsMap,
		)
		if err != nil {
			return err
		}
		return nil
	}

	if pb.PrimitiveComposer.IsAwait() && !method.IsAwaitable() {
		return fmt.Errorf("method %s is not awaitable", method.GetName())
	}
	if pb.PrimitiveComposer.IsAwait() && !method.IsAwaitable() {
		return fmt.Errorf("method %s is not awaitable", method.GetName())
	}
	analysisInput := formulation.NewMethodAnalysisInput(
		method,
		svc,
		true,
		[]formulation.ColumnDescriptor{},
		pb.PrimitiveComposer.IsAwait(),
	)
	analyser := formulation.NewMethodAnalyzer()
	methodAnalysisOutput, analysisErr := analyser.AnalyzeUnaryAction(analysisInput)
	if analysisErr != nil {
		return analysisErr
	}
	err = pb.buildRequestContextFromMapArray(node, tbl, nil, updateValOnlyRowsMap)
	if err != nil {
		return err
	}
	columnHandles := []parserutil.ColumnHandle{}
	if len(node.SelectExprs) > 0 {
		starColumns, _ := methodAnalysisOutput.GetOrderedStarColumnsNames()
		columnHandles, err = parserutil.ExtractUpdateReturningColumnNames(
			node,
			starColumns,
			handlerCtx.GetASTFormatter(),
		)
		if err != nil {
			return err
		}
	}
	err = pb.analyzeUnaryAction(
		pbi,
		handlerCtx,
		node,
		nil,
		tbl,
		columnHandles,
		methodAnalysisOutput,
	)
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
	paramMap, paramMapExists := pbi.GetAnnotatedAST().GetWhereParamMapsEntry(node.Where)
	isAwait := pb.hasAwaitComment(node.Comments)
	err := pb.inferHeirarchyAndPersist(handlerCtx, node, paramMap, false, isAwait)
	if err != nil {
		return err
	}
	tbl, err := pb.PrimitiveComposer.GetTable(node)
	if err != nil {
		return err
	}

	if tbl.IsPhysicalTable() {
		return nil
	}

	if !paramMapExists {
		return fmt.Errorf("where parameters not found; should be anlaysed a priori")
	}
	method, err := tbl.GetMethod()
	if err != nil {
		return err
	}

	if pb.PrimitiveComposer.IsAwait() && !method.IsAwaitable() {
		return fmt.Errorf("method %s is not awaitable", method.GetName())
	}
	if pb.PrimitiveComposer.IsAwait() && len(node.SelectExprs) > 0 {
		return fmt.Errorf("returning from asynchronous delete is disallowed")
	}
	svc, err := tbl.GetService()
	if err != nil {
		return err
	}

	if pb.PrimitiveComposer.IsAwait() && !method.IsAwaitable() {
		return fmt.Errorf("method %s is not awaitable", method.GetName())
	}
	if pb.PrimitiveComposer.IsAwait() && !method.IsAwaitable() {
		return fmt.Errorf("method %s is not awaitable", method.GetName())
	}
	analysisInput := formulation.NewMethodAnalysisInput(
		method,
		svc,
		true,
		[]formulation.ColumnDescriptor{},
		pb.PrimitiveComposer.IsAwait(),
	)
	analyser := formulation.NewMethodAnalyzer()
	methodAnalysisOutput, analysisErr := analyser.AnalyzeUnaryAction(analysisInput)
	if analysisErr != nil {
		return analysisErr
	}
	err = pb.buildRequestContext(node, tbl, nil, nil)
	if err != nil {
		return err
	}
	columnHandles := []parserutil.ColumnHandle{}
	if len(node.SelectExprs) > 0 {
		if pb.PrimitiveComposer.IsAwait() {
			return fmt.Errorf("delete with returning not allowed in await mode")
		}
		starColumns, _ := methodAnalysisOutput.GetOrderedStarColumnsNames()
		columnHandles, err = parserutil.ExtractDeleteReturningColumnNames(
			node,
			starColumns,
			handlerCtx.GetASTFormatter(),
		)
		if err != nil {
			return err
		}
	}
	err = pb.analyzeUnaryAction(
		pbi,
		handlerCtx,
		node,
		node.Where,
		tbl,
		columnHandles,
		methodAnalysisOutput,
	)
	if err != nil {
		return err
	}
	pb.PrimitiveComposer.SetTable(node, tbl)
	return nil
}

func (pb *standardPrimitiveGenerator) analyzeDescribe(pbi planbuilderinput.PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetDescribeTable()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Describe", pbi.GetStatement())
	}
	var err error
	// DESCRIBE has no await or comments at this time
	err = pb.inferHeirarchyAndPersist(handlerCtx, node, nil, true, false)
	if err != nil {
		return err
	}
	tbl, err := pb.PrimitiveComposer.GetTable(node)
	if err != nil {
		return err
	}
	_, isView := tbl.GetView()
	if isView {
		return nil
	}
	prov, err := tbl.GetProvider()
	if err != nil {
		return err
	}
	currentService, err := tbl.GetServiceStr()
	if err != nil {
		return err
	}
	currentResource, err := tbl.GetResourceStr()
	if err != nil {
		return err
	}
	_, err = checkResource(handlerCtx, prov, currentService, currentResource)
	if err != nil {
		return err
	}
	return nil
}

func (pb *standardPrimitiveGenerator) analyzeSleep(pbi planbuilderinput.PlanBuilderInput) error {
	// handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetSleep()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Sleep", pbi.GetStatement())
	}
	sleepDuration, err := parserutil.ExtractSleepDuration(node)
	if err != nil {
		return err
	}
	if sleepDuration <= 0 {
		return fmt.Errorf("sleep duration %d not allowed, must be > 0", sleepDuration)
	}
	graph := pb.PrimitiveComposer.GetGraphHolder()
	pb.PrimitiveComposer.SetRoot(
		graph.CreatePrimitiveNode(
			primitive.NewLocalPrimitive(
				//nolint:revive // acceptable for now
				func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
					time.Sleep(time.Duration(sleepDuration) * time.Millisecond)
					return internaldto.NewExecutorOutput(
						nil, nil, nil,
						internaldto.NewBackendMessages(
							[]string{
								fmt.Sprintf("Success: slept for %d milliseconds", sleepDuration),
							},
						), nil)
				},
			),
		),
	)
	return err
}

func (pb *standardPrimitiveGenerator) AnalyzeRegistry(pbi planbuilderinput.PlanBuilderInput) error {
	_, ok := pbi.GetRegistry()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Registry", pbi.GetStatement())
	}
	return nil
}

//nolint:funlen,gocognit,gocyclo,cyclop,govet // TODO: review
func (pb *standardPrimitiveGenerator) analyzeShow(
	pbi planbuilderinput.PlanBuilderInput) error {
	var err error
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetShow()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Show", pbi.GetStatement())
	}
	if sel, ok := planbuilderinput.IsPGSetupQuery(pbi); ok {
		if sel != nil {
			bldr := primitivebuilder.NewNativeSelect(pb.PrimitiveComposer.GetGraphHolder(), handlerCtx, sel)
			pb.PrimitiveComposer.SetBuilder(bldr)
			return nil
		}
		return pb.AnalyzeNop(pbi)
	}
	pb.parseComments(node.Comments)
	err = pb.inferProviderForShow(node, handlerCtx)
	if err != nil {
		return err
	}
	nodeTypeUpperCase := strings.ToUpper(node.Type)
	if pb.PrimitiveComposer.GetProvider() != nil {
		pb.PrimitiveComposer.SetLikeAbleColumns(pb.PrimitiveComposer.GetProvider().GetLikeableColumns(nodeTypeUpperCase))
	}
	colNames, err := parserutil.ExtractShowColNames(node.ShowTablesOpt)
	if err != nil {
		return err
	}
	colUsages, err := parserutil.ExtractShowColUsage(node.ShowTablesOpt)
	if err != nil {
		return err
	}
	switch nodeTypeUpperCase {
	case "AUTH":
		// TODO
	case "INSERT":
		// SHOW has no comments or concept of await
		err = pb.inferHeirarchyAndPersist(handlerCtx, node, nil, false, false)
		if err != nil {
			return err
		}
	case "METHODS":
		// SHOW has no comments or concept of await
		err = pb.inferHeirarchyAndPersist(handlerCtx, node, nil, false, false)
		if err != nil {
			return err
		}
		tbl, err := pb.PrimitiveComposer.GetTable(node)
		if err != nil {
			return err
		}
		currentService, err := tbl.GetServiceStr()
		if err != nil {
			return err
		}
		currentResource, err := tbl.GetResourceStr()
		if err != nil {
			return err
		}
		_, err = checkResource(handlerCtx, pb.PrimitiveComposer.GetProvider(), currentService, currentResource)
		if err != nil {
			return err
		}
		if node.ShowTablesOpt != nil {
			meth := formulation.NewEmptyOperationStore()
			err = pb.analyzeShowFilter(node, meth)
			if err != nil {
				return err
			}
		}
		return nil
	case "PROVIDERS":
		// TODO
	case "RESOURCES":
		prov, err := handlerCtx.GetProvider(node.OnTable.Qualifier.GetRawVal())
		if err != nil {
			return err
		}
		pb.PrimitiveComposer.SetProvider(prov)
		_, err = pb.assembleResources(handlerCtx, pb.PrimitiveComposer.GetProvider(), node.OnTable.Name.GetRawVal())
		if err != nil {
			return err
		}
		for _, col := range colNames {
			if !formulation.ResourceKeyExists(col) {
				return fmt.Errorf("SHOW key = '%s' does NOT exist", col)
			}
		}
		for _, colUsage := range colUsages {
			if !formulation.ResourceKeyExists(colUsage.ColName.Name.GetRawVal()) {
				return fmt.Errorf("SHOW key = '%s' does NOT exist", colUsage.ColName.Name.GetRawVal())
			}
			usageErr := parserutil.CheckSQLParserTypeVsResourceColumn(colUsage)
			if usageErr != nil {
				return usageErr
			}
		}
		if node.ShowTablesOpt != nil {
			rsc := formulation.NewEmptyResource()
			err = pb.analyzeShowFilter(node, rsc)
			if err != nil {
				return err
			}
		}
	case "SERVICES":
		prov, err := handlerCtx.GetProvider(node.OnTable.Name.GetRawVal())
		if err != nil {
			return err
		}
		pb.PrimitiveComposer.SetProvider(prov)
		for _, col := range colNames {
			if !formulation.ServiceKeyExists(col) {
				return fmt.Errorf("SHOW key = '%s' does NOT exist", col)
			}
		}
		for _, colUsage := range colUsages {
			if !formulation.ServiceKeyExists(colUsage.ColName.Name.GetRawVal()) {
				return fmt.Errorf("SHOW key = '%s' does NOT exist", colUsage.ColName.Name.GetRawVal())
			}
			usageErr := parserutil.CheckSQLParserTypeVsServiceColumn(colUsage)
			if usageErr != nil {
				return usageErr
			}
		}
		if node.ShowTablesOpt != nil {
			svc := formulation.NewEmptyProviderService()
			err = pb.analyzeShowFilter(node, svc)
			if err != nil {
				return err
			}
		}
	default:
		err = fmt.Errorf("SHOW statement not supported for '%s'", nodeTypeUpperCase)
	}
	return err
}
