package intrinsic

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
)

type mutationPresentationError struct {
	cause   error
	message string
}

func (e mutationPresentationError) Error() string { return e.message }
func (e mutationPresentationError) Unwrap() error { return e.cause }
func mutationHTTPFailure(ctx queryContext, verb string, err error) error {
	var response omnisdk.HTTPStatusError
	if !errors.As(err, &response) {
		return err
	}
	if output, ok := ctx.(interface{ GetOutErrFile() io.Writer }); ok {
		diagnostic := fmt.Sprintf(
			"http response status code: %d, response body: %s", response.StatusCode(), response.ResponseBody(),
		)
		if response.ResponseBody() == "" {
			diagnostic = fmt.Sprintf("http response status code: %d, response body is nil", response.StatusCode())
		}
		//nolint:errcheck // Diagnostics do not determine the outcome.
		fmt.Fprintln(output.GetOutErrFile(), diagnostic)
	}
	message := fmt.Sprintf(
		"%s over HTTP error: %d %s", verb, response.StatusCode(), strings.ToUpper(http.StatusText(response.StatusCode())),
	)
	return mutationPresentationError{cause: err, message: message}
}
