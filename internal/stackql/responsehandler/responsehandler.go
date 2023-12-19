package responsehandler

import (
	"fmt"
	"os"

	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
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
	if response.GetMessages() != nil {
		for _, msg := range response.GetMessages() {
			handlerCtx.GetOutErrFile().Write([]byte(msg + fmt.Sprintln(""))) //nolint:errcheck // outstream write
		}
	}
	sqlResult := response.GetSQLResult()
	sqlErr := response.GetError()
	if sqlResult != nil && sqlErr == nil {
		outputWriter, err = output.GetOutputWriter(
			handlerCtx.GetOutfile(),
			handlerCtx.GetOutErrFile(),
			internaldto.OutputContext{
				RuntimeContext: handlerCtx.GetRuntimeContext(),
				Result:         sqlResult,
			},
		)
		if outputWriter == nil || err != nil {
			handleEmptyWriter(outputWriter, err)
			return err
		}
		outputWriter.Write(sqlResult) //nolint:errcheck // outstream write
	} else if sqlErr != nil {
		outputWriter, err = output.GetOutputWriter(
			handlerCtx.GetOutfile(),
			handlerCtx.GetOutErrFile(),
			internaldto.OutputContext{
				RuntimeContext: handlerCtx.GetRuntimeContext(),
				Result:         sqlResult,
			},
		)
		if outputWriter == nil || err != nil {
			handleEmptyWriter(outputWriter, err)
			return sqlErr
		}
		outputWriter.WriteError(sqlErr, handlerCtx.GetErrorPresentation()) //nolint:errcheck // outstream write
		return sqlErr
	}
	return err
}
