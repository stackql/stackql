package primitivebuilder

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/jeroenrinzema/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/httpexec"
	"github.com/stackql/stackql/internal/stackql/httpmiddleware"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/go-openapistackql/openapistackql"

	"vitess.io/vitess/go/sqltypes"
	querypb "vitess.io/vitess/go/vt/proto/query"

	log "github.com/sirupsen/logrus"
)

var ()

type Builder interface {
	Build() error

	GetRoot() primitivegraph.PrimitiveNode

	GetTail() primitivegraph.PrimitiveNode
}

type SingleSelectAcquire struct {
	primitiveBuilder           *PrimitiveBuilder
	query                      string
	handlerCtx                 *handler.HandlerContext
	tableMeta                  taxonomy.ExtendedTableMetadata
	tabulation                 openapistackql.Tabulation
	drmCfg                     drm.DRMConfig
	insertPreparedStatementCtx *drm.PreparedStatementCtx
	selectPreparedStatementCtx *drm.PreparedStatementCtx
	txnCtrlCtr                 *dto.TxnControlCounters
	rowSort                    func(map[string]map[string]interface{}) []string
	root                       primitivegraph.PrimitiveNode
}

type NullaryAction struct {
	primitiveBuilder *PrimitiveBuilder
	query            string
	handlerCtx       *handler.HandlerContext
	tableMeta        taxonomy.ExtendedTableMetadata
	tabulation       openapistackql.Tabulation
	drmCfg           drm.DRMConfig
	txnCtrlCtr       *dto.TxnControlCounters
	root             primitivegraph.PrimitiveNode
}

type SingleSelect struct {
	primitiveBuilder           *PrimitiveBuilder
	query                      string
	handlerCtx                 *handler.HandlerContext
	tableMeta                  taxonomy.ExtendedTableMetadata
	tabulation                 openapistackql.Tabulation
	drmCfg                     drm.DRMConfig
	insertPreparedStatementCtx *drm.PreparedStatementCtx
	selectPreparedStatementCtx *drm.PreparedStatementCtx
	txnCtrlCtr                 *dto.TxnControlCounters
	rowSort                    func(map[string]map[string]interface{}) []string
	root                       primitivegraph.PrimitiveNode
}

type SubTreeBuilder struct {
	children []Builder
}

type DiamondBuilder struct {
	SubTreeBuilder
	parentBuilder            Builder
	graph                    *primitivegraph.PrimitiveGraph
	root, tailRoot, tailTail primitivegraph.PrimitiveNode
	sqlEngine                sqlengine.SQLEngine
	shouldCollectGarbage     bool
	txnControlCounterSlice   []dto.TxnControlCounters
}

func NewSubTreeBuilder(children []Builder) *SubTreeBuilder {
	return &SubTreeBuilder{
		children: children,
	}
}

func NewDiamondBuilder(parent Builder, children []Builder, graph *primitivegraph.PrimitiveGraph, sqlEngine sqlengine.SQLEngine, shouldCollectGarbage bool) *DiamondBuilder {
	return &DiamondBuilder{
		SubTreeBuilder:       SubTreeBuilder{children: children},
		parentBuilder:        parent,
		graph:                graph,
		sqlEngine:            sqlEngine,
		shouldCollectGarbage: shouldCollectGarbage,
	}
}

func (st *SubTreeBuilder) Build() error {
	for _, child := range st.children {
		err := child.Build()
		if err != nil {
			return err
		}
	}
	return nil
}

func (st *SubTreeBuilder) GetRoot() primitivegraph.PrimitiveNode {
	return st.children[0].GetRoot()
}

func (st *SubTreeBuilder) GetTail() primitivegraph.PrimitiveNode {
	return st.children[len(st.children)-1].GetTail()
}

func (db *DiamondBuilder) Build() error {
	for _, child := range db.children {
		err := child.Build()
		if err != nil {
			return err
		}
	}
	db.root = db.graph.CreatePrimitiveNode(NewPassThroughPrimitive(db.sqlEngine, db.graph.GetTxnControlCounterSlice(), false))
	if db.parentBuilder != nil {
		err := db.parentBuilder.Build()
		if err != nil {
			return err
		}
		db.tailRoot = db.parentBuilder.GetRoot()
		db.tailTail = db.parentBuilder.GetTail()
	} else {
		db.tailRoot = db.graph.CreatePrimitiveNode(NewPassThroughPrimitive(db.sqlEngine, db.graph.GetTxnControlCounterSlice(), db.shouldCollectGarbage))
		db.tailTail = db.tailRoot
	}
	for _, child := range db.children {
		root := child.GetRoot()
		tail := child.GetTail()
		db.graph.NewDependency(db.root, root, 1.0)
		db.graph.NewDependency(tail, db.tailRoot, 1.0)
		// db.tail.Primitive = child.GetTail().Primitive
	}
	return nil
}

func (db *DiamondBuilder) GetRoot() primitivegraph.PrimitiveNode {
	return db.root
}

func (db *DiamondBuilder) GetTail() primitivegraph.PrimitiveNode {
	return db.tailTail
}

type SingleAcquireAndSelect struct {
	primitiveBuilder *PrimitiveBuilder
	acquireBuilder   *SingleSelectAcquire
	selectBuilder    *SingleSelect
}

type Join struct {
	lhsPb, rhsPb *PrimitiveBuilder
	lhs, rhs     Builder
	handlerCtx   *handler.HandlerContext
	rowSort      func(map[string]map[string]interface{}) []string
}

func NewSingleSelectAcquire(pb *PrimitiveBuilder, handlerCtx *handler.HandlerContext, tableMeta taxonomy.ExtendedTableMetadata, insertCtx *drm.PreparedStatementCtx, selectCtx *drm.PreparedStatementCtx, rowSort func(map[string]map[string]interface{}) []string) *SingleSelectAcquire {
	var tcc *dto.TxnControlCounters
	if selectCtx != nil {
		tcc = selectCtx.TxnCtrlCtrs
	}
	return &SingleSelectAcquire{
		primitiveBuilder:           pb,
		handlerCtx:                 handlerCtx,
		tableMeta:                  tableMeta,
		rowSort:                    rowSort,
		drmCfg:                     handlerCtx.DrmConfig,
		insertPreparedStatementCtx: insertCtx,
		selectPreparedStatementCtx: selectCtx,
		txnCtrlCtr:                 tcc,
	}
}

func NewSingleSelect(pb *PrimitiveBuilder, handlerCtx *handler.HandlerContext, tableMeta taxonomy.ExtendedTableMetadata, insertCtx *drm.PreparedStatementCtx, selectCtx *drm.PreparedStatementCtx, rowSort func(map[string]map[string]interface{}) []string) *SingleSelect {
	return &SingleSelect{
		primitiveBuilder:           pb,
		handlerCtx:                 handlerCtx,
		tableMeta:                  tableMeta,
		rowSort:                    rowSort,
		drmCfg:                     handlerCtx.DrmConfig,
		insertPreparedStatementCtx: insertCtx,
		selectPreparedStatementCtx: selectCtx,
		txnCtrlCtr:                 selectCtx.TxnCtrlCtrs,
	}
}

type Union struct {
	unionCtx         *drm.PreparedStatementCtx
	primitiveBuilder *PrimitiveBuilder
	handlerCtx       *handler.HandlerContext
	drmCfg           drm.DRMConfig
	lhs              *drm.PreparedStatementCtx
	rhs              []*drm.PreparedStatementCtx
	root, tail       primitivegraph.PrimitiveNode
}

func (un *Union) Build() error {
	unionEx := func(pc primitive.IPrimitiveCtx) dto.ExecutorOutput {
		us := drm.NewPreparedStatementParameterized(un.unionCtx, nil, false)
		i := 0
		us.AddChild(i, drm.NewPreparedStatementParameterized(un.lhs, nil, true))
		for _, rhsElement := range un.rhs {
			i++
			us.AddChild(i, drm.NewPreparedStatementParameterized(rhsElement, nil, true))
		}
		return prepareGolangResult(un.handlerCtx.SQLEngine, us, un.lhs.NonControlColumns, un.drmCfg)
	}
	graph := un.primitiveBuilder.GetGraph()
	unionNode := graph.CreatePrimitiveNode(NewLocalPrimitive(unionEx))
	un.root = unionNode
	return nil
}

func NewUnion(pb *PrimitiveBuilder, handlerCtx *handler.HandlerContext, unionCtx *drm.PreparedStatementCtx, lhs *drm.PreparedStatementCtx, rhs []*drm.PreparedStatementCtx) *Union {
	return &Union{
		primitiveBuilder: pb,
		handlerCtx:       handlerCtx,
		drmCfg:           handlerCtx.DrmConfig,
		unionCtx:         unionCtx,
		lhs:              lhs,
		rhs:              rhs,
	}
}

func NewSingleAcquireAndSelect(pb *PrimitiveBuilder, handlerCtx *handler.HandlerContext, tableMeta taxonomy.ExtendedTableMetadata, insertCtx *drm.PreparedStatementCtx, selectCtx *drm.PreparedStatementCtx, rowSort func(map[string]map[string]interface{}) []string) *SingleAcquireAndSelect {
	return &SingleAcquireAndSelect{
		primitiveBuilder: pb,
		acquireBuilder:   NewSingleSelectAcquire(pb, handlerCtx, tableMeta, insertCtx, selectCtx, rowSort),
		selectBuilder:    NewSingleSelect(pb, handlerCtx, tableMeta, insertCtx, selectCtx, rowSort),
	}
}

func NewJoin(lhsPb *PrimitiveBuilder, rhsPb *PrimitiveBuilder, handlerCtx *handler.HandlerContext, rowSort func(map[string]map[string]interface{}) []string) *Join {
	return &Join{
		lhsPb:      lhsPb,
		rhsPb:      rhsPb,
		handlerCtx: handlerCtx,
		rowSort:    rowSort,
	}
}

func (ss *SingleSelect) Build() error {

	selectEx := func(pc primitive.IPrimitiveCtx) dto.ExecutorOutput {

		// select phase
		log.Infoln(fmt.Sprintf("running select with control parameters: %v", ss.selectPreparedStatementCtx.TxnCtrlCtrs))

		return prepareGolangResult(ss.handlerCtx.SQLEngine, drm.NewPreparedStatementParameterized(ss.selectPreparedStatementCtx, nil, true), ss.selectPreparedStatementCtx.NonControlColumns, ss.drmCfg)
	}
	graph := ss.primitiveBuilder.GetGraph()
	selectNode := graph.CreatePrimitiveNode(NewLocalPrimitive(selectEx))
	ss.root = selectNode

	return nil
}

func prepareGolangResult(sqlEngine sqlengine.SQLEngine, stmtCtx drm.PreparedStatementParameterized, nonControlColumns []drm.ColumnMetadata, drmCfg drm.DRMConfig) dto.ExecutorOutput {
	r, sqlErr := drmCfg.QueryDML(
		sqlEngine,
		stmtCtx,
	)
	log.Infoln(fmt.Sprintf("select result = %v, error = %v", r, sqlErr))
	altKeys := make(map[string]map[string]interface{})
	rawRows := make(map[int]map[int]interface{})
	var ks []int
	i := 0
	var keyArr []string
	var ifArr []interface{}
	for i < len(nonControlColumns) {
		x := nonControlColumns[i]
		y := drmCfg.GetGolangValue(x.GetType())
		ifArr = append(ifArr, y)
		keyArr = append(keyArr, x.Column.GetIdentifier())
		i++
	}
	if r != nil {
		i := 0
		for r.Next() {
			errScan := r.Scan(ifArr...)
			if errScan != nil {
				log.Infoln(fmt.Sprintf("%v", errScan))
			}
			for ord, val := range ifArr {
				log.Infoln(fmt.Sprintf("col #%d '%s':  %v  type: %T", ord, nonControlColumns[ord].GetName(), val, val))
			}
			im := make(map[string]interface{})
			imRaw := make(map[int]interface{})
			for ord, key := range keyArr {
				val := ifArr[ord]
				ev := drmCfg.ExtractFromGolangValue(val)
				im[key] = ev
				imRaw[ord] = ev
			}
			altKeys[strconv.Itoa(i)] = im
			rawRows[i] = imRaw
			ks = append(ks, i)
			i++
		}

		for ord := range ks {
			val := altKeys[strconv.Itoa(ord)]
			log.Infoln(fmt.Sprintf("row #%d:  %v  type: %T", ord, val, val))
		}
	}
	var cNames []string
	for _, v := range nonControlColumns {
		cNames = append(cNames, v.Column.GetIdentifier())
	}
	rowSort := func(m map[string]map[string]interface{}) []string {
		var arr []int
		for k, _ := range m {
			ord, _ := strconv.Atoi(k)
			arr = append(arr, ord)
		}
		sort.Ints(arr)
		var rv []string
		for _, v := range arr {
			rv = append(rv, strconv.Itoa(v))
		}
		return rv
	}
	rv := util.PrepareResultSet(dto.NewPrepareResultSetPlusRawDTO(nil, altKeys, cNames, rowSort, nil, nil, rawRows))
	if rv.GetSQLResult() == nil {

		resVal := &sqltypes.Result{
			Fields: make([]*querypb.Field, len(nonControlColumns)),
		}

		var colz []string
		for _, col := range nonControlColumns {
			colz = append(colz, col.GetIdentifier())
		}

		for f := range resVal.Fields {
			resVal.Fields[f] = &querypb.Field{
				Name: cNames[f],
			}
		}
		rv.GetSQLResult = func() sqldata.ISQLResultStream { return util.GetHeaderOnlyResultStream(colz) }
	}
	return rv
}

func (ss *SingleSelectAcquire) Build() error {
	prov, err := ss.tableMeta.GetProvider()
	if err != nil {
		return err
	}
	ex := func(pc primitive.IPrimitiveCtx) dto.ExecutorOutput {
		ss.primitiveBuilder.graph.AddTxnControlCounters(*ss.primitiveBuilder.txnCtrlCtrs)
		mr := prov.InferMaxResultsElement(ss.tableMeta.HeirarchyObjects.Method)
		if mr != nil {
			// TODO: infer param position and act accordingly
			ok := true
			if ok && ss.handlerCtx.RuntimeContext.HTTPMaxResults > 0 {
				for i, param := range ss.tableMeta.HttpArmoury.RequestParams {
					// param.Context.SetQueryParam("maxResults", strconv.Itoa(ss.handlerCtx.RuntimeContext.HTTPMaxResults))
					q := param.Request.URL.Query()
					q.Set("maxResults", strconv.Itoa(ss.handlerCtx.RuntimeContext.HTTPMaxResults))
					param.Request.URL.RawQuery = q.Encode()
					ss.tableMeta.HttpArmoury.RequestParams[i] = param
				}
			}
		}
		for _, reqCtx := range ss.tableMeta.HttpArmoury.RequestParams {
			response, apiErr := httpmiddleware.HttpApiCallFromRequest(*(ss.handlerCtx), prov, reqCtx.Request.Clone(context.Background()))
			housekeepingDone := false
			npt := prov.InferNextPageResponseElement(ss.tableMeta.HeirarchyObjects.Method)
			nptKey := prov.InferNextPageRequestElement(ss.tableMeta.HeirarchyObjects.Method)
			for {
				if apiErr != nil {
					return util.PrepareResultSet(dto.NewPrepareResultSetDTO(nil, nil, nil, ss.rowSort, apiErr, nil))
				}
				target, err := httpexec.ProcessHttpResponse(response)
				if err != nil {
					return dto.NewErroneousExecutorOutput(err)
				}
				log.Infoln(fmt.Sprintf("target = %v", target))
				var items interface{}
				var ok bool
				switch pl := target.(type) {
				case map[string]interface{}:
					if ss.tableMeta.SelectItemsKey != "" {
						items, ok = pl[ss.tableMeta.SelectItemsKey]
					} else {
						items = []interface{}{
							pl,
						}
						ok = true
					}
				case []interface{}:
					items = pl
					ok = true
				case nil:
					return dto.ExecutorOutput{}
				}
				keys := make(map[string]map[string]interface{})

				if ok {
					iArr, ok := items.([]interface{})
					if ok && len(iArr) > 0 {
						if !housekeepingDone && ss.insertPreparedStatementCtx != nil {
							_, err = ss.handlerCtx.SQLEngine.Exec(ss.insertPreparedStatementCtx.GetGCHousekeepingQueries())
							housekeepingDone = true
						}
						if err != nil {
							return dto.NewErroneousExecutorOutput(err)
						}

						for i := range iArr {
							item, ok := iArr[i].(map[string]interface{})
							if !ok {
								if iArr[i] != nil {
									item = map[string]interface{}{openapistackql.AnonymousColumnName: iArr[i]}
									ok = true
								}
							}
							if ok {

								log.Infoln(fmt.Sprintf("running insert with control parameters: %v", ss.insertPreparedStatementCtx.TxnCtrlCtrs))
								r, err := ss.drmCfg.ExecuteInsertDML(ss.handlerCtx.SQLEngine, ss.insertPreparedStatementCtx, item)
								log.Infoln(fmt.Sprintf("insert result = %v, error = %v", r, err))
								if err != nil {
									return dto.NewErroneousExecutorOutput(err)
								}
								keys[strconv.Itoa(i)] = item
							}
						}
					}
				}
				if npt == nil || nptKey == nil {
					break
				}
				tk := extractNextPageToken(target, npt.Name)
				if tk == "" {
					break
				}
				q := reqCtx.Request.URL.Query()
				q.Set(nptKey.Name, tk)
				reqCtx.Request.URL.RawQuery = q.Encode()
				response, apiErr = httpmiddleware.HttpApiCallFromRequest(*(ss.handlerCtx), prov, reqCtx.Request)
			}
			if reqCtx.Request != nil {
				q := reqCtx.Request.URL.Query()
				q.Del(nptKey.Name)
				reqCtx.Request.URL.RawQuery = q.Encode()
			}
		}
		return dto.ExecutorOutput{}
	}

	prep := func() *drm.PreparedStatementCtx {
		return ss.selectPreparedStatementCtx
	}
	insertPrim := NewHTTPRestPrimitive(
		prov,
		ex,
		prep,
		ss.txnCtrlCtr,
	)
	graph := ss.primitiveBuilder.GetGraph()
	insertNode := graph.CreatePrimitiveNode(insertPrim)
	ss.root = insertNode

	return nil
}

func extractNextPageToken(body interface{}, tokenKey string) string {
	switch target := body.(type) {
	case map[string]interface{}:
		nextPageToken, ok := target[tokenKey]
		if !ok || nextPageToken == "" {
			log.Infoln("breaking out")
			return ""
		}
		tk, ok := nextPageToken.(string)
		if !ok {
			log.Infoln("breaking out")
			return ""
		}
		return tk
	}
	return ""
}

func (ss *SingleAcquireAndSelect) GetRoot() primitivegraph.PrimitiveNode {
	return ss.acquireBuilder.GetRoot()
}

func (ss *SingleAcquireAndSelect) GetTail() primitivegraph.PrimitiveNode {
	return ss.selectBuilder.GetTail()
}

func (ss *SingleAcquireAndSelect) Build() error {
	err := ss.acquireBuilder.Build()
	if err != nil {
		return err
	}
	err = ss.selectBuilder.Build()
	if err != nil {
		return err
	}
	graph := ss.primitiveBuilder.GetGraph()
	graph.NewDependency(ss.acquireBuilder.root, ss.selectBuilder.root, 1.0)
	return nil
}

func (ss *SingleSelect) GetRoot() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *SingleSelect) GetTail() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *Union) GetRoot() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *Union) GetTail() primitivegraph.PrimitiveNode {
	return ss.tail
}

func (ss *SingleSelectAcquire) GetRoot() primitivegraph.PrimitiveNode {
	return ss.root
}

func (ss *SingleSelectAcquire) GetTail() primitivegraph.PrimitiveNode {
	return ss.root
}

func (j *Join) Build() error {
	return nil
}

func (j *Join) getErrNode() primitivegraph.PrimitiveNode {
	graph := j.lhsPb.GetGraph()
	return graph.CreatePrimitiveNode(
		NewLocalPrimitive(
			func(pc primitive.IPrimitiveCtx) dto.ExecutorOutput {
				return util.GenerateSimpleErroneousOutput(fmt.Errorf("joins not yet supported"))
			},
		))
}

func (j *Join) GetRoot() primitivegraph.PrimitiveNode {
	return j.getErrNode()
}

func (j *Join) GetTail() primitivegraph.PrimitiveNode {
	return j.getErrNode()
}
