package httpmiddleware

import (
	"fmt"
	"net/http"

	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/httpexec"
	"github.com/stackql/stackql/internal/stackql/provider"
)

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

func HttpApiCallFromRequest(handlerCtx handler.HandlerContext, prov provider.IProvider, request *http.Request) (*http.Response, error) {
	httpClient, httpClientErr := getAuthenticatedClient(handlerCtx, prov)
	if httpClientErr != nil {
		return nil, httpClientErr
	}
	request.Header.Del("Authorization")
	r, err := httpClient.Do(request)
	if handlerCtx.RuntimeContext.HTTPLogEnabled {
		if r != nil {
			handlerCtx.OutErrFile.Write([]byte(fmt.Sprintln(fmt.Sprintf("http response status: %s", r.Status))))
		} else {
			handlerCtx.OutErrFile.Write([]byte(fmt.Sprintln("http response came buck null")))
		}
	}
	if err != nil {
		if handlerCtx.RuntimeContext.HTTPLogEnabled {
			handlerCtx.OutErrFile.Write([]byte(fmt.Sprintln(fmt.Sprintf("http response error: %s", err.Error()))))
		}
		return nil, err
	}
	return r, err
}

// DEPRECATED
func HttpApiCall(handlerCtx handler.HandlerContext, prov provider.IProvider, requestCtx httpexec.IHttpContext) (*http.Response, error) {
	httpClient, httpClientErr := getAuthenticatedClient(handlerCtx, prov)
	if httpClientErr != nil {
		return nil, httpClientErr
	}
	r, err := httpexec.HTTPApiCall(httpClient, requestCtx)
	if handlerCtx.RuntimeContext.HTTPLogEnabled {
		if r != nil {
			handlerCtx.OutErrFile.Write([]byte(fmt.Sprintln(fmt.Sprintf("http response status: %s", r.Status))))
		} else {
			handlerCtx.OutErrFile.Write([]byte(fmt.Sprintln("http response came buck null")))
		}
	}
	if err != nil {
		if handlerCtx.RuntimeContext.HTTPLogEnabled {
			handlerCtx.OutErrFile.Write([]byte(fmt.Sprintln(fmt.Sprintf("http response error: %s", err.Error()))))
		}
		return nil, err
	}
	return r, err
}
