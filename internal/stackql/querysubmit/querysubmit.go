package querysubmit

import (
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/planbuilder"
)

func SubmitQuery(handlerCtx handler.HandlerContext) internaldto.ExecutorOutput {
	logging.GetLogger().Debugln("SubmitQuery() invoked...")
	plan, err := planbuilder.BuildPlanFromContext(handlerCtx)
	if err != nil {
		return internaldto.NewExecutorOutput(nil, nil, nil, nil, err)
	}
	pl := internaldto.NewBasicPrimitiveContext(
		nil,
		handlerCtx.GetOutfile(),
		handlerCtx.GetOutErrFile(),
	)
	return plan.Instructions.Execute(pl)
}
