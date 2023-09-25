package dependencyplanner

import (
	"fmt"
	"strings"

	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/go-openapistackql/pkg/media"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/astanalysis/annotatedast"
	"github.com/stackql/stackql/internal/stackql/astindirect"
	"github.com/stackql/stackql/internal/stackql/astvisit"
	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/dataflow"
	"github.com/stackql/stackql/internal/stackql/docparser"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/primitivebuilder"
	"github.com/stackql/stackql/internal/stackql/primitivecomposer"
	"github.com/stackql/stackql/internal/stackql/sqlrewrite"
	"github.com/stackql/stackql/internal/stackql/sqlstream"
	"github.com/stackql/stackql/internal/stackql/streaming"
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
	bldr          primitivebuilder.Builder
	selCtx        drm.PreparedStatementCtx
	defaultStream streaming.MapStream
	annMap        taxonomy.AnnotationCtxMap
	//
	prepStmtOffset int
	//
	isElideRead bool
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
		annotatedAST:       annotatedAST,
		handlerCtx:         handlerCtx,
		dataflowCollection: dataflowCollection,
		colRefs:            colRefs,
		rewrittenWhere:     rewrittenWhere,
		sqlStatement:       sqlStatement,
		tblz:               tblz,
		primitiveComposer:  primitiveComposer,
		discoGenIDs:        make(map[sqlparser.SQLNode]int),
		defaultStream:      streaming.NewStandardMapStream(),
		annMap:             make(taxonomy.AnnotationCtxMap),
		tcc:                tcc,
		tccSetAheadOfTime:  tccSetAheadOfTime,
	}, nil
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
	for _, unit := range units {
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
			if isView || isSubquery {
				dp.annMap[tableExpr] = annotation
				continue
			}
			dp.annMap[tableExpr] = annotation
			insPsc, _, insErr := dp.processOrphan(tableExpr, annotation, dp.defaultStream)
			if insErr != nil {
				return insErr
			}
			stream := streaming.NewNopMapStream()
			err = dp.orchestrate(annotation, insPsc, dp.defaultStream, stream)
			if err != nil {
				return err
			}
		case dataflow.WeaklyConnectedComponent:
			weaklyConnectedComponentCount++
			orderedNodes, oErr := unit.GetOrderedNodes()
			if oErr != nil {
				return oErr
			}
			logging.GetLogger().Infof("%v\n", orderedNodes)
			edges, eErr := unit.GetEdges()
			if eErr != nil {
				return eErr
			}
			logging.GetLogger().Infof("%v\n", edges)
			edgeCount := len(edges)
			// TODO: test this
			if edgeCount > dp.handlerCtx.GetRuntimeContext().DataflowDependencyMax {
				return fmt.Errorf(
					"data flow: cannot accomodate table dependencies of this complexity: supplied = %d, max = 1",
					edgeCount)
			}
			idsVisited := make(map[int64]struct{})
			for _, n := range orderedNodes {
				if _, ok := idsVisited[n.ID()]; ok {
					continue
				}
				idsVisited[n.ID()] = struct{}{}
				tableExpr := n.GetTableExpr()
				annotation := n.GetAnnotation()
				dp.annMap[tableExpr] = annotation
				for _, e := range edges {
					//nolint:nestif // TODO: refactor
					if e.From().ID() == n.ID() {
						//
						insPsc, tcc, insErr := dp.processOrphan(tableExpr, annotation, dp.defaultStream)
						if insErr != nil {
							return insErr
						}
						toNode := e.GetDest()
						toTableExpr := toNode.GetTableExpr()
						toAnnotation := toNode.GetAnnotation()
						stream, streamErr := dp.getStreamFromEdge(e, toAnnotation, tcc)
						if streamErr != nil {
							return streamErr
						}
						err = dp.orchestrate(annotation, insPsc, dp.defaultStream, stream)
						if err != nil {
							return err
						}
						//
						dp.annMap[toTableExpr] = toAnnotation
						toAnnotation.SetDynamic()
						insPsc, _, err = dp.processOrphan(toTableExpr, toAnnotation, stream)
						if err != nil {
							return err
						}
						err = dp.orchestrate(toAnnotation, insPsc, stream, streaming.NewNopMapStream())
						if err != nil {
							return err
						}
						idsVisited[toNode.ID()] = struct{}{}
					}
				}
			}
		default:
			return fmt.Errorf("cannot support dependency unit of type = '%T'", unit)
		}
	}
	if weaklyConnectedComponentCount > 1 {
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
	drmCfg, err := drm.GetDRMConfig(
		dp.handlerCtx.GetSQLSystem(),
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
	)
	dp.selCtx = selCtx
	return nil
}

func (dp *standardDependencyPlanner) processOrphan(
	sqlNode sqlparser.SQLNode,
	annotationCtx taxonomy.AnnotationCtx,
	inStream streaming.MapStream,
) (drm.PreparedStatementCtx, internaldto.TxnControlCounters, error) {
	anTab, tcc, err := dp.processAcquire(sqlNode, annotationCtx, inStream)
	if err != nil {
		return nil, nil, err
	}
	_, isSQLDataSource := annotationCtx.GetTableMeta().GetSQLDataSource()
	var opStore openapistackql.OperationStore
	if !isSQLDataSource {
		opStore, err = annotationCtx.GetTableMeta().GetMethod()
		if err != nil {
			return nil, nil, err
		}
	} else {
		// Persist SQL mirror table here prior to generating insert DML
		drmCfg := dp.handlerCtx.GetDrmConfig()
		ddl, ddlErr := drmCfg.GenerateDDL(anTab, opStore, 0, false)
		if ddlErr != nil {
			return nil, nil, ddlErr
		}
		err = dp.handlerCtx.GetSQLEngine().ExecInTxn(ddl)
		if err != nil {
			return nil, nil, err
		}
	}
	insPsc, err := dp.primitiveComposer.GetDRMConfig().GenerateInsertDML(anTab, opStore, tcc)
	return insPsc, tcc, err
}

func (dp *standardDependencyPlanner) orchestrate(
	annotationCtx taxonomy.AnnotationCtx,
	insPsc drm.PreparedStatementCtx,
	inStream streaming.MapStream,
	outStream streaming.MapStream,
) error {
	rc, err := tableinsertioncontainer.NewTableInsertionContainer(
		annotationCtx.GetTableMeta(),
		dp.handlerCtx.GetSQLEngine())
	if err != nil {
		return err
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
			return tErr
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
		)
	}
	dp.execSlice = append(dp.execSlice, builder)
	dp.tableSlice = append(dp.tableSlice, rc)
	err = annotationCtx.Prepare(dp.handlerCtx, inStream)
	return err
}

func (dp *standardDependencyPlanner) processAcquire(
	sqlNode sqlparser.SQLNode,
	annotationCtx taxonomy.AnnotationCtx,
	stream streaming.MapStream, //nolint:unparam,revive // TODO: remove this
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
	m, err := annotationCtx.GetTableMeta().GetMethod()
	if err != nil {
		return util.NewAnnotatedTabulation(nil, nil, "", ""), nil, err
	}
	tab := annotationCtx.GetSchema().Tabulate(false)
	_, mediaType, err := m.GetResponseBodySchemaAndMediaType()
	if err != nil {
		return util.NewAnnotatedTabulation(nil, nil, "", ""), nil, err
	}
	switch mediaType {
	case media.MediaTypeTextXML, media.MediaTypeXML:
		tab = tab.RenameColumnsToXml()
	}
	anTab := util.NewAnnotatedTabulation(
		tab,
		annotationCtx.GetHIDs(),
		inputTableName,
		annotationCtx.GetTableMeta().GetAlias())

	discoGenID, err := docparser.OpenapiStackQLTabulationsPersistor(
		m,
		[]util.AnnotatedTabulation{anTab},
		dp.primitiveComposer.GetSQLEngine(),
		prov.GetName(),
		dp.handlerCtx.GetNamespaceCollection(),
		dp.handlerCtx.GetControlAttributes(),
		dp.handlerCtx.GetSQLSystem(),
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
		dp.tcc = dp.tcc.CloneAndIncrementInsertID()
	}
	dp.secondaryTccs = append(dp.secondaryTccs, dp.tcc)
	return anTab, dp.tcc, nil
}

func (dp *standardDependencyPlanner) getStreamFromEdge(
	e dataflow.Edge,
	ac taxonomy.AnnotationCtx,
	tcc internaldto.TxnControlCounters,
) (streaming.MapStream, error) {
	if e.IsSQL() {
		selectCtx, err := dp.generateSelectDML(e, tcc)
		if err != nil {
			return nil, err
		}
		ann := e.GetSource().GetAnnotation()
		meta := ann.GetTableMeta()
		insertContainer, err := tableinsertioncontainer.NewTableInsertionContainer(meta, dp.handlerCtx.GetSQLEngine())
		if err != nil {
			return nil, err
		}
		return sqlstream.NewSimpleSQLMapStream(
			selectCtx,
			insertContainer,
			dp.handlerCtx.GetDrmConfig(),
			dp.handlerCtx.GetSQLEngine(),
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
	params := ac.GetParameters()
	staticParams := make(map[string]interface{})
	for k, v := range params {
		if _, ok := incomingCols[k]; !ok {
			staticParams[k] = v
			incomingCols[k] = struct{}{}
		}
	}
	if len(staticParams) > 0 {
		staticParams, err = util.TransformSQLRawParameters(staticParams)
		if err != nil {
			return nil, err
		}
	}
	return streaming.NewSimpleProjectionMapStream(projection, staticParams), nil
}

func (dp *standardDependencyPlanner) generateSelectDML(
	e dataflow.Edge,
	tcc internaldto.TxnControlCounters,
) (drm.PreparedStatementCtx, error) {
	ann := e.GetSource().GetAnnotation()
	columnDescriptors, err := e.GetColumnDescriptors()
	if err != nil {
		return nil, err
	}
	alias := ann.GetTableMeta().GetAlias()
	tn, err := dp.handlerCtx.GetDrmConfig().GetTable(ann.GetHIDs(), dp.tcc.GetGenID())
	if err != nil {
		return nil, err
	}
	tableName := fmt.Sprintf(`"%s"`, tn.GetName())
	if alias != "" {
		tableName = fmt.Sprintf("%s AS %s", tableName, alias)
	}
	relationalColumns := dp.handlerCtx.GetDrmConfig().OpenapiColumnsToRelationalColumns(columnDescriptors)
	rewriteInput := sqlrewrite.NewStandardSQLRewriteInput(
		dp.handlerCtx.GetDrmConfig(),
		relationalColumns,
		tcc,
		"",
		"",
		dp.secondaryTccs,
		dp.tblz,
		tableName,
		nil,
		dp.handlerCtx.GetNamespaceCollection(),
		nil,
	)
	return sqlrewrite.GenerateRewrittenSelectDML(rewriteInput)
}
