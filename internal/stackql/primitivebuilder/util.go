package primitivebuilder

import (
	"github.com/stackql/stackql/internal/stackql/internaldto"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"github.com/stackql/stackql/internal/stackql/util"
)

func generateSuccessMessagesFromHeirarchy(meta tablemetadata.ExtendedTableMetadata) []string {
	successMsgs := []string{
		"The operation completed successfully",
	}
	m, methodErr := meta.GetMethod()
	prov, err := meta.GetProvider()
	if methodErr == nil && err == nil && m != nil && prov != nil && prov.GetProviderString() == "google" {
		if m.APIMethod == "select" || m.APIMethod == "get" || m.APIMethod == "list" || m.APIMethod == "aggregatedList" {
			successMsgs = []string{
				"The operation completed successfully, consider using a SELECT statement if you are performing an operation that returns data, see https://docs.stackql.io/language-spec/select for more information",
			}
		}
	}
	return successMsgs
}

func generateResultIfNeededfunc(resultMap map[string]map[string]interface{}, body map[string]interface{}, msg *internaldto.BackendMessages, err error, isShowResults bool) internaldto.ExecutorOutput {
	if isShowResults {
		return util.PrepareResultSet(internaldto.NewPrepareResultSetDTO(nil, resultMap, nil, nil, nil, nil))
	}
	return internaldto.NewExecutorOutput(nil, body, nil, msg, err)
}
