package netutils

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
)

type HTTPContext interface {
	GetCABundle() string
	GetHTTPProxyHost() string
	GetHTTPProxyUser() string
	GetHTTPProxyPassword() string
	GetHTTPProxyPort() int
	GetHTTPProxyScheme() string
	GetAPIRequestTimeout() int
	GetTLSAllowInsecure() bool
}

func GetRoundTripper(httpCtx HTTPContext, existingTransport http.RoundTripper) http.RoundTripper {
	return getRoundTripper(httpCtx, existingTransport)
}

func getRoundTripper(httpCtx HTTPContext, existingTransport http.RoundTripper) http.RoundTripper {
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
	if httpCtx.GetCABundle() != "" {
		rootCAs, err := getCertPool(httpCtx.GetCABundle())
		if err == nil {
			config := &tls.Config{
				InsecureSkipVerify: httpCtx.GetTLSAllowInsecure(), //nolint:gosec // intentional, if contraindicated
				RootCAs:            rootCAs,
			}
			tr.TLSClientConfig = config
		}
	} else if httpCtx.GetTLSAllowInsecure() {
		config := &tls.Config{
			InsecureSkipVerify: httpCtx.GetTLSAllowInsecure(), //nolint:gosec // intentional, if contraindicated
		}
		tr.TLSClientConfig = config
	}
	host := httpCtx.GetHTTPProxyHost()
	if host != "" {
		if httpCtx.GetHTTPProxyPort() > 0 {
			host = fmt.Sprintf("%s:%d", httpCtx.GetHTTPProxyHost(), httpCtx.GetHTTPProxyPort())
		}
		var usr *url.Userinfo
		if httpCtx.GetHTTPProxyUser() != "" {
			usr = url.UserPassword(httpCtx.GetHTTPProxyUser(), httpCtx.GetHTTPProxyPassword())
		}
		proxyURL := &url.URL{
			Host:   host,
			Scheme: httpCtx.GetHTTPProxyScheme(),
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

func GetHTTPClient(httpCtx HTTPContext, existingClient *http.Client) *http.Client {
	return getHTTPClient(httpCtx, existingClient)
}

func getHTTPClient(httpCtx HTTPContext, existingClient *http.Client) *http.Client {
	var rt http.RoundTripper
	if existingClient != nil && existingClient.Transport != nil {
		rt = existingClient.Transport
	}
	return &http.Client{
		Timeout:   time.Second * time.Duration(httpCtx.GetAPIRequestTimeout()),
		Transport: getRoundTripper(httpCtx, rt),
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
