package requests

import (
	"sort"
	"strings"

	"encoding/json"

	"github.com/stackql/any-sdk/anysdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/parserutil"
	"github.com/stackql/stackql/internal/stackql/provider"
)

type requestBodyParam struct {
	Key string
	Val interface{}
}

func parseRequestBodyParam(k string, v interface{}, method anysdk.OperationStore) *requestBodyParam {
	trimmedKey, trimmedKeyErr := method.RenameRequestBodyAttribute(k)
	var parsedVal interface{}
	if trimmedKey != k && trimmedKeyErr == nil { //nolint:nestif // keep for now
		switch vt := v.(type) {
		case string:
			var js map[string]interface{}
			var jArr []interface{}
			//nolint:gocritic // keep for now
			if json.Unmarshal([]byte(vt), &js) == nil {
				parsedVal = js
			} else if json.Unmarshal([]byte(vt), &jArr) == nil {
				parsedVal = jArr
			} else {
				parsedVal = vt
			}
		case *sqlparser.FuncExpr:
			if strings.ToLower(vt.Name.GetRawVal()) == "string" && len(vt.Exprs) == 1 {
				pv, err := parserutil.GetStringFromStringFunc(vt)
				if err == nil {
					parsedVal = pv
				} else {
					parsedVal = vt
				}
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

//nolint:revive // not super complex
func SplitHTTPParameters(
	prov provider.IProvider,
	sqlParamMap map[int]map[string]interface{},
	method anysdk.OperationStore,
) ([]anysdk.HttpParameters, error) {
	var retVal []anysdk.HttpParameters
	var rowKeys []int
	requestSchema, _ := method.GetRequestBodySchema()
	responseSchema, _ := method.GetRequestBodySchema()
	for idx := range sqlParamMap {
		rowKeys = append(rowKeys, idx)
	}
	sort.Ints(rowKeys)
	for _, key := range rowKeys {
		sqlRow := sqlParamMap[key]
		reqMap := anysdk.NewHttpParameters(method)
		for k, v := range sqlRow {
			if param, ok := method.GetOperationParameter(k); ok {
				reqMap.StoreParameter(param, v)
			} else {
				if requestSchema != nil {
					rbp := parseRequestBodyParam(k, v, method)
					if rbp != nil {
						reqMap.SetRequestBodyParam(rbp.Key, rbp.Val)
						continue
					}
				}
				reqMap.SetServerParam(k, method.GetService(), v)
			}
			if responseSchema != nil && responseSchema.FindByPath(k, nil) != nil {
				reqMap.SetResponseBodyParam(k, v)
			}
		}
		retVal = append(retVal, reqMap)
	}
	return retVal, nil
}
