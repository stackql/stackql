package httpexec_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	. "github.com/stackql/stackql/internal/stackql/httpexec"

	"github.com/stackql/stackql/internal/test/testhttpapi"
	"github.com/stackql/stackql/internal/test/testutil"

	log "github.com/sirupsen/logrus"
)

func TestMain(m *testing.M) {
	log.SetOutput(ioutil.Discard)
	os.Exit(m.Run())
}

func validateContextualisedHTTPCallLightweight(t *testing.T, requestCtx IHttpContext, expectations testhttpapi.ExpectationStore, handlerFunc http.HandlerFunc) {
	handler := testhttpapi.GetRequestTestHandler(t, expectations, handlerFunc)
	http.DefaultClient.Transport = testhttpapi.NewHandlerTransport(http.HandlerFunc(handler), nil)
	response, err := HTTPApiCall(http.DefaultClient, requestCtx)
	testhttpapi.ValidateHTTPResponseAndErr(t, response, err)
}

func validateContextualisedHTTPCallHeavyweight(t *testing.T, requestCtx IHttpContext, expectations testhttpapi.ExpectationStore) {
	testhttpapi.StartServer(t, expectations)
	response, err := HTTPApiCall(http.DefaultClient, requestCtx)
	testhttpapi.ValidateHTTPResponseAndErr(t, response, err)
}

func TestBasicHTTPSExamples(t *testing.T) {
	// Recipe for testing with the httptest server -- heavyweight test with rich response object.
	s := httptest.NewServer(http.HandlerFunc(testhttpapi.DefaultHandler))
	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatalf("failed to parse httptest.Server URL: %v", err)
	}
	http.DefaultClient.Transport = testhttpapi.NewRewriteTransport(nil, u)
	resp, err := http.Get("https://google.com/path-one")
	if err != nil {
		t.Fatalf("failed to send first request: %v", err)
	}
	fmt.Println("[First Response]")
	resp.Write(os.Stdout)

	fmt.Print("\n", strings.Repeat("-", 80), "\n\n")

	// Heavyweight test on httpexec request context.
	requestCtx := CreateNonTemplatedHttpContext("GET", "https://google.com/path-one", nil)
	expectations := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", nil, "google.com", "bog-standard-reponse", nil)
	ex := testhttpapi.NewExpectationStore(1)
	ex.Put("google.com/path-one", *expectations)
	validateContextualisedHTTPCallHeavyweight(t, requestCtx, ex)

	// Lightweight test on httpexec request context.
	ex.Put("google.com/path-one", *expectations)
	lightweightRequestCtx := CreateNonTemplatedHttpContext("GET", "https://google.com/path-one", nil)
	validateContextualisedHTTPCallLightweight(t, lightweightRequestCtx, ex, nil)
}

func TestContextualisedHTTPSCall(t *testing.T) {

	requestCtx := CreateNonTemplatedHttpContext("GET", "https://google.com/path-one", nil)
	expectations := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", nil, "google.com", "bog-standard-reponse", nil)
	ex := testhttpapi.NewExpectationStore(1)
	ex.Put("google.com/path-one", *expectations)
	validateContextualisedHTTPCallHeavyweight(t, requestCtx, ex)
}

func TestContextualisedHTTPSCallLightweight(t *testing.T) {

	requestCtx := CreateNonTemplatedHttpContext("GET", "https://google.com/path-three", nil)
	expectations := testhttpapi.NewHTTPRequestExpectations(nil, nil, "GET", nil, "google.com", "bog-standard-response", nil)
	ex := testhttpapi.NewExpectationStore(1)
	ex.Put("google.com/path-three", *expectations)
	validateContextualisedHTTPCallLightweight(t, requestCtx, ex, nil)
}

func TestContextualisedRewrittenHTTPSCall(t *testing.T) {

	requestBody := `{ "data": { "key1": "value1", "key2": 2, "key3": { "k4": "v4" }, "key5": [ "v5", "v6" ]  } }`
	responseBody := `{ "response": { "key1": "all good"  } }`
	rb := testutil.CreateReadCloserFromString(requestBody)
	eb := testutil.CreateReadCloserFromString(requestBody)
	inURL := testhttpapi.NewURL("https", "google.com", "/create-widget")
	requestCtx := CreateNonTemplatedHttpContext("POST", fmt.Sprintf("%s://%s%s", inURL.Scheme, inURL.Host, inURL.Path), make(http.Header))
	requestCtx.SetHeader("Content-Type", "application/json")
	requestCtx.SetBody(rb)
	expectedHeaders := http.Header{"Content-Type": []string{"application/json"}}
	expectations := testhttpapi.NewHTTPRequestExpectations(eb, expectedHeaders, "POST", nil, "google.com", responseBody, make(http.Header))
	ex := testhttpapi.NewExpectationStore(1)
	ex.Put("google.com/create-widget", *expectations)
	validateContextualisedHTTPCallHeavyweight(t, requestCtx, ex)
}
