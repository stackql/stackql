package netutils

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/stackql/stackql/internal/stackql/dto"
)

func GetRoundTripper(runtimeCtx dto.RuntimeCtx, existingTransport http.RoundTripper) http.RoundTripper {
	return getRoundTripper(runtimeCtx, existingTransport)
}

func getRoundTripper(runtimeCtx dto.RuntimeCtx, existingTransport http.RoundTripper) http.RoundTripper {
	var tr *http.Transport
	var rt http.RoundTripper
	if existingTransport != nil {
		switch exTR := existingTransport.(type) {
		case *http.Transport:
			tr = exTR.Clone()
		default:
			rt = exTR
		}
	} else {
		tr = &http.Transport{}
	}
	if runtimeCtx.CABundle != "" {
		rootCAs, err := getCertPool(runtimeCtx.CABundle)
		if err == nil {
			config := &tls.Config{
				InsecureSkipVerify: runtimeCtx.AllowInsecure, //nolint:gosec // intentional, if contraindicated
				RootCAs:            rootCAs,
			}
			tr.TLSClientConfig = config
		}
	} else if runtimeCtx.AllowInsecure {
		config := &tls.Config{
			InsecureSkipVerify: runtimeCtx.AllowInsecure, //nolint:gosec // intentional, if contraindicated
		}
		tr.TLSClientConfig = config
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
		proxyURL := &url.URL{
			Host:   host,
			Scheme: runtimeCtx.HTTPProxyScheme,
			User:   usr,
		}
		if tr != nil {
			tr.Proxy = http.ProxyURL(proxyURL)
		}
	}
	if tr != nil {
		rt = tr
	}
	return rt
}

func GetHTTPClient(runtimeCtx dto.RuntimeCtx, existingClient *http.Client) *http.Client {
	return getHTTPClient(runtimeCtx, existingClient)
}

func getHTTPClient(runtimeCtx dto.RuntimeCtx, existingClient *http.Client) *http.Client {
	var rt http.RoundTripper
	if existingClient != nil && existingClient.Transport != nil {
		rt = existingClient.Transport
	}
	return &http.Client{
		Timeout:   time.Second * time.Duration(runtimeCtx.APIRequestTimeout),
		Transport: getRoundTripper(runtimeCtx, rt),
	}
}

func getCertPool(localCaBundlePath string) (*x509.CertPool, error) {
	var lb []byte
	var err error
	if localCaBundlePath != "" {
		lb, err = os.ReadFile(localCaBundlePath)
		if err != nil {
			return nil, err
		}
	}
	sp, err := x509.SystemCertPool()
	if err == nil && sp != nil {
		if lb != nil {
			sp.AppendCertsFromPEM(lb)
		}
		return sp, nil
	}
	vp := x509.NewCertPool()
	if lb != nil {
		vp.AppendCertsFromPEM(lb)
	}
	return vp, nil
}
