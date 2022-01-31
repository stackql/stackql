package querysubmit

import (
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/planbuilder"

	log "github.com/sirupsen/logrus"
)

func SubmitQuery(handlerCtx *handler.HandlerContext) dto.ExecutorOutput {
	log.Debugln("SubmitQuery() invoked...")
	plan, err := planbuilder.BuildPlanFromContext(handlerCtx)
	if err != nil {
		return dto.NewExecutorOutput(nil, nil, nil, nil, err)
	}
	pl := dto.NewBasicPrimitiveContext(
		nil,
		handlerCtx.Outfile,
		handlerCtx.OutErrFile,
	)
	return plan.Instructions.Execute(pl)
}
