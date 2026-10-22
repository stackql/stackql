package responsehandler

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"

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
				Query:          handlerCtx.GetRawQuery(),
				StartTime:      handlerCtx.GetQueryStartTime(),
			},
		)
		if outputWriter == nil || err != nil {
			handleEmptyWriter(outputWriter, err)
			return err
		}
		if writeErr := outputWriter.Write(sqlResult); writeErr != nil {
			//nolint:errcheck // Preserve the original stream failure.
			outputWriter.WriteError(
				presentationError(writeErr),
				handlerCtx.GetErrorPresentation(),
			)
			return writeErr
		}
	} else if sqlErr != nil {
		outputWriter, err = output.GetOutputWriter(
			handlerCtx.GetOutfile(),
			handlerCtx.GetOutErrFile(),
			internaldto.OutputContext{
				RuntimeContext: handlerCtx.GetRuntimeContext(),
				Result:         sqlResult,
				Query:          handlerCtx.GetRawQuery(),
				StartTime:      handlerCtx.GetQueryStartTime(),
			},
		)
		if outputWriter == nil || err != nil {
			handleEmptyWriter(outputWriter, err)
			return sqlErr
		}
		//nolint:errcheck // Preserve the original query error.
		outputWriter.WriteError(
			presentationError(sqlErr),
			handlerCtx.GetErrorPresentation(),
		)
		return sqlErr
	}
	return err
}

// Keep the established CLI diagnostic while retaining typed SDK errors for
// programmatic consumers and transport/effect classification.
func presentationError(err error) error {
	var response omnisdk.HTTPStatusError
	if !errors.As(err, &response) {
		return err
	}
	diagnostic := fmt.Sprintf(
		"http response status code: %d, response body: %s",
		response.StatusCode(), response.ResponseBody(),
	)
	if response.ResponseBody() == "" {
		diagnostic = fmt.Sprintf("http response status code: %d, response body is nil", response.StatusCode())
	}
	return errors.New(strings.Replace(err.Error(), response.Error(), diagnostic, 1))
}
