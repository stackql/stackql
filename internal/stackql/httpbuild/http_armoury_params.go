package httpbuild

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql/internal/stackql/internaldto"
)

type HTTPArmouryParameters interface {
	Encode() string
	GetBodyBytes() []byte
	GetHeader() http.Header
	GetParameters() *openapistackql.HttpParameters
	GetQuery() url.Values
	GetRequest() *http.Request
	SetBodyBytes(b []byte)
	SetHeaderKV(k string, v []string)
	SetNextPage(ops *openapistackql.OperationStore, token string, tokenKey internaldto.HTTPElement) (*http.Request, error)
	SetParameters(*openapistackql.HttpParameters)
	SetRawQuery(string)
	SetRequest(*http.Request)
	SetRequestBodyMap(openapistackql.BodyMap)
	ToFlatMap() (map[string]interface{}, error)
}

func NewHTTPArmouryParameters() HTTPArmouryParameters {
	return &standardHTTPArmouryParameters{
		header: make(http.Header),
	}
}

type standardHTTPArmouryParameters struct {
	header     http.Header
	parameters *openapistackql.HttpParameters
	request    *http.Request
	bodyBytes  []byte
}

func (hap *standardHTTPArmouryParameters) GetQuery() url.Values {
	return hap.request.URL.Query()
}

func (hap *standardHTTPArmouryParameters) SetRawQuery(q string) {
	hap.request.URL.RawQuery = q
}

func (hap *standardHTTPArmouryParameters) SetRequest(req *http.Request) {
	hap.request = req
}

func (hap *standardHTTPArmouryParameters) GetRequest() *http.Request {
	return hap.request
}

func (hap *standardHTTPArmouryParameters) SetRequestBodyMap(body openapistackql.BodyMap) {
	hap.parameters.RequestBody = body
}

func (hap *standardHTTPArmouryParameters) SetParameters(p *openapistackql.HttpParameters) {
	hap.parameters = p
}

func (hap *standardHTTPArmouryParameters) GetParameters() *openapistackql.HttpParameters {
	return hap.parameters
}

func (hap *standardHTTPArmouryParameters) GetHeader() http.Header {
	return hap.header
}

func (hap *standardHTTPArmouryParameters) SetHeaderKV(k string, v []string) {
	hap.header[k] = v
}

func (hap *standardHTTPArmouryParameters) SetBodyBytes(b []byte) {
	hap.bodyBytes = b
}

func (hap *standardHTTPArmouryParameters) GetBodyBytes() []byte {
	return hap.bodyBytes
}

func (hap *standardHTTPArmouryParameters) ToFlatMap() (map[string]interface{}, error) {
	return hap.toFlatMap()
}

func (hap *standardHTTPArmouryParameters) toFlatMap() (map[string]interface{}, error) {
	if hap.parameters != nil {
		return hap.parameters.ToFlatMap()
	}
	return make(map[string]interface{}), nil
}

func (hap *standardHTTPArmouryParameters) Encode() string {
	if hap.parameters != nil {
		return hap.parameters.Encode()
	}
	return ""
}

func (hap *standardHTTPArmouryParameters) SetNextPage(ops *openapistackql.OperationStore, token string, tokenKey internaldto.HTTPElement) (*http.Request, error) {
	rv := hap.request.Clone(hap.request.Context())
	switch tokenKey.GetType() {
	case internaldto.QueryParam:
		q := hap.request.URL.Query()
		q.Set(tokenKey.GetName(), token)
		rv.URL.RawQuery = q.Encode()
		return rv, nil
	case internaldto.RequestString:
		u, err := url.Parse(token)
		if err != nil {
			return nil, err
		}
		rv.URL = u
		return rv, nil
	case internaldto.BodyAttribute:
		bm := make(map[string]interface{})
		for k, v := range hap.parameters.RequestBody {
			bm[k] = v
		}
		tokenName := tokenKey.GetName()
		bm[tokenName] = token
		b, err := ops.MarshalBody(bm, ops.Request)
		if err != nil {
			return nil, err
		}
		rv.Body = io.NopCloser(bytes.NewBuffer(b))
		rv.ContentLength = int64(len(b))
		return rv, nil
	default:
		return nil, fmt.Errorf("cannot accomodate pagaination for http element type = %+v", tokenKey.GetType())
	}
}
