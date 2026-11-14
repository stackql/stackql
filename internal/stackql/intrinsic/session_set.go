package intrinsic

import (
	"fmt"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/sessionconfig"
)

func sessionSetFunc(ctx queryContext, node *sqlparser.Set) (func() internaldto.ExecutorOutput, bool) {
	host, ok := ctx.(interface {
		SetConfigAtPath(string, interface{}, string) error
	})
	if !ok {
		return nil, false
	}
	return func() internaldto.ExecutorOutput {
		if len(node.Exprs) == 0 {
			return internaldto.NewErroneousExecutorOutput(fmt.Errorf("no set expressions found"))
		}
		for _, expression := range node.Exprs {
			if err := sessionconfig.Apply(host, expression); err != nil {
				return internaldto.NewErroneousExecutorOutput(fmt.Errorf("%w: %w", sessionconfig.ErrSessionConfiguration, err))
			}
		}
		return internaldto.NewNopEmptyExecutorOutput(nil)
	}, true
}
