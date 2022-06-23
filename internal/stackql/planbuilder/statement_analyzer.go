package planbuilder

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/stackql/stackql/internal/stackql/astvisit"
	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/docparser"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/httpbuild"
	"github.com/stackql/stackql/internal/stackql/iqlerror"
	"github.com/stackql/stackql/internal/stackql/iqlutil"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivebuilder"
	"github.com/stackql/stackql/internal/stackql/provider"
	"github.com/stackql/stackql/internal/stackql/relational"
	"github.com/stackql/stackql/internal/stackql/suffix"
	"github.com/stackql/stackql/internal/stackql/symtab"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/go-openapistackql/openapistackql"

	"vitess.io/vitess/go/vt/sqlparser"

	log "github.com/sirupsen/logrus"
)

func (p *primitiveGenerator) analyzeStatement(pbi PlanBuilderInput) error {
	var err error
	statement := pbi.GetStatement()
	switch stmt := statement.(type) {
	case *sqlparser.Auth:
		return p.analyzeAuth(pbi)
	case *sqlparser.AuthRevoke:
		return p.analyzeAuthRevoke(pbi)
	case *sqlparser.Begin:
		return iqlerror.GetStatementNotSupportedError("TRANSACTION: BEGIN")
	case *sqlparser.Commit:
		return iqlerror.GetStatementNotSupportedError("TRANSACTION: COMMIT")
	case *sqlparser.DBDDL:
		return iqlerror.GetStatementNotSupportedError(fmt.Sprintf("unsupported: Database DDL %v", sqlparser.String(stmt)))
	case *sqlparser.DDL:
		return iqlerror.GetStatementNotSupportedError("DDL")
	case *sqlparser.Delete:
		return p.analyzeDelete(pbi)
	case *sqlparser.DescribeTable:
		return p.analyzeDescribe(pbi)
	case *sqlparser.Exec:
		return p.analyzeExec(pbi)
	case *sqlparser.Explain:
		return iqlerror.GetStatementNotSupportedError("EXPLAIN")
	case *sqlparser.Insert:
		return p.analyzeInsert(pbi)
	case *sqlparser.OtherRead, *sqlparser.OtherAdmin:
		return iqlerror.GetStatementNotSupportedError("OTHER")
	case *sqlparser.Registry:
		return p.analyzeRegistry(pbi)
	case *sqlparser.Rollback:
		return iqlerror.GetStatementNotSupportedError("TRANSACTION: ROLLBACK")
	case *sqlparser.Savepoint:
		return iqlerror.GetStatementNotSupportedError("TRANSACTION: SAVEPOINT")
	case *sqlparser.Select:
		return p.analyzeSelect(pbi)
	case *sqlparser.Set:
		return iqlerror.GetStatementNotSupportedError("SET")
	case *sqlparser.SetTransaction:
		return iqlerror.GetStatementNotSupportedError("SET TRANSACTION")
	case *sqlparser.Show:
		return p.analyzeShow(pbi)
	case *sqlparser.Sleep:
		return p.analyzeSleep(pbi)
	case *sqlparser.SRollback:
		return iqlerror.GetStatementNotSupportedError("TRANSACTION: SROLLBACK")
	case *sqlparser.Release:
		return iqlerror.GetStatementNotSupportedError("TRANSACTION: RELEASE")
	case *sqlparser.Union:
		return p.analyzeUnion(pbi)
	case *sqlparser.Update:
		return iqlerror.GetStatementNotSupportedError("UPDATE")
	case *sqlparser.Use:
		return p.analyzeUse(pbi)
	}
	return err
}

func (p *primitiveGenerator) analyzeUse(pbi PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetUse()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Use", pbi.GetStatement())
	}
	prov, pErr := handlerCtx.GetProvider(node.DBName.GetRawVal())
	if pErr != nil {
		return pErr
	}
	p.PrimitiveBuilder.SetProvider(prov)
	return nil
}

func (p *primitiveGenerator) analyzeUnion(pbi PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetUnion()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Union", pbi.GetStatement())
	}
	unionQuery := astvisit.GenerateUnionTemplateQuery(node)
	i := 0
	leaf, err := p.PrimitiveBuilder.GetSymTab().NewLeaf(i)
	if err != nil {
		return err
	}
	pChild := p.addChildPrimitiveGenerator(node.FirstStatement, leaf)
	err = pChild.analyzeSelectStatement(NewPlanBuilderInput(handlerCtx, node.FirstStatement, nil, nil, nil, nil, nil))
	if err != nil {
		return err
	}
	var selectStatementContexts []*drm.PreparedStatementCtx
	for _, rhsStmt := range node.UnionSelects {
		i++
		leaf, err := p.PrimitiveBuilder.GetSymTab().NewLeaf(i)
		if err != nil {
			return err
		}
		pChild := p.addChildPrimitiveGenerator(rhsStmt.Statement, leaf)
		err = pChild.analyzeSelectStatement(NewPlanBuilderInput(handlerCtx, rhsStmt.Statement, nil, nil, nil, nil, nil))
		if err != nil {
			return err
		}
		ctx := pChild.PrimitiveBuilder.GetSelectPreparedStatementCtx()
		ctx.SetKind(rhsStmt.Type)
		selectStatementContexts = append(selectStatementContexts, ctx)
	}

	bldr := primitivebuilder.NewUnion(
		p.PrimitiveBuilder.GetGraph(),
		handlerCtx,
		drm.NewQueryOnlyPreparedStatementCtx(unionQuery),
		pChild.PrimitiveBuilder.GetSelectPreparedStatementCtx(),
		selectStatementContexts,
	)
	p.PrimitiveBuilder.SetBuilder(bldr)

	return nil
}

func (p *primitiveGenerator) analyzeSelectStatement(pbi PlanBuilderInput) error {
	node := pbi.GetStatement()
	switch node.(type) {
	case *sqlparser.Select:
		return p.analyzeSelect(pbi)
	case *sqlparser.ParenSelect:
		return p.analyzeSelectStatement(pbi)
	case *sqlparser.Union:
		return p.analyzeUnion(pbi)
	}
	return nil
}

func (p *primitiveGenerator) analyzeAuth(pbi PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	authNode, ok := pbi.GetAuth()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Auth", pbi.GetStatement())
	}
	provider, pErr := handlerCtx.GetProvider(authNode.Provider)
	if pErr != nil {
		return pErr
	}
	p.PrimitiveBuilder.SetProvider(provider)
	return nil
}

func (p *primitiveGenerator) analyzeAuthRevoke(pbi PlanBuilderInput) error {
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
	return fmt.Errorf(`Auth revoke for Google Failed; improper auth method: "%s" specified`, authCtx.Type)
}

func checkResource(handlerCtx *handler.HandlerContext, prov provider.IProvider, service string, resource string) (*openapistackql.Resource, error) {
	return prov.GetResource(service, resource, handlerCtx.RuntimeContext)
}

func (pb *primitiveGenerator) assembleResources(handlerCtx *handler.HandlerContext, prov provider.IProvider, service string) (map[string]*openapistackql.Resource, error) {
	rm, err := prov.GetResourcesMap(service, handlerCtx.RuntimeContext)
	if err != nil {
		return nil, err
	}
	return rm, err
}

func (pb *primitiveGenerator) analyzeShowFilter(node *sqlparser.Show, table openapistackql.ITable) error {
	showFilter := node.ShowTablesOpt.Filter
	if showFilter == nil {
		return nil
	}
	if showFilter.Like != "" {
		likeRegexp, err := regexp.Compile(iqlutil.TranslateLikeToRegexPattern(showFilter.Like))
		if err != nil {
			return fmt.Errorf("cannot compile like string '%s': %s", showFilter.Like, err.Error())
		}
		tableFilter := pb.PrimitiveBuilder.GetTableFilter()
		for _, col := range pb.PrimitiveBuilder.GetLikeAbleColumns() {
			tableFilter = relational.OrTableFilters(tableFilter, relational.ConstructLikePredicateFilter(col, likeRegexp, false))
		}
		pb.PrimitiveBuilder.SetTableFilter(relational.OrTableFilters(pb.PrimitiveBuilder.GetTableFilter(), tableFilter))
	} else if showFilter.Filter != nil {
		tableFilter, err := pb.traverseShowFilter(table, node, showFilter.Filter)
		if err != nil {
			return err
		}
		pb.PrimitiveBuilder.SetTableFilter(tableFilter)
	}
	return nil
}

func (pb *primitiveGenerator) traverseShowFilter(table openapistackql.ITable, node *sqlparser.Show, filter sqlparser.Expr) (func(openapistackql.ITable) (openapistackql.ITable, error), error) {
	var retVal func(openapistackql.ITable) (openapistackql.ITable, error)
	switch filter := filter.(type) {
	case *sqlparser.ComparisonExpr:
		return pb.comparisonExprToFilterFunc(table, node, filter)
	case *sqlparser.AndExpr:
		log.Infoln("complex AND expr detected")
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
		log.Infoln("complex OR expr detected")
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
		return nil, fmt.Errorf("unsupported constraint in openapistackql filter: %v", sqlparser.String(filter))
	default:
		return nil, fmt.Errorf("unsupported constraint in openapistackql filter: %v", sqlparser.String(filter))
	}
	return retVal, nil
}

func (pb *primitiveGenerator) traverseWhereFilter(node sqlparser.SQLNode, requiredParameters, optionalParameters *suffix.ParameterSuffixMap) (sqlparser.Expr, []string, error) {
	switch node := node.(type) {
	case *sqlparser.ComparisonExpr:
		exp, cn, err := pb.whereComparisonExprCopyAndReWrite(node, requiredParameters, optionalParameters)
		return exp, []string{cn}, err
	case *sqlparser.AndExpr:
		log.Infoln("complex AND expr detected")
		lhs, lParams, lhErr := pb.traverseWhereFilter(node.Left, requiredParameters, optionalParameters)
		rhs, rParams, rhErr := pb.traverseWhereFilter(node.Right, requiredParameters, optionalParameters)
		if lhErr != nil {
			return nil, nil, lhErr
		}
		if rhErr != nil {
			return nil, nil, rhErr
		}
		for _, v := range rParams {
			lParams = append(lParams, v)
		}
		return &sqlparser.AndExpr{Left: lhs, Right: rhs}, lParams, nil
	case *sqlparser.OrExpr:
		log.Infoln("complex OR expr detected")
		lhs, lParams, lhErr := pb.traverseWhereFilter(node.Left, requiredParameters, optionalParameters)
		rhs, rParams, rhErr := pb.traverseWhereFilter(node.Right, requiredParameters, optionalParameters)
		if lhErr != nil {
			return nil, nil, lhErr
		}
		if rhErr != nil {
			return nil, nil, rhErr
		}
		for _, v := range rParams {
			lParams = append(lParams, v)
		}
		return &sqlparser.OrExpr{Left: lhs, Right: rhs}, lParams, nil
	case *sqlparser.FuncExpr:
		return nil, nil, fmt.Errorf("unsupported constraint in openapistackql filter: %v", sqlparser.String(node))
	case *sqlparser.IsExpr:
		return &sqlparser.IsExpr{
			Operator: node.Operator,
			Expr:     node.Expr,
		}, nil, nil
	default:
		return nil, nil, fmt.Errorf("unsupported constraint in openapistackql filter: %v", sqlparser.String(node))
	}
	return nil, nil, fmt.Errorf("unsupported constraint in openapistackql filter: %v", sqlparser.String(node))
}

func (pb *primitiveGenerator) whereComparisonExprCopyAndReWrite(expr *sqlparser.ComparisonExpr, requiredParameters, optionalParameters *suffix.ParameterSuffixMap) (sqlparser.Expr, string, error) {
	qualifiedName, ok := expr.Left.(*sqlparser.ColName)
	if !ok {
		return nil, "", fmt.Errorf("unexpected: %v", sqlparser.String(expr))
	}
	colName := dto.GeneratePutativelyUniqueColumnID(qualifiedName.Qualifier, qualifiedName.Name.GetRawVal())
	symTabEntry, symTabErr := pb.PrimitiveBuilder.GetSymbol(colName)
	_, requiredParamPresent := requiredParameters.Get(colName)
	_, optionalParamPresent := optionalParameters.Get(colName)
	log.Infoln(fmt.Sprintf("symTabEntry = %v", symTabEntry))
	if symTabErr != nil && !(requiredParamPresent || optionalParamPresent) {
		return nil, colName, symTabErr
	}
	if requiredParamPresent {
		requiredParameters.Delete(colName)
	}
	if optionalParamPresent {
		optionalParameters.Delete(colName)
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
		paramMAtchStr := ""
		switch rhs := expr.Right.(type) {
		case *sqlparser.SQLVal:
			paramMAtchStr = string(rhs.Val)
		}
		switch rhs := expr.Left.(type) {
		case *sqlparser.SQLVal:
			paramMAtchStr = string(rhs.Val)
		}
		newRhs := &sqlparser.SQLVal{
			Type: sqlparser.StrVal,
			Val:  []byte(fmt.Sprintf("%%%s%%", paramMAtchStr)),
		}
		return &sqlparser.OrExpr{
			Left: &sqlparser.ComparisonExpr{
				Left:     expr.Left,
				Right:    newRhs,
				Operator: sqlparser.LikeStr,
				Escape:   nil,
			},
			Right: &sqlparser.ComparisonExpr{
				Left: expr.Right,
				Right: &sqlparser.BinaryExpr{
					Left: &sqlparser.BinaryExpr{
						Left: &sqlparser.SQLVal{
							Type: sqlparser.StrVal,
							Val:  []byte("%"),
						},
						Right:    expr.Left,
						Operator: sqlparser.BitOrStr,
					},
					Right: &sqlparser.SQLVal{
						Type: sqlparser.StrVal,
						Val:  []byte("%"),
					},
					Operator: sqlparser.BitOrStr,
				},
				Operator: sqlparser.LikeStr,
				Escape:   nil,
			},
		}, colName, nil
	}
	return &sqlparser.ComparisonExpr{
		Left:     &sqlparser.SQLVal{Type: sqlparser.IntVal, Val: []byte("1")},
		Right:    &sqlparser.SQLVal{Type: sqlparser.IntVal, Val: []byte("1")},
		Operator: expr.Operator,
		Escape:   expr.Escape,
	}, colName, nil
}

func (pb *primitiveGenerator) resolveMethods(where *sqlparser.Where) error {
	requiredParameters := suffix.NewParameterSuffixMap()
	// remainingRequiredParameters := suffix.NewParameterSuffixMap()
	optionalParameters := suffix.NewParameterSuffixMap()
	for _, tb := range pb.PrimitiveBuilder.GetTables() {
		tbID := tb.GetUniqueId()
		method, err := tb.GetMethod()
		if err != nil {
			return err
		}
		for k, v := range method.GetRequiredParameters() {
			key := fmt.Sprintf("%s.%s", tbID, k)
			_, keyExists := requiredParameters.Get(key)
			if keyExists {
				return fmt.Errorf("key already is required: %s", k)
			}
			requiredParameters.Put(key, v)
		}
		for k, vOpt := range method.GetOptionalParameters() {
			key := fmt.Sprintf("%s.%s", tbID, k)
			_, keyExists := optionalParameters.Get(key)
			if keyExists {
				return fmt.Errorf("key already is optional: %s", k)
			}
			optionalParameters.Put(key, vOpt)
		}
	}
	return nil
}

func (pb *primitiveGenerator) extractParamsFromWhere(where *sqlparser.Where) (map[string]interface{}, error) {
	if where == nil {
		return map[string]interface{}{}, nil
	}
	return map[string]interface{}{}, nil
}

func (pb *primitiveGenerator) analyzeWhere(where *sqlparser.Where) (*sqlparser.Where, []string, error) {
	requiredParameters := suffix.NewParameterSuffixMap()
	remainingRequiredParameters := suffix.NewParameterSuffixMap()
	optionalParameters := suffix.NewParameterSuffixMap()
	tbVisited := map[*taxonomy.ExtendedTableMetadata]struct{}{}
	for _, tb := range pb.PrimitiveBuilder.GetTables() {
		if _, ok := tbVisited[tb]; ok {
			continue
		}
		tbVisited[tb] = struct{}{}
		tbID := tb.GetUniqueId()
		method, err := tb.GetMethod()
		if err != nil {
			return nil, nil, err
		}
		reqParams := method.GetRequiredParameters()
		for k, v := range reqParams {
			key := fmt.Sprintf("%s.%s", tbID, k)
			_, keyExists := requiredParameters.Get(key)
			if keyExists {
				return nil, nil, fmt.Errorf("key already is required: %s", k)
			}
			requiredParameters.Put(key, v)
		}
		for k, vOpt := range method.GetOptionalParameters() {
			key := fmt.Sprintf("%s.%s", tbID, k)
			_, keyExists := optionalParameters.Get(key)
			if keyExists {
				return nil, nil, fmt.Errorf("key already is optional: %s", k)
			}
			optionalParameters.Put(key, vOpt)
		}
	}
	var retVal sqlparser.Expr
	var paramsSupplied []string
	var err error
	if where != nil {
		retVal, paramsSupplied, err = pb.traverseWhereFilter(where.Expr, requiredParameters, optionalParameters)
		if err != nil {
			return nil, paramsSupplied, err
		}
	}

	for l, w := range requiredParameters.GetAll() {
		remainingRequiredParameters.Put(l, w)
	}

	if remainingRequiredParameters.Size() > 0 {
		if where == nil {
			return nil, paramsSupplied, fmt.Errorf("WHERE clause not supplied, run DESCRIBE EXTENDED for the resource to see required parameters")
		}
		var keys []string
		for k := range remainingRequiredParameters.GetAll() {
			keys = append(keys, k)
		}
		return nil, paramsSupplied, fmt.Errorf("query cannot be executed, missing required parameters: { %s }", strings.Join(keys, ", "))
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

func (p *primitiveGenerator) parseComments(comments sqlparser.Comments) {
	if comments != nil {
		p.PrimitiveBuilder.SetCommentDirectives(sqlparser.ExtractCommentDirectives(comments))
		p.PrimitiveBuilder.SetAwait(p.PrimitiveBuilder.GetCommentDirectives().IsSet("AWAIT"))
	}
}

func (p *primitiveGenerator) persistHerarchyToBuilder(heirarchy *taxonomy.HeirarchyObjects, node sqlparser.SQLNode) {
	p.PrimitiveBuilder.SetTable(node, taxonomy.NewExtendedTableMetadata(heirarchy, taxonomy.GetAliasFromStatement(node)))
}

func (p *primitiveGenerator) analyzeUnaryExec(handlerCtx *handler.HandlerContext, node *sqlparser.Exec, selectNode *sqlparser.Select, cols []parserutil.ColumnHandle) (*taxonomy.ExtendedTableMetadata, error) {
	err := p.inferHeirarchyAndPersist(handlerCtx, node, nil)
	if err != nil {
		return nil, err
	}
	p.parseComments(node.Comments)

	meta, err := p.PrimitiveBuilder.GetTable(node)
	if err != nil {
		return nil, err
	}

	method, err := meta.GetMethod()
	if err != nil {
		return nil, err
	}

	if p.PrimitiveBuilder.IsAwait() && !method.IsAwaitable() {
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
		log.Debugln(fmt.Sprintf("param = %v", param))
		_, err := extractVarDefFromExec(node, k)
		if err != nil {
			return nil, fmt.Errorf("required param not supplied for exec: %s", err.Error())
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
	rStr, err := meta.GetResourceStr()
	if err != nil {
		return nil, err
	}
	log.Infoln(fmt.Sprintf("provider = '%s', service = '%s', resource = '%s'", prov.GetProviderString(), svcStr, rStr))
	requestSchema, err := method.GetRequestBodySchema()
	// requestSchema, err := prov.GetObjectSchema(svcStr, rStr, method.Request.BodyMediaType)
	if err != nil && method.Request != nil {
		return nil, err
	}
	var execPayload *dto.ExecPayload
	if node.OptExecPayload != nil {
		mediaType := "application/json"
		if method.Request != nil && method.Request.BodyMediaType != "" {
			mediaType = method.Request.BodyMediaType
		}
		execPayload, err = p.parseExecPayload(node.OptExecPayload, mediaType)
		if err != nil {
			return nil, err
		}
		err = p.analyzeSchemaVsMap(handlerCtx, requestSchema, execPayload.PayloadMap, method)
		if err != nil {
			return nil, err
		}
	}
	rsc, err := meta.GetResource()
	if err != nil {
		return nil, err
	}
	_, err = p.buildRequestContext(handlerCtx, node, meta, httpbuild.NewExecContext(execPayload, rsc), nil)
	if err != nil {
		return nil, err
	}
	p.PrimitiveBuilder.SetTable(node, meta)

	// parse response with SQL
	if method.IsNullary() && !p.PrimitiveBuilder.IsAwait() {
		return meta, nil
	}
	if selectNode != nil {
		return meta, p.analyzeUnarySelection(handlerCtx, selectNode, selectNode.Where, meta, cols)
	}
	return meta, p.analyzeUnarySelection(handlerCtx, node, nil, meta, cols)
}

func (p *primitiveGenerator) analyzeNop(pbi PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	p.PrimitiveBuilder.SetBuilder(
		primitivebuilder.NewNopBuilder(
			p.PrimitiveBuilder.GetGraph(),
			p.PrimitiveBuilder.GetTxnCtrlCtrs(),
			handlerCtx,
			handlerCtx.SQLEngine,
		),
	)
	err := p.PrimitiveBuilder.GetBuilder().Build()
	return err
}

func (p *primitiveGenerator) analyzeExec(pbi PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetExec()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Exec", pbi.GetStatement())
	}
	tbl, err := p.analyzeUnaryExec(handlerCtx, node, nil, nil)
	if err != nil {
		log.Infoln(fmt.Sprintf("error analyzing EXEC as selection: '%s'", err.Error()))
		return err
	} else {
		m, err := tbl.GetMethod()
		if err != nil {
			return err
		}
		if m.IsNullary() && !p.PrimitiveBuilder.IsAwait() {
			p.PrimitiveBuilder.SetBuilder(primitivebuilder.NewSingleSelectAcquire(p.PrimitiveBuilder.GetGraph(), handlerCtx, tbl, p.PrimitiveBuilder.GetInsertPreparedStatementCtx(), nil))
			return nil
		}
		p.PrimitiveBuilder.SetBuilder(primitivebuilder.NewSingleAcquireAndSelect(p.PrimitiveBuilder.GetGraph(), p.PrimitiveBuilder.GetTxnCtrlCtrs(), handlerCtx, tbl, p.PrimitiveBuilder.GetInsertPreparedStatementCtx(), p.PrimitiveBuilder.GetSelectPreparedStatementCtx(), nil))
	}
	return nil
}

func (p *primitiveGenerator) parseExecPayload(node *sqlparser.ExecVarDef, payloadType string) (*dto.ExecPayload, error) {
	var b []byte
	m := make(map[string][]string)
	var pm map[string]interface{}
	switch val := node.Val.(type) {
	case *sqlparser.SQLVal:
		b = val.Val
	default:
		return nil, fmt.Errorf("payload map of SQL type = '%T' not allowed", val)
	}
	switch payloadType {
	case constants.JsonStr, "application/json":
		m["Content-Type"] = []string{"application/json"}
		err := json.Unmarshal(b, &pm)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("payload map of declared type = '%T' not allowed", payloadType)
	}
	return &dto.ExecPayload{
		Payload:    b,
		Header:     m,
		PayloadMap: pm,
	}, nil
}

func contains(slice []interface{}, elem interface{}) bool {
	for _, a := range slice {
		if a == elem {
			return true
		}
	}
	return false
}

func (p *primitiveGenerator) analyzeSchemaVsMap(handlerCtx *handler.HandlerContext, schema *openapistackql.Schema, payload map[string]interface{}, method *openapistackql.OperationStore) error {
	requiredElements := make(map[string]bool)
	schemas, err := schema.GetProperties()
	if err != nil {
		return err
	}
	for k, _ := range schemas {
		if schema.IsRequired(k) {
			requiredElements[k] = true
		}
	}
	for k, v := range payload {
		ss, err := schema.GetProperty(k)
		if err != nil {
			return fmt.Errorf("schema does not possess payload key '%s'", k)
		}
		switch val := v.(type) {
		case map[string]interface{}:
			delete(requiredElements, k)
			var err error
			err = p.analyzeSchemaVsMap(handlerCtx, ss, val, method)
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
					err := p.analyzeSchemaVsMap(handlerCtx, arraySchema, item, method)
					if err != nil {
						return err
					}
				case string:
					if arraySchema.Type != "string" {
						return fmt.Errorf("array at key '%s' expected to contain elemenst of type 'string' but instead they are type '%T'", k, item)
					}
				default:
					return fmt.Errorf("array at key '%s' does not contain recognisable type '%T'", k, item)
				}
			}
		case string:
			if ss.Type != "string" {
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
		for k, _ := range requiredElements {
			missingKeys = append(missingKeys, k)
		}
		return fmt.Errorf("required elements not included in suplied object; the following keys are missing: %s.", strings.Join(missingKeys, ", "))
	}
	return nil
}

func isPGSetupQuery(q string) bool {
	if q == "select relname, nspname, relkind from pg_catalog.pg_class c, pg_catalog.pg_namespace n where relkind in ('r', 'v', 'm', 'f') and nspname not in ('pg_catalog', 'information_schema', 'pg_toast', 'pg_temp_1') and n.oid = relnamespace order by nspname, relname" {
		return true
	}
	if q == "select oid, typbasetype from pg_type where typname = 'lo'" {
		return true
	}
	return false
}

func (p *primitiveGenerator) analyzeSelect(pbi PlanBuilderInput) error {

	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetSelect()
	if !ok {
		return fmt.Errorf("could not cast statement of type '%T' to required Select", pbi.GetStatement())
	}

	if isPGSetupQuery(handlerCtx.RawQuery) {
		return p.analyzeNop(pbi)
	}

	var pChild *primitiveGenerator
	var err error

	paramMap := astvisit.ExtractParamsFromWhereClause(node.Where)

	router := parserutil.NewParameterRouter(pbi.GetAliasedTables(), pbi.GetAssignedAliasedColumns(), paramMap, pbi.GetColRefs())

	v := astvisit.NewTableRouteAstVisitor(pbi.handlerCtx, router)

	err = v.Visit(pbi.GetStatement())

	if err != nil {
		return err
	}

	tblz := v.GetTableMap()
	annotations := v.GetAnnotations()
	colRefs := pbi.GetColRefs()

	for k, v := range tblz {
		p.PrimitiveBuilder.SetTable(k, v)
	}

	for i, fromExpr := range node.From {
		var leafKey interface{} = i
		switch from := fromExpr.(type) {
		case *sqlparser.AliasedTableExpr:
			if from.As.GetRawVal() != "" {
				leafKey = from.As.GetRawVal()
			}
		}

		leaf, err := p.PrimitiveBuilder.GetSymTab().NewLeaf(leafKey)
		if err != nil {
			return err
		}
		pChild = p.addChildPrimitiveGenerator(fromExpr, leaf)

		for _, tbl := range tblz {
			//
			svc, err := tbl.GetService()
			if err != nil {
				return err
			}
			for _, sv := range svc.Servers {
				for k := range sv.Variables {
					colEntry := symtab.NewSymTabEntry(
						pChild.PrimitiveBuilder.GetDRMConfig().GetRelationalType("string"),
						"",
						"server",
					)
					uid := fmt.Sprintf("%s.%s", tbl.GetUniqueId(), k)
					pChild.PrimitiveBuilder.SetSymbol(uid, colEntry)
				}
				break
			}

			if err != nil {
				return err
			}
			//
			responseSchema, err := tbl.GetSelectableObjectSchema()
			if err != nil {
				return err
			}
			cols, err := responseSchema.GetProperties()
			if err != nil {
				return err
			}
			if len(cols) == 0 {
				cols = openapistackql.Schemas{openapistackql.AnonymousColumnName: responseSchema}
			}
			for colName, colSchema := range cols {
				if colSchema == nil {
					return fmt.Errorf("could not infer column information")
				}
				colEntry := symtab.NewSymTabEntry(
					pChild.PrimitiveBuilder.GetDRMConfig().GetRelationalType(colSchema.Type),
					colSchema,
					"",
				)
				uid := fmt.Sprintf("%s.%s", tbl.GetUniqueId(), colName)
				pChild.PrimitiveBuilder.SetSymbol(uid, colEntry)
			}
		}
	}
	// TODO: fix this hack
	var rewrittenWhere *sqlparser.Where
	var paramsPresent []string
	if len(node.From) == 1 {
		switch ft := node.From[0].(type) {
		case *sqlparser.ExecSubquery:
			log.Infoln(fmt.Sprintf("%v", ft))
		default:
			rewrittenWhere, paramsPresent, err = p.analyzeWhere(node.Where)
			if err != nil {
				return err
			}
			p.PrimitiveBuilder.SetWhere(rewrittenWhere)
		}
	}
	log.Debugf("len(paramsPresent) = %d\n", len(paramsPresent))

	// END TODO
	if len(node.From) == 1 {
		switch ft := node.From[0].(type) {
		case *sqlparser.JoinTableExpr, *sqlparser.AliasedTableExpr:
			var execSlice []primitivebuilder.Builder
			var primaryTcc, tcc *dto.TxnControlCounters
			var secondaryTccs []*dto.TxnControlCounters
			var tableSlice []*taxonomy.ExtendedTableMetadata
			discoGenIDs := make(map[sqlparser.SQLNode]int)
			for k, v := range annotations {
				pr, err := v.TableMeta.GetProvider()
				if err != nil {
					return err
				}
				prov, err := v.TableMeta.GetProviderObject()
				if err != nil {
					return err
				}
				svc, err := v.TableMeta.GetService()
				if err != nil {
					return err
				}
				m, err := v.TableMeta.GetMethod()
				if err != nil {
					return err
				}
				tab := v.Schema.Tabulate(false)
				_, mediaType, err := m.GetResponseBodySchemaAndMediaType()
				if err != nil {
					return err
				}
				switch mediaType {
				case openapistackql.MediaTypeTextXML, openapistackql.MediaTypeXML:
					tab = tab.RenameColumnsToXml()
				}
				anTab := util.NewAnnotatedTabulation(tab, v.HIDs, v.TableMeta.Alias)

				discoGenId, err := docparser.OpenapiStackQLTabulationsPersistor(prov, svc, []util.AnnotatedTabulation{anTab}, p.PrimitiveBuilder.GetSQLEngine(), prov.Name)
				if err != nil {
					return err
				}
				discoGenIDs[k] = discoGenId
				// router.GetAvailableParameters()
				parametersCleaned, err := util.TransformSQLRawParameters(v.Parameters)
				if err != nil {
					return err
				}
				httpArmoury, err := httpbuild.BuildHTTPRequestCtxFromAnnotation(handlerCtx, parametersCleaned, pr, m, svc, nil, nil)
				if err != nil {
					return err
				}
				v.TableMeta.HttpArmoury = httpArmoury
				tableDTO, err := p.PrimitiveBuilder.GetDRMConfig().GetCurrentTable(v.HIDs, handlerCtx.SQLEngine)
				if err != nil {
					return err
				}
				if tcc == nil {
					tcc = dto.NewTxnControlCounters(p.PrimitiveBuilder.GetTxnCounterManager(), tableDTO.GetDiscoveryID())
					primaryTcc = tcc
				} else {
					tcc = tcc.CloneAndIncrementInsertID()
					tcc.DiscoveryGenerationId = discoGenId
					secondaryTccs = append(secondaryTccs, tcc)
				}
				insPsc, err := p.PrimitiveBuilder.GetDRMConfig().GenerateInsertDML(anTab, tcc)
				if err != nil {
					return err
				}
				builder := primitivebuilder.NewSingleSelectAcquire(p.PrimitiveBuilder.GetGraph(), handlerCtx, v.TableMeta, insPsc, nil)
				execSlice = append(execSlice, builder)
				tableSlice = append(tableSlice, v.TableMeta)
			}
			rewrittenWhereStr := astvisit.GenerateModifiedWhereClause(rewrittenWhere)
			log.Debugf("rewrittenWhereStr = '%s'", rewrittenWhereStr)
			v := astvisit.NewQueryRewriteAstVisitor(
				handlerCtx,
				tblz,
				tableSlice,
				annotations,
				discoGenIDs,
				colRefs,
				drm.GetGoogleV1SQLiteConfig(),
				primaryTcc,
				secondaryTccs,
				rewrittenWhereStr,
			)
			err = v.Visit(pbi.GetStatement())
			if err != nil {
				return err
			}
			selCtx, err := v.GenerateSelectDML()
			if err != nil {
				return err
			}
			selBld := primitivebuilder.NewSingleSelect(p.PrimitiveBuilder.GetGraph(), handlerCtx, selCtx, nil)
			bld := primitivebuilder.NewMultipleAcquireAndSelect(p.PrimitiveBuilder.GetGraph(), execSlice, selBld)
			pChild.PrimitiveBuilder.SetBuilder(bld)
			p.PrimitiveBuilder.SetSelectPreparedStatementCtx(selCtx)
			return nil
		case *sqlparser.ExecSubquery:
			cols, err := parserutil.ExtractSelectColumnNames(node)
			if err != nil {
				return err
			}
			tbl, err := pChild.analyzeUnaryExec(handlerCtx, ft.Exec, node, cols)
			if err != nil {
				return err
			}
			pChild.PrimitiveBuilder.SetBuilder(primitivebuilder.NewSingleAcquireAndSelect(pChild.PrimitiveBuilder.GetGraph(), pChild.PrimitiveBuilder.GetTxnCtrlCtrs(), handlerCtx, tbl, pChild.PrimitiveBuilder.GetInsertPreparedStatementCtx(), pChild.PrimitiveBuilder.GetSelectPreparedStatementCtx(), nil))
			return nil
		}

	}
	return fmt.Errorf("cannot process complex select just yet")
}

func (p *primitiveGenerator) buildRequestContext(handlerCtx *handler.HandlerContext, node sqlparser.SQLNode, meta *taxonomy.ExtendedTableMetadata, execContext *httpbuild.ExecContext, rowsToInsert map[int]map[int]interface{}) (*httpbuild.HTTPArmoury, error) {
	m, err := meta.GetMethod()
	if err != nil {
		return nil, err
	}
	prov, err := meta.GetProvider()
	if err != nil {
		return nil, err
	}
	svc, err := meta.GetService()
	if err != nil {
		return nil, err
	}
	httpArmoury, err := httpbuild.BuildHTTPRequestCtx(handlerCtx, node, prov, m, svc, rowsToInsert, execContext)
	if err != nil {
		return nil, err
	}
	meta.HttpArmoury = httpArmoury
	return httpArmoury, err
}

func (p *primitiveGenerator) analyzeInsert(pbi PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetInsert()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Insert", pbi.GetStatement())
	}
	err := p.inferHeirarchyAndPersist(handlerCtx, node, pbi.paramsPlaceheld.GetStringified())
	if err != nil {
		return err
	}
	tbl, err := p.PrimitiveBuilder.GetTable(node)
	if err != nil {
		return err
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
	insertValOnlyRows, nonValCols, err := parserutil.ExtractInsertValColumnsPlusPlaceHolders(node)
	if err != nil {
		return err
	}
	p.PrimitiveBuilder.SetInsertValOnlyRows(insertValOnlyRows)
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
	}

	p.parseComments(node.Comments)

	method, err := tbl.GetMethod()
	if err != nil {
		return err
	}

	if p.PrimitiveBuilder.IsAwait() && !method.IsAwaitable() {
		return fmt.Errorf("method %s is not awaitable", method.GetName())
	}

	_, err = checkResource(handlerCtx, prov, currentService, currentResource)
	if err != nil {
		return err
	}

	_, err = p.buildRequestContext(handlerCtx, node, tbl, nil, insertValOnlyRows)
	if err != nil {
		return err
	}
	p.PrimitiveBuilder.SetTable(node, tbl)
	return nil
}

func (p *primitiveGenerator) inferHeirarchyAndPersist(handlerCtx *handler.HandlerContext, node sqlparser.SQLNode, parameters map[string]interface{}) error {
	heirarchy, _, err := taxonomy.GetHeirarchyFromStatement(handlerCtx, node, parameters)
	if err != nil {
		return err
	}
	p.persistHerarchyToBuilder(heirarchy, node)
	return err
}

func (p *primitiveGenerator) analyzeDelete(pbi PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetDelete()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Delete", pbi.GetStatement())
	}
	p.parseComments(node.Comments)
	paramMap := astvisit.ExtractParamsFromWhereClause(node.Where)

	err := p.inferHeirarchyAndPersist(handlerCtx, node, paramMap.GetStringified())
	if err != nil {
		return err
	}
	tbl, err := p.PrimitiveBuilder.GetTable(node)
	if err != nil {
		return err
	}
	prov, err := tbl.GetProvider()
	if err != nil {
		return err
	}
	method, err := tbl.GetMethod()
	if err != nil {
		return err
	}

	if p.PrimitiveBuilder.IsAwait() && !method.IsAwaitable() {
		return fmt.Errorf("method %s is not awaitable", method.GetName())
	}
	if p.PrimitiveBuilder.IsAwait() && !method.IsAwaitable() {
		return fmt.Errorf("method %s is not awaitable", method.GetName())
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
	schema, _, err := method.GetResponseBodySchemaAndMediaType()
	if err != nil {
		log.Infof("no response schema for delete: %s \n", err.Error())
	}
	if schema != nil {
		_, _, whereErr := p.analyzeWhere(node.Where)
		if whereErr != nil {
			return whereErr
		}
	}
	colPrefix := prov.GetDefaultKeyForDeleteItems() + "[]."
	whereNames, err := parserutil.ExtractWhereColNames(node.Where)
	if err != nil {
		return err
	}
	for _, w := range whereNames {
		ok := method.KeyExists(w)
		if ok {
			continue
		}
		if schema == nil {
			return fmt.Errorf("cannot locate parameter '%s'", w)
		}
		log.Infoln(fmt.Sprintf("w = '%s'", w))
		foundSchemaPrefixed := schema.FindByPath(colPrefix+w, nil)
		foundSchema := schema.FindByPath(w, nil)
		if foundSchemaPrefixed == nil && foundSchema == nil {
			return fmt.Errorf("DELETE Where element = '%s' is NOT present in data returned from provider", w)
		}
	}
	if err != nil {
		return err
	}
	_, err = p.buildRequestContext(handlerCtx, node, tbl, nil, nil)
	if err != nil {
		return err
	}
	p.PrimitiveBuilder.SetTable(node, tbl)
	return err
}

func (p *primitiveGenerator) analyzeDescribe(pbi PlanBuilderInput) error {
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetDescribeTable()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Describe", pbi.GetStatement())
	}
	var err error
	err = p.inferHeirarchyAndPersist(handlerCtx, node, nil)
	if err != nil {
		return err
	}
	tbl, err := p.PrimitiveBuilder.GetTable(node)
	if err != nil {
		return err
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

func (p *primitiveGenerator) analyzeSleep(pbi PlanBuilderInput) error {
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
	graph := p.PrimitiveBuilder.GetGraph()
	p.PrimitiveBuilder.SetRoot(
		graph.CreatePrimitiveNode(
			primitivebuilder.NewLocalPrimitive(
				func(pc primitive.IPrimitiveCtx) dto.ExecutorOutput {
					time.Sleep(time.Duration(sleepDuration) * time.Millisecond)
					msgs := dto.BackendMessages{
						WorkingMessages: []string{
							fmt.Sprintf("Success: slept for %d milliseconds", sleepDuration),
						},
					}
					return dto.NewExecutorOutput(nil, nil, nil, &msgs, nil)
				},
			),
		),
	)
	return err
}

func (p *primitiveGenerator) analyzeRegistry(pbi PlanBuilderInput) error {
	_, ok := pbi.GetRegistry()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Registry", pbi.GetStatement())
	}
	return nil
}

func (p *primitiveGenerator) analyzeShow(pbi PlanBuilderInput) error {
	var err error
	handlerCtx := pbi.GetHandlerCtx()
	node, ok := pbi.GetShow()
	if !ok {
		return fmt.Errorf("could not cast node of type '%T' to required Show", pbi.GetStatement())
	}
	p.parseComments(node.Comments)
	err = p.inferProviderForShow(node, handlerCtx)
	if err != nil {
		return err
	}
	nodeTypeUpperCase := strings.ToUpper(node.Type)
	if p.PrimitiveBuilder.GetProvider() != nil {
		p.PrimitiveBuilder.SetLikeAbleColumns(p.PrimitiveBuilder.GetProvider().GetLikeableColumns(nodeTypeUpperCase))
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
		err = p.inferHeirarchyAndPersist(handlerCtx, node, nil)
		if err != nil {
			return err
		}
	case "METHODS":
		err = p.inferHeirarchyAndPersist(handlerCtx, node, nil)
		if err != nil {
			return err
		}
		tbl, err := p.PrimitiveBuilder.GetTable(node)
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
		_, err = checkResource(handlerCtx, p.PrimitiveBuilder.GetProvider(), currentService, currentResource)
		if err != nil {
			return err
		}
		if node.ShowTablesOpt != nil {
			meth := &openapistackql.OperationStore{}
			err = p.analyzeShowFilter(node, meth)
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
		p.PrimitiveBuilder.SetProvider(prov)
		_, err = p.assembleResources(handlerCtx, p.PrimitiveBuilder.GetProvider(), node.OnTable.Name.GetRawVal())
		if err != nil {
			return err
		}
		for _, col := range colNames {
			if !openapistackql.ResourceKeyExists(col) {
				return fmt.Errorf("SHOW key = '%s' does NOT exist", col)
			}
		}
		for _, colUsage := range colUsages {
			if !openapistackql.ResourceKeyExists(colUsage.ColName.Name.GetRawVal()) {
				return fmt.Errorf("SHOW key = '%s' does NOT exist", colUsage.ColName.Name.GetRawVal())
			}
			usageErr := parserutil.CheckSqlParserTypeVsResourceColumn(colUsage)
			if usageErr != nil {
				return usageErr
			}
		}
		if node.ShowTablesOpt != nil {
			rsc := &openapistackql.Resource{}
			err = p.analyzeShowFilter(node, rsc)
			if err != nil {
				return err
			}
		}
	case "SERVICES":
		prov, err := handlerCtx.GetProvider(node.OnTable.Name.GetRawVal())
		if err != nil {
			return err
		}
		p.PrimitiveBuilder.SetProvider(prov)
		for _, col := range colNames {
			if !openapistackql.ServiceKeyExists(col) {
				return fmt.Errorf("SHOW key = '%s' does NOT exist", col)
			}
		}
		for _, colUsage := range colUsages {
			if !openapistackql.ServiceKeyExists(colUsage.ColName.Name.GetRawVal()) {
				return fmt.Errorf("SHOW key = '%s' does NOT exist", colUsage.ColName.Name.GetRawVal())
			}
			usageErr := parserutil.CheckSqlParserTypeVsServiceColumn(colUsage)
			if usageErr != nil {
				return usageErr
			}
		}
		if node.ShowTablesOpt != nil {
			svc := &openapistackql.ProviderService{}
			err = p.analyzeShowFilter(node, svc)
			if err != nil {
				return err
			}
		}
	default:
		err = fmt.Errorf("SHOW statement not supported for '%s'", nodeTypeUpperCase)
	}
	return err
}
