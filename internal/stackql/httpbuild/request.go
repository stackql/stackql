package httpbuild

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/provider"
	"github.com/stackql/stackql/internal/stackql/requests"
	"github.com/stackql/stackql/internal/stackql/streaming"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/go-openapistackql/openapistackql"

	"vitess.io/vitess/go/vt/sqlparser"
)

type ExecContext struct {
	ExecPayload *dto.ExecPayload
	Resource    *openapistackql.Resource
}

func NewExecContext(payload *dto.ExecPayload, rsc *openapistackql.Resource) *ExecContext {
	return &ExecContext{
		ExecPayload: payload,
		Resource:    rsc,
	}
}

type HTTPArmouryParameters struct {
	Header     http.Header
	Parameters *openapistackql.HttpParameters
	Request    *http.Request
	BodyBytes  []byte
}

func (hap HTTPArmouryParameters) ToFlatMap() (map[string]interface{}, error) {
	return hap.toFlatMap()
}

func (hap HTTPArmouryParameters) toFlatMap() (map[string]interface{}, error) {
	if hap.Parameters != nil {
		return hap.Parameters.ToFlatMap()
	}
	return make(map[string]interface{}), nil
}

func (hap HTTPArmouryParameters) Encode() string {
	if hap.Parameters != nil {
		return hap.Parameters.Encode()
	}
	return ""
}

func (hap HTTPArmouryParameters) SetNextPage(ops *openapistackql.OperationStore, token string, tokenKey *dto.HTTPElement) (*http.Request, error) {
	rv := hap.Request.Clone(hap.Request.Context())
	switch tokenKey.Type {
	case dto.QueryParam:
		q := hap.Request.URL.Query()
		q.Set(tokenKey.Name, token)
		rv.URL.RawQuery = q.Encode()
		return rv, nil
	case dto.RequestString:
		u, err := url.Parse(token)
		if err != nil {
			return nil, err
		}
		rv.URL = u
		return rv, nil
	case dto.BodyAttribute:
		bm := make(map[string]interface{})
		for k, v := range hap.Parameters.RequestBody {
			bm[k] = v
		}
		bm[tokenKey.Name] = token
		b, err := ops.MarshalBody(bm, ops.Request)
		if err != nil {
			return nil, err
		}
		rv.Body = io.NopCloser(bytes.NewBuffer(b))
		rv.ContentLength = int64(len(b))
		return rv, nil
	default:
		return nil, fmt.Errorf("cannot accomodate pagaination for http element type = %+v", tokenKey.Type)
	}
}

type HTTPArmoury interface {
	AddRequestParams(HTTPArmouryParameters)
	GetRequestParams() []HTTPArmouryParameters
	GetRequestSchema() *openapistackql.Schema
	GetResponseSchema() *openapistackql.Schema
	SetRequestParams([]HTTPArmouryParameters)
	SetRequestSchema(*openapistackql.Schema)
	SetResponseSchema(*openapistackql.Schema)
}

type StandardHTTPArmoury struct {
	RequestParams  []HTTPArmouryParameters
	RequestSchema  *openapistackql.Schema
	ResponseSchema *openapistackql.Schema
}

func (ih *StandardHTTPArmoury) GetRequestParams() []HTTPArmouryParameters {
	return ih.RequestParams
}

func (ih *StandardHTTPArmoury) SetRequestParams(ps []HTTPArmouryParameters) {
	ih.RequestParams = ps
}

func (ih *StandardHTTPArmoury) AddRequestParams(p HTTPArmouryParameters) {
	ih.RequestParams = append(ih.RequestParams, p)
}

func (ih *StandardHTTPArmoury) SetRequestSchema(s *openapistackql.Schema) {
	ih.RequestSchema = s
}

func (ih *StandardHTTPArmoury) SetResponseSchema(s *openapistackql.Schema) {
	ih.ResponseSchema = s
}

func (ih *StandardHTTPArmoury) GetRequestSchema() *openapistackql.Schema {
	return ih.RequestSchema
}

func (ih *StandardHTTPArmoury) GetResponseSchema() *openapistackql.Schema {
	return ih.ResponseSchema
}

func NewHTTPArmouryParameters() HTTPArmouryParameters {
	return HTTPArmouryParameters{
		Header: make(http.Header),
	}
}

func NewHTTPArmoury() HTTPArmoury {
	return &StandardHTTPArmoury{}
}

func BuildHTTPRequestCtx(node sqlparser.SQLNode, prov provider.IProvider, m *openapistackql.OperationStore, svc *openapistackql.Service, insertValOnlyRows map[int]map[int]interface{}, execContext *ExecContext) (HTTPArmoury, error) {
	var err error
	httpArmoury := NewHTTPArmoury()
	var requestSchema, responseSchema *openapistackql.Schema
	if m.Request != nil && m.Request.Schema != nil {
		requestSchema = m.Request.Schema
	}
	if m.Response != nil && m.Response.Schema != nil {
		responseSchema = m.Response.Schema
	}
	httpArmoury.SetRequestSchema(requestSchema)
	httpArmoury.SetResponseSchema(responseSchema)
	paramMap, err := util.ExtractSQLNodeParams(node, insertValOnlyRows)
	if err != nil {
		return nil, err
	}
	paramList, err := requests.SplitHttpParameters(prov, paramMap, m)
	if err != nil {
		return nil, err
	}
	for _, prms := range paramList {
		params := prms
		pm := NewHTTPArmouryParameters()
		if err != nil {
			return nil, err
		}
		if execContext != nil && execContext.ExecPayload != nil {
			pm.BodyBytes = execContext.ExecPayload.Payload
			for j, v := range execContext.ExecPayload.Header {
				pm.Header[j] = v
			}
			params.RequestBody = execContext.ExecPayload.PayloadMap
		} else if params.RequestBody != nil && len(params.RequestBody) != 0 {
			b, err := json.Marshal(params.RequestBody)
			if err != nil {
				return nil, err
			}
			pm.BodyBytes = b
			pm.Header["Content-Type"] = []string{m.Request.BodyMediaType}
		}
		if m.Response != nil {
			if m.Response.BodyMediaType != "" && prov.GetProviderString() != "aws" {
				pm.Header["Accept"] = []string{m.Response.BodyMediaType}
			}
		}
		pm.Parameters = params
		httpArmoury.AddRequestParams(pm)
	}
	secondPassParams := httpArmoury.GetRequestParams()
	pr, err := prov.GetProvider()
	if err != nil {
		return nil, err
	}
	for i, param := range secondPassParams {
		p := param
		if len(p.Parameters.RequestBody) == 0 {
			p.Parameters.RequestBody = nil
		}
		var baseRequestCtx *http.Request
		switch node := node.(type) {
		case *sqlparser.Delete, *sqlparser.Exec, *sqlparser.Insert, *sqlparser.Select, *sqlparser.Update:
			baseRequestCtx, err = getRequest(pr, svc, m, p.Parameters)
			if err != nil {
				return nil, err
			}
			for k, v := range p.Header {
				for _, vi := range v {
					baseRequestCtx.Header.Set(k, vi)
				}
			}
			p.Request = baseRequestCtx
		default:
			return nil, fmt.Errorf("cannot create http primitive for sql node of type %T", node)
		}
		if err != nil {
			return nil, err
		}
		logging.GetLogger().Infoln(fmt.Sprintf("pre transform: httpArmoury.RequestParams[%d] = %s", i, string(p.BodyBytes)))
		logging.GetLogger().Infoln(fmt.Sprintf("post transform: httpArmoury.RequestParams[%d] = %s", i, string(p.BodyBytes)))
		secondPassParams[i] = p
	}
	httpArmoury.SetRequestParams(secondPassParams)
	if err != nil {
		return nil, err
	}
	return httpArmoury, nil
}

func awsContextHousekeeping(ctx context.Context, svc *openapistackql.Service, parameters map[string]interface{}) context.Context {
	ctx = context.WithValue(ctx, "service", svc.GetName())
	if region, ok := parameters["region"]; ok {
		if regionStr, ok := region.(string); ok {
			ctx = context.WithValue(ctx, "region", regionStr)
		}
	}
	return ctx
}

func getRequest(prov *openapistackql.Provider, svc *openapistackql.Service, method *openapistackql.OperationStore, httpParams *openapistackql.HttpParameters) (*http.Request, error) {
	params, err := httpParams.ToFlatMap()
	if err != nil {
		return nil, err
	}
	validationParams, err := method.Parameterize(prov, svc, httpParams, httpParams.RequestBody)
	if err != nil {
		return nil, err
	}
	request := validationParams.Request
	ctx := awsContextHousekeeping(request.Context(), svc, params)
	request = request.WithContext(ctx)
	return request, nil
}

func BuildHTTPRequestCtxFromAnnotation(parameters streaming.MapStream, prov provider.IProvider, m *openapistackql.OperationStore, svc *openapistackql.Service, insertValOnlyRows map[int]map[int]interface{}, execContext *ExecContext) (HTTPArmoury, error) {
	var err error
	httpArmoury := NewHTTPArmoury()
	var requestSchema, responseSchema *openapistackql.Schema
	if m.Request != nil && m.Request.Schema != nil {
		requestSchema = m.Request.Schema
	}
	if m.Response != nil && m.Response.Schema != nil {
		responseSchema = m.Response.Schema
	}
	httpArmoury.SetRequestSchema(requestSchema)
	httpArmoury.SetResponseSchema(responseSchema)

	paramMap := make(map[int]map[string]interface{})
	i := 0
	for {
		out, err := parameters.Read()
		for _, m := range out {
			paramMap[i] = m
			i++
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	paramList, err := requests.SplitHttpParameters(prov, paramMap, m)
	if err != nil {
		return nil, err
	}
	for _, prms := range paramList {
		params := prms
		pm := NewHTTPArmouryParameters()
		if err != nil {
			return nil, err
		}
		if execContext != nil && execContext.ExecPayload != nil {
			pm.BodyBytes = execContext.ExecPayload.Payload
			for j, v := range execContext.ExecPayload.Header {
				pm.Header[j] = v
			}
			params.RequestBody = execContext.ExecPayload.PayloadMap
		} else if params.RequestBody != nil && len(params.RequestBody) != 0 {
			b, err := json.Marshal(params.RequestBody)
			if err != nil {
				return nil, err
			}
			pm.BodyBytes = b
			pm.Header["Content-Type"] = []string{m.Request.BodyMediaType}
		}
		if m.Response != nil {
			if m.Response.BodyMediaType != "" && prov.GetProviderString() != "aws" {
				pm.Header["Accept"] = []string{m.Response.BodyMediaType}
			}
		}
		pm.Parameters = params
		httpArmoury.AddRequestParams(pm)
	}
	secondPassParams := httpArmoury.GetRequestParams()
	pr, err := prov.GetProvider()
	if err != nil {
		return nil, err
	}
	for i, param := range secondPassParams {
		p := param
		if len(p.Parameters.RequestBody) == 0 {
			p.Parameters.RequestBody = nil
		}
		var baseRequestCtx *http.Request
		baseRequestCtx, err = getRequest(pr, svc, m, p.Parameters)
		if err != nil {
			return nil, err
		}
		for k, v := range p.Header {
			for _, vi := range v {
				baseRequestCtx.Header.Set(k, vi)
			}
		}

		p.Request = baseRequestCtx
		logging.GetLogger().Infoln(fmt.Sprintf("pre transform: httpArmoury.RequestParams[%d] = %s", i, string(p.BodyBytes)))
		logging.GetLogger().Infoln(fmt.Sprintf("post transform: httpArmoury.RequestParams[%d] = %s", i, string(p.BodyBytes)))
		secondPassParams[i] = p
	}
	httpArmoury.SetRequestParams(secondPassParams)
	if err != nil {
		return nil, err
	}
	return httpArmoury, nil
}
