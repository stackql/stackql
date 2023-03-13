package httpbuild

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/provider"
	"github.com/stackql/stackql/internal/stackql/requests"
	"github.com/stackql/stackql/internal/stackql/streaming"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/go-openapistackql/openapistackql"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

func BuildHTTPRequestCtx(node sqlparser.SQLNode, prov provider.IProvider, m openapistackql.OperationStore, svc openapistackql.Service, insertValOnlyRows map[int]map[int]interface{}, execContext ExecContext) (HTTPArmoury, error) {
	var err error
	httpArmoury := NewHTTPArmoury()
	var requestSchema, responseSchema openapistackql.Schema
	req, reqExists := m.GetRequest()
	if reqExists && req.GetSchema() != nil {
		requestSchema = req.GetSchema()
	}
	res, resExists := m.GetResponse()
	if resExists && res.GetSchema() != nil {
		responseSchema = res.GetSchema()
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
		if execContext != nil && execContext.GetExecPayload() != nil {
			pm.SetBodyBytes(execContext.GetExecPayload().GetPayload())
			for j, v := range execContext.GetExecPayload().GetHeader() {
				pm.SetHeaderKV(j, v)
			}
			params.SetRequestBody(execContext.GetExecPayload().GetPayloadMap())
		} else if params.GetRequestBody() != nil && len(params.GetRequestBody()) != 0 {
			b, err := json.Marshal(params.GetRequestBody())
			if err != nil {
				return nil, err
			}
			pm.SetBodyBytes(b)
			req, reqExists := m.GetRequest()
			if reqExists {
				pm.SetHeaderKV("Content-Type", []string{req.GetBodyMediaType()})
			}
		}
		resp, respExists := m.GetResponse()
		if respExists {
			if resp.GetBodyMediaType() != "" && prov.GetProviderString() != "aws" {
				pm.SetHeaderKV("Accept", []string{resp.GetBodyMediaType()})
			}
		}
		pm.SetParameters(params)
		httpArmoury.AddRequestParams(pm)
	}
	secondPassParams := httpArmoury.GetRequestParams()
	pr, err := prov.GetProvider()
	if err != nil {
		return nil, err
	}
	for i, param := range secondPassParams {
		p := param
		if len(p.GetParameters().GetRequestBody()) == 0 {
			p.SetRequestBodyMap(nil)
		}
		var baseRequestCtx *http.Request
		switch node := node.(type) {
		case *sqlparser.Delete, *sqlparser.Exec, *sqlparser.Insert, *sqlparser.Select, *sqlparser.Update:
			baseRequestCtx, err = getRequest(pr, svc, m, p.GetParameters())
			if err != nil {
				return nil, err
			}
			for k, v := range p.GetHeader() {
				for _, vi := range v {
					baseRequestCtx.Header.Set(k, vi)
				}
			}
			p.SetRequest(baseRequestCtx)
		default:
			return nil, fmt.Errorf("cannot create http primitive for sql node of type %T", node)
		}
		if err != nil {
			return nil, err
		}
		logging.GetLogger().Infoln(fmt.Sprintf("pre transform: httpArmoury.RequestParams[%d] = %s", i, string(p.GetBodyBytes())))
		logging.GetLogger().Infoln(fmt.Sprintf("post transform: httpArmoury.RequestParams[%d] = %s", i, string(p.GetBodyBytes())))
		secondPassParams[i] = p
	}
	httpArmoury.SetRequestParams(secondPassParams)
	if err != nil {
		return nil, err
	}
	return httpArmoury, nil
}

func awsContextHousekeeping(ctx context.Context, svc openapistackql.Service, parameters map[string]interface{}) context.Context {
	ctx = context.WithValue(ctx, "service", svc.GetName())
	if region, ok := parameters["region"]; ok {
		if regionStr, ok := region.(string); ok {
			ctx = context.WithValue(ctx, "region", regionStr)
		}
	}
	return ctx
}

func getRequest(prov openapistackql.Provider, svc openapistackql.Service, method openapistackql.OperationStore, httpParams openapistackql.HttpParameters) (*http.Request, error) {
	params, err := httpParams.ToFlatMap()
	if err != nil {
		return nil, err
	}
	validationParams, err := method.Parameterize(prov, svc, httpParams, httpParams.GetRequestBody())
	if err != nil {
		return nil, err
	}
	request := validationParams.Request
	ctx := awsContextHousekeeping(request.Context(), svc, params)
	request = request.WithContext(ctx)
	return request, nil
}

func BuildHTTPRequestCtxFromAnnotation(parameters streaming.MapStream, prov provider.IProvider, m openapistackql.OperationStore, svc openapistackql.Service, insertValOnlyRows map[int]map[int]interface{}, execContext ExecContext) (HTTPArmoury, error) {
	var err error
	httpArmoury := NewHTTPArmoury()
	var requestSchema, responseSchema openapistackql.Schema
	req, reqExists := m.GetRequest()
	if reqExists && req.GetSchema() != nil {
		requestSchema = req.GetSchema()
	}
	resp, respExists := m.GetResponse()
	if respExists && resp.GetSchema() != nil {
		responseSchema = resp.GetSchema()
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
		if execContext != nil && execContext.GetExecPayload() != nil {
			pm.SetBodyBytes(execContext.GetExecPayload().GetPayload())
			for j, v := range execContext.GetExecPayload().GetHeader() {
				pm.SetHeaderKV(j, v)
			}
			params.SetRequestBody(execContext.GetExecPayload().GetPayloadMap())
		} else if params.GetRequestBody() != nil && len(params.GetRequestBody()) != 0 {
			b, err := json.Marshal(params.GetRequestBody())
			if err != nil {
				return nil, err
			}
			pm.SetBodyBytes(b)
			req, reqExists := m.GetRequest()
			if reqExists {
				pm.SetHeaderKV("Content-Type", []string{req.GetBodyMediaType()})
			}
		}
		resp, respExists := m.GetResponse()
		if respExists {
			if resp.GetBodyMediaType() != "" && prov.GetProviderString() != "aws" {
				pm.SetHeaderKV("Accept", []string{resp.GetBodyMediaType()})
			}
		}
		pm.SetParameters(params)
		httpArmoury.AddRequestParams(pm)
	}
	secondPassParams := httpArmoury.GetRequestParams()
	pr, err := prov.GetProvider()
	if err != nil {
		return nil, err
	}
	for i, param := range secondPassParams {
		p := param
		if len(p.GetParameters().GetRequestBody()) == 0 {
			p.SetRequestBodyMap(nil)
		}
		var baseRequestCtx *http.Request
		baseRequestCtx, err = getRequest(pr, svc, m, p.GetParameters())
		if err != nil {
			return nil, err
		}
		for k, v := range p.GetHeader() {
			for _, vi := range v {
				baseRequestCtx.Header.Set(k, vi)
			}
		}

		p.SetRequest(baseRequestCtx)
		logging.GetLogger().Infoln(fmt.Sprintf("pre transform: httpArmoury.RequestParams[%d] = %s", i, string(p.GetBodyBytes())))
		logging.GetLogger().Infoln(fmt.Sprintf("post transform: httpArmoury.RequestParams[%d] = %s", i, string(p.GetBodyBytes())))
		secondPassParams[i] = p
	}
	httpArmoury.SetRequestParams(secondPassParams)
	if err != nil {
		return nil, err
	}
	return httpArmoury, nil
}
