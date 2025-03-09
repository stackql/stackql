package internaldto

import (
	"github.com/stackql/any-sdk/pkg/streaming"
	"github.com/stackql/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql/internal/stackql/acid/binlog"
)

var (
	_ ExecutorOutput  = &standardExecutorOutput{}
	_ BackendMessages = &standardBackendMessages{}
)

type BackendMessages interface {
	AppendMessages([]string)
	GetMessages() []string
}

func newBackendMessages(msgs []string) BackendMessages {
	return &standardBackendMessages{
		WorkingMessages: msgs,
	}
}

func NewBackendMessages(msgs []string) BackendMessages {
	return newBackendMessages(msgs)
}

type standardBackendMessages struct {
	WorkingMessages []string
}

func (m *standardBackendMessages) AppendMessages(msg []string) {
	m.WorkingMessages = append(m.WorkingMessages, msg...)
}

func (m *standardBackendMessages) GetMessages() []string {
	return m.WorkingMessages
}

type ExecutorOutput interface {
	GetSQLResult() sqldata.ISQLResultStream
	GetRawResult() IRawResultStream
	GetOutputBody() map[string]interface{}
	GetStream() streaming.MapStream
	SetStream(s streaming.MapStream)
	ResultToMap() (IRawResultStream, error)
	GetError() error
	SetSQLResultFn(f func() sqldata.ISQLResultStream)
	SetRawResultFn(f func() IRawResultStream)
	SetOutputBodyFn(f func() map[string]interface{})
	GetMessages() []string
	AppendMessages(m []string)
	GetUndoLog() (binlog.LogEntry, bool)
	GetRedoLog() (binlog.LogEntry, bool)
	SetUndoLog(binlog.LogEntry)
	SetRedoLog(binlog.LogEntry)
	WithUndoLog(binlog.LogEntry) ExecutorOutput
	WithRedoLog(binlog.LogEntry) ExecutorOutput
}

type standardExecutorOutput struct {
	getSQLResult  func() sqldata.ISQLResultStream
	getRawResult  func() IRawResultStream
	getOutputBody func() map[string]interface{}
	stream        streaming.MapStream
	Msg           BackendMessages
	redoLog       binlog.LogEntry
	undoLog       binlog.LogEntry
	Err           error
}

func (ex *standardExecutorOutput) SetRedoLog(log binlog.LogEntry) {
	ex.redoLog = log
}

func (ex *standardExecutorOutput) SetUndoLog(log binlog.LogEntry) {
	ex.undoLog = log
}

func (ex *standardExecutorOutput) WithUndoLog(log binlog.LogEntry) ExecutorOutput {
	ex.undoLog = log
	return ex
}

func (ex *standardExecutorOutput) WithRedoLog(log binlog.LogEntry) ExecutorOutput {
	ex.redoLog = log
	return ex
}

func (ex *standardExecutorOutput) GetRedoLog() (binlog.LogEntry, bool) {
	return ex.redoLog, ex.redoLog != nil
}

func (ex *standardExecutorOutput) GetUndoLog() (binlog.LogEntry, bool) {
	return ex.undoLog, ex.undoLog != nil
}

func (ex *standardExecutorOutput) GetSQLResult() sqldata.ISQLResultStream {
	return ex.getSQLResult()
}

func (ex *standardExecutorOutput) GetMessages() []string {
	return ex.Msg.GetMessages()
}

func (ex *standardExecutorOutput) AppendMessages(m []string) {
	ex.Msg.AppendMessages(m)
}

func (ex *standardExecutorOutput) GetRawResult() IRawResultStream {
	return ex.getRawResult()
}

func (ex *standardExecutorOutput) GetOutputBody() map[string]interface{} {
	return ex.getOutputBody()
}

func (ex *standardExecutorOutput) SetSQLResultFn(f func() sqldata.ISQLResultStream) {
	ex.getSQLResult = f
}

func (ex *standardExecutorOutput) SetRawResultFn(f func() IRawResultStream) {
	ex.getRawResult = f
}

func (ex *standardExecutorOutput) SetOutputBodyFn(f func() map[string]interface{}) {
	ex.getOutputBody = f
}

func (ex *standardExecutorOutput) ResultToMap() (IRawResultStream, error) {
	return ex.getRawResult(), nil
}

func (ex *standardExecutorOutput) SetStream(s streaming.MapStream) {
	ex.stream = s
}

func (ex *standardExecutorOutput) GetStream() streaming.MapStream {
	return ex.stream
}

func (ex *standardExecutorOutput) GetError() error {
	return ex.Err
}

func NewExecutorOutput(
	result sqldata.ISQLResultStream,
	body map[string]interface{},
	rawResult map[int]map[int]interface{},
	msg BackendMessages,
	err error,
) ExecutorOutput {
	return newExecutorOutput(result, body, rawResult, msg, err)
}

func newExecutorOutput(
	result sqldata.ISQLResultStream,
	body map[string]interface{},
	rawResult map[int]map[int]interface{},
	msg BackendMessages,
	err error,
) ExecutorOutput {
	if msg == nil {
		msg = newBackendMessages([]string{})
	}
	return &standardExecutorOutput{
		getSQLResult: func() sqldata.ISQLResultStream { return result },
		getRawResult: func() IRawResultStream {
			if rawResult == nil {
				return createSimpleRawResultStream(make(map[int]map[int]interface{}))
			}
			return createSimpleRawResultStream(rawResult)
		},
		getOutputBody: func() map[string]interface{} { return body },
		Msg:           msg,
		Err:           err,
	}
}

func NewErroneousExecutorOutput(err error) ExecutorOutput {
	return newExecutorOutput(nil, nil, nil, nil, err)
}

func NewEmptyExecutorOutput() ExecutorOutput {
	return newExecutorOutput(nil, nil, nil, nil, nil)
}

func NewNopEmptyExecutorOutput(messages []string) ExecutorOutput {
	return newExecutorOutput(nil, nil, nil, newBackendMessages(messages), nil)
}
