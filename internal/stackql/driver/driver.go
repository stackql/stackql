package driver

import (
	"bytes"
	"context"
	"fmt"

	"github.com/stackql/any-sdk/pkg/logging"
	"github.com/stackql/any-sdk/public/sqlengine"
	"github.com/stackql/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql/internal/stackql/acid/tsm_physio"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/paramdecoder"
	"github.com/stackql/stackql/internal/stackql/queryshape"
	"github.com/stackql/stackql/internal/stackql/responsehandler"
	"github.com/stackql/stackql/internal/stackql/util"
	"github.com/stackql/stackql/pkg/txncounter"

	sqlbackend "github.com/stackql/psql-wire/pkg/sqlbackend"
)

var (
	_ StackQLDriver                    = &basicStackQLDriver{}
	_ sqlbackend.IExtendedQueryBackend = &basicStackQLDriver{}
	_ sqlbackend.SQLBackendFactory     = &basicStackQLDriverFactory{}
	_ StackQLDriverFactory             = &basicStackQLDriverFactory{}
)

type StackQLDriverFactory interface {
	NewSQLDriver() (StackQLDriver, error)
}

type basicStackQLDriverFactory struct {
	isCaptureDebug bool
	handlerCtx     handler.HandlerContext
}

func (sdf *basicStackQLDriverFactory) NewSQLBackend() (sqlbackend.ISQLBackend, error) {
	return sdf.newSQLDriver()
}

func (sdf *basicStackQLDriverFactory) NewSQLDriver() (StackQLDriver, error) {
	return sdf.newSQLDriver()
}

func (sdf *basicStackQLDriverFactory) newSQLDriver() (StackQLDriver, error) {
	txCtr, err := getTxnCounterManager(sdf.handlerCtx.GetSQLEngine())
	if err != nil {
		return nil, err
	}
	txnOrchestrator, orcErr := tsm_physio.NewOrchestrator(sdf.handlerCtx)
	if orcErr != nil {
		return nil, orcErr
	}
	tsmInstance, walError := tsm_physio.GetTSM(sdf.handlerCtx)
	if walError != nil {
		return nil, walError
	}
	sdf.handlerCtx.SetTSM(tsmInstance)
	clonedCtx := sdf.handlerCtx.Clone()
	clonedCtx.SetTxnCounterMgr(txCtr)
	buf := bytes.NewBuffer([]byte{})
	if sdf.isCaptureDebug {
		logging.GetLogger().Debugln("debug mode enabled")
		clonedCtx.SetOutErrFile(buf)
	}
	rv := &basicStackQLDriver{
		debugBuf:        buf,
		handlerCtx:      clonedCtx,
		txnOrchestrator: txnOrchestrator,
		shapeInferrer:   queryshape.NewInferrer(clonedCtx),
		paramDecoder:    paramdecoder.NewDecoder(),
		stmtCache:       make(map[string]*stmtMeta),
		portalCache:     make(map[string]*portalMeta),
	}
	return rv, nil
}

func NewStackQLDriverFactory(handlerCtx handler.HandlerContext, isCaptureDebug bool) sqlbackend.SQLBackendFactory {
	return &basicStackQLDriverFactory{
		isCaptureDebug: isCaptureDebug,
		handlerCtx:     handlerCtx,
	}
}

func getTxnCounterManager(sqlEngine sqlengine.SQLEngine) (txncounter.Manager, error) {
	genID, err := sqlEngine.GetCurrentGenerationID()
	if err != nil {
		genID, err = sqlEngine.GetNextGenerationID()
		if err != nil {
			return nil, err
		}
	}
	sessionID, err := sqlEngine.GetNextSessionID(genID)
	if err != nil {
		return nil, err
	}
	return txncounter.NewTxnCounterManager(genID, sessionID), nil
}

// StackQLDriver lifetimes map to the concept of "session".
// It is responsible for handling queries
// and their bounding transactions.
type StackQLDriver interface {
	sqlbackend.ISQLBackend
	ProcessDryRun(string)
	ProcessQuery(string)
}

func (dr *basicStackQLDriver) ProcessDryRun(query string) {
	resultMap := map[string]map[string]interface{}{
		"1": {
			"query": query,
		},
	}
	logging.GetLogger().Debugln("dryrun query underway...")
	response := util.PrepareResultSet(internaldto.NewPrepareResultSetDTO(nil, resultMap, nil, nil, nil, nil,
		dr.handlerCtx.GetTypingConfig(),
	))
	responsehandler.HandleResponse(dr.handlerCtx, response) //nolint:errcheck // TODO: investigate
}

func (dr *basicStackQLDriver) ProcessQuery(query string) {
	clonedCtx := dr.handlerCtx.Clone()
	clonedCtx.SetRawQuery(query)
	responses, ok := dr.processQueryOrQueries(clonedCtx)
	if ok {
		for _, r := range responses {
			responsehandler.HandleResponse(clonedCtx, r) //nolint:errcheck // TODO: investigate
		}
	}
}

type stmtMeta struct {
	query     string
	paramOIDs []uint32
	columns   []sqldata.ISQLColumn
}

type portalMeta struct {
	stmtName string
}

type basicStackQLDriver struct {
	debugBuf        *bytes.Buffer
	handlerCtx      handler.HandlerContext
	txnOrchestrator tsm_physio.Orchestrator
	shapeInferrer   queryshape.Inferrer
	paramDecoder    paramdecoder.Decoder
	stmtCache       map[string]*stmtMeta
	portalCache     map[string]*portalMeta
}

func (dr *basicStackQLDriver) GetDebugStr() string {
	if dr.debugBuf != nil {
		return dr.debugBuf.String()
	}
	return ""
}

func (dr *basicStackQLDriver) CloneSQLBackend() sqlbackend.ISQLBackend {
	return &basicStackQLDriver{
		handlerCtx: dr.handlerCtx.Clone(),
	}
}

//nolint:revive // TODO: review
func (dr *basicStackQLDriver) HandleSimpleQuery(ctx context.Context, query string) (sqldata.ISQLResultStream, error) {
	dr.handlerCtx.SetRawQuery(query)
	res, ok := dr.processQueryOrQueries(dr.handlerCtx)
	if !ok {
		return nil, fmt.Errorf("no SQLresults available")
	}
	r := res[0]
	if r.GetError() != nil {
		return nil, fmt.Errorf("query returns error: %w", r.GetError())
	}
	return r.GetSQLResult(), nil
}

func (dr *basicStackQLDriver) SplitCompoundQuery(s string) ([]string, error) {
	res := []string{}
	var beg int
	var inDoubleQuotes bool

	for i := 0; i < len(s); i++ {
		if s[i] == ';' && !inDoubleQuotes {
			res = append(res, s[beg:i])
			beg = i + 1
		} else if s[i] == '"' {
			if !inDoubleQuotes {
				inDoubleQuotes = true
			} else if i > 0 && s[i-1] != '\\' {
				inDoubleQuotes = false
			}
		}
	}
	return append(res, s[beg:]), nil
}

func NewStackQLDriver(handlerCtx handler.HandlerContext) (StackQLDriver, error) {
	txnOrchestrator, orcErr := tsm_physio.NewOrchestrator(handlerCtx)
	if orcErr != nil {
		return nil, orcErr
	}
	tsmInstance, walError := tsm_physio.GetTSM(handlerCtx)
	if walError != nil {
		return nil, walError
	}
	handlerCtx.SetTSM(tsmInstance)
	return &basicStackQLDriver{
		handlerCtx:      handlerCtx,
		txnOrchestrator: txnOrchestrator,
		shapeInferrer:   queryshape.NewInferrer(handlerCtx),
		paramDecoder:    paramdecoder.NewDecoder(),
		stmtCache:       make(map[string]*stmtMeta),
		portalCache:     make(map[string]*portalMeta),
	}, nil
}

func (dr *basicStackQLDriver) HandleParse(
	ctx context.Context, stmtName string, query string, paramOIDs []uint32,
) ([]uint32, error) {
	// Infer result columns at parse time and cache for Describe/Execute.
	columns := dr.shapeInferrer.InferResultColumns(query)
	dr.stmtCache[stmtName] = &stmtMeta{
		query:     query,
		paramOIDs: paramOIDs,
		columns:   columns,
	}
	return paramOIDs, nil
}

func (dr *basicStackQLDriver) HandleBind(
	ctx context.Context, portalName string, stmtName string,
	paramFormats []int16, paramValues [][]byte, resultFormats []int16,
) error {
	dr.portalCache[portalName] = &portalMeta{
		stmtName: stmtName,
	}
	return nil
}

func (dr *basicStackQLDriver) HandleDescribeStatement(
	ctx context.Context, stmtName string, query string, paramOIDs []uint32,
) ([]uint32, []sqldata.ISQLColumn, error) {
	if cached, ok := dr.stmtCache[stmtName]; ok {
		return cached.paramOIDs, cached.columns, nil
	}
	// Fallback: infer on the fly (shouldn't happen if Parse was called first).
	columns := dr.shapeInferrer.InferResultColumns(query)
	return paramOIDs, columns, nil
}

func (dr *basicStackQLDriver) HandleDescribePortal(
	ctx context.Context, portalName string, stmtName string, query string, paramOIDs []uint32,
) ([]sqldata.ISQLColumn, error) {
	if portal, portalFound := dr.portalCache[portalName]; portalFound {
		if cached, stmtFound := dr.stmtCache[portal.stmtName]; stmtFound {
			return cached.columns, nil
		}
	}
	return dr.shapeInferrer.InferResultColumns(query), nil
}

func (dr *basicStackQLDriver) HandleExecute(
	ctx context.Context, portalName string, stmtName string, query string,
	paramFormats []int16, paramValues [][]byte, resultFormats []int16, maxRows int32,
) (sqldata.ISQLResultStream, error) {
	// Look up cached param OIDs for format-aware decoding.
	var paramOIDs []uint32
	if portal, portalFound := dr.portalCache[portalName]; portalFound {
		if cached, stmtFound := dr.stmtCache[portal.stmtName]; stmtFound {
			paramOIDs = cached.paramOIDs
		}
	}
	// Decode params (handles both text and binary formats).
	decodedStrings, err := dr.paramDecoder.DecodeParams(paramOIDs, paramFormats, paramValues)
	if err != nil {
		return nil, fmt.Errorf("parameter decoding error: %w", err)
	}
	resolved := queryshape.SubstituteDecodedParams(query, decodedStrings)
	return dr.HandleSimpleQuery(ctx, resolved)
}

func (dr *basicStackQLDriver) HandleCloseStatement(ctx context.Context, stmtName string) error {
	delete(dr.stmtCache, stmtName)
	return nil
}

func (dr *basicStackQLDriver) HandleClosePortal(ctx context.Context, portalName string) error {
	delete(dr.portalCache, portalName)
	return nil
}

func (dr *basicStackQLDriver) processQueryOrQueries(
	handlerCtx handler.HandlerContext,
) ([]internaldto.ExecutorOutput, bool) {
	return dr.txnOrchestrator.ProcessQueryOrQueries(handlerCtx)
}
