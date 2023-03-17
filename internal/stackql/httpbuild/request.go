package httpbuild

import (
	"context"
	"encoding/json"
	"errors"
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

//nolint:funlen,gocognit // TODO: review
func BuildHTTPRequestCtx(
	node sqlparser.SQLNode,
	prov provider.IProvider,
	m openapistackql.OperationStore,
	svc openapistackql.Service,
	insertValOnlyRows map[int]map[int]interface{},
	execContext ExecContext,
) (HTTPArmoury, error) {
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
	paramList, err := requests.SplitHTTPParameters(prov, paramMap, m)
	if err != nil {
		return nil, err
	}
	//nolint:dupl // TODO: review
	for _, prms := range paramList {
		params := prms
		pm := NewHTTPArmouryParameters()
		if execContext != nil && execContext.GetExecPayload() != nil {
			pm.SetBodyBytes(execContext.GetExecPayload().GetPayload())
			for j, v := range execContext.GetExecPayload().GetHeader() {
				pm.SetHeaderKV(j, v)
			}
			params.SetRequestBody(execContext.GetExecPayload().GetPayloadMap())
		} else if params.GetRequestBody() != nil && len(params.GetRequestBody()) != 0 {
			b, bErr := json.Marshal(params.GetRequestBody())
			if bErr != nil {
				return nil, bErr
			}
			pm.SetBodyBytes(b)
			req, reqExists := m.GetRequest() //nolint:govet // intentional shadowing
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
		logging.GetLogger().Infoln(
			fmt.Sprintf(
				"pre transform: httpArmoury.RequestParams[%d] = %s", i, string(p.GetBodyBytes())))
		logging.GetLogger().Infoln(
			fmt.Sprintf(
				"post transform: httpArmoury.RequestParams[%d] = %s", i, string(p.GetBodyBytes())))
		secondPassParams[i] = p
	}
	httpArmoury.SetRequestParams(secondPassParams)
	if err != nil {
		return nil, err
	}
	return httpArmoury, nil
}

func awsContextHousekeeping(
	ctx context.Context,
	svc openapistackql.Service,
	parameters map[string]interface{},
) context.Context {
	ctx = context.WithValue(ctx, "service", svc.GetName()) //nolint:revive,staticcheck // TODO: add custom context type
	if region, ok := parameters["region"]; ok {
		if regionStr, rOk := region.(string); rOk {
			ctx = context.WithValue(ctx, "region", regionStr) //nolint:revive,staticcheck // TODO: add custom context type
		}
	}
	return ctx
}

func getRequest(
	prov openapistackql.Provider,
	svc openapistackql.Service,
	method openapistackql.OperationStore,
	httpParams openapistackql.HttpParameters,
) (*http.Request, error) {
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

//nolint:funlen,gocognit // acceptable
func BuildHTTPRequestCtxFromAnnotation(
	parameters streaming.MapStream, prov provider.IProvider,
	m openapistackql.OperationStore, svc openapistackql.Service,
	insertValOnlyRows map[int]map[int]interface{},
	execContext ExecContext) (HTTPArmoury, error) {
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
		out, oErr := parameters.Read()
		for _, m := range out {
			paramMap[i] = m
			i++
		}
		if errors.Is(oErr, io.EOF) {
			break
		}
		if oErr != nil {
			return nil, oErr
		}
	}
	paramList, err := requests.SplitHTTPParameters(prov, paramMap, m)
	if err != nil {
		return nil, err
	}
	for _, prms := range paramList { //nolint:dupl // TODO: refactor
		params := prms
		pm := NewHTTPArmouryParameters()
		if execContext != nil && execContext.GetExecPayload() != nil {
			pm.SetBodyBytes(execContext.GetExecPayload().GetPayload())
			for j, v := range execContext.GetExecPayload().GetHeader() {
				pm.SetHeaderKV(j, v)
			}
			params.SetRequestBody(execContext.GetExecPayload().GetPayloadMap())
		} else if params.GetRequestBody() != nil && len(params.GetRequestBody()) != 0 {
			b, jErr := json.Marshal(params.GetRequestBody())
			if jErr != nil {
				return nil, jErr
			}
			pm.SetBodyBytes(b)
			req, reqExists := m.GetRequest() //nolint:govet // intentional
			if reqExists {
				pm.SetHeaderKV("Content-Type", []string{req.GetBodyMediaType()})
			}
		}
		resp, respExists := m.GetResponse() //nolint:govet // intentional
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
		logging.GetLogger().Infoln(
			fmt.Sprintf("pre transform: httpArmoury.RequestParams[%d] = %s",
				i, string(p.GetBodyBytes())))
		logging.GetLogger().Infoln(
			fmt.Sprintf("post transform: httpArmoury.RequestParams[%d] = %s",
				i, string(p.GetBodyBytes())))
		secondPassParams[i] = p
	}
	httpArmoury.SetRequestParams(secondPassParams)
	if err != nil {
		return nil, err
	}
	return httpArmoury, nil
}
