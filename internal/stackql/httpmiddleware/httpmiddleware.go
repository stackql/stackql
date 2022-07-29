package httpmiddleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/stackql/stackql/internal/stackql/handler"
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

func HttpApiCallFromRequest(handlerCtx handler.HandlerContext, prov provider.IProvider, request *http.Request) (*http.Response, error) {
	httpClient, httpClientErr := getAuthenticatedClient(handlerCtx, prov)
	if httpClientErr != nil {
		return nil, httpClientErr
	}
	request.Header.Del("Authorization")
	if handlerCtx.RuntimeContext.HTTPLogEnabled {
		urlStr := ""
		if request != nil && request.URL != nil {
			urlStr = request.URL.String()
		}
		handlerCtx.OutErrFile.Write([]byte(fmt.Sprintf("http request url: %s\n", urlStr)))
		body := request.Body
		if body != nil {
			b, err := io.ReadAll(body)
			if err != nil {
				handlerCtx.OutErrFile.Write([]byte(fmt.Sprintf("error inpecting http request body: %s\n", err.Error())))
			}
			bodyStr := string(b)
			request.Body = io.NopCloser(bytes.NewBuffer(b))
			handlerCtx.OutErrFile.Write([]byte(fmt.Sprintf("http request body = '%s'\n", bodyStr)))
		}
	}
	r, err := httpClient.Do(request)
	if handlerCtx.RuntimeContext.HTTPLogEnabled {
		if r != nil {
			handlerCtx.OutErrFile.Write([]byte(fmt.Sprintf("http response status: %s\n", r.Status)))
		} else {
			handlerCtx.OutErrFile.Write([]byte("http response came buck null\n"))
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
