package httpmiddleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/any-sdk/pkg/requesttranslate"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/provider"
)

func GetAuthenticatedClient(handlerCtx handler.HandlerContext, prov provider.IProvider) (*http.Client, error) {
	return getAuthenticatedClient(handlerCtx, prov)
}

func getAuthenticatedClient(handlerCtx handler.HandlerContext, prov provider.IProvider) (*http.Client, error) {
	authCtx, authErr := handlerCtx.GetAuthContext(prov.GetProviderString())
	if authErr != nil {
		return nil, authErr
	}
	httpClient, httpClientErr := prov.Auth(authCtx, authCtx.Type, false)
	if httpClientErr != nil {
		return nil, httpClientErr
	}
	return httpClient, nil
}

//nolint:gocognit // acceptable
func HTTPApiCallFromRequest(
	handlerCtx handler.HandlerContext,
	prov provider.IProvider,
	method anysdk.OperationStore,
	request *http.Request,
) (*http.Response, error) {
	httpClient, httpClientErr := getAuthenticatedClient(handlerCtx, prov)
	if httpClientErr != nil {
		return nil, httpClientErr
	}
	request.Header.Del("Authorization")
	requestTranslator, err := requesttranslate.NewRequestTranslator(method.GetRequestTranslateAlgorithm())
	if err != nil {
		return nil, err
	}
	translatedRequest, err := requestTranslator.Translate(request)
	if err != nil {
		return nil, err
	}
	if handlerCtx.GetRuntimeContext().HTTPLogEnabled {
		urlStr := ""
		methodStr := ""
		if translatedRequest != nil && translatedRequest.URL != nil {
			urlStr = translatedRequest.URL.String()
			methodStr = translatedRequest.Method
		}
		//nolint:errcheck // output stream
		handlerCtx.GetOutErrFile().Write([]byte(fmt.Sprintf("http request url: '%s', method: '%s'\n", urlStr, methodStr)))
		body := translatedRequest.Body
		if body != nil {
			b, bErr := io.ReadAll(body)
			if bErr != nil {
				//nolint:errcheck // output stream
				handlerCtx.GetOutErrFile().Write([]byte(fmt.Sprintf("error inpecting http request body: %s\n", bErr.Error())))
			}
			bodyStr := string(b)
			translatedRequest.Body = io.NopCloser(bytes.NewBuffer(b))
			//nolint:errcheck // output stream
			handlerCtx.GetOutErrFile().Write([]byte(fmt.Sprintf("http request body = '%s'\n", bodyStr)))
		}
	}
	walObj, _ := handlerCtx.GetTSM()
	logging.GetLogger().Debugf("Proof of invariant: walObj = %v", walObj)
	r, err := httpClient.Do(translatedRequest)
	if handlerCtx.GetRuntimeContext().HTTPLogEnabled { //nolint:nestif // acceptable
		if r != nil {
			//nolint:errcheck // output stream
			handlerCtx.GetOutErrFile().Write([]byte(fmt.Sprintf("http response status: %s\n", r.Status)))
			if r.StatusCode >= 300 || handlerCtx.GetRuntimeContext().VerboseFlag {
				if r.Body != nil {
					bodyBytes, bErr := io.ReadAll(r.Body)
					if bErr != nil {
						return nil, bErr
					}
					//nolint:errcheck // output stream
					handlerCtx.GetOutErrFile().Write([]byte(fmt.Sprintf("http error response body: %s\n", string(bodyBytes))))
					r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
				}
			}
		} else {
			//nolint:errcheck // output stream
			handlerCtx.GetOutErrFile().Write([]byte("http response came buck null\n"))
		}
	}
	if err != nil {
		if handlerCtx.GetRuntimeContext().HTTPLogEnabled {
			//nolint:errcheck // output stream
			handlerCtx.GetOutErrFile().Write([]byte(
				fmt.Sprintln(fmt.Sprintf("http response error: %s", err.Error()))), //nolint:gosimple,lll // TODO: sweep through this sort of nonsense
			)
		}
		return nil, err
	}
	return r, err
}
