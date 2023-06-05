package driver

import (
	"context"
	"fmt"
	"strings"

	"github.com/stackql/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql/internal/stackql/acid/transact"
	"github.com/stackql/stackql/internal/stackql/acid/txn_context"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/responsehandler"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/util"
	"github.com/stackql/stackql/pkg/txncounter"

	sqlbackend "github.com/stackql/psql-wire/pkg/sqlbackend"
)

var (
	_               StackQLDriver                = &basicStackQLDriver{}
	_               sqlbackend.SQLBackendFactory = &basicStackQLDriverFactory{}
	_               StackQLDriverFactory         = &basicStackQLDriverFactory{}
	noParentMessage string                       = "no parent transaction manager available" //nolint:gochecknoglobals,revive,lll // permissable
)

type StackQLDriverFactory interface {
	NewSQLDriver() (StackQLDriver, error)
}

type basicStackQLDriverFactory struct {
	handlerCtx handler.HandlerContext
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
	txnProvider, txnProviderErr := transact.GetProviderInstance(sdf.handlerCtx.GetTxnCoordinatorCtx())
	if txnProviderErr != nil {
		return nil, txnProviderErr
	}
	txnCoordinator, txnCoordinatorErr := txnProvider.NewTxnCoordinator()
	if txnCoordinatorErr != nil {
		return nil, txnCoordinatorErr
	}
	clonedCtx := sdf.handlerCtx.Clone()
	clonedCtx.SetTxnCounterMgr(txCtr)
	rv := &basicStackQLDriver{
		handlerCtx:     clonedCtx,
		txnCoordinator: txnCoordinator,
	}
	return rv, nil
}

func NewStackQLDriverFactory(handlerCtx handler.HandlerContext) sqlbackend.SQLBackendFactory {
	return &basicStackQLDriverFactory{
		handlerCtx: handlerCtx,
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

type basicStackQLDriver struct {
	handlerCtx     handler.HandlerContext
	txnCoordinator transact.Coordinator
}

func (dr *basicStackQLDriver) CloneSQLBackend() sqlbackend.ISQLBackend {
	return &basicStackQLDriver{
		handlerCtx: dr.handlerCtx.Clone(),
	}
}

//nolint:revive // TODO: review
func (dr *basicStackQLDriver) HandleSimpleQuery(ctx context.Context, query string) (sqldata.ISQLResultStream, error) {
	dr.handlerCtx.SetRawQuery(query)
	// if strings.Count(query, ";") > 1 {
	// 	return nil, fmt.Errorf("only support single queries in server mode at this time")
	// }
	res, ok := dr.processQueryOrQueries(dr.handlerCtx)
	if !ok {
		return nil, fmt.Errorf("no SQLresults available")
	}
	r := res[0]
	if r.GetError() != nil {
		return nil, r.GetError()
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
	txnProvider, txnProviderErr := transact.GetProviderInstance(
		handlerCtx.GetTxnCoordinatorCtx())
	if txnProviderErr != nil {
		return nil, txnProviderErr
	}
	txnCoordinator, txnCoordinatorErr := txnProvider.NewTxnCoordinator()
	if txnCoordinatorErr != nil {
		return nil, txnCoordinatorErr
	}
	return &basicStackQLDriver{
		handlerCtx:     handlerCtx,
		txnCoordinator: txnCoordinator,
	}, nil
}

//nolint:gocognit // TODO: review
func (dr *basicStackQLDriver) processQueryOrQueries(
	handlerCtx handler.HandlerContext,
) ([]internaldto.ExecutorOutput, bool) {
	var retVal []internaldto.ExecutorOutput
	cmdString := handlerCtx.GetRawQuery()
	for _, s := range strings.Split(cmdString, ";") {
		if s == "" {
			continue
		}
		clonedCtx := handlerCtx.Clone()
		clonedCtx.SetQuery(s)
		transactStatement := transact.NewStatement(s, clonedCtx, txn_context.NewTransactionContext(dr.txnCoordinator.Depth()))
		prepareErr := transactStatement.Prepare()
		if prepareErr != nil {
			retVal = append(retVal, internaldto.NewErroneousExecutorOutput(prepareErr))
			continue
		}
		isReadOnly := transactStatement.IsReadOnly()
		// TODO: implement eager execution for non-mutating statements
		//       and lazy execution for mutating statements.
		// TODO: implement transaction stack.
		if transactStatement.IsBegin() { //nolint:gocritic,nestif // TODO: review
			txnCoordinator, beginErr := dr.txnCoordinator.Begin()
			if beginErr != nil {
				retVal = append(retVal, internaldto.NewErroneousExecutorOutput(beginErr))
				continue
			}
			dr.txnCoordinator = txnCoordinator
			retVal = append(retVal, internaldto.NewNopEmptyExecutorOutput([]string{"OK"}))
			continue
		} else if transactStatement.IsCommit() {
			commitCoDomain := dr.txnCoordinator.Commit()
			commitErr, commitErrExists := commitCoDomain.GetError()
			if commitErrExists {
				retVal = append(retVal, internaldto.NewErroneousExecutorOutput(commitErr))
				undoLog, undoLogExists := commitCoDomain.GetUndoLog()
				if undoLogExists && undoLog != nil {
					humanReadable := undoLog.GetHumanReadable()
					if len(humanReadable) > 0 {
						displayUndoLogs := make([]string, len(humanReadable))
						for i, h := range humanReadable {
							displayUndoLogs[i] = fmt.Sprintf("UNDO required: %s", h)
						}
						retVal = append(retVal, internaldto.NewNopEmptyExecutorOutput(displayUndoLogs))
					}
				}
				continue
			}
			retVal = append(retVal, commitCoDomain.GetExecutorOutput()...)
			parent, hasParent := dr.txnCoordinator.GetParent()
			if hasParent {
				dr.txnCoordinator = parent
				retVal = append(retVal, internaldto.NewNopEmptyExecutorOutput([]string{"OK"}))
				continue
			}
			noParentErr := fmt.Errorf(noParentMessage)
			retVal = append(retVal, internaldto.NewErroneousExecutorOutput(noParentErr))
			continue
		} else if transactStatement.IsRollback() {
			rollbackErr := dr.txnCoordinator.Rollback()
			if rollbackErr != nil {
				retVal = append(retVal, internaldto.NewErroneousExecutorOutput(rollbackErr))
			}
			parent, hasParent := dr.txnCoordinator.GetParent()
			if hasParent {
				dr.txnCoordinator = parent
				retVal = append(retVal, internaldto.NewNopEmptyExecutorOutput([]string{"Rollback OK"}))
				continue
			}
			retVal = append(
				retVal,
				internaldto.NewErroneousExecutorOutput(
					fmt.Errorf(noParentMessage)),
			)
			continue
		}
		if isReadOnly || dr.txnCoordinator.IsRoot() {
			stmtOutput := transactStatement.Execute()
			retVal = append(retVal, stmtOutput)
		} else {
			dr.txnCoordinator.Enqueue(transactStatement) //nolint:errcheck // TODO: investigate
			retVal = append(retVal, internaldto.NewNopEmptyExecutorOutput([]string{"mutating statement queued"}))
		}
	}
	return retVal, len(retVal) > 0
}
