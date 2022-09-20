package router

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/astvisit"
	"github.com/stackql/stackql/internal/stackql/dataflow"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
	"vitess.io/vitess/go/vt/sqlparser"
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
	// invalidateParams(params map[string]interface{}) error

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
	Route(tb sqlparser.TableExpr, handler *handler.HandlerContext) (taxonomy.AnnotationCtx, error)

	// Detects:
	//   - Dependency cycle.
	AnalyzeDependencies() error

	GetOnConditionsToRewrite() map[*sqlparser.ComparisonExpr]struct{}

	GetOnConditionDataFlows() (dataflow.DataFlowCollection, error)
}

type StandardParameterRouter struct {
	tablesAliasMap                parserutil.TableAliasMap
	tableMap                      parserutil.TableExprMap
	onParamMap                    parserutil.ParameterMap
	whereParamMap                 parserutil.ParameterMap
	colRefs                       parserutil.ColTableMap
	comparisonToTableDependencies parserutil.ComparisonTableMap
	tableToComparisonDependencies parserutil.ComparisonTableMap
	tableToAnnotationCtx          map[sqlparser.TableExpr]taxonomy.AnnotationCtx
	invalidatedParams             map[string]interface{}
}

func NewParameterRouter(
	tablesAliasMap parserutil.TableAliasMap,
	tableMap parserutil.TableExprMap,
	whereParamMap parserutil.ParameterMap,
	onParamMap parserutil.ParameterMap,
	colRefs parserutil.ColTableMap,
) ParameterRouter {
	return &StandardParameterRouter{
		tablesAliasMap:                tablesAliasMap,
		tableMap:                      tableMap,
		whereParamMap:                 whereParamMap,
		onParamMap:                    onParamMap,
		colRefs:                       colRefs,
		invalidatedParams:             make(map[string]interface{}),
		comparisonToTableDependencies: make(parserutil.ComparisonTableMap),
		tableToComparisonDependencies: make(parserutil.ComparisonTableMap),
		tableToAnnotationCtx:          make(map[sqlparser.TableExpr]taxonomy.AnnotationCtx),
	}
}

func (pr *StandardParameterRouter) AnalyzeDependencies() error {
	return nil
}

// This has been obviated for the time being.
// Essentially, not required so long as:
//   - in params
//   - result set data
// ...are guaranteed present in same table.
func (pr *StandardParameterRouter) GetOnConditionsToRewrite() map[*sqlparser.ComparisonExpr]struct{} {
	rv := make(map[*sqlparser.ComparisonExpr]struct{})
	for k, _ := range pr.comparisonToTableDependencies {
		logging.GetLogger().Debugf("%v\n", k)
	}
	return rv
}

func (pr *StandardParameterRouter) extractDataFlowDependency(input sqlparser.Expr) (taxonomy.AnnotationCtx, sqlparser.TableExpr, error) {
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

func (pr *StandardParameterRouter) extractFromFunctionExpr(f *sqlparser.FuncExpr) (taxonomy.AnnotationCtx, sqlparser.TableExpr, error) {
	sv := astvisit.NewLeftoverReferencesAstVisitor(
		pr.colRefs,
		pr.tableToAnnotationCtx,
	)
	sv.Visit(f)
	tbz := sv.GetTablesFoundThisIteration()
	if len(tbz) != 1 {
		return nil, nil, fmt.Errorf("cannot accomodate this")
	}
	for k, v := range tbz {
		return v, k, nil
	}
	return nil, nil, fmt.Errorf("cannot accomodate this")
}

func (pr *StandardParameterRouter) GetOnConditionDataFlows() (dataflow.DataFlowCollection, error) {
	rv := dataflow.NewStandardDataFlowCollection()
	for k, destinationTable := range pr.comparisonToTableDependencies {
		selfTableCited := false
		destHierarchy, ok := pr.tableToAnnotationCtx[destinationTable]
		if !ok {
			return nil, fmt.Errorf("table expression '%s' has not been assigned to hierarchy", sqlparser.String(destinationTable))
		}
		var dependencyTable sqlparser.TableExpr
		var dependency taxonomy.AnnotationCtx
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
				dependency = lhr
				dependencyTable = candidateTable
				srcExpr = k.Left
			}
		case *sqlparser.FuncExpr:
			annCtx, te, err := pr.extractFromFunctionExpr(l)
			if err != nil {
				return nil, err
			}
			dependency = annCtx
			dependencyTable = te
		}
		switch r := k.Right.(type) {
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
				dependency = rhr
				dependencyTable = candidateTable
			}
		case *sqlparser.FuncExpr:
			annCtx, te, err := pr.extractFromFunctionExpr(r)
			if err != nil {
				return nil, err
			}
			dependency = annCtx
			dependencyTable = te
		}
		if !selfTableCited {
			return nil, fmt.Errorf("table join ON comparison '%s' referencing incomplete", sqlparser.String(k))
		}
		// rv[dependency] = destHierarchy

		srcVertex := dataflow.NewStandardDataFlowVertex(dependency, dependencyTable, rv.GetNextID())
		destVertex := dataflow.NewStandardDataFlowVertex(destHierarchy, destinationTable, rv.GetNextID())

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
	for k, v := range pr.tableToAnnotationCtx {
		rv.AddVertex(dataflow.NewStandardDataFlowVertex(v, k, rv.GetNextID()))
	}
	return rv, nil
}

func (pr *StandardParameterRouter) getAvailableParameters(tb sqlparser.TableExpr) parserutil.TableParameterCoupling {
	rv := parserutil.NewTableParameterCoupling()
	for k, v := range pr.whereParamMap.GetMap() {
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
		rv.Add(k, v, parserutil.WhereParam)
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
		switch val := val.(type) {
		case *sqlparser.ColName:
			logging.GetLogger().Debugf("%v\n", val)
			rhsAlias := val.Qualifier.GetRawVal()
			logging.GetLogger().Debugf("%v\n", rhsAlias)
			foundTable, ok := pr.tablesAliasMap[rhsAlias]
			if ok && foundTable != tb {
				//
			}
		}
		rv.Add(k, v, parserutil.JoinOnParam)
	}
	return rv
}

func (pr *StandardParameterRouter) invalidateParams(params map[string]interface{}) error {
	for k, v := range params {
		err := pr.invalidate(k, v)
		if err != nil {
			return err
		}
	}
	return nil
}

func (pr *StandardParameterRouter) isInvalidated(key string) bool {
	_, ok := pr.invalidatedParams[key]
	return ok
}

func (pr *StandardParameterRouter) invalidate(key string, val interface{}) error {
	if pr.isInvalidated(key) {
		return fmt.Errorf("parameter '%s' already invalidated", key)
	}
	pr.invalidatedParams[key] = val
	return nil
}

// Route will map columnar input to a supplied
// parser table object.
// Columnar input may come from either where clause
// or on conditions.
// TODO: Get rid of the dead set mess that is where paramters in preference.
func (pr *StandardParameterRouter) Route(tb sqlparser.TableExpr, handlerCtx *handler.HandlerContext) (taxonomy.AnnotationCtx, error) {
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
	// After executing GetHeirarchyFromStatement(), we know:
	//   - Any remaining param is not required.
	//   - Any "on" param that was consumed:
	//      - Can / must be from removed join conditions in a rewrite. [Requires Join in router for later rewrite].
	//      - Defines a sequencing and data flow dependency unless RHS is a literal. [Create new object to represent].
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
	stringParams := tpc.GetStringified()
	notOnParams := tpc.GetNotOnCoupling()
	notOnStringParams := notOnParams.GetStringified()
	hr, remainingParams, err := taxonomy.GetHeirarchyFromStatement(handlerCtx, tb, notOnStringParams)
	if err != nil {
		hr, remainingParams, err = taxonomy.GetHeirarchyFromStatement(handlerCtx, tb, stringParams)
	} else {
		// if the where pearamters are sufficient, then need to switch
		// the Table - Paramater coupling object
		tpc = notOnParams
	}
	logging.GetLogger().Infof("hr = '%+v', remainingParams = '%+v', err = '%+v'", hr, remainingParams, err)
	if err != nil {
		return nil, err
	}
	reconstitutedConsumedParams, err := tpc.ReconstituteConsumedParams(remainingParams)
	if err != nil {
		return nil, err
	}
	abbreviatedConsumedMap, err := reconstitutedConsumedParams.AbbreviateMap()
	if err != nil {
		return nil, err
	}
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
			return nil, fmt.Errorf("data flow violation detected: ON comparison expression '%s' is a  dependency for tables '%s' and '%s'", sqlparser.String(p), sqlparser.String(existingTable), sqlparser.String(tb))
		}
		pr.comparisonToTableDependencies[p] = tb
		logging.GetLogger().Infof("%v", kv)
	}
	m := taxonomy.NewExtendedTableMetadata(hr, taxonomy.GetAliasFromStatement(tb))
	// store relationship from sqlparser table expression to
	// hierarchy.  This enables e2e relationship
	// from expression to hierarchy.
	// eg: "on" clause to openapi method
	ac, err := obtainAnnotationCtx(handlerCtx.SQLEngine, m, abbreviatedConsumedMap)
	pr.tableToAnnotationCtx[tb] = ac
	return ac, err
}
