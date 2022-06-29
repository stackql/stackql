package httpbuild

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/provider"
	"github.com/stackql/stackql/internal/stackql/requests"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/go-openapistackql/openapistackql"

	"vitess.io/vitess/go/vt/sqlparser"

	log "github.com/sirupsen/logrus"
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
	Parameters *dto.HttpParameters
	Request    *http.Request
	BodyBytes  []byte
}

func (hap HTTPArmouryParameters) ToFlatMap() (map[string]interface{}, error) {
	if hap.Parameters != nil {
		return hap.Parameters.ToFlatMap()
	}
	return make(map[string]interface{}), nil
}

func (hap HTTPArmouryParameters) SetNextPage(token string, tokenKey *dto.HTTPElement) (*http.Request, error) {
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
	default:
		return nil, fmt.Errorf("cannot accomodate pagaination for http element type = %+v", tokenKey.Type)
	}
}

type HTTPArmoury struct {
	RequestParams  []HTTPArmouryParameters
	RequestSchema  *openapistackql.Schema
	ResponseSchema *openapistackql.Schema
}

func NewHTTPArmouryParameters() HTTPArmouryParameters {
	return HTTPArmouryParameters{
		Header: make(http.Header),
	}
}

func NewHTTPArmoury() HTTPArmoury {
	return HTTPArmoury{}
}

func BuildHTTPRequestCtx(handlerCtx *handler.HandlerContext, node sqlparser.SQLNode, prov provider.IProvider, m *openapistackql.OperationStore, svc *openapistackql.Service, insertValOnlyRows map[int]map[int]interface{}, execContext *ExecContext) (*HTTPArmoury, error) {
	var err error
	httpArmoury := NewHTTPArmoury()
	var requestSchema, responseSchema *openapistackql.Schema
	if m.Request != nil && m.Request.Schema != nil {
		requestSchema = m.Request.Schema
	}
	if m.Response != nil && m.Response.Schema != nil {
		responseSchema = m.Response.Schema
	}
	httpArmoury.RequestSchema = requestSchema
	httpArmoury.ResponseSchema = responseSchema
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
		httpArmoury.RequestParams = append(httpArmoury.RequestParams, pm)
	}
	for i, param := range httpArmoury.RequestParams {
		p := param
		if len(p.Parameters.RequestBody) == 0 {
			p.Parameters.RequestBody = nil
		}
		var baseRequestCtx *http.Request
		switch node := node.(type) {
		case *sqlparser.Delete, *sqlparser.Exec, *sqlparser.Insert, *sqlparser.Select:
			baseRequestCtx, err = getRequest(svc, m, p.Parameters)
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
		log.Infoln(fmt.Sprintf("pre transform: httpArmoury.RequestParams[%d] = %s", i, string(p.BodyBytes)))
		if handlerCtx.RuntimeContext.HTTPLogEnabled {
			// url, _ := p.Context.GetUrl()
			// handlerCtx.OutErrFile.Write([]byte(fmt.Sprintln(fmt.Sprintf("http request url: %s", url))))
		}
		log.Infoln(fmt.Sprintf("post transform: httpArmoury.RequestParams[%d] = %s", i, string(p.BodyBytes)))
		httpArmoury.RequestParams[i] = p
	}
	if err != nil {
		return nil, err
	}
	return &httpArmoury, nil
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

func getRequest(svc *openapistackql.Service, method *openapistackql.OperationStore, httpParams *dto.HttpParameters) (*http.Request, error) {
	params, err := httpParams.ToFlatMap()
	if err != nil {
		return nil, err
	}
	validationParams, err := method.Parameterize(svc, params, httpParams.RequestBody)
	if err != nil {
		return nil, err
	}
	request := validationParams.Request
	ctx := awsContextHousekeeping(request.Context(), svc, params)
	request = request.WithContext(ctx)
	return request, nil
}

func BuildHTTPRequestCtxFromAnnotation(handlerCtx *handler.HandlerContext, parameters map[string]interface{}, prov provider.IProvider, m *openapistackql.OperationStore, svc *openapistackql.Service, insertValOnlyRows map[int]map[int]interface{}, execContext *ExecContext) (*HTTPArmoury, error) {
	var err error
	httpArmoury := NewHTTPArmoury()
	var requestSchema, responseSchema *openapistackql.Schema
	if m.Request != nil && m.Request.Schema != nil {
		requestSchema = m.Request.Schema
	}
	if m.Response != nil && m.Response.Schema != nil {
		responseSchema = m.Response.Schema
	}
	httpArmoury.RequestSchema = requestSchema
	httpArmoury.ResponseSchema = responseSchema
	paramMap := map[int]map[string]interface{}{0: parameters}
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
		httpArmoury.RequestParams = append(httpArmoury.RequestParams, pm)
	}
	for i, param := range httpArmoury.RequestParams {
		p := param
		if len(p.Parameters.RequestBody) == 0 {
			p.Parameters.RequestBody = nil
		}
		var baseRequestCtx *http.Request
		baseRequestCtx, err = getRequest(svc, m, p.Parameters)
		if err != nil {
			return nil, err
		}
		for k, v := range p.Header {
			for _, vi := range v {
				baseRequestCtx.Header.Set(k, vi)
			}
		}

		p.Request = baseRequestCtx
		log.Infoln(fmt.Sprintf("pre transform: httpArmoury.RequestParams[%d] = %s", i, string(p.BodyBytes)))
		if handlerCtx.RuntimeContext.HTTPLogEnabled {
			// url, _ := p.Context.GetUrl()
			// handlerCtx.OutErrFile.Write([]byte(fmt.Sprintln(fmt.Sprintf("http request url: %s", url))))
		}
		log.Infoln(fmt.Sprintf("post transform: httpArmoury.RequestParams[%d] = %s", i, string(p.BodyBytes)))
		httpArmoury.RequestParams[i] = p
	}
	if err != nil {
		return nil, err
	}
	return &httpArmoury, nil
}
