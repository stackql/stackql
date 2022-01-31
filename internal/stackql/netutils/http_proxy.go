package netutils

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/stackql/stackql/internal/stackql/dto"
)

func GetHttpClient(runtimeCtx dto.RuntimeCtx, existingClient *http.Client) *http.Client {
	var tr *http.Transport
	var rt http.RoundTripper
	if existingClient != nil && existingClient.Transport != nil {
		switch exTR := existingClient.Transport.(type) {
		case *http.Transport:
			tr = exTR.Clone()
		default:
			rt = exTR
		}
	} else {
		tr = &http.Transport{}
	}
	host := runtimeCtx.HTTPProxyHost
	if host != "" {
		if runtimeCtx.HTTPProxyPort > 0 {
			host = fmt.Sprintf("%s:%d", runtimeCtx.HTTPProxyHost, runtimeCtx.HTTPProxyPort)
		}
		var usr *url.Userinfo
		if runtimeCtx.HTTPProxyUser != "" {
			usr = url.UserPassword(runtimeCtx.HTTPProxyUser, runtimeCtx.HTTPProxyPassword)
		}
		proxyUrl := &url.URL{
			Host:   host,
			Scheme: runtimeCtx.HTTPProxyScheme,
			User:   usr,
		}
		if tr != nil {
			tr.Proxy = http.ProxyURL(proxyUrl)
		}
	}
	if tr != nil {
		rt = tr
	}
	return &http.Client{
		Timeout:   time.Second * time.Duration(runtimeCtx.APIRequestTimeout),
		Transport: rt,
	}
}
