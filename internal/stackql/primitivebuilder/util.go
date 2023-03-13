package primitivebuilder

import (
	"fmt"

	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/tablemetadata"
	"github.com/stackql/stackql/internal/stackql/util"
)

func generateSuccessMessagesFromHeirarchy(meta tablemetadata.ExtendedTableMetadata, isAwait bool) []string {
	baseSuccessString := "The operation completed successfully"
	if !isAwait {
		baseSuccessString = "The operation was despatched successfully"
	}
	successMsgs := []string{
		baseSuccessString,
	}
	m, methodErr := meta.GetMethod()
	prov, err := meta.GetProvider()
	if methodErr == nil && err == nil && m != nil && prov != nil && prov.GetProviderString() == "google" {
		if m.GetAPIMethod() == "select" || m.GetAPIMethod() == "get" || m.GetAPIMethod() == "list" || m.GetAPIMethod() == "aggregatedList" {
			successMsgs = []string{
				fmt.Sprintf("%s, consider using a SELECT statement if you are performing an operation that returns data, see https://docs.stackql.io/language-spec/select for more information", baseSuccessString),
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
