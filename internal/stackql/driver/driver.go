package driver

import (
	"context"
	"fmt"
	"strings"

	"github.com/stackql/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/querysubmit"
	"github.com/stackql/stackql/internal/stackql/responsehandler"
	"github.com/stackql/stackql/internal/stackql/util"

	sqlbackend "github.com/stackql/psql-wire/pkg/sqlbackend"
)

var (
	_ StackQLDriver = &basicStackQLDriver{}
)

type StackQLDriver interface {
	sqlbackend.ISQLBackend
	ProcessDryRun(handlerCtx handler.HandlerContext)
	ProcessQuery(handlerCtx handler.HandlerContext)
}

func (dr *basicStackQLDriver) ProcessDryRun(handlerCtx handler.HandlerContext) {
	resultMap := map[string]map[string]interface{}{
		"1": {
			"query": handlerCtx.GetRawQuery(),
		},
	}
	logging.GetLogger().Debugln("dryrun query underway...")
	response := util.PrepareResultSet(internaldto.NewPrepareResultSetDTO(nil, resultMap, nil, nil, nil, nil))
	responsehandler.HandleResponse(handlerCtx, response)
}

func throwErr(err error, handlerCtx handler.HandlerContext) {
	response := internaldto.NewExecutorOutput(nil, nil, nil, nil, err)
	responsehandler.HandleResponse(handlerCtx, response)
}

func (dr *basicStackQLDriver) ProcessQuery(handlerCtx handler.HandlerContext) {
	responses, ok := processQueryOrQueries(handlerCtx)
	if ok {
		for _, r := range responses {
			responsehandler.HandleResponse(handlerCtx, r)
		}
	}
}

type basicStackQLDriver struct {
	handlerCtx handler.HandlerContext
}

func (sbs *basicStackQLDriver) HandleSimpleQuery(ctx context.Context, query string) (sqldata.ISQLResultStream, error) {
	sbs.handlerCtx.SetRawQuery(query)
	// if strings.Count(query, ";") > 1 {
	// 	return nil, fmt.Errorf("only support single queries in server mode at this time")
	// }
	res, ok := processQueryOrQueries(sbs.handlerCtx)
	if !ok {
		return nil, fmt.Errorf("no SQLresults available")
	}
	r := res[0]
	if r.Err != nil {
		return nil, r.Err
	}
	return r.GetSQLResult(), nil
}

func (sb *basicStackQLDriver) SplitCompoundQuery(s string) ([]string, error) {
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
	return &basicStackQLDriver{
		handlerCtx: handlerCtx,
	}, nil
}

func processQueryOrQueries(handlerCtx handler.HandlerContext) ([]internaldto.ExecutorOutput, bool) {
	var retVal []internaldto.ExecutorOutput
	cmdString := handlerCtx.GetRawQuery()
	querySubmitter := querysubmit.NewQuerySubmitter()
	for _, s := range strings.Split(cmdString, ";") {
		if s == "" {
			continue
		}
		handlerCtx.SetQuery(s)
		retVal = append(retVal, querySubmitter.SubmitQuery(handlerCtx))
	}
	return retVal, true
}
