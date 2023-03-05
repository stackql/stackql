package internaldto

import (
	"github.com/stackql/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql/internal/stackql/streaming"
)

type BackendMessages struct {
	WorkingMessages []string
}

type ExecutorOutput struct {
	GetSQLResult  func() sqldata.ISQLResultStream
	GetRawResult  func() IRawResultStream
	GetOutputBody func() map[string]interface{}
	stream        streaming.MapStream
	Msg           *BackendMessages
	Err           error
}

func (ex ExecutorOutput) ResultToMap() (IRawResultStream, error) {
	return ex.GetRawResult(), nil
}

func (ex ExecutorOutput) SetStream(s streaming.MapStream) {
	ex.stream = s
}

func (ex ExecutorOutput) GetStream() streaming.MapStream {
	return ex.stream
}

func NewExecutorOutput(result sqldata.ISQLResultStream, body map[string]interface{}, rawResult map[int]map[int]interface{}, msg *BackendMessages, err error) ExecutorOutput {
	return newExecutorOutput(result, body, rawResult, msg, err)
}

func newExecutorOutput(result sqldata.ISQLResultStream, body map[string]interface{}, rawResult map[int]map[int]interface{}, msg *BackendMessages, err error) ExecutorOutput {
	return ExecutorOutput{
		GetSQLResult: func() sqldata.ISQLResultStream { return result },
		GetRawResult: func() IRawResultStream {
			if rawResult == nil {
				return createSimpleRawResultStream(make(map[int]map[int]interface{}))
			}
			return createSimpleRawResultStream(rawResult)
		},
		GetOutputBody: func() map[string]interface{} { return body },
		Msg:           msg,
		Err:           err,
	}
}

func NewErroneousExecutorOutput(err error) ExecutorOutput {
	return newExecutorOutput(nil, nil, nil, nil, err)
}
