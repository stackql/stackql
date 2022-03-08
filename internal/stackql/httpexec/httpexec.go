package httpexec

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"net/url"

	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql/internal/stackql/util"

	log "github.com/sirupsen/logrus"
)

type IHttpContext interface {
	RemoveQueryParam(string)
	GetHeaders() http.Header
	GetMethod() string
	GetTemplateUrl() string
	GetUrl() (string, error)
	SetHeader(string, string)
	SetQueryParam(string, string)
	SetHeaders(http.Header)
	SetMethod(string)
	SetUrl(string)
	SetBody(io.Reader)
	GetBody() io.Reader
}

type BasicHttpContext struct {
	method      string
	templateUrl string
	url         string
	headers     http.Header
	body        io.Reader
	queryParams map[string]string
}

func CreateTemplatedHttpContext(method string, templateUrl string, headers http.Header) IHttpContext {
	return &BasicHttpContext{
		method:      method,
		templateUrl: templateUrl,
		headers:     headers,
		queryParams: make(map[string]string),
	}
}

func CreateNonTemplatedHttpContext(method string, url string, headers http.Header) IHttpContext {
	return &BasicHttpContext{
		method:      method,
		url:         url,
		headers:     headers,
		queryParams: make(map[string]string),
	}
}

func (bc *BasicHttpContext) GetMethod() string {
	return bc.method
}

func (bc *BasicHttpContext) GetHeaders() http.Header {
	return bc.headers
}

func (bc *BasicHttpContext) GetUrl() (string, error) {
	urlObj, err := url.Parse(bc.url)
	if err != nil {
		return "", err
	}
	q := urlObj.Query()
	for k, v := range bc.queryParams {
		q.Set(k, v)
	}
	urlObj.RawQuery = q.Encode()
	return urlObj.String(), nil
}

func (bc *BasicHttpContext) SetBody(body io.Reader) {
	bc.body = body
}

func (bc *BasicHttpContext) GetBody() io.Reader {
	return bc.body
}

func (bc *BasicHttpContext) GetTemplateUrl() string {
	return bc.templateUrl
}

func (bc *BasicHttpContext) SetMethod(method string) {
	bc.method = method
}

func (bc *BasicHttpContext) SetUrl(url string) {
	bc.url = url
}

func (bc *BasicHttpContext) SetHeaders(headers http.Header) {
	bc.headers = headers
}

func (bc *BasicHttpContext) SetHeader(k string, v string) {
	if headerVals, ok := bc.headers[k]; ok {
		bc.headers[k] = append(headerVals, v)
	}
	bc.headers[k] = []string{v}
}

func (bc *BasicHttpContext) SetQueryParam(k string, v string) {
	bc.queryParams[k] = v
}

func (bc *BasicHttpContext) RemoveQueryParam(k string) {
	delete(bc.queryParams, k)
}

func getErroneousBody(response *http.Response) string {
	nullErr := "empty response body"
	if response.Body == nil {
		return nullErr
	}
	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nullErr
	}
	bodyString := string(bodyBytes)
	return bodyString
}

func HTTPApiCall(httpClient *http.Client, requestCtx IHttpContext) (*http.Response, error) {
	urlStr, err := requestCtx.GetUrl()
	if err != nil {
		return nil, err
	}
	body := requestCtx.GetBody()
	var bytez []byte
	if body != nil {
		buf := new(bytes.Buffer)
		buf.ReadFrom(body)
		bytez = buf.Bytes()
	}
	log.Infoln(fmt.Sprintf("request body = %s", string(bytez)))
	req, requestErr := http.NewRequest(requestCtx.GetMethod(), urlStr, bytes.NewReader(bytez))
	for k, v := range requestCtx.GetHeaders() {
		for i := range v {
			req.Header.Set(k, v[i])
		}
	}
	if requestErr != nil {
		return nil, requestErr
	}
	log.Infoln(fmt.Sprintf("http request = %v", req))
	response, reponseErr := httpClient.Do(req)
	log.Infoln(fmt.Sprintf("http response = %v", response))
	if reponseErr != nil {
		log.Infoln(fmt.Errorf("error for request method = %v, url = %v", req.Method, req.URL))
		return response, reponseErr
	}
	if response != nil && (response.StatusCode >= 400) {
		log.Infoln(fmt.Errorf("code-dictated error for request method = %v, url = %v", req.Method, req.URL))
		return response, fmt.Errorf("API error, status code %d: %s", response.StatusCode, getErroneousBody(response))
	}
	return response, nil
}

func getResponseMediaType(r *http.Response) (string, error) {
	rt := r.Header.Get("Content-Type")
	var mediaType string
	var err error
	if rt != "" {
		mediaType, _, err = mime.ParseMediaType(rt)
		if err != nil {
			return "", err
		}
		return mediaType, nil
	}
	return "", nil
}

func marshalResponse(r *http.Response) (interface{}, error) {
	body := r.Body
	if body != nil {
		defer body.Close()
	} else {
		return nil, nil
	}
	var target interface{}
	mediaType, err := getResponseMediaType(r)
	if err != nil {
		return nil, err
	}
	switch mediaType {
	case openapistackql.MediaTypeJson:
		err = json.NewDecoder(body).Decode(&target)
	case openapistackql.MediaTypeXML:
		err = xml.NewDecoder(body).Decode(&target)
	case openapistackql.MediaTypeOctetStream:
		target, err = io.ReadAll(body)
	case openapistackql.MediaTypeTextPlain, openapistackql.MediaTypeHTML:
		var b []byte
		b, err = io.ReadAll(body)
		if err == nil {
			target = string(b)
		}
	default:
		target, err = io.ReadAll(body)
	}
	return target, err
}

func ProcessHttpResponse(response *http.Response) (interface{}, error) {
	target, err := marshalResponse(response)
	if err == nil && response.StatusCode >= 400 {
		err = fmt.Errorf(fmt.Sprintf("HTTP response error: %s", string(util.InterfaceToBytes(target, true))))
	}
	if err == io.EOF {
		if response.StatusCode >= 200 && response.StatusCode < 300 {
			return map[string]interface{}{"result": "The Operation Completed Successfully"}, nil
		}
	}
	switch rv := target.(type) {
	case string, int:
		return map[string]interface{}{openapistackql.AnonymousColumnName: []interface{}{rv}}, nil
	}
	return target, err
}

func DeprecatedProcessHttpResponse(response *http.Response) (map[string]interface{}, error) {
	target, err := ProcessHttpResponse(response)
	if err != nil {
		return nil, err
	}
	switch rv := target.(type) {
	case map[string]interface{}:
		return rv, nil
	case nil:
		return nil, nil
	case string:
		return map[string]interface{}{openapistackql.AnonymousColumnName: rv}, nil
	case []byte:
		return map[string]interface{}{openapistackql.AnonymousColumnName: string(rv)}, nil
	default:
		return nil, fmt.Errorf("DeprecatedProcessHttpResponse() cannot acccept response of type %T", rv)
	}
}
