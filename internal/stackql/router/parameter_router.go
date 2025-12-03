package router

import (
	"fmt"

	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/any-sdk/pkg/logging"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/astanalysis/annotatedast"
	"github.com/stackql/stackql/internal/stackql/astindirect"
	"github.com/stackql/stackql/internal/stackql/astvisit"
	"github.com/stackql/stackql/internal/stackql/dataflow"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/router/obtain_context"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"github.com/stackql/stackql/internal/stackql/tablenamespace"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
)

var (
	_ ParameterRouter = &standardParameterRouter{}
)

// Parameter router supports
// mapping columnar input to
// tabular output.
// This is for dealing with parser objects, prior to assignment
// of openapi schemas.
// The storage medium for constituents is abstracted.
// As of now this is a multi stage object, violates Single functionality.
type ParameterRouter interface {

	// Obtains parameters that are unammbiguous (eg: aliased, unique col name)
	// or potential matches for a supplied table.
	// getAvailableParameters(tb sqlparser.TableExpr) *parserutil.TableParameterCoupling

	// Records the fact that parameters have been assigned to a table and
	// cannot be used elsewhere.
	// splitParams(params map[string]interface{}) error

	// First pass assignment of columnar objects
	// to tables, only for HTTP method parameters.  All data accrual is done herein:
	//   - SQL parser table objects mapped to hierarchy.
	//   - Data flow dependencies identified and persisted.
	//   - Hierarchies may be persisted for analysis.
	// Detects bi-directional data flow errors and returns error if so.
	// Returns:
	//   - Hierarchy.
	//   - Columnar objects definitely assigned as HTTP method parameters.
	//   - Error if applicable.
	Route(tb sqlparser.TableExpr, handler handler.HandlerContext, isAwait bool) (taxonomy.AnnotationCtx, error)

	// Detects:
	//   - Dependency cycle.
	AnalyzeDependencies() error

	GetOnConditionsToRewrite() map[*sqlparser.ComparisonExpr]struct{}

	GetOnConditionDataFlows() (dataflow.Collection, error)
}

type standardParameterRouter struct {
	annotatedAST                  annotatedast.AnnotatedAst
	tablesAliasMap                parserutil.TableAliasMap
	tableMap                      parserutil.TableExprMap
	onParamMap                    parserutil.ParameterMap
	whereParamMap                 parserutil.ParameterMap
	colRefs                       parserutil.ColTableMap
	comparisonToTableDependencies parserutil.ComparisonTableMap
	tableToComparisonDependencies parserutil.ComparisonTableMap
	tableToAnnotationCtx          map[sqlparser.TableExpr]taxonomy.AnnotationCtx
	invalidatedParams             map[string]interface{}
	namespaceCollection           tablenamespace.Collection
	astFormatter                  sqlparser.NodeFormatter
	dataFlowCfg                   dto.DataFlowCfg
}

func NewParameterRouter(
	annotatedAST annotatedast.AnnotatedAst,
	tablesAliasMap parserutil.TableAliasMap,
	tableMap parserutil.TableExprMap,
	whereParamMap parserutil.ParameterMap,
	onParamMap parserutil.ParameterMap,
	colRefs parserutil.ColTableMap,
	namespaceCollection tablenamespace.Collection,
	astFormatter sqlparser.NodeFormatter,
	dataflowCfg dto.DataFlowCfg,
) ParameterRouter {
	return &standardParameterRouter{
		tablesAliasMap:                tablesAliasMap,
		tableMap:                      tableMap,
		whereParamMap:                 whereParamMap,
		onParamMap:                    onParamMap,
		colRefs:                       colRefs,
		invalidatedParams:             make(map[string]interface{}),
		comparisonToTableDependencies: make(parserutil.ComparisonTableMap),
		tableToComparisonDependencies: make(parserutil.ComparisonTableMap),
		tableToAnnotationCtx:          make(map[sqlparser.TableExpr]taxonomy.AnnotationCtx),
		namespaceCollection:           namespaceCollection,
		astFormatter:                  astFormatter,
		annotatedAST:                  annotatedAST,
		dataFlowCfg:                   dataflowCfg,
	}
}

func (pr *standardParameterRouter) AnalyzeDependencies() error {
	return nil
}

// This has been obviated for the time being.
// Essentially, not required so long as:
//   - in params
//   - result set data
//
// ...are guaranteed present in same table.
func (pr *standardParameterRouter) GetOnConditionsToRewrite() map[*sqlparser.ComparisonExpr]struct{} {
	rv := make(map[*sqlparser.ComparisonExpr]struct{})
	for k := range pr.comparisonToTableDependencies {
		logging.GetLogger().Debugf("%v\n", k)
	}
	return rv
}

func (pr *standardParameterRouter) extractDataFlowDependency(
	input sqlparser.Expr,
) (taxonomy.AnnotationCtx, sqlparser.TableExpr, error) {
	switch l := input.(type) {
	case *sqlparser.ColName:
		// leave unknown for now -- bit of a mess
		ref, err := parserutil.NewColumnarReference(l, parserutil.UnknownParam)
		if err != nil {
			return nil, nil, err
		}
		tb, ok := pr.colRefs[ref]
		if !ok {
			return nil, nil, fmt.Errorf("unassigned column in ON condition dataflow; please alias column '%s'", l.GetRawVal())
		}
		hr, ok := pr.tableToAnnotationCtx[tb]
		if !ok {
			return nil, nil, fmt.Errorf("cannot assign hierarchy for column '%s'", l.GetRawVal())
		}
		return hr, tb, nil
	default:
		return nil, nil, fmt.Errorf("cannot accomodate ON condition of type = '%T'", l)
	}
}

func (pr *standardParameterRouter) extractFromFunctionExpr(
	f *sqlparser.FuncExpr,
) (taxonomy.AnnotationCtx, sqlparser.TableExpr, error) {
	sv := astvisit.NewLeftoverReferencesAstVisitor(
		pr.annotatedAST,
		pr.colRefs,
		pr.tableToAnnotationCtx,
	)
	sv.Visit(f) //nolint:errcheck // TODO: review
	tbz := sv.GetTablesFoundThisIteration()
	if len(tbz) != 1 {
		return nil, nil, fmt.Errorf("cannot accomodate this")
	}
	for k, v := range tbz {
		return v, k, nil
	}
	return nil, nil, fmt.Errorf("cannot accomodate this")
}

type paramSplitter interface {
	split() (bool, error)
	getSplitDataFlowCollection() dataflow.Collection
	getSplitAnnotationContextMap() taxonomy.AnnotationCtxSplitMap
}

type standardParamSplitter struct {
	alreadySplit              map[string]struct{}
	splitParamMap             map[sqlparser.TableExpr]map[string][]any
	paramToTableExprMap       map[string]sqlparser.TableExpr
	splitAnnotationContextMap taxonomy.AnnotationCtxSplitMap
	tableToAnnotationCtx      map[sqlparser.TableExpr]taxonomy.AnnotationCtx
	dataflowCollection        dataflow.Collection
}

func newParamSplitter(
	tableToAnnotationCtx map[sqlparser.TableExpr]taxonomy.AnnotationCtx,
	dataFlowCfg dto.DataFlowCfg,
) paramSplitter {
	splitParamMap := make(map[sqlparser.TableExpr]map[string][]any)
	for k := range tableToAnnotationCtx {
		splitParamMap[k] = make(map[string][]any)
	}
	return &standardParamSplitter{
		alreadySplit:              make(map[string]struct{}),
		splitParamMap:             splitParamMap,
		paramToTableExprMap:       make(map[string]sqlparser.TableExpr),
		splitAnnotationContextMap: taxonomy.NewAnnotationCtxSplitMap(),
		tableToAnnotationCtx:      tableToAnnotationCtx,
		dataflowCollection:        dataflow.NewStandardDataFlowCollection(dataFlowCfg),
	}
}

func (sp *standardParamSplitter) getSplitAnnotationContextMap() taxonomy.AnnotationCtxSplitMap {
	return sp.splitAnnotationContextMap
}

func (sp *standardParamSplitter) split() (bool, error) {
	var tableEquivalencyID int64
	// Rist, see if any dependencies need splitting
	var isAnythingSplit bool
	for k, v := range sp.tableToAnnotationCtx {
		tableEquivalencyID++ // start at 1 for > 0 logic
		var skipBaseAdd bool
		for k1, param := range v.GetParameters() {
			sp.paramToTableExprMap[k1] = k
			paramSlice, isSplit := sp.splitSingleParam(param)
			sp.splitParamMap[k][k1] = paramSlice
			if isSplit {
				skipBaseAdd = true
				isAnythingSplit = true
			}
		}
		_, assembleErr := sp.assembleSplitParams(k, tableEquivalencyID)
		if assembleErr != nil {
			return false, assembleErr
		}
		if !skipBaseAdd {
			sp.dataflowCollection.AddVertex(sp.dataflowCollection.UpsertStandardDataFlowVertex(v, k))
		}
	}
	return isAnythingSplit, nil
}

func (sp *standardParamSplitter) getSplitDataFlowCollection() dataflow.Collection {
	return sp.dataflowCollection
}

func (sp *standardParamSplitter) assembleSplitParams(
	tableExpr sqlparser.TableExpr,
	tableEquivalencyID int64,
) (bool, error) {
	rawAnnotationCtx, ok := sp.tableToAnnotationCtx[tableExpr]
	if !ok {
		return false, fmt.Errorf("table expression '%s' has not been assigned to hierarchy", sqlparser.String(tableExpr))
	}
	combinationComposerObj := newCombinationComposer()
	splitParams := sp.splitParamMap[tableExpr]
	analysisErr := combinationComposerObj.analyse(splitParams)
	if analysisErr != nil {
		return false, analysisErr
	}
	combinations := combinationComposerObj.getCombinations()
	_, isAnythingSplit := len(combinations), combinationComposerObj.getIsAnythingSplit()
	for _, paramCombination := range combinations {
		com := paramCombination
		splitAnnotationCtx := taxonomy.NewStaticStandardAnnotationCtx(
			rawAnnotationCtx.GetSchema(),
			rawAnnotationCtx.GetHIDs(),
			rawAnnotationCtx.GetTableMeta().Clone(),
			com,
			rawAnnotationCtx.IsAwait(),
		)
		sp.splitAnnotationContextMap.Put(rawAnnotationCtx, splitAnnotationCtx)
		// TODO: this has gotta replace the original and also be duplicated
		sourceVertexIteration := sp.dataflowCollection.UpsertStandardDataFlowVertex(splitAnnotationCtx, tableExpr)
		sourceVertexIteration.SetEquivalencyGroup(tableEquivalencyID)
		sp.dataflowCollection.AddVertex(sourceVertexIteration)
	}
	return isAnythingSplit, nil
}

func (sp *standardParamSplitter) splitSingleParam(
	param any,
) ([]any, bool) {
	var isSplit bool
	var rv []any
	switch param := param.(type) { //nolint:gocritic // TODO: review
	case parserutil.ParameterMetadata:
		rhs := param.GetVal()
		switch rhs := rhs.(type) {
		case sqlparser.ValTuple:
			// TODO: fix update anomale for dataflow graph!!!
			for _, valTmp := range rhs {
				val := valTmp
				rv = append(rv, val)
				isSplit = true
			}
		default:
			rv = append(rv, param)
		}
	}
	return rv, isSplit
}

//nolint:funlen,gocognit,nestif // inherently complex functionality
func (pr *standardParameterRouter) GetOnConditionDataFlows() (dataflow.Collection, error) {
	paramSplitterObj := newParamSplitter(pr.tableToAnnotationCtx, pr.dataFlowCfg)
	isInitiallySplit, splitErr := paramSplitterObj.split()
	if isInitiallySplit {
		logging.GetLogger().Debugf("dataflow required initial splitting")
	}
	if splitErr != nil {
		return nil, splitErr
	}
	rv := paramSplitterObj.getSplitDataFlowCollection()
	splitAnnotationContextMap := paramSplitterObj.getSplitAnnotationContextMap()

	for k, destinationTable := range pr.comparisonToTableDependencies {
		selfTableCited := false
		destHierarchy, ok := pr.tableToAnnotationCtx[destinationTable]
		if !ok {
			return nil, fmt.Errorf(
				"table expression '%s' has not been assigned to hierarchy", sqlparser.String(destinationTable))
		}
		var dependencyTable sqlparser.TableExpr
		var dependencies []taxonomy.AnnotationCtx
		var destinations []taxonomy.AnnotationCtx

		var destColumn *sqlparser.ColName
		var srcExpr sqlparser.Expr
		switch l := k.Left.(type) {
		case *sqlparser.ColName:
			lhr, candidateTable, err := pr.extractDataFlowDependency(l)
			if err != nil {
				return nil, err
			}
			if destHierarchy == lhr {
				selfTableCited = true
				destColumn = l
				srcExpr = k.Right
			} else {
				splitDependencies, isSplit := splitAnnotationContextMap.Get(lhr)
				if isSplit {
					dependencies = append(dependencies, splitDependencies...)
				} else {
					dependencies = append(dependencies, lhr)
				}
				splitDestinations, isDestinationSplit := splitAnnotationContextMap.Get(destHierarchy)
				if isDestinationSplit {
					destinations = append(destinations, splitDestinations...)
				} else {
					destinations = append(destinations, destHierarchy)
				}
				dependencyTable = candidateTable
				srcExpr = k.Left
			}
		case *sqlparser.FuncExpr:
			annCtx, te, err := pr.extractFromFunctionExpr(l)
			if err != nil {
				return nil, err
			}
			splitDependencies, isSplit := splitAnnotationContextMap.Get(annCtx)
			if isSplit {
				dependencies = append(dependencies, splitDependencies...)
			} else {
				dependencies = append(dependencies, annCtx)
			}
			splitDestinations, isDestinationSplit := splitAnnotationContextMap.Get(destHierarchy)
			if isDestinationSplit {
				destinations = append(destinations, splitDestinations...)
			} else {
				destinations = append(destinations, destHierarchy)
			}
			dependencyTable = te
		}
		switch r := k.Right.(type) {
		case *sqlparser.SQLVal:
			// no dataflow dependencies, do zero
			continue
		case *sqlparser.ColName:
			rhr, candidateTable, err := pr.extractDataFlowDependency(r)
			if err != nil {
				return nil, err
			}
			if destHierarchy == rhr {
				if selfTableCited {
					return nil, fmt.Errorf("table join ON comparison '%s' is self referencing", sqlparser.String(k))
				}
				selfTableCited = true
				destColumn = r
				srcExpr = k.Left
			} else {
				splitDependencies, isSplit := splitAnnotationContextMap.Get(rhr)
				if isSplit {
					dependencies = append(dependencies, splitDependencies...)
				} else {
					dependencies = append(dependencies, rhr)
				}
				splitDestinations, isDestinationSplit := splitAnnotationContextMap.Get(destHierarchy)
				if isDestinationSplit {
					destinations = append(destinations, splitDestinations...)
				} else {
					destinations = append(destinations, destHierarchy)
				}
				dependencyTable = candidateTable
			}
		case *sqlparser.FuncExpr:
			annCtx, te, err := pr.extractFromFunctionExpr(r)
			if err != nil {
				return nil, err
			}
			splitDependencies, isSplit := splitAnnotationContextMap.Get(annCtx)
			if isSplit {
				dependencies = append(dependencies, splitDependencies...)
			} else {
				dependencies = append(dependencies, annCtx)
			}
			splitDestinations, isDestinationSplit := splitAnnotationContextMap.Get(destHierarchy)
			if isDestinationSplit {
				destinations = append(destinations, splitDestinations...)
			} else {
				destinations = append(destinations, destHierarchy)
			}
			dependencyTable = te
		}
		if !selfTableCited {
			return nil, fmt.Errorf("table join ON comparison '%s' referencing incomplete", sqlparser.String(k))
		}

		for i, dependency := range dependencies {
			srcVertex := rv.UpsertStandardDataFlowVertex(dependency, dependencyTable)
			destination := destHierarchy
			if i < len(destinations) {
				destination = destinations[i]
			}
			destVertex := rv.UpsertStandardDataFlowVertex(destination, destinationTable)

			err := rv.AddOrUpdateEdge(
				srcVertex,
				destVertex,
				k,
				srcExpr,
				destColumn,
			)
			if err != nil {
				return nil, err
			}
		}
	}
	return rv.WithSplitAnnotationContextMap(splitAnnotationContextMap), nil
}

//nolint:gocognit // who cares
func (pr *standardParameterRouter) getAvailableParameters(
	tb sqlparser.TableExpr,
) parserutil.TableParameterCoupling {
	rv := parserutil.NewTableParameterCoupling()
	minKeyMap := make(map[string]int)
	for k, v := range pr.whereParamMap.GetMap() {
		key := k.String()
		tableAlias := k.Alias()
		ordinal := v.GetOrdinal()
		foundTable, ok := pr.tablesAliasMap[tableAlias]
		if ok && foundTable != tb {
			continue
		}
		if pr.isInvalidated(key) {
			continue
		}
		ref, ok := pr.colRefs[k]
		if ok && ref != tb {
			continue
		}
		existingOrdinal, ordinalOk := minKeyMap[key]
		if ordinalOk {
			_, isPlaceholder := v.(*parserutil.PlaceholderParameterMetadata)
			if isPlaceholder {
				continue
			}
			if existingOrdinal < ordinal {
				continue
			}
			rv.DeleteByOrdinal(existingOrdinal)
		}
		rv.Add(k, v, parserutil.WhereParam) //nolint:errcheck // no issue
		minKeyMap[key] = ordinal
	}
	for k, v := range pr.onParamMap.GetMap() {
		key := k.String()
		tableAlias := k.Alias()
		foundTable, ok := pr.tablesAliasMap[tableAlias]
		if ok && foundTable != tb {
			continue
		}
		if pr.isInvalidated(key) {
			continue
		}
		ref, ok := pr.colRefs[k]
		if ok && ref != tb {
			continue
		}
		val := v.GetVal()
		switch val := val.(type) { //nolint:gocritic // TODO: review
		case *sqlparser.ColName:
			logging.GetLogger().Debugf("%v\n", val)
			rhsAlias := val.Qualifier.GetRawVal()
			logging.GetLogger().Debugf("%v\n", rhsAlias)
		}
		rv.Add(k, v, parserutil.JoinOnParam) //nolint:errcheck // no issue
	}
	return rv
}

func (pr *standardParameterRouter) isInvalidated(key string) bool {
	_, ok := pr.invalidatedParams[key]
	return ok
}

// Route will map columnar input to a supplied
// parser table object.
// Columnar input may come from either where clause
// or on conditions.
func (pr *standardParameterRouter) Route(
	tb sqlparser.TableExpr,
	handlerCtx handler.HandlerContext,
	isAwait bool,
) (taxonomy.AnnotationCtx, error) {
	return pr.route(tb, handlerCtx, isAwait)
}

//nolint:funlen,gocognit,govet,gocyclo,cyclop // inherently complex functionality
func (pr *standardParameterRouter) route(
	tb sqlparser.TableExpr,
	handlerCtx handler.HandlerContext,
	isAwait bool,
) (taxonomy.AnnotationCtx, error) {
	// TODO: Get rid of the dead set mess that is where paramters in preference.
	for k, v := range pr.whereParamMap.GetMap() {
		logging.GetLogger().Infof("%v\n", v)
		alias := k.Alias()
		if alias == "" {
			continue
		}
		t, ok := pr.tablesAliasMap[alias]
		if !ok {
			return nil, fmt.Errorf("alias '%s' does not map to any table expression", alias)
		}
		if t == tb {
			ref, ok := pr.colRefs[k]
			if ok && ref != t {
				return nil, fmt.Errorf("failed parameter routing, cannot re-assign")
			}
			pr.colRefs[k] = t
		}
	}
	for k, v := range pr.onParamMap.GetMap() {
		logging.GetLogger().Infof("%v\n", v)
		alias := k.Alias()
		if alias == "" {
			continue
		}
		t, ok := pr.tablesAliasMap[alias]
		if !ok {
			return nil, fmt.Errorf("alias '%s' does not map to any table expression", alias)
		}
		if t == tb {
			ref, ok := pr.colRefs[k]
			if ok && ref != t {
				return nil, fmt.Errorf("failed parameter routing, cannot re-assign")
			}
			pr.colRefs[k] = t
		}
	}
	// These are "available parameters"
	tpc := pr.getAvailableParameters(tb)
	runParamters := tpc.Clone()
	// After executing GetHeirarchyFromStatement(), we know:
	//   - Any remaining param is not required.
	//   - Any "on" param that was consumed:
	//      - Can / must be from removed join conditions in a rewrite. [Requires Join in router for later rewrite].
	//      - Defines a sequencing and data flow dependencies unless RHS is a literal. [Create new object to represent].
	// TODO: In order to do this, we can, for each table:
	//   1. [*] Subtract the remaining parameters returned by GetHeirarchyFromStatement()
	//      from the available parameters.  Will need reversible string to object translation.
	//   2. [*] Identify "on" parameters that were consumed as per item #1.
	//      We are free to change the "table parameter coupling" API to accomodate
	//      items #1 and #2.
	//   3. [*] If #2 is consumed, then:
	//        - [*] Tag the "on" comparison as being incident to the table.
	//        - [*] Tag the "on" comparison for later rewrite to NOP.
	//      Probably some
	//      new data structure to accomodate this.
	// And then, once all tables are done and also therefore, all hierarchies are present:
	//   a) [ ] Assign all remaining on parameters based on schema.
	//   b) [ ] Represent assignments as edges from table to on condition.
	//   d) [ ] Throw error for disallowed scenarios:
	//          - Dual outgoing from ON object.
	//   e) [ ] Rewrite NOP on clauses.
	//   f) [ ] Catalogue and return dataflows (somehow)
	// stringParams := tpc.GetStringified()
	notOnParams := runParamters.Clone().GetNotOnCoupling()
	priorNotOnParameters := notOnParams.Clone()
	priorParameters := runParamters.Clone()
	// notOnStringParams := notOnParams.GetStringified()
	// TODO: add parent params into the mix here.
	// Note: CTEs are converted to subqueries at AST level, so they follow
	// the normal subquery path (handled via GetSubquery() in ObtainAnnotationCtx).
	var hr tablemetadata.HeirarchyObjects
	var err error
	hr, err = taxonomy.GetHeirarchyFromStatement(handlerCtx, tb, notOnParams, false, isAwait)
	if err != nil {
		hr, err = taxonomy.GetHeirarchyFromStatement(handlerCtx, tb, runParamters, false, isAwait)
	} else {
		// If the where parameters are sufficient, then need to switch
		// the Table - Paramater coupling object
		runParamters = notOnParams
		priorParameters = priorNotOnParameters
	}
	// logging.GetLogger().Infof("hr = '%+v', remainingParams = '%+v', err = '%+v'", hr, remainingParams, err)
	if err != nil {
		return nil, err
	}
	// reconstitutedConsumedParams, err := tpc.ReconstituteConsumedParams(remainingParams)
	// if err != nil {
	// 	return nil, err
	// }
	// reconstitutedConsumedParams := tpc.Minus(runParamters)
	// reconstitutedConsumedParams := priorParameters.Minus(runParamters)
	logging.GetLogger().Debugf("%v\n", priorParameters)
	// TODO: need to get ALL the required stuff in here,
	//       BUT not send the wrong things for dataflow analysis.
	reconstitutedConsumedParams := runParamters
	abbreviatedConsumedMap, err := reconstitutedConsumedParams.AbbreviateMap()
	if err != nil {
		return nil, err
	}
	// if err != nil {
	// 	return nil, err
	// }
	// TODO: fix this mess and make it global
	//       so that ancestor params can be correctly consumed!!!
	// err = pr.splitParams(abbreviatedConsumedMap)
	// if err != nil {
	// 	return nil, err
	// }
	onConsumed := reconstitutedConsumedParams.GetOnCoupling()
	pms := onConsumed.GetAllParameters()
	logging.GetLogger().Infof("onConsumed = '%+v'", onConsumed)
	for _, kv := range pms {
		// In this stanza:
		//   1. [*] mark comparisons for rewriting
		//   2. [*] some sequencing data to be stored
		p := kv.V.GetParent()
		existingTable, ok := pr.comparisonToTableDependencies[p]
		if ok {
			return nil, fmt.Errorf(
				"data flow violation detected: ON comparison expression '%s' is a  dependencies for tables '%s' and '%s'",
				sqlparser.String(p), sqlparser.String(existingTable), sqlparser.String(tb))
		}
		pr.comparisonToTableDependencies[p] = tb
		logging.GetLogger().Infof("%v", kv)
	}
	indirect, _ := pr.annotatedAST.GetIndirect(tb)
	currentIndirect := indirect
	// TODO: elide all non selected indirects
	var alreadyMatched bool
	for {
		if currentIndirect == nil {
			break
		}
		ind, matches := currentIndirect.MatchOnParams(abbreviatedConsumedMap)
		if matches && !alreadyMatched {
			indirect = ind
			alreadyMatched = true
		} else {
			// elide this indirect
			currentIndirect.SetElide(true)
		}
		var hasNext bool
		currentIndirect, hasNext = currentIndirect.Next()
		if !hasNext {
			break
		}
		logging.GetLogger().Infof("nextIndirect = %v", currentIndirect)
	}
	hrView, hrViewPresent := hr.GetHeirarchyIDs().GetView()
	if indirect == nil && hrViewPresent { //nolint:nestif // TODO: review
		if hrView.IsMaterialized() { //nolint:gocritic // TODO: review
			indirect, err = astindirect.NewMaterializedViewIndirect(hrView, handlerCtx.GetSQLSystem())
			if err != nil {
				return nil, err
			}
			err = indirect.Parse()
			if err != nil {
				return nil, err
			}
		} else if hrView.IsTable() {
			indirect, err = astindirect.NewPhysicalTableIndirect(hrView, handlerCtx.GetSQLSystem())
			if err != nil {
				return nil, err
			}
			err = indirect.Parse()
			if err != nil {
				return nil, err
			}
		} else {
			indirect, err = astindirect.NewViewIndirect(hrView)
			if err != nil {
				return nil, err
			}
			err = indirect.Parse()
			if err != nil {
				return nil, err
			}
		}
	}
	m := tablemetadata.NewExtendedTableMetadata(
		hr,
		taxonomy.GetTableNameFromStatement(tb, pr.astFormatter),
		taxonomy.GetAliasFromStatement(tb)).WithIndirect(indirect)

	// store relationship from sqlparser table expression to
	// hierarchy.  This enables e2e relationship
	// from expression to hierarchy.
	// eg: "on" clause to openapi method
	ac, err := obtain_context.ObtainAnnotationCtx(
		handlerCtx.GetSQLSystem(), m, abbreviatedConsumedMap, pr.namespaceCollection)
	pr.tableToAnnotationCtx[tb] = ac
	return ac, err
}
