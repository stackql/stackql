package responsehandler

import (
	"fmt"
	"os"

	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/output"
)

func handleEmptyWriter(outputWriter output.IOutputWriter, err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}
	if outputWriter == nil {
		fmt.Fprintln(os.Stderr, "Unable to obtain output writer")
		return
	}
}

func HandleResponse(handlerCtx handler.HandlerContext, response internaldto.ExecutorOutput) error {
	var outputWriter output.IOutputWriter
	var err error
	logging.GetLogger().Debugln(fmt.Sprintf("response from query = '%v'", response.GetSQLResult()))
	if response.Msg != nil {
		for _, msg := range response.Msg.WorkingMessages {
			handlerCtx.GetOutfile().Write([]byte(msg + fmt.Sprintln("")))
		}
	}
	if response.GetSQLResult() != nil && response.GetSQLResult() != nil && response.Err == nil {
		outputWriter, err = output.GetOutputWriter(
			handlerCtx.GetOutfile(),
			handlerCtx.GetOutErrFile(),
			internaldto.OutputContext{
				RuntimeContext: handlerCtx.GetRuntimeContext(),
				Result:         response.GetSQLResult(),
			},
		)
		if outputWriter == nil || err != nil {
			handleEmptyWriter(outputWriter, err)
			return err
		}
		outputWriter.Write(response.GetSQLResult())
	} else if response.Err != nil {
		outputWriter, err = output.GetOutputWriter(
			handlerCtx.GetOutfile(),
			handlerCtx.GetOutErrFile(),
			internaldto.OutputContext{
				RuntimeContext: handlerCtx.GetRuntimeContext(),
				Result:         response.GetSQLResult(),
			},
		)
		if outputWriter == nil || err != nil {
			handleEmptyWriter(outputWriter, err)
			return response.Err
		}
		outputWriter.WriteError(response.Err, handlerCtx.GetErrorPresentation())
		return response.Err
	}
	return err
}
