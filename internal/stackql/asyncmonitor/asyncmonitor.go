package asyncmonitor

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/httpmiddleware"
	"github.com/stackql/stackql/internal/stackql/primitive"
	"github.com/stackql/stackql/internal/stackql/provider"
	"github.com/stackql/stackql/internal/stackql/taxonomy"
	"github.com/stackql/stackql/internal/stackql/util"

	log "github.com/sirupsen/logrus"
	"vitess.io/vitess/go/vt/sqlparser"
)

var MonitorPollIntervalSeconds int = 10

type IAsyncMonitor interface {
	GetMonitorPrimitive(heirarchy *taxonomy.HeirarchyObjects, precursor primitive.IPrimitive, initialCtx primitive.IPrimitiveCtx, comments sqlparser.CommentDirectives) (primitive.IPrimitive, error)
}

type AsyncHttpMonitorPrimitive struct {
	handlerCtx          *handler.HandlerContext
	heirarchy           *taxonomy.HeirarchyObjects
	initialCtx          primitive.IPrimitiveCtx
	precursor           primitive.IPrimitive
	transferPayload     map[string]interface{}
	executor            func(pc primitive.IPrimitiveCtx, initalBody interface{}) dto.ExecutorOutput
	elapsedSeconds      int
	pollIntervalSeconds int
	noStatus            bool
	id                  int64
	comments            sqlparser.CommentDirectives
}

func (pr *AsyncHttpMonitorPrimitive) SetTxnId(id int) {
}

func (pr *AsyncHttpMonitorPrimitive) IncidentData(fromId int64, input dto.ExecutorOutput) error {
	return pr.precursor.IncidentData(fromId, input)
}

func (pr *AsyncHttpMonitorPrimitive) SetInputAlias(alias string, id int64) error {
	return pr.precursor.SetInputAlias(alias, id)
}

func (pr *AsyncHttpMonitorPrimitive) Optimise() error {
	return nil
}

func (asm *AsyncHttpMonitorPrimitive) Execute(pc primitive.IPrimitiveCtx) dto.ExecutorOutput {
	if asm.executor != nil {
		if pc == nil {
			pc = asm.initialCtx
		}
		pr := asm.precursor.Execute(pc)
		if pr.Err != nil || asm.executor == nil {
			return pr
		}
		prStr := asm.heirarchy.Provider.GetProviderString()
		// seems pointless
		_, err := asm.initialCtx.GetAuthContext(prStr)
		if err != nil {
			return dto.NewExecutorOutput(nil, nil, nil, nil, err)
		}
		//
		asyP := dto.NewBasicPrimitiveContext(
			asm.initialCtx.GetAuthContext,
			pc.GetWriter(),
			pc.GetErrWriter(),
		)
		return asm.executor(asyP, pr.GetOutputBody())
	}
	return dto.NewExecutorOutput(nil, nil, nil, nil, nil)
}

func (pr *AsyncHttpMonitorPrimitive) ID() int64 {
	return pr.id
}

func (pr *AsyncHttpMonitorPrimitive) GetInputFromAlias(string) (dto.ExecutorOutput, bool) {
	var rv dto.ExecutorOutput
	return rv, false
}

func (pr *AsyncHttpMonitorPrimitive) SetExecutor(ex func(pc primitive.IPrimitiveCtx) dto.ExecutorOutput) error {
	return fmt.Errorf("AsyncHttpMonitorPrimitive does not support SetExecutor()")
}

func NewAsyncMonitor(handlerCtx *handler.HandlerContext, prov provider.IProvider) (IAsyncMonitor, error) {
	switch prov.GetProviderString() {
	case "google":
		return newGoogleAsyncMonitor(handlerCtx, prov, prov.GetVersion())
	}
	return nil, fmt.Errorf("async operation monitor for provider = '%s', api version = '%s' currently not supported", prov.GetProviderString(), prov.GetVersion())
}

func newGoogleAsyncMonitor(handlerCtx *handler.HandlerContext, prov provider.IProvider, version string) (IAsyncMonitor, error) {
	switch version {
	default:
		return &DefaultGoogleAsyncMonitor{
			handlerCtx: handlerCtx,
			provider:   prov,
		}, nil
	}
}

type DefaultGoogleAsyncMonitor struct {
	handlerCtx *handler.HandlerContext
	provider   provider.IProvider
	precursor  primitive.IPrimitive
}

func (gm *DefaultGoogleAsyncMonitor) GetMonitorPrimitive(heirarchy *taxonomy.HeirarchyObjects, precursor primitive.IPrimitive, initialCtx primitive.IPrimitiveCtx, comments sqlparser.CommentDirectives) (primitive.IPrimitive, error) {
	switch strings.ToLower(heirarchy.Provider.GetVersion()) {
	default:
		return gm.getV1Monitor(heirarchy, precursor, initialCtx, comments)
	}
}

func getOperationDescriptor(body map[string]interface{}) string {
	operationDescriptor := "operation"
	if body == nil {
		return operationDescriptor
	}
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

func (gm *DefaultGoogleAsyncMonitor) getV1Monitor(heirarchy *taxonomy.HeirarchyObjects, precursor primitive.IPrimitive, initialCtx primitive.IPrimitiveCtx, comments sqlparser.CommentDirectives) (primitive.IPrimitive, error) {
	asyncPrim := AsyncHttpMonitorPrimitive{
		handlerCtx:          gm.handlerCtx,
		heirarchy:           heirarchy,
		initialCtx:          initialCtx,
		precursor:           precursor,
		elapsedSeconds:      0,
		pollIntervalSeconds: MonitorPollIntervalSeconds,
		comments:            comments,
	}
	if comments != nil {
		asyncPrim.noStatus = comments.IsSet("NOSTATUS")
	}
	m := heirarchy.Method
	if m.IsAwaitable() {
		asyncPrim.executor = func(pc primitive.IPrimitiveCtx, bd interface{}) dto.ExecutorOutput {
			body, ok := bd.(map[string]interface{})
			if !ok {
				return dto.NewExecutorOutput(nil, nil, nil, nil, fmt.Errorf("cannot execute monitor: response body of type '%T' unreadable", bd))
			}
			if pc == nil {
				return dto.NewExecutorOutput(nil, nil, nil, nil, fmt.Errorf("cannot execute monitor: nil plan primitive"))
			}
			if body == nil {
				return dto.NewExecutorOutput(nil, nil, nil, nil, fmt.Errorf("cannot execute monitor: no body present"))
			}
			log.Infoln(fmt.Sprintf("body = %v", body))

			operationDescriptor := getOperationDescriptor(body)
			endTime, endTimeOk := body["endTime"]
			if endTimeOk && endTime != "" {
				return prepareReultSet(&asyncPrim, pc, body, operationDescriptor)
			}
			url, ok := body["selfLink"]
			if !ok {
				return dto.NewExecutorOutput(nil, nil, nil, nil, fmt.Errorf("cannot execute monitor: no 'selfLink' property present"))
			}
			prStr := heirarchy.Provider.GetProviderString()
			authCtx, err := pc.GetAuthContext(prStr)
			if err != nil {
				return dto.NewExecutorOutput(nil, nil, nil, nil, err)
			}
			if authCtx == nil {
				return dto.NewExecutorOutput(nil, nil, nil, nil, fmt.Errorf("cannot execute monitor: no auth context"))
			}
			time.Sleep(time.Duration(asyncPrim.pollIntervalSeconds) * time.Second)
			asyncPrim.elapsedSeconds += asyncPrim.pollIntervalSeconds
			if !asyncPrim.noStatus {
				pc.GetWriter().Write([]byte(fmt.Sprintf("%s in progress, %d seconds elapsed", operationDescriptor, asyncPrim.elapsedSeconds) + fmt.Sprintln("")))
			}
			req, err := getMonitorRequest(url.(string))
			if err != nil {
				return dto.NewExecutorOutput(nil, nil, nil, nil, err)
			}
			response, apiErr := httpmiddleware.HttpApiCallFromRequest(*gm.handlerCtx, gm.provider, m, req)
			if apiErr != nil {
				return dto.NewExecutorOutput(nil, nil, nil, nil, apiErr)
			}
			target, err := heirarchy.Method.DeprecatedProcessResponse(response)
			gm.handlerCtx.LogHTTPResponseMap(target)
			if err != nil {
				return dto.NewExecutorOutput(nil, nil, nil, nil, err)
			}
			return asyncPrim.executor(dto.NewBasicPrimitiveContext(
				pc.GetAuthContext,
				pc.GetWriter(),
				pc.GetErrWriter(),
			),
				target)
		}
		return &asyncPrim, nil
	}
	return nil, fmt.Errorf("method %s is not awaitable", heirarchy.Method.GetName())
}

func prepareReultSet(prim *AsyncHttpMonitorPrimitive, pc primitive.IPrimitiveCtx, target map[string]interface{}, operationDescriptor string) dto.ExecutorOutput {
	payload := dto.PrepareResultSetDTO{
		OutputBody:  target,
		Msg:         nil,
		RowMap:      nil,
		ColumnOrder: nil,
		RowSort:     nil,
		Err:         nil,
	}
	if !prim.noStatus {
		pc.GetWriter().Write([]byte(fmt.Sprintf("%s complete", operationDescriptor) + fmt.Sprintln("")))
	}
	return util.PrepareResultSet(payload)
}

func getMonitorRequest(urlStr string) (*http.Request, error) {
	return http.NewRequest(
		"GET",
		urlStr,
		nil,
	)
}
