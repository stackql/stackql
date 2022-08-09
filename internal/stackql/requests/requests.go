package requests

import (
	"sort"
	"strings"

	"encoding/json"

	"github.com/stackql/stackql/internal/stackql/constants"
	"github.com/stackql/stackql/internal/stackql/httpparameters"
	"github.com/stackql/stackql/internal/stackql/provider"

	"github.com/stackql/go-openapistackql/openapistackql"
)

type requestBodyParam struct {
	Key string
	Val interface{}
}

func parseRequestBodyParam(k string, v interface{}) *requestBodyParam {
	trimmedKey := strings.TrimPrefix(k, constants.RequestBodyBaseKey)
	var parsedVal interface{}
	if trimmedKey != k {
		switch vt := v.(type) {
		case string:
			var js map[string]interface{}
			var jArr []interface{}
			if json.Unmarshal([]byte(vt), &js) == nil {
				parsedVal = js
			} else if json.Unmarshal([]byte(vt), &jArr) == nil {
				parsedVal = jArr
			} else {
				parsedVal = vt
			}
		default:
			parsedVal = vt
		}
		return &requestBodyParam{
			Key: trimmedKey,
			Val: parsedVal,
		}
	}
	return nil
}

func SplitHttpParameters(prov provider.IProvider, sqlParamMap map[int]map[string]interface{}, method *openapistackql.OperationStore) ([]*httpparameters.HttpParameters, error) {
	var retVal []*httpparameters.HttpParameters
	var rowKeys []int
	requestSchema, _ := method.GetRequestBodySchema()
	responseSchema, _ := method.GetRequestBodySchema()
	for idx, _ := range sqlParamMap {
		rowKeys = append(rowKeys, idx)
	}
	sort.Ints(rowKeys)
	for _, key := range rowKeys {
		sqlRow := sqlParamMap[key]
		pr, err := prov.GetProvider()
		if err != nil {
			return nil, err
		}
		reqMap := httpparameters.NewHttpParameters(pr, method)
		for k, v := range sqlRow {
			if param, ok := method.GetOperationParameter(k); ok {
				reqMap.StoreParameter(param, v)
			} else {
				if requestSchema != nil {
					rbp := parseRequestBodyParam(k, v)
					if rbp != nil {
						reqMap.RequestBody[rbp.Key] = rbp.Val
						continue
					}
				}
				reqMap.ServerParams[k] = httpparameters.NewParameterBinding(&openapistackql.Parameter{In: "server"}, v)
			}
			if responseSchema != nil && responseSchema.FindByPath(k, nil) != nil {
				reqMap.ResponseBody[k] = v
			}
		}
		retVal = append(retVal, reqMap)
	}
	return retVal, nil
}
