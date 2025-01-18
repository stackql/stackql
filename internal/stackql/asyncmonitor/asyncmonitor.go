package asyncmonitor

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/any-sdk/pkg/logging"
	"github.com/stackql/stackql/internal/stackql/acid/binlog"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/httpmiddleware"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/provider"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

var (
	MonitorPollIntervalSeconds int = 10 //nolint:revive,gochecknoglobals // TODO: global vars refactor
)

type IAsyncMonitor interface {
	GetMonitorPrimitive(
		prov provider.IProvider,
		op anysdk.OperationStore,
		precursor primitive.IPrimitive,
		initialCtx primitive.IPrimitiveCtx,
		comments sqlparser.CommentDirectives,
	) (primitive.IPrimitive, error)
}

type AsyncHTTPMonitorPrimitive struct {
	handlerCtx          handler.HandlerContext
	prov                provider.IProvider
	op                  anysdk.OperationStore
	initialCtx          primitive.IPrimitiveCtx
	precursor           primitive.IPrimitive
	executor            func(pc primitive.IPrimitiveCtx, initalBody interface{}) internaldto.ExecutorOutput
	elapsedSeconds      int
	pollIntervalSeconds int
	noStatus            bool
	id                  int64
	comments            sqlparser.CommentDirectives
}

func (pr *AsyncHTTPMonitorPrimitive) SetTxnID(_ int) {
}

func (pr *AsyncHTTPMonitorPrimitive) IsReadOnly() bool {
	return false
}

func (pr *AsyncHTTPMonitorPrimitive) GetRedoLog() (binlog.LogEntry, bool) {
	return nil, false
}

func (pr *AsyncHTTPMonitorPrimitive) GetUndoLog() (binlog.LogEntry, bool) {
	return nil, false
}

func (pr *AsyncHTTPMonitorPrimitive) WithDebugName(_ string) primitive.IPrimitive {
	return pr
}

func (pr *AsyncHTTPMonitorPrimitive) SetUndoLog(_ binlog.LogEntry) {
}

func (pr *AsyncHTTPMonitorPrimitive) SetRedoLog(_ binlog.LogEntry) {
}

func (pr *AsyncHTTPMonitorPrimitive) IncidentData(fromID int64, input internaldto.ExecutorOutput) error {
	return pr.precursor.IncidentData(fromID, input)
}

func (pr *AsyncHTTPMonitorPrimitive) SetInputAlias(alias string, id int64) error {
	return pr.precursor.SetInputAlias(alias, id)
}

func (pr *AsyncHTTPMonitorPrimitive) Optimise() error {
	return nil
}

func (pr *AsyncHTTPMonitorPrimitive) Execute(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput {
	if pr.executor != nil {
		if pc == nil {
			pc = pr.initialCtx
		}
		subPr := pr.precursor.Execute(pc)
		if subPr.GetError() != nil || pr.executor == nil {
			return subPr
		}
		prStr := pr.prov.GetProviderString()
		// seems pointless
		_, err := pr.initialCtx.GetAuthContext(prStr)
		if err != nil {
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, err)
		}
		//
		asyP := internaldto.NewBasicPrimitiveContext(
			pr.initialCtx.GetAuthContext,
			pc.GetWriter(),
			pc.GetErrWriter(),
		)
		return pr.executor(asyP, subPr.GetOutputBody())
	}
	return internaldto.NewExecutorOutput(nil, nil, nil, nil, nil)
}

func (pr *AsyncHTTPMonitorPrimitive) ID() int64 {
	return pr.id
}

func (pr *AsyncHTTPMonitorPrimitive) GetInputFromAlias(string) (internaldto.ExecutorOutput, bool) {
	var rv internaldto.ExecutorOutput
	return rv, false
}

func (pr *AsyncHTTPMonitorPrimitive) SetExecutor(_ func(pc primitive.IPrimitiveCtx) internaldto.ExecutorOutput) error {
	return fmt.Errorf("AsyncHTTPMonitorPrimitive does not support SetExecutor()")
}

func NewAsyncMonitor(
	handlerCtx handler.HandlerContext,
	prov provider.IProvider,
	op anysdk.OperationStore,
) (IAsyncMonitor, error) {
	//nolint:gocritic //TODO: refactor
	switch prov.GetProviderString() {
	case "google":
		return newGoogleAsyncMonitor(handlerCtx, prov, op, prov.GetVersion())
	}
	return nil, fmt.Errorf(
		"async operation monitor for provider = '%s', api version = '%s' currently not supported",
		prov.GetProviderString(), prov.GetVersion())
}

func newGoogleAsyncMonitor(
	handlerCtx handler.HandlerContext,
	prov provider.IProvider,
	op anysdk.OperationStore,
	version string, //nolint:unparam // TODO: refactor
) (IAsyncMonitor, error) {
	//nolint:gocritic //TODO: refactor
	switch version {
	default:
		return &DefaultGoogleAsyncMonitor{
			handlerCtx: handlerCtx,
			prov:       prov,
			op:         op,
		}, nil
	}
}

type DefaultGoogleAsyncMonitor struct {
	handlerCtx handler.HandlerContext
	prov       provider.IProvider
	op         anysdk.OperationStore
}

func (gm *DefaultGoogleAsyncMonitor) GetMonitorPrimitive(
	prov provider.IProvider,
	op anysdk.OperationStore,
	precursor primitive.IPrimitive,
	initialCtx primitive.IPrimitiveCtx,
	comments sqlparser.CommentDirectives,
) (primitive.IPrimitive, error) {
	//nolint:gocritic,staticcheck //TODO: refactor
	switch strings.ToLower(prov.GetVersion()) {
	default:
		return gm.getV1Monitor(prov, op, precursor, initialCtx, comments)
	}
}

func getOperationDescriptor(body map[string]interface{}) string {
	operationDescriptor := "operation"
	if body == nil {
		return operationDescriptor
	}
	//nolint:nestif,govet // TODO: refactor
	if descriptor, ok := body["kind"]; ok {
		if descriptorStr, ok := descriptor.(string); ok {
			operationDescriptor = descriptorStr
			if typeElem, ok := body["operationType"]; ok {
				if typeStr, ok := typeElem.(string); ok {
					operationDescriptor = fmt.Sprintf("%s: %s", descriptorStr, typeStr)
				}
			}
		}
	}
	return operationDescriptor
}

//nolint:gocognit,funlen // review later
func (gm *DefaultGoogleAsyncMonitor) getV1Monitor(
	prov provider.IProvider,
	op anysdk.OperationStore,
	precursor primitive.IPrimitive,
	initialCtx primitive.IPrimitiveCtx,
	comments sqlparser.CommentDirectives,
) (primitive.IPrimitive, error) {
	asyncPrim := AsyncHTTPMonitorPrimitive{
		handlerCtx:          gm.handlerCtx,
		prov:                prov,
		op:                  op,
		initialCtx:          initialCtx,
		precursor:           precursor,
		elapsedSeconds:      0,
		pollIntervalSeconds: MonitorPollIntervalSeconds,
		comments:            comments,
	}
	if comments != nil {
		asyncPrim.noStatus = comments.IsSet("NOSTATUS")
	}
	m := gm.op
	if m.IsAwaitable() { //nolint:nestif // encapulation probably sufficient
		asyncPrim.executor = func(pc primitive.IPrimitiveCtx, bd interface{}) internaldto.ExecutorOutput {
			body, ok := bd.(map[string]interface{})
			if !ok {
				return internaldto.NewExecutorOutput(
					nil,
					nil,
					nil,
					nil,
					fmt.Errorf("cannot execute monitor: response body of type '%T' unreadable", bd),
				)
			}
			if pc == nil {
				return internaldto.NewExecutorOutput(nil, nil, nil, nil, fmt.Errorf("cannot execute monitor: nil plan primitive"))
			}
			if body == nil {
				return internaldto.NewExecutorOutput(nil, nil, nil, nil, fmt.Errorf("cannot execute monitor: no body present"))
			}
			logging.GetLogger().Infoln(fmt.Sprintf("body = %v", body))

			operationDescriptor := getOperationDescriptor(body)
			endTime, endTimeOk := body["endTime"]
			if endTimeOk && endTime != "" {
				return prepareReultSet(&asyncPrim, pc, body, operationDescriptor)
			}
			url, ok := body["selfLink"]
			if !ok {
				return internaldto.NewExecutorOutput(
					nil,
					nil,
					nil,
					nil,
					fmt.Errorf("cannot execute monitor: no 'selfLink' property present"),
				)
			}
			prStr := gm.prov.GetProviderString()
			authCtx, err := pc.GetAuthContext(prStr)
			if err != nil {
				return internaldto.NewExecutorOutput(nil, nil, nil, nil, err)
			}
			if authCtx == nil {
				return internaldto.NewExecutorOutput(nil, nil, nil, nil, fmt.Errorf("cannot execute monitor: no auth context"))
			}
			time.Sleep(time.Duration(asyncPrim.pollIntervalSeconds) * time.Second)
			asyncPrim.elapsedSeconds += asyncPrim.pollIntervalSeconds
			if !asyncPrim.noStatus {
				//nolint:errcheck //TODO: handle error
				pc.GetWriter().Write(
					[]byte(
						fmt.Sprintf(
							"%s in progress, %d seconds elapsed",
							operationDescriptor,
							asyncPrim.elapsedSeconds,
						) + fmt.Sprintln(""),
					),
				)
			}
			req, err := getMonitorRequest(url.(string))
			if req != nil && req.Body != nil {
				defer req.Body.Close()
			}
			if err != nil {
				return internaldto.NewExecutorOutput(nil, nil, nil, nil, err)
			}
			response, apiErr := httpmiddleware.HTTPApiCallFromRequest(gm.handlerCtx.Clone(), gm.prov, m, req)
			if response != nil && response.Body != nil {
				defer response.Body.Close()
			}
			if apiErr != nil {
				return internaldto.NewExecutorOutput(nil, nil, nil, nil, apiErr)
			}
			target, err := m.DeprecatedProcessResponse(response)
			gm.handlerCtx.LogHTTPResponseMap(target)
			if err != nil {
				return internaldto.NewExecutorOutput(nil, nil, nil, nil, err)
			}
			return asyncPrim.executor(internaldto.NewBasicPrimitiveContext(
				pc.GetAuthContext,
				pc.GetWriter(),
				pc.GetErrWriter(),
			),
				target)
		}
		return &asyncPrim, nil
	}
	return nil, fmt.Errorf("method %s is not awaitable", m.GetName())
}

func prepareReultSet(
	prim *AsyncHTTPMonitorPrimitive,
	pc primitive.IPrimitiveCtx,
	target map[string]interface{},
	operationDescriptor string,
) internaldto.ExecutorOutput {
	payload := internaldto.PrepareResultSetDTO{
		OutputBody:  target,
		Msg:         nil,
		RowMap:      nil,
		ColumnOrder: nil,
		RowSort:     nil,
		Err:         nil,
	}
	if !prim.noStatus {
		//nolint:errcheck //TODO: handle error
		pc.GetWriter().Write([]byte(fmt.Sprintf("%s complete", operationDescriptor) + fmt.Sprintln("")))
	}
	return util.PrepareResultSet(payload)
}

func getMonitorRequest(urlStr string) (*http.Request, error) {
	req, err := http.NewRequest(
		http.MethodGet,
		urlStr,
		nil,
	)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(context.Background())
	return req, nil
}
