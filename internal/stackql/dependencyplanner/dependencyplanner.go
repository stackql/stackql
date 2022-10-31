package dependencyplanner

import (
	"fmt"

	"github.com/stackql/go-openapistackql/pkg/media"
	"github.com/stackql/stackql/internal/stackql/astvisit"
	"github.com/stackql/stackql/internal/stackql/dataflow"
	"github.com/stackql/stackql/internal/stackql/docparser"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/handler"
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
	"vitess.io/vitess/go/vt/sqlparser"
)

type DependencyPlanner interface {
	Plan() error
	GetBldr() primitivebuilder.Builder
	GetSelectCtx() *drm.PreparedStatementCtx
}

type StandardDependencyPlanner struct {
	dataflowCollection dataflow.DataFlowCollection
	colRefs            parserutil.ColTableMap
	handlerCtx         *handler.HandlerContext
	execSlice          []primitivebuilder.Builder
	primaryTcc, tcc    *dto.TxnControlCounters
	primitiveComposer  primitivecomposer.PrimitiveComposer
	rewrittenWhere     *sqlparser.Where
	secondaryTccs      []*dto.TxnControlCounters
	sqlStatement       sqlparser.SQLNode
	tableSlice         []tableinsertioncontainer.TableInsertionContainer
	tblz               taxonomy.TblMap
	discoGenIDs        map[sqlparser.SQLNode]int

	//
	bldr          primitivebuilder.Builder
	selCtx        *drm.PreparedStatementCtx
	defaultStream streaming.MapStream
	annMap        taxonomy.AnnotationCtxMap
}

func NewStandardDependencyPlanner(
	handlerCtx *handler.HandlerContext,
	dataflowCollection dataflow.DataFlowCollection,
	colRefs parserutil.ColTableMap,
	rewrittenWhere *sqlparser.Where,
	sqlStatement sqlparser.SQLNode,
	tblz taxonomy.TblMap,
	primitiveComposer primitivecomposer.PrimitiveComposer,
	tcc *dto.TxnControlCounters,
) (DependencyPlanner, error) {
	if tcc == nil {
		return nil, fmt.Errorf("violation of StandardDependencyPlanner invariant: txn counter cannot be nil")
	}
	return &StandardDependencyPlanner{
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
	}, nil
}

func (dp *StandardDependencyPlanner) GetBldr() primitivebuilder.Builder {
	return dp.bldr
}

func (dp *StandardDependencyPlanner) GetSelectCtx() *drm.PreparedStatementCtx {
	return dp.selCtx
}

func (dp *StandardDependencyPlanner) Plan() error {
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
		case dataflow.DataFlowVertex:
			inDegree := dp.dataflowCollection.InDegree(unit)
			outDegree := dp.dataflowCollection.OutDegree(unit)
			if inDegree == 0 && outDegree > 0 {
				// TODO: start builder
				logging.GetLogger().Infof("\n")
			}
			if inDegree != 0 || outDegree != 0 {
				return fmt.Errorf("cannot currently execute data dependent tables with inDegree = %d and/or outDegree = %d", inDegree, outDegree)
			}
			tableExpr := unit.GetTableExpr()
			annotation := unit.GetAnnotation()
			dp.annMap[tableExpr] = annotation
			insPsc, _, err := dp.processOrphan(tableExpr, annotation, dp.defaultStream)
			if err != nil {
				return err
			}
			stream := streaming.NewNopMapStream()
			err = dp.orchestrate(annotation, insPsc, dp.defaultStream, stream)
			if err != nil {
				return err
			}
		case dataflow.DataFlowWeaklyConnectedComponent:
			weaklyConnectedComponentCount++
			orderedNodes, err := unit.GetOrderedNodes()
			if err != nil {
				return err
			}
			logging.GetLogger().Infof("%v\n", orderedNodes)
			edges, err := unit.GetEdges()
			if err != nil {
				return err
			}
			logging.GetLogger().Infof("%v\n", edges)
			edgeCount := len(edges)
			if edgeCount > 1 {
				return fmt.Errorf("data flow: cannot accomodate table dependencies of this complexity: supplied = %d, max = 1", edgeCount)
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
					if e.From().ID() == n.ID() {

						//
						insPsc, tcc, err := dp.processOrphan(tableExpr, annotation, dp.defaultStream)
						if err != nil {
							return err
						}
						toNode := e.GetDest()
						toTableExpr := toNode.GetTableExpr()
						toAnnotation := toNode.GetAnnotation()
						stream, err := dp.getStreamFromEdge(e, toAnnotation, tcc)
						if err != nil {
							return err
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
		return fmt.Errorf("data flow: there are too many weakly connected components; found = %d, max = 1", weaklyConnectedComponentCount)
	}
	rewrittenWhereStr := astvisit.GenerateModifiedWhereClause(dp.rewrittenWhere, dp.handlerCtx.GetNamespaceCollection())
	logging.GetLogger().Debugf("rewrittenWhereStr = '%s'", rewrittenWhereStr)
	drmCfg, err := drm.GetGoogleV1SQLiteConfig(dp.handlerCtx.SQLEngine, dp.handlerCtx.GetNamespaceCollection(), dp.handlerCtx.ControlAttributes)
	if err != nil {
		return err
	}
	v := astvisit.NewQueryRewriteAstVisitor(
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
	)
	err = v.Visit(dp.sqlStatement)
	if err != nil {
		return err
	}
	selCtx, err := v.GenerateSelectDML()
	if err != nil {
		return err
	}
	selBld := primitivebuilder.NewSingleSelect(
		dp.primitiveComposer.GetGraph(),
		dp.handlerCtx,
		selCtx,
		dp.tableSlice,
		nil,
		streaming.NewNopMapStream(),
	)
	// TODO: make this finer grained STAT
	dp.bldr = primitivebuilder.NewDependentMultipleAcquireAndSelect(dp.primitiveComposer.GetGraph(), dp.execSlice, selBld)
	dp.selCtx = selCtx
	return nil
}

func (dp *StandardDependencyPlanner) processOrphan(sqlNode sqlparser.SQLNode, annotationCtx taxonomy.AnnotationCtx, inStream streaming.MapStream) (*drm.PreparedStatementCtx, *dto.TxnControlCounters, error) {
	anTab, tcc, err := dp.processAcquire(sqlNode, annotationCtx, inStream)
	if err != nil {
		return nil, nil, err
	}
	os, err := annotationCtx.GetTableMeta().GetMethod()
	if err != nil {
		return nil, nil, err
	}
	insPsc, err := dp.primitiveComposer.GetDRMConfig().GenerateInsertDML(anTab, os, tcc)
	return insPsc, tcc, err
}

func (dp *StandardDependencyPlanner) orchestrate(
	annotationCtx taxonomy.AnnotationCtx,
	insPsc *drm.PreparedStatementCtx,
	inStream streaming.MapStream,
	outStream streaming.MapStream,
) error {
	rc := tableinsertioncontainer.NewTableInsertionContainer(annotationCtx.GetTableMeta(), dp.handlerCtx.SQLEngine)
	builder := primitivebuilder.NewSingleSelectAcquire(
		dp.primitiveComposer.GetGraph(),
		dp.handlerCtx,
		rc,
		insPsc,
		nil,
		outStream,
	)
	dp.execSlice = append(dp.execSlice, builder)
	dp.tableSlice = append(dp.tableSlice, rc)
	err := annotationCtx.Prepare(dp.handlerCtx, inStream)
	return err
}

func (dp *StandardDependencyPlanner) processAcquire(
	sqlNode sqlparser.SQLNode,
	annotationCtx taxonomy.AnnotationCtx,
	stream streaming.MapStream,
) (util.AnnotatedTabulation, *dto.TxnControlCounters, error) {
	prov, err := annotationCtx.GetTableMeta().GetProviderObject()
	if err != nil {
		return util.NewAnnotatedTabulation(nil, nil, ""), nil, err
	}
	m, err := annotationCtx.GetTableMeta().GetMethod()
	if err != nil {
		return util.NewAnnotatedTabulation(nil, nil, ""), nil, err
	}
	tab := annotationCtx.GetSchema().Tabulate(false)
	_, mediaType, err := m.GetResponseBodySchemaAndMediaType()
	if err != nil {
		return util.NewAnnotatedTabulation(nil, nil, ""), nil, err
	}
	switch mediaType {
	case media.MediaTypeTextXML, media.MediaTypeXML:
		tab = tab.RenameColumnsToXml()
	}
	anTab := util.NewAnnotatedTabulation(tab, annotationCtx.GetHIDs(), annotationCtx.GetTableMeta().Alias)

	discoGenId, err := docparser.OpenapiStackQLTabulationsPersistor(m, []util.AnnotatedTabulation{anTab}, dp.primitiveComposer.GetSQLEngine(), prov.Name, dp.handlerCtx.GetNamespaceCollection(), dp.handlerCtx.ControlAttributes)
	if err != nil {
		return util.NewAnnotatedTabulation(nil, nil, ""), nil, err
	}
	dp.discoGenIDs[sqlNode] = discoGenId
	if dp.tcc == nil {
		return util.NewAnnotatedTabulation(nil, nil, ""), nil, fmt.Errorf("nil counters disallowed in dependency planner")
	} else {
		dp.tcc = dp.tcc.CloneAndIncrementInsertID()
		dp.secondaryTccs = append(dp.secondaryTccs, dp.tcc)
	}
	return anTab, dp.tcc, nil
}

func (dp *StandardDependencyPlanner) getStreamFromEdge(e dataflow.DataFlowEdge, ac taxonomy.AnnotationCtx, tcc *dto.TxnControlCounters) (streaming.MapStream, error) {
	if e.IsSQL() {
		selectCtx, err := dp.generateSelectDML(e, tcc)
		if err != nil {
			return nil, err
		}
		ann := e.GetSource().GetAnnotation()
		meta := ann.GetTableMeta()
		insertContainer := tableinsertioncontainer.NewTableInsertionContainer(meta, dp.handlerCtx.SQLEngine)
		return sqlstream.NewSimpleSQLMapStream(selectCtx, insertContainer, dp.handlerCtx.DrmConfig, dp.handlerCtx.SQLEngine), nil
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

func (dp *StandardDependencyPlanner) generateSelectDML(e dataflow.DataFlowEdge, tcc *dto.TxnControlCounters) (*drm.PreparedStatementCtx, error) {
	ann := e.GetSource().GetAnnotation()
	columnDescriptors, err := e.GetColumnDescriptors()
	if err != nil {
		return nil, err
	}
	alias := ann.GetTableMeta().Alias
	tn, err := dp.handlerCtx.DrmConfig.GetTableName(ann.GetHIDs(), dp.tcc.GenId)
	if err != nil {
		return nil, err
	}
	tableName := fmt.Sprintf(`"%s"`, tn)
	if alias != "" {
		tableName = fmt.Sprintf("%s AS %s", tableName, alias)
	}
	rewriteInput := sqlrewrite.NewStandardSQLRewriteInput(
		dp.handlerCtx.DrmConfig,
		columnDescriptors,
		tcc,
		"",
		"",
		dp.secondaryTccs,
		dp.tblz,
		tableName,
		nil,
		dp.handlerCtx.GetNamespaceCollection(),
	)
	return sqlrewrite.GenerateSelectDML(rewriteInput)
}
