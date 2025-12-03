package dependencyplanner

import (
	"fmt"
	"strings"

	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/any-sdk/pkg/constants"
	"github.com/stackql/any-sdk/pkg/logging"
	"github.com/stackql/any-sdk/pkg/media"
	"github.com/stackql/any-sdk/pkg/streaming"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/astanalysis/annotatedast"
	"github.com/stackql/stackql/internal/stackql/astindirect"
	"github.com/stackql/stackql/internal/stackql/astvisit"
	"github.com/stackql/stackql/internal/stackql/dataflow"
	"github.com/stackql/stackql/internal/stackql/docparser"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/primitivebuilder"
	"github.com/stackql/stackql/internal/stackql/primitivecomposer"
	"github.com/stackql/stackql/internal/stackql/sqlrewrite"
	"github.com/stackql/stackql/internal/stackql/sqlstream"
	"github.com/stackql/stackql/internal/stackql/tableinsertioncontainer"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
	"github.com/stackql/stackql/internal/stackql/util"
)

type DependencyPlanner interface {
	Plan() error
	GetBldr() primitivebuilder.Builder
	GetSelectCtx() drm.PreparedStatementCtx
	WithPrepStmtOffset(offset int) DependencyPlanner
	WithElideRead(isElideRead bool) DependencyPlanner
}

type standardDependencyPlanner struct {
	annotatedAST       annotatedast.AnnotatedAst
	dataflowCollection dataflow.Collection
	colRefs            parserutil.ColTableMap
	handlerCtx         handler.HandlerContext
	execSlice          []primitivebuilder.Builder
	primaryTcc, tcc    internaldto.TxnControlCounters
	primitiveComposer  primitivecomposer.PrimitiveComposer
	rewrittenWhere     *sqlparser.Where
	secondaryTccs      []internaldto.TxnControlCounters
	sqlStatement       *sqlparser.Select
	tableSlice         []tableinsertioncontainer.TableInsertionContainer
	tblz               taxonomy.TblMap
	discoGenIDs        map[sqlparser.SQLNode]int
	tccSetAheadOfTime  bool

	//
	bldr                 primitivebuilder.Builder
	selCtx               drm.PreparedStatementCtx
	defaultStream        streaming.MapStream
	annMap               taxonomy.AnnotationCtxMap
	equivalencyGroupTCCs map[int64]internaldto.TxnControlCounters
	//
	prepStmtOffset int
	//
	isElideRead bool
	//
	dataflowToEdges map[int][]int
	nodeIDIdxMap    map[int64]int
}

func NewStandardDependencyPlanner(
	annotatedAST annotatedast.AnnotatedAst,
	handlerCtx handler.HandlerContext,
	dataflowCollection dataflow.Collection,
	colRefs parserutil.ColTableMap,
	rewrittenWhere *sqlparser.Where,
	sqlStatement *sqlparser.Select,
	tblz taxonomy.TblMap,
	primitiveComposer primitivecomposer.PrimitiveComposer,
	tcc internaldto.TxnControlCounters,
	tccSetAheadOfTime bool,
) (DependencyPlanner, error) {
	if tcc == nil {
		return nil, fmt.Errorf("violation of standardDependencyPlanner invariant: txn counter cannot be nil")
	}
	return &standardDependencyPlanner{
		annotatedAST:         annotatedAST,
		handlerCtx:           handlerCtx,
		dataflowCollection:   dataflowCollection,
		colRefs:              colRefs,
		rewrittenWhere:       rewrittenWhere,
		sqlStatement:         sqlStatement,
		tblz:                 tblz,
		primitiveComposer:    primitiveComposer,
		discoGenIDs:          make(map[sqlparser.SQLNode]int),
		defaultStream:        streaming.NewStandardMapStream(),
		annMap:               make(taxonomy.AnnotationCtxMap),
		tcc:                  tcc,
		tccSetAheadOfTime:    tccSetAheadOfTime,
		equivalencyGroupTCCs: make(map[int64]internaldto.TxnControlCounters),
		dataflowToEdges:      make(map[int][]int),
		nodeIDIdxMap:         make(map[int64]int),
	}, nil
}

func (dp *standardDependencyPlanner) dataflowEdgeExists(from, to int) bool {
	edges, ok := dp.dataflowToEdges[to]
	if !ok {
		return false
	}
	for _, e := range edges {
		if e == from {
			return true
		}
	}
	return false
}

func (dp *standardDependencyPlanner) WithPrepStmtOffset(offset int) DependencyPlanner {
	dp.prepStmtOffset = offset
	return dp
}

func (dp *standardDependencyPlanner) WithElideRead(isElideRead bool) DependencyPlanner {
	dp.isElideRead = isElideRead
	return dp
}

func (dp *standardDependencyPlanner) GetBldr() primitivebuilder.Builder {
	return dp.bldr
}

func (dp *standardDependencyPlanner) GetSelectCtx() drm.PreparedStatementCtx {
	return dp.selCtx
}

//nolint:funlen,gocognit,gocyclo,cyclop // inherently complex function
func (dp *standardDependencyPlanner) Plan() error {
	err := dp.dataflowCollection.Sort()
	if err != nil {
		return err
	}
	units, err := dp.dataflowCollection.GetAllUnits()
	if err != nil {
		return err
	}
	// TODO: lift this restriction once all traversal algorithms are adequate
	weaklyConnectedComponentCount := 0
	for _, u := range units {
		unit := u
		switch unit := unit.(type) {
		case dataflow.Vertex:
			inDegree := dp.dataflowCollection.InDegree(unit)
			outDegree := dp.dataflowCollection.OutDegree(unit)
			if inDegree == 0 && outDegree > 0 {
				// TODO: start builder
				logging.GetLogger().Infof("\n")
			}
			if inDegree != 0 || outDegree != 0 {
				return fmt.Errorf(
					"cannot currently execute data dependent tables with inDegree = %d and/or outDegree = %d",
					inDegree, outDegree)
			}
			tableExpr := unit.GetTableExpr()
			annotation := unit.GetAnnotation()
			_, isView := annotation.GetView()
			_, isSubquery := annotation.GetSubquery()
			// Note: CTEs are converted to subqueries at AST level, so isSubquery handles them.
			if isView || isSubquery {
				dp.annMap[tableExpr] = annotation
				continue
			}
			dp.annMap[tableExpr] = annotation
			connectorStream := streaming.NewStandardMapStream()
			insPsc, _, insErr := dp.processOrphan(tableExpr, annotation, unit)
			if insErr != nil {
				return insErr
			}
			stream := streaming.NewNopMapStream()
			idx, orcErr := dp.orchestrate(unit.GetEquivalencyGroup(), annotation, insPsc, connectorStream, stream)
			if orcErr != nil {
				return orcErr
			}
			dp.nodeIDIdxMap[unit.ID()] = idx
		case dataflow.WeaklyConnectedComponent:
			weaklyConnectedComponentCount++
			orderedNodes, oErr := unit.GetOrderedNodes()
			if oErr != nil {
				return oErr
			}
			dp.nodeIDIdxMap = make(map[int64]int)
			logging.GetLogger().Infof("%v\n", orderedNodes)
			edges, eErr := unit.GetEdges()
			if eErr != nil {
				return eErr
			}
			logging.GetLogger().Infof("%v\n", edges)
			edgeCount := len(edges)
			// TODO: test this
			dependencyMax := dp.handlerCtx.GetRuntimeContext().DataflowDependencyMax
			if edgeCount > dependencyMax {
				return fmt.Errorf(
					"data flow: cannot accomodate table dependencies of this complexity: supplied = %d, max = %d",
					edgeCount, dependencyMax)
			}
			idsVisited := make(map[int64]struct{})
			// first pass: set up AOT stuff
			//    - stream per edge.
			edgeStreams := make(map[dataflow.Edge]streaming.MapStream)
			nodeStreamCollections := NewStreamDependecyCollection()
			insertPrepearedStatements := make(map[int64]drm.PreparedStatementCtx)
			var orderedEdges []dataflow.Edge
			for _, n := range orderedNodes {
				if _, ok := idsVisited[n.ID()]; ok {
					continue
				}
				idsVisited[n.ID()] = struct{}{}
				tableExpr := n.GetTableExpr()
				annotation := n.GetAnnotation()
				dp.annMap[tableExpr] = annotation
				for _, e := range edges {
					if e.From().ID() == n.ID() {
						insPsc, tcc, insErr := dp.processOrphan(tableExpr, annotation, n)
						if insErr != nil {
							return insErr
						}
						insertPrepearedStatements[n.ID()] = insPsc
						toNode := e.GetDest()
						toAnnotation := toNode.GetAnnotation().Clone() // this bodge protects split source vertices
						toTableExpr := toNode.GetTableExpr()
						stream, streamErr := dp.getStreamFromEdge(e, toAnnotation, tcc)
						if streamErr != nil {
							return streamErr
						}
						edgeStreams[e] = stream
						nodeStreamCollections.Add(n.ID(), toNode.ID(), stream)
						toInsPsc, _, toErr := dp.processOrphan(toTableExpr, toAnnotation, toNode)
						if toErr != nil {
							return toErr
						}
						insertPrepearedStatements[toNode.ID()] = toInsPsc
						orderedEdges = append(orderedEdges, e)
					}
				}
			}
			// second pass: connect streams
			//     - edge collection per node
			for _, e := range orderedEdges {
				fromNode := e.GetSource()
				toNode := e.GetDest()
				fromAnnotation := fromNode.GetAnnotation()
				toAnnotation := toNode.GetAnnotation().Clone() // this bodge protects split source vertices
				toTableExpr := toNode.GetTableExpr()
				departingSourceNodeStream := nodeStreamCollections.GetDeparting(fromNode.ID())
				arrivingDestinationNodeStream := nodeStreamCollections.GetArriving(toNode.ID())
				arrivingSourceNodeStream := nodeStreamCollections.GetArriving(fromNode.ID())
				departingDestinationNodeStream := nodeStreamCollections.GetDeparting(toNode.ID())
				insPsc, pscExists := insertPrepearedStatements[fromNode.ID()]
				if !pscExists {
					return fmt.Errorf("unknown insert prepared statement")
				}
				toInsPsc, pscExists := insertPrepearedStatements[toNode.ID()]
				if !pscExists {
					return fmt.Errorf("unknown insert prepared statement")
				}
				fromIdx, fromBuilderExists := dp.nodeIDIdxMap[fromNode.ID()]
				if !fromBuilderExists {
					var fromErr error
					fromIdx, fromErr = dp.orchestrate(-1, fromAnnotation, insPsc, arrivingSourceNodeStream, departingSourceNodeStream)
					if fromErr != nil {
						return fromErr
					}
					dp.nodeIDIdxMap[fromNode.ID()] = fromIdx
				}
				toIdx, toBuilderExists := dp.nodeIDIdxMap[e.To().ID()]
				if !toBuilderExists {
					dp.annMap[toTableExpr] = toAnnotation
					toAnnotation.SetDynamic()
					var toErr error
					toIdx, toErr = dp.orchestrate(
						-1, toAnnotation, toInsPsc, arrivingDestinationNodeStream, departingDestinationNodeStream)
					if toErr != nil {
						return toErr
					}
					dp.nodeIDIdxMap[e.To().ID()] = toIdx
				}
				if !dp.dataflowEdgeExists(fromIdx, toIdx) {
					dp.dataflowToEdges[toIdx] = append(dp.dataflowToEdges[toIdx], fromIdx)
				}
			}
			for _, n := range orderedNodes {
				// another pass for AOT dataflows; to wit, on clauses
				if _, ok := idsVisited[n.ID()]; ok {
					continue
				}
				for _, e := range edges {
					toIdx, toFound := dp.nodeIDIdxMap[e.To().ID()]
					if !toFound {
						return fmt.Errorf("unknown to node index")
					}
					fromIdx, fromFound := dp.nodeIDIdxMap[e.From().ID()]
					if !fromFound {
						return fmt.Errorf("unknown from node index")
					}
					dp.dataflowToEdges[toIdx] = append(dp.dataflowToEdges[toIdx], fromIdx)
				}
			}
		default:
			return fmt.Errorf("cannot support dependency unit of type = '%T'", unit)
		}
	}
	maxWeaklyConnectedComponents := dp.handlerCtx.GetRuntimeContext().DataflowComponentsMax
	if weaklyConnectedComponentCount > maxWeaklyConnectedComponents {
		return fmt.Errorf(
			"data flow: there are too many weakly connected components; found = %d, max = 1",
			weaklyConnectedComponentCount)
	}
	rewrittenWhereStr := astvisit.GenerateModifiedWhereClause(
		dp.annotatedAST,
		dp.rewrittenWhere,
		dp.handlerCtx.GetSQLSystem(),
		dp.handlerCtx.GetASTFormatter(),
		dp.handlerCtx.GetNamespaceCollection())
	rewrittenWhereStr, err = dp.handlerCtx.GetSQLSystem().SanitizeWhereQueryString(rewrittenWhereStr)
	if err != nil {
		return err
	}
	logging.GetLogger().Debugf("rewrittenWhereStr = '%s'", rewrittenWhereStr)
	drmCfg, err := drm.GenerateDRMConfig(
		dp.handlerCtx.GetSQLSystem(),
		dp.handlerCtx.GetPersistenceSystem(),
		dp.handlerCtx.GetTypingConfig(),
		dp.handlerCtx.GetNamespaceCollection(),
		dp.handlerCtx.GetControlAttributes())
	if err != nil {
		return err
	}
	v := astvisit.NewQueryRewriteAstVisitor(
		dp.annotatedAST,
		dp.handlerCtx,
		dp.tblz,
		dp.tableSlice,
		dp.annMap,
		dp.discoGenIDs,
		dp.colRefs,
		drmCfg,
		dp.primaryTcc,
		dp.secondaryTccs,
		rewrittenWhereStr,
		drmCfg.GetNamespaceCollection(),
	).WithFormatter(drmCfg.GetSQLSystem().GetASTFormatter()).WithPrepStmtOffset(dp.prepStmtOffset)
	err = v.Visit(dp.sqlStatement)
	if err != nil {
		return err
	}
	selCtx, err := v.GenerateSelectDML()
	if err != nil {
		return err
	}
	selBld := primitivebuilder.NewSingleSelect(
		dp.primitiveComposer.GetGraphHolder(),
		dp.handlerCtx,
		selCtx,
		dp.tableSlice,
		nil,
		streaming.NewNopMapStream(),
	)
	selIndirect, indirectErr := astindirect.NewParserSelectIndirect(dp.sqlStatement, selCtx)
	if indirectErr != nil {
		return indirectErr
	}
	dp.annotatedAST.SetSelectIndirect(dp.sqlStatement, selIndirect)
	if dp.isElideRead {
		selBld = primitivebuilder.NewNopBuilder(
			dp.primitiveComposer.GetGraphHolder(),
			dp.primitiveComposer.GetTxnCtrlCtrs(),
			dp.handlerCtx,
			dp.handlerCtx.GetSQLEngine(),
			[]string{},
		)
	}
	// TODO: make this finer grained STAT
	dp.bldr = primitivebuilder.NewDependentMultipleAcquireAndSelect(
		dp.primitiveComposer.GetGraphHolder(),
		dp.execSlice,
		selBld,
		dp.dataflowToEdges,
		dp.handlerCtx.GetSQLSystem(),
	)
	dp.selCtx = selCtx
	return nil
}

func (dp *standardDependencyPlanner) processOrphan(
	sqlNode sqlparser.SQLNode,
	annotationCtx taxonomy.AnnotationCtx,
	vertex dataflow.Vertex,
) (drm.PreparedStatementCtx, internaldto.TxnControlCounters, error) {
	anTab, tcc, err := dp.processAcquire(sqlNode, annotationCtx, vertex)
	if err != nil {
		return nil, nil, err
	}

	tableMetadata := annotationCtx.GetTableMeta()

	_, isSQLDataSource := annotationCtx.GetTableMeta().GetSQLDataSource()
	var opStore anysdk.StandardOperationStore
	if !isSQLDataSource {
		opStore, err = annotationCtx.GetTableMeta().GetMethod()
		if err != nil {
			return nil, nil, err
		}
	} else {
		// Persist SQL mirror table here prior to generating insert DML
		drmCfg := dp.handlerCtx.GetDrmConfig()
		ddl, ddlErr := drmCfg.GenerateDDL(anTab, nil, nil, nil, opStore, annotationCtx.IsAwait(), 0, false, false)
		if ddlErr != nil {
			return nil, nil, ddlErr
		}
		err = dp.handlerCtx.GetSQLEngine().ExecInTxn(ddl)
		if err != nil {
			return nil, nil, err
		}

		insPsc, insPscErr := dp.primitiveComposer.GetDRMConfig().GenerateInsertDML(
			anTab,
			nil,
			nil,
			nil,
			opStore,
			tcc,
			false,
			annotationCtx.IsAwait(),
		)
		return insPsc, tcc, insPscErr
	}

	anySdkProv, anySdkPrvErr := tableMetadata.GetProviderObject()
	if anySdkPrvErr != nil {
		return nil, nil, anySdkPrvErr
	}
	svc, svcErr := tableMetadata.GetService()
	if svcErr != nil {
		return nil, nil, svcErr
	}
	resource, resourceErr := tableMetadata.GetResource()
	if resourceErr != nil {
		return nil, nil, resourceErr
	}
	insPsc, err := dp.primitiveComposer.GetDRMConfig().GenerateInsertDML(
		anTab,
		anySdkProv,
		svc,
		resource,
		opStore,
		tcc,
		false,
		annotationCtx.IsAwait(),
	)
	return insPsc, tcc, err
}

func (dp *standardDependencyPlanner) orchestrate(
	equivalencyGroupID int64,
	annotationCtx taxonomy.AnnotationCtx,
	insPsc drm.PreparedStatementCtx,
	inStream streaming.MapStream,
	outStream streaming.MapStream,
) (int, error) {
	rc, err := tableinsertioncontainer.NewTableInsertionContainer(
		annotationCtx.GetTableMeta(),
		dp.handlerCtx.GetSQLEngine(),
		dp.handlerCtx.GetTxnCounterMgr(),
	)
	if equivalencyGroupID > 0 {
		tcc, ok := dp.equivalencyGroupTCCs[equivalencyGroupID]
		if ok {
			tn, _ := rc.GetTableTxnCounters()
			setErr := rc.SetTableTxnCounters(tn, tcc)
			if setErr != nil {
				return -1, setErr
			}
		}
	}
	if err != nil {
		return -1, err
	}
	sqlDataSource, isSQLDataSource := annotationCtx.GetTableMeta().GetSQLDataSource()
	var builder primitivebuilder.Builder
	if isSQLDataSource {
		// TODO: generate query properly with ordered columns
		// starColNames := range insPsc.GetNonControlColumns()
		var colNames []string
		colz := insPsc.GetNonControlColumns()
		for _, col := range colz {
			//
			colNames = append(colNames, fmt.Sprintf(`"%s"`, col.GetIdentifier()))
		}
		projectionStr := strings.Join(colNames, ", ")
		_, tErr := dp.handlerCtx.GetDrmConfig().GetCurrentTable(annotationCtx.GetHIDs())
		if tErr != nil {
			return -1, tErr
		}
		tableName := annotationCtx.GetHIDs().GetSQLDataSourceTableName()
		// targetTableName := annotationCtx.GetHIDs().GetStackQLTableName()
		query := fmt.Sprintf(`SELECT %s FROM %s`, projectionStr, tableName)
		if sqlDataSource.GetDBName() == constants.SQLDbNameSnowflake {
			query = strings.ReplaceAll(query, `"`, ``)
		}
		builder = primitivebuilder.NewSQLDataSourceSingleSelectAcquire(
			dp.primitiveComposer.GetGraphHolder(),
			dp.handlerCtx,
			rc,
			query,
			nil,
			insPsc,
			nil,
			outStream,
		)
	} else {
		builder = primitivebuilder.NewSingleSelectAcquire(
			dp.primitiveComposer.GetGraphHolder(),
			dp.handlerCtx,
			rc,
			insPsc,
			nil,
			outStream,
			false, // returning hardcoded to false for now
		)
	}
	dp.execSlice = append(dp.execSlice, builder)
	idx := len(dp.execSlice) - 1
	dp.tableSlice = append(dp.tableSlice, rc)
	err = annotationCtx.Prepare(dp.handlerCtx, inStream)
	return idx, err
}

//nolint:gocognit,funlen // live with it
func (dp *standardDependencyPlanner) processAcquire(
	sqlNode sqlparser.SQLNode,
	annotationCtx taxonomy.AnnotationCtx,
	vertex dataflow.Vertex,
) (util.AnnotatedTabulation, internaldto.TxnControlCounters, error) {
	inputTableName, err := annotationCtx.GetInputTableName()
	inputProviderString := annotationCtx.GetHIDs().GetProviderStr()
	sqlDataSource, isSQLDataSource := dp.handlerCtx.GetSQLDataSource(inputProviderString)
	if isSQLDataSource {
		if dp.tcc == nil {
			return util.NewAnnotatedTabulation(nil, nil, "", ""),
				nil,
				fmt.Errorf("nil counters disallowed in dependency planner")
		}
		if !dp.tccSetAheadOfTime {
			dp.tcc = dp.tcc.CloneAndIncrementInsertID()
		}
		dp.secondaryTccs = append(dp.secondaryTccs, dp.tcc)
		anTab := util.NewAnnotatedTabulation(
			nil, annotationCtx.GetHIDs(),
			inputTableName,
			annotationCtx.GetTableMeta().GetAlias())
		anTab.SetSQLDataSource(sqlDataSource)
		return anTab, dp.tcc, nil
	}
	if err != nil {
		return util.NewAnnotatedTabulation(nil, nil, "", ""), nil, err
	}
	prov, err := annotationCtx.GetTableMeta().GetProviderObject()
	if err != nil {
		return util.NewAnnotatedTabulation(nil, nil, "", ""), nil, err
	}
	svc, err := annotationCtx.GetTableMeta().GetService()
	if err != nil {
		return util.NewAnnotatedTabulation(nil, nil, "", ""), nil, err
	}
	resource, rscErr := annotationCtx.GetTableMeta().GetResource()
	if rscErr != nil {
		return util.NewAnnotatedTabulation(nil, nil, "", ""), nil, rscErr
	}
	m, err := annotationCtx.GetTableMeta().GetMethod()
	if err != nil {
		return util.NewAnnotatedTabulation(nil, nil, "", ""), nil, err
	}
	selectItemsKey := annotationCtx.GetTableMeta().GetSelectItemsKey()
	if selectItemsKey == "" {
		selectItemsKey = m.GetSelectItemsKey()
	}
	var defaultColName string
	if selectItemsKey != "" {
		defaultColName = util.TrimSelectItemsKey(selectItemsKey)
	}
	tab := annotationCtx.GetSchema().Tabulate(false, defaultColName)
	_, mediaType, err := m.GetResponseBodySchemaAndMediaType()
	if err != nil {
		return util.NewAnnotatedTabulation(nil, nil, "", ""), nil, err
	}
	switch mediaType {
	case media.MediaTypeTextXML, media.MediaTypeXML:
		tab = tab.RenameColumnsToXml()
	}
	// TODO: support defaulting columns to where parameters
	anTab := util.NewAnnotatedTabulation(
		tab,
		annotationCtx.GetHIDs(),
		inputTableName,
		annotationCtx.GetTableMeta().GetAlias()).WithParameters(annotationCtx.GetParameters())

	discoGenID, err := docparser.OpenapiStackQLTabulationsPersistor(
		prov,
		svc,
		resource,
		m,
		annotationCtx.IsAwait(),
		[]util.AnnotatedTabulation{anTab},
		dp.primitiveComposer.GetSQLEngine(),
		prov.GetName(),
		dp.handlerCtx.GetNamespaceCollection(),
		dp.handlerCtx.GetControlAttributes(),
		dp.handlerCtx.GetSQLSystem(),
		dp.handlerCtx.GetPersistenceSystem(),
		dp.handlerCtx.GetTypingConfig(),
	)
	if err != nil {
		return util.NewAnnotatedTabulation(nil, nil, "", ""), nil, err
	}
	dp.discoGenIDs[sqlNode] = discoGenID
	if dp.tcc == nil {
		return util.NewAnnotatedTabulation(nil, nil, "", ""), nil, fmt.Errorf("nil counters disallowed in dependency planner")
	}
	if !dp.tccSetAheadOfTime {
		if vertex.GetEquivalencyGroup() > 0 {
			tcc, ok := dp.equivalencyGroupTCCs[vertex.GetEquivalencyGroup()]
			if ok {
				dp.tcc = tcc.Clone()
			} else {
				dp.tcc = dp.tcc.CloneAndIncrementInsertID()
				dp.secondaryTccs = append(dp.secondaryTccs, dp.tcc)
				dp.equivalencyGroupTCCs[vertex.GetEquivalencyGroup()] = dp.tcc
			}
			return anTab, dp.tcc, nil
		}
		dp.tcc = dp.tcc.CloneAndIncrementInsertID()
	}
	dp.secondaryTccs = append(dp.secondaryTccs, dp.tcc)
	return anTab, dp.tcc, nil
}

func (dp *standardDependencyPlanner) isVectorParam(param interface{}) bool {
	paramMeta, isParamMeta := param.(parserutil.ParameterMetadata)
	if isParamMeta {
		val := paramMeta.GetVal()
		_, valIsSQLVal := val.(sqlparser.ValTuple)
		if valIsSQLVal {
			return true
		}
	}
	return false
}

//nolint:gocognit,nestif // live with it
func (dp *standardDependencyPlanner) getStreamFromEdge(
	e dataflow.Edge,
	toAc taxonomy.AnnotationCtx,
	tcc internaldto.TxnControlCounters,
) (streaming.MapStream, error) {
	if e.IsSQL() {
		selectCtx, err := dp.generateSelectDML(e, tcc)
		if err != nil {
			return nil, err
		}
		ann := e.GetSource().GetAnnotation()
		meta := ann.GetTableMeta()
		insertContainer, err := tableinsertioncontainer.NewTableInsertionContainer(meta,
			dp.handlerCtx.GetSQLEngine(),
			dp.handlerCtx.GetTxnCounterMgr())
		if err != nil {
			return nil, err
		}
		toParams := toAc.GetParameters()

		transformedToStaticParams, paramTransformErr := util.TransformSQLRawParameters(toParams, false)
		if paramTransformErr != nil {
			return nil, paramTransformErr
		}
		for k, v := range transformedToStaticParams {
			hasSourceAttr := e.HasSourceAttribute(k)
			if !hasSourceAttr {
				vIsUnpopulated := k == v
				if vIsUnpopulated {
					delete(transformedToStaticParams, k)
				}
			}
		}
		return sqlstream.NewSimpleSQLMapStream(
			selectCtx,
			insertContainer,
			dp.handlerCtx.GetDrmConfig(),
			dp.handlerCtx.GetSQLEngine(),
			transformedToStaticParams,
		), nil
	}
	projection, err := e.GetProjection()
	if err != nil {
		return nil, err
	}
	incomingCols := make(map[string]struct{})
	for _, v := range projection {
		incomingCols[v] = struct{}{}
	}
	params := toAc.GetParameters()
	staticParams := make(map[string]interface{})
	for k, v := range params {
		isVector := dp.isVectorParam(v)
		if _, ok := incomingCols[k]; !ok && !isVector {
			staticParams[k] = v
			incomingCols[k] = struct{}{}
		}
	}
	if len(staticParams) > 0 {
		staticParams, err = util.TransformSQLRawParameters(staticParams, false)
		if err != nil {
			return nil, err
		}
	}
	return streaming.NewSimpleProjectionMapStream(projection, staticParams), nil
}

// This is very naive but safe because there is only ever one source table.
func (dp *standardDependencyPlanner) harvestFilter(sourceAnnotation taxonomy.AnnotationCtx) (string, bool) {
	var valz []string
	for k, p := range sourceAnnotation.GetParameters() {
		//nolint:gocritic // fine with this
		switch pt := p.(type) {
		case *sqlparser.SQLVal:
			val := string(pt.Val)
			s := fmt.Sprintf(`"%s" = '%s' `, k, val)
			valz = append(valz, s)
		}
	}
	if len(valz) == 0 {
		return "", false
	}
	rv := strings.Join(valz, " AND ")
	return rv, true
}

func (dp *standardDependencyPlanner) generateSelectDML(
	e dataflow.Edge,
	tcc internaldto.TxnControlCounters,
) (drm.PreparedStatementCtx, error) {
	ann := e.GetSource().GetAnnotation()
	whereStr, _ := dp.harvestFilter(ann)
	columnDescriptors, err := e.GetColumnDescriptors()
	if err != nil {
		return nil, err
	}
	discoID, discoIDErr := dp.handlerCtx.GetSQLEngine().GetCurrentDiscoveryGenerationID(ann.GetHIDs().GetProviderStr())
	if discoIDErr != nil {
		return nil, fmt.Errorf("error generating select dml: %w", discoIDErr)
	}
	alias := ann.GetTableMeta().GetAlias()
	tn, err := dp.handlerCtx.GetDrmConfig().GetTable(ann.GetHIDs(), discoID)
	if err != nil {
		return nil, err
	}
	tableName := fmt.Sprintf(`"%s"`, tn.GetName())
	// TODO: obtain namespace prefix for postgres
	tableSchema := tn.GetNameSpace()
	if tableSchema != "" {
		tableName = fmt.Sprintf(`"%s"."%s"`, tableSchema, tn.GetName())
	}
	if alias != "" {
		tableName = fmt.Sprintf("%s AS %s", tableName, alias)
	}
	relationalColumns := dp.handlerCtx.GetDrmConfig().OpenapiColumnsToRelationalColumns(columnDescriptors)
	rewriteInput := sqlrewrite.NewStandardSQLRewriteInput(
		dp.handlerCtx.GetDrmConfig(),
		relationalColumns,
		tcc,
		"",
		whereStr,
		dp.secondaryTccs,
		dp.tblz,
		tableName, // this is the sole from string
		nil,
		dp.handlerCtx.GetNamespaceCollection(),
		nil,
		map[string]interface{}{},
	).WithSelectQualifier("DISTINCT")
	return sqlrewrite.GenerateRewrittenSelectDML(rewriteInput)
}
