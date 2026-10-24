package intrinsic

import (
	"context"
	"fmt"
	"time"

	"github.com/stackql-labs/omnisdk/pkg/omnisdk"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/parserutil"
)

func sleepFunc(node *sqlparser.Sleep) func() internaldto.ExecutorOutput {
	return func() internaldto.ExecutorOutput {
		duration, err := parserutil.ExtractSleepDuration(node)
		if err != nil {
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, err)
		}
		planned, err := omnisdk.NewSleep(time.Duration(duration) * time.Millisecond)
		if err != nil {
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, err)
		}
		rows, err := planned.Open(context.Background())
		if err != nil {
			return internaldto.NewExecutorOutput(nil, nil, nil, nil, err)
		}
		defer rows.Close()
		for rows.Next() {
		}
		return internaldto.NewExecutorOutput(nil, nil, nil,
			internaldto.NewBackendMessages([]string{fmt.Sprintf("Success: slept for %d milliseconds", duration)}), rows.Err())
	}
}
