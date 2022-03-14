package planbuilder

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/stackql/stackql/internal/stackql/astvisit"
	"github.com/stackql/stackql/internal/stackql/constants"
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
	"github.com/stackql/stackql/internal/stackql/symtab"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/go-openapistackql/openapistackql"

	"vitess.io/vitess/go/vt/sqlparser"

	log "github.com/sirupsen/logrus"
)

func (p *primitiveGenerator) analyzeStatement(handlerCtx *handler.HandlerContext, statement sqlparser.SQLNode) error {
	var err error
	switch stmt := statement.(type) {
	case *sqlparser.Auth:
		return p.analyzeAuth(handlerCtx, stmt)
	case *sqlparser.AuthRevoke:
		return p.analyzeAuthRevoke(handlerCtx, stmt)
	case *sqlparser.Begin:
		return iqlerror.GetStatementNotSupportedError("TRANSACTION: BEGIN")
	case *sqlparser.Commit:
		return iqlerror.GetStatementNotSupportedError("TRANSACTION: COMMIT")
	case *sqlparser.DBDDL:
		return iqlerror.GetStatementNotSupportedError(fmt.Sprintf("unsupported: Database DDL %v", sqlparser.String(stmt)))
	case *sqlparser.DDL:
		return iqlerror.GetStatementNotSupportedError("DDL")
	case *sqlparser.Delete:
		return p.analyzeDelete(handlerCtx, stmt)
	case *sqlparser.DescribeTable:
		return p.analyzeDescribe(handlerCtx, stmt)
	case *sqlparser.Exec:
		return p.analyzeExec(handlerCtx, stmt)
	case *sqlparser.Explain:
		return iqlerror.GetStatementNotSupportedError("EXPLAIN")
	case *sqlparser.Insert:
		return p.analyzeInsert(handlerCtx, stmt)
	case *sqlparser.OtherRead, *sqlparser.OtherAdmin:
		return iqlerror.GetStatementNotSupportedError("OTHER")
	case *sqlparser.Registry:
		return p.analyzeRegistry(handlerCtx, stmt)
	case *sqlparser.Rollback:
		return iqlerror.GetStatementNotSupportedError("TRANSACTION: ROLLBACK")
	case *sqlparser.Savepoint:
		return iqlerror.GetStatementNotSupportedError("TRANSACTION: SAVEPOINT")
	case *sqlparser.Select:
		return p.analyzeSelect(handlerCtx, stmt)
	case *sqlparser.Set:
		return iqlerror.GetStatementNotSupportedError("SET")
	case *sqlparser.SetTransaction:
		return iqlerror.GetStatementNotSupportedError("SET TRANSACTION")
	case *sqlparser.Show:
		return p.analyzeShow(handlerCtx, stmt)
	case *sqlparser.Sleep:
		return p.analyzeSleep(handlerCtx, stmt)
	case *sqlparser.SRollback:
		return iqlerror.GetStatementNotSupportedError("TRANSACTION: SROLLBACK")
	case *sqlparser.Release:
		return iqlerror.GetStatementNotSupportedError("TRANSACTION: RELEASE")
	case *sqlparser.Union:
		return p.analyzeUnion(handlerCtx, stmt)
	case *sqlparser.Update:
		return iqlerror.GetStatementNotSupportedError("UPDATE")
	case *sqlparser.Use:
		return p.analyzeUse(handlerCtx, stmt)
	}
	return err
}

func (p *primitiveGenerator) analyzeUse(handlerCtx *handler.HandlerContext, node *sqlparser.Use) error {
	prov, pErr := handlerCtx.GetProvider(node.DBName.GetRawVal())
	if pErr != nil {
		return pErr
	}
	p.PrimitiveBuilder.SetProvider(prov)
	return nil
}

func (p *primitiveGenerator) analyzeUnion(handlerCtx *handler.HandlerContext, node *sqlparser.Union) error {
	unionQuery := astvisit.GenerateUnionTemplateQuery(node)
	i := 0
	leaf, err := p.PrimitiveBuilder.GetSymTab().NewLeaf(i)
	if err != nil {
		return err
	}
	pChild := p.addChildPrimitiveGenerator(node.FirstStatement, leaf)
	err = pChild.analyzeSelectStatement(handlerCtx, node.FirstStatement)
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
		err = pChild.analyzeSelectStatement(handlerCtx, rhsStmt.Statement)
		if err != nil {
			return err
		}
		ctx := pChild.PrimitiveBuilder.GetSelectPreparedStatementCtx()
		ctx.Kind = rhsStmt.Type
		selectStatementContexts = append(selectStatementContexts, ctx)
	}

	bldr := primitivebuilder.NewUnion(
		p.PrimitiveBuilder,
		handlerCtx,
		drm.NewQueryOnlyPreparedStatementCtx(unionQuery),
		pChild.PrimitiveBuilder.GetSelectPreparedStatementCtx(),
		selectStatementContexts,
	)
	p.PrimitiveBuilder.SetBuilder(bldr)

	return nil
}

func (p *primitiveGenerator) analyzeSelectStatement(handlerCtx *handler.HandlerContext, node sqlparser.SelectStatement) error {
	switch node := node.(type) {
	case *sqlparser.Select:
		return p.analyzeSelect(handlerCtx, node)
	case *sqlparser.ParenSelect:
		return p.analyzeSelectStatement(handlerCtx, node.Select)
	case *sqlparser.Union:
		return p.analyzeUnion(handlerCtx, node)
	}
	return nil
}

func (p *primitiveGenerator) analyzeAuth(handlerCtx *handler.HandlerContext, node *sqlparser.Auth) error {
	provider, pErr := handlerCtx.GetProvider(node.Provider)
	if pErr != nil {
		return pErr
	}
	p.PrimitiveBuilder.SetProvider(provider)
	return nil
}

func (p *primitiveGenerator) analyzeAuthRevoke(handlerCtx *handler.HandlerContext, node *sqlparser.AuthRevoke) error {
	authCtx, authErr := handlerCtx.GetAuthContext(node.Provider)
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

// DEPRECATED
func (pb *primitiveGenerator) traverseWhereFilterDeprecated(table *openapistackql.OperationStore, node sqlparser.Expr, schema *openapistackql.Schema, requiredParameters map[string]*openapistackql.Parameter) (func(openapistackql.ITable) (openapistackql.ITable, error), error) {
	var retVal func(openapistackql.ITable) (openapistackql.ITable, error)
	switch node := node.(type) {
	case *sqlparser.ComparisonExpr:
		return pb.whereComparisonExprToFilterFunc(node, table, schema, requiredParameters)
	case *sqlparser.AndExpr:
		log.Infoln("complex AND expr detected")
		lhs, lhErr := pb.traverseWhereFilterDeprecated(table, node.Left, schema, requiredParameters)
		rhs, rhErr := pb.traverseWhereFilterDeprecated(table, node.Right, schema, requiredParameters)
		if lhErr != nil {
			return nil, lhErr
		}
		if rhErr != nil {
			return nil, rhErr
		}
		return relational.AndTableFilters(lhs, rhs), nil
	case *sqlparser.OrExpr:
		log.Infoln("complex OR expr detected")
		lhs, lhErr := pb.traverseWhereFilterDeprecated(table, node.Left, schema, requiredParameters)
		rhs, rhErr := pb.traverseWhereFilterDeprecated(table, node.Right, schema, requiredParameters)
		if lhErr != nil {
			return nil, lhErr
		}
		if rhErr != nil {
			return nil, rhErr
		}
		return relational.OrTableFilters(lhs, rhs), nil
	case *sqlparser.FuncExpr:
		return nil, fmt.Errorf("unsupported constraint in openapistackql filter: %v", sqlparser.String(node))
	default:
		return nil, fmt.Errorf("unsupported constraint in openapistackql filter: %v", sqlparser.String(node))
	}
	return retVal, nil
}

func (pb *primitiveGenerator) traverseWhereFilter(node sqlparser.SQLNode, requiredParameters, optionalParameters map[string]*openapistackql.Parameter) (sqlparser.Expr, error) {
	switch node := node.(type) {
	case *sqlparser.ComparisonExpr:
		return pb.whereComparisonExprCopyAndReWrite(node, requiredParameters, optionalParameters)
	case *sqlparser.AndExpr:
		log.Infoln("complex AND expr detected")
		lhs, lhErr := pb.traverseWhereFilter(node.Left, requiredParameters, optionalParameters)
		rhs, rhErr := pb.traverseWhereFilter(node.Right, requiredParameters, optionalParameters)
		if lhErr != nil {
			return nil, lhErr
		}
		if rhErr != nil {
			return nil, rhErr
		}
		return &sqlparser.AndExpr{Left: lhs, Right: rhs}, nil
	case *sqlparser.OrExpr:
		log.Infoln("complex OR expr detected")
		lhs, lhErr := pb.traverseWhereFilter(node.Left, requiredParameters, optionalParameters)
		rhs, rhErr := pb.traverseWhereFilter(node.Right, requiredParameters, optionalParameters)
		if lhErr != nil {
			return nil, lhErr
		}
		if rhErr != nil {
			return nil, rhErr
		}
		return &sqlparser.OrExpr{Left: lhs, Right: rhs}, nil
	case *sqlparser.FuncExpr:
		return nil, fmt.Errorf("unsupported constraint in openapistackql filter: %v", sqlparser.String(node))
	case *sqlparser.IsExpr:
		return &sqlparser.IsExpr{
			Operator: node.Operator,
			Expr:     node.Expr,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported constraint in openapistackql filter: %v", sqlparser.String(node))
	}
	return nil, fmt.Errorf("unsupported constraint in openapistackql filter: %v", sqlparser.String(node))
}

func (pb *primitiveGenerator) whereComparisonExprCopyAndReWrite(expr *sqlparser.ComparisonExpr, requiredParameters, optionalParameters map[string]*openapistackql.Parameter) (sqlparser.Expr, error) {
	qualifiedName, ok := expr.Left.(*sqlparser.ColName)
	if !ok {
		return nil, fmt.Errorf("unexpected: %v", sqlparser.String(expr))
	}
	colName := qualifiedName.Name.GetRawVal()
	symTabEntry, symTabErr := pb.PrimitiveBuilder.GetSymbol(colName)
	_, requiredParamPresent := requiredParameters[colName]
	_, optionalParamPresent := optionalParameters[colName]
	log.Infoln(fmt.Sprintf("symTabEntry = %v", symTabEntry))
	if symTabErr != nil && !(requiredParamPresent || optionalParamPresent) {
		return nil, symTabErr
	}
	if requiredParamPresent {
		delete(requiredParameters, colName)
	}
	if optionalParamPresent {
		delete(optionalParameters, colName)
	}
	if symTabErr == nil && symTabEntry.In != "server" {
		if !(requiredParamPresent || optionalParamPresent) {
			return &sqlparser.ComparisonExpr{
				Left:     expr.Left,
				Right:    expr.Right,
				Operator: expr.Operator,
				Escape:   expr.Escape,
			}, nil
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
		}, nil
	}
	return &sqlparser.ComparisonExpr{
		Left:     &sqlparser.SQLVal{Type: sqlparser.IntVal, Val: []byte("1")},
		Right:    &sqlparser.SQLVal{Type: sqlparser.IntVal, Val: []byte("1")},
		Operator: expr.Operator,
		Escape:   expr.Escape,
	}, nil
}

// DEPRECATED
func (pb *primitiveGenerator) whereComparisonExprToFilterFunc(expr *sqlparser.ComparisonExpr, table *openapistackql.OperationStore, schema *openapistackql.Schema, requiredParameters map[string]*openapistackql.Parameter) (func(openapistackql.ITable) (openapistackql.ITable, error), error) {
	qualifiedName, ok := expr.Left.(*sqlparser.ColName)
	if !ok {
		return nil, fmt.Errorf("unexpected: %v", sqlparser.String(expr))
	}
	colName := qualifiedName.Name.GetRawVal()
	tableContainsKey := table.KeyExists(colName)
	var subSchema *openapistackql.Schema
	if schema != nil {
		subSchema = schema.FindByPath(colName, nil)
	}
	if !tableContainsKey && subSchema == nil {
		return nil, fmt.Errorf("col name = '%s' not found in resource name = '%s'", colName, table.GetName())
	}
	delete(requiredParameters, colName)
	if tableContainsKey && subSchema != nil && !subSchema.ReadOnly {
		log.Infoln(fmt.Sprintf("tableContainsKey && subSchema = %v", subSchema))
		return nil, fmt.Errorf("col name = '%s' ambiguous for resource name = '%s'", colName, table.GetName())
	}
	val, ok := expr.Right.(*sqlparser.SQLVal)
	if !ok {
		return nil, fmt.Errorf("unexpected: %v", sqlparser.String(expr))
	}
	//StrVal is varbinary, we do not support varchar since we would have to implement all collation types
	if val.Type != sqlparser.IntVal && val.Type != sqlparser.StrVal {
		return nil, fmt.Errorf("unexpected: %v", sqlparser.String(expr))
	}
	pv, err := sqlparser.NewPlanValue(val)
	if err != nil {
		return nil, err
	}
	resolved, err := pv.ResolveValue(nil)
	log.Debugln(fmt.Sprintf("resolved = %v", resolved))
	if err != nil {
		return nil, err
	}
	return nil, nil
}

// DEPRECATED
func (pb *primitiveGenerator) analyzeSingleTableWhere(where *sqlparser.Where, schema *openapistackql.Schema) error {
	remainingRequiredParameters := make(map[string]*openapistackql.Parameter)
	for _, v := range pb.PrimitiveBuilder.GetTables() {
		method, err := v.GetMethod()
		if err != nil {
			return err
		}
		requiredParameters := method.GetRequiredParameters()
		if where != nil {
			pb.traverseWhereFilterDeprecated(method, where.Expr, schema, requiredParameters)
		}
		for l, w := range requiredParameters {
			rscStr, _ := v.GetResourceStr()
			remainingRequiredParameters[fmt.Sprintf("%s.%s", rscStr, l)] = w
		}
		var colUsages []parserutil.ColumnUsageMetadata
		if where != nil {
			colUsages, err = parserutil.GetColumnUsageTypes(where.Expr)
		}
		if err != nil {
			return err
		}
		err = parserutil.CheckColUsagesAgainstTable(colUsages, method)
		if err != nil {
			return err
		}
	}
	if len(remainingRequiredParameters) > 0 {
		var keys []string
		for k := range remainingRequiredParameters {
			keys = append(keys, k)
		}
		return fmt.Errorf("Query cannot be executed, missing required parameters: { %s }", strings.Join(keys, ", "))
	}
	return nil
}

func (pb *primitiveGenerator) analyzeWhere(where *sqlparser.Where, schema *openapistackql.Schema) (*sqlparser.Where, error) {
	requiredParameters := make(map[string]*openapistackql.Parameter)
	remainingRequiredParameters := make(map[string]*openapistackql.Parameter)
	optionalParameters := make(map[string]*openapistackql.Parameter)
	for _, v := range pb.PrimitiveBuilder.GetTables() {
		method, err := v.GetMethod()
		if err != nil {
			return nil, err
		}
		for k, v := range method.GetRequiredParameters() {
			_, keyExists := requiredParameters[k]
			if keyExists {
				return nil, fmt.Errorf("key already is required: %s", k)
			}
			requiredParameters[k] = v
		}
		for k, v := range method.GetOptionalParameters() {
			_, keyExists := optionalParameters[k]
			if keyExists {
				return nil, fmt.Errorf("key already is optional: %s", k)
			}
			optionalParameters[k] = v
		}
	}
	var retVal sqlparser.Expr
	var err error
	if where != nil {
		retVal, err = pb.traverseWhereFilter(where.Expr, requiredParameters, optionalParameters)
		if err != nil {
			return nil, err
		}
	}

	for l, w := range requiredParameters {
		remainingRequiredParameters[fmt.Sprintf("%s", l)] = w
	}

	if len(remainingRequiredParameters) > 0 {
		if where == nil {
			return nil, fmt.Errorf("WHERE clause not supplied, run DESCRIBE EXTENDED for the resource to see required parameters")
		}
		var keys []string
		for k := range remainingRequiredParameters {
			keys = append(keys, k)
		}
		return nil, fmt.Errorf("Query cannot be executed, missing required parameters: { %s }", strings.Join(keys, ", "))
	}
	if where == nil {
		return nil, nil
	}
	return &sqlparser.Where{Type: where.Type, Expr: retVal}, nil
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
	err := p.inferHeirarchyAndPersist(handlerCtx, node)
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
		execPayload, err = p.parseExecPayload(node.OptExecPayload, method.Request.BodyMediaType)
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
	_, err = p.buildRequestContext(handlerCtx, node, &meta, httpbuild.NewExecContext(execPayload, rsc), nil)
	if err != nil {
		return nil, err
	}
	p.PrimitiveBuilder.SetTable(node, meta)

	// parse response with SQL
	if method.IsNullary() && !p.PrimitiveBuilder.IsAwait() {
		return &meta, nil
	}
	if selectNode != nil {
		return &meta, p.analyzeUnarySelection(handlerCtx, selectNode, selectNode.Where, &meta, cols)
	}
	return &meta, p.analyzeUnarySelection(handlerCtx, node, nil, &meta, cols)
}

func (p *primitiveGenerator) analyzeExec(handlerCtx *handler.HandlerContext, node *sqlparser.Exec) error {
	tbl, err := p.analyzeUnaryExec(handlerCtx, node, nil, nil)
	if err != nil {
		log.Infoln(fmt.Sprintf("error analyzing EXEC as selection: '%s'", err.Error()))
	} else {
		m, err := tbl.GetMethod()
		if err != nil {
			return err
		}
		if m.IsNullary() && !p.PrimitiveBuilder.IsAwait() {
			p.PrimitiveBuilder.SetBuilder(primitivebuilder.NewSingleSelectAcquire(p.PrimitiveBuilder, handlerCtx, *tbl, p.PrimitiveBuilder.GetInsertPreparedStatementCtx(), p.PrimitiveBuilder.GetSelectPreparedStatementCtx(), nil))
			return nil
		}
		p.PrimitiveBuilder.SetBuilder(primitivebuilder.NewSingleAcquireAndSelect(p.PrimitiveBuilder, handlerCtx, *tbl, p.PrimitiveBuilder.GetInsertPreparedStatementCtx(), p.PrimitiveBuilder.GetSelectPreparedStatementCtx(), nil))
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

func (p *primitiveGenerator) analyzeSelect(handlerCtx *handler.HandlerContext, node *sqlparser.Select) error {

	for i, fromExpr := range node.From {
		var leafKey interface{} = i
		switch tbl := fromExpr.(type) {
		case *sqlparser.AliasedTableExpr:
			if tbl.As.GetRawVal() != "" {
				leafKey = tbl.As.GetRawVal()
			}
		}
		leaf, err := p.PrimitiveBuilder.GetSymTab().NewLeaf(leafKey)
		if err != nil {
			return err
		}
		pChild := p.addChildPrimitiveGenerator(fromExpr, leaf)
		var tbl *taxonomy.ExtendedTableMetadata
		switch from := fromExpr.(type) {
		case *sqlparser.ExecSubquery:
			log.Infoln(fmt.Sprintf("from = %v", from))
			tbl, err = pChild.analyzeTableExpr(handlerCtx, from)
		default:
			tbl, err = pChild.analyzeTableExpr(handlerCtx, from)
		}

		if err != nil {
			return err
		}
		svc, err := tbl.GetService()
		if err != nil {
			return err
		}
		// svc.Servers
		for _, sv := range svc.Servers {
			for k := range sv.Variables {
				colEntry := symtab.NewSymTabEntry(
					pChild.PrimitiveBuilder.GetDRMConfig().GetRelationalType("string"),
					"",
					"server",
				)
				pChild.PrimitiveBuilder.SetSymbol(k, colEntry)
			}
			break
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
			pChild.PrimitiveBuilder.SetSymbol(colName, colEntry)
		}
		if len(node.From) == 1 {
			switch ft := node.From[0].(type) {
			case *sqlparser.JoinTableExpr:
				tbl, err := pChild.analyzeTableExpr(handlerCtx, ft.LeftExpr)
				if err != nil {
					return err
				}
				err = pChild.analyzeSelectDetail(handlerCtx, node, tbl)
				if err != nil {
					return err
				}
				rhsPb := newRootPrimitiveGenerator(pChild.PrimitiveBuilder.GetAst(), handlerCtx, pChild.PrimitiveBuilder.GetGraph())
				tbl, err = rhsPb.analyzeTableExpr(handlerCtx, ft.RightExpr)
				if err != nil {
					return err
				}
				err = rhsPb.analyzeSelectDetail(handlerCtx, node, tbl)
				if err != nil {
					return err
				}
				pChild.PrimitiveBuilder.SetBuilder(primitivebuilder.NewJoin(pChild.PrimitiveBuilder, rhsPb.PrimitiveBuilder, handlerCtx, nil))
				return nil
			case *sqlparser.AliasedTableExpr:
				tbl, err := pChild.analyzeTableExpr(handlerCtx, node.From[0])
				if err != nil {
					return err
				}
				err = pChild.analyzeSelectDetail(handlerCtx, node, tbl)
				if err != nil {
					return err
				}
				pChild.PrimitiveBuilder.SetBuilder(primitivebuilder.NewSingleAcquireAndSelect(pChild.PrimitiveBuilder, handlerCtx, *tbl, pChild.PrimitiveBuilder.GetInsertPreparedStatementCtx(), pChild.PrimitiveBuilder.GetSelectPreparedStatementCtx(), nil))
				p.PrimitiveBuilder.SetSelectPreparedStatementCtx(pChild.PrimitiveBuilder.GetSelectPreparedStatementCtx())
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

				pChild.PrimitiveBuilder.SetBuilder(primitivebuilder.NewSingleAcquireAndSelect(pChild.PrimitiveBuilder, handlerCtx, *tbl, pChild.PrimitiveBuilder.GetInsertPreparedStatementCtx(), pChild.PrimitiveBuilder.GetSelectPreparedStatementCtx(), nil))
				return nil
			}
		}
	}
	return fmt.Errorf("cannot process complex select just yet")
}

func (p *primitiveGenerator) analyzeSelectDetail(handlerCtx *handler.HandlerContext, node *sqlparser.Select, tbl *taxonomy.ExtendedTableMetadata) error {
	var err error
	valOnlyCols, nonValCols := parserutil.ExtractSelectValColumns(node)
	p.PrimitiveBuilder.SetValOnlyCols(valOnlyCols)
	svcStr, _ := tbl.GetServiceStr()
	rStr, _ := tbl.GetResourceStr()
	if rStr == "dual" { // some bizarre artifact of vitess.io, indicates no table supplied
		tbl.IsLocallyExecutable = true
		if svcStr == "" {
			if nonValCols == 0 && node.Where == nil {
				log.Infoln("val only select looks ok")
				return nil
			}
			err = fmt.Errorf("select values inadequate: expected 0 non-val columns but got %d", nonValCols)
		}
		return err
	}
	cols, err := parserutil.ExtractSelectColumnNames(node)
	if err != nil {
		return err
	}

	responseSchema, err := tbl.GetResponseSchema()
	if err != nil {
		return err
	}

	rewrittenWhere, whereErr := p.analyzeWhere(node.Where, responseSchema)
	if whereErr != nil {
		return whereErr
	}
	p.PrimitiveBuilder.SetWhere(rewrittenWhere)

	err = p.analyzeUnarySelection(handlerCtx, node, rewrittenWhere, tbl, cols)
	if err != nil {
		return err
	}

	_, err = tbl.GetProvider()
	if err != nil {
		return err
	}
	method, err := tbl.GetMethod()
	if err != nil {
		return err
	}

	// TODO: get rid of prefix garbage
	colPrefix := tbl.SelectItemsKey + "[]."

	whereNames, err := parserutil.ExtractWhereColNames(node.Where)
	if err != nil {
		return err
	}
	for _, w := range whereNames {
		_, ok := method.Parameters[w]
		if ok {
			continue
		}
		log.Infoln(fmt.Sprintf("w = '%s'", w))
		foundSchemaPrefixed := responseSchema.FindByPath(colPrefix+w, nil)
		foundSchema := responseSchema.FindByPath(w, nil)
		if foundSchemaPrefixed == nil && foundSchema == nil {
			return fmt.Errorf("SELECT Where element = '%s' is NOT present in data returned from provider", w)
		}
	}
	if err != nil {
		return err
	}
	havingNames, err := parserutil.ExtractWhereColNames(node.Having)
	if err != nil {
		return err
	}
	for _, w := range havingNames {
		_, ok := method.Parameters[w]
		if ok {
			continue
		}
		log.Infoln(fmt.Sprintf("w = '%s'", w))
		foundSchemaPrefixed := responseSchema.FindByPath(colPrefix+w, nil)
		foundSchema := responseSchema.FindByPath(w, nil)
		if foundSchemaPrefixed == nil && foundSchema == nil {
			return fmt.Errorf("SELECT HAVING element = '%s' is NOT present in data returned from provider", w)
		}
	}
	if err != nil {
		return err
	}
	_, err = p.buildRequestContext(handlerCtx, node, tbl, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

func (p *primitiveGenerator) analyzeTableExpr(handlerCtx *handler.HandlerContext, node sqlparser.TableExpr) (*taxonomy.ExtendedTableMetadata, error) {
	var nodeToPersist sqlparser.SQLNode = node
	switch node := node.(type) {
	case *sqlparser.ExecSubquery:
		nodeToPersist = node.Exec
	}
	err := p.inferHeirarchyAndPersist(handlerCtx, nodeToPersist)
	if err != nil {
		return nil, err
	}
	tbl, err := p.PrimitiveBuilder.GetTable(nodeToPersist)
	if err != nil {
		return nil, err
	}
	_, err = tbl.GetProvider()
	if err != nil {
		return nil, err
	}
	method, err := tbl.GetMethod()
	if err != nil {
		return nil, err
	}
	_, err = tbl.GetServiceStr()
	if err != nil {
		return nil, err
	}
	_, err = tbl.GetResourceStr()
	if err != nil {
		return nil, err
	}
	schema := method.Response.Schema
	unsuitableSchemaMsg := "schema unsuitable for select query"
	// log.Infoln(fmt.Sprintf("schema.ID = %v", schema.ID))
	log.Infoln(fmt.Sprintf("schema.Items = %v", schema.Items))
	log.Infoln(fmt.Sprintf("schema.Properties = %v", schema.Properties))
	var itemObjS *openapistackql.Schema
	itemObjS, tbl.SelectItemsKey, err = schema.GetSelectSchema(tbl.LookupSelectItemsKey())
	if itemObjS == nil || err != nil {
		return nil, fmt.Errorf(unsuitableSchemaMsg)
	}
	return &tbl, nil
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

func (p *primitiveGenerator) analyzeInsert(handlerCtx *handler.HandlerContext, node *sqlparser.Insert) error {
	err := p.inferHeirarchyAndPersist(handlerCtx, node)
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

	_, err = p.buildRequestContext(handlerCtx, node, &tbl, nil, insertValOnlyRows)
	if err != nil {
		return err
	}
	p.PrimitiveBuilder.SetTable(node, tbl)
	return nil
}

func (p *primitiveGenerator) inferHeirarchyAndPersist(handlerCtx *handler.HandlerContext, node sqlparser.SQLNode) error {
	heirarchy, err := taxonomy.GetHeirarchyFromStatement(handlerCtx, node)
	if err != nil {
		return err
	}
	p.persistHerarchyToBuilder(heirarchy, node)
	return err
}

func (p *primitiveGenerator) analyzeDelete(handlerCtx *handler.HandlerContext, node *sqlparser.Delete) error {
	p.parseComments(node.Comments)
	err := p.inferHeirarchyAndPersist(handlerCtx, node)
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
	schema, err := method.GetResponseBodySchema()
	if err != nil {
		log.Infof("no response schema for delete: %s \n", err.Error())
	}
	if schema != nil {
		whereErr := p.analyzeSingleTableWhere(node.Where, schema)
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
	_, err = p.buildRequestContext(handlerCtx, node, &tbl, nil, nil)
	if err != nil {
		return err
	}
	p.PrimitiveBuilder.SetTable(node, tbl)
	return err
}

func (p *primitiveGenerator) analyzeDescribe(handlerCtx *handler.HandlerContext, node *sqlparser.DescribeTable) error {
	var err error
	err = p.inferHeirarchyAndPersist(handlerCtx, node)
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

func (p *primitiveGenerator) analyzeSleep(handlerCtx *handler.HandlerContext, node *sqlparser.Sleep) error {
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

func (p *primitiveGenerator) analyzeRegistry(handlerCtx *handler.HandlerContext, node *sqlparser.Registry) error {
	var err error
	return err
}

func (p *primitiveGenerator) analyzeShow(handlerCtx *handler.HandlerContext, node *sqlparser.Show) error {
	var err error
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
		err = p.inferHeirarchyAndPersist(handlerCtx, node)
		if err != nil {
			return err
		}
	case "METHODS":
		err = p.inferHeirarchyAndPersist(handlerCtx, node)
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
