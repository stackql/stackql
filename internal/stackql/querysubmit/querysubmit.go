package querysubmit

import (
	"github.com/stackql/stackql/internal/stackql/handler"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/planbuilder"
)

var (
	_ QuerySubmitter = &basicQuerySubmitter{}
)

type QuerySubmitter interface {
	SubmitQuery(handlerCtx handler.HandlerContext) internaldto.ExecutorOutput
}

func NewQuerySubmitter() QuerySubmitter {
	return &basicQuerySubmitter{}
}

type basicQuerySubmitter struct{}

func (qs *basicQuerySubmitter) SubmitQuery(handlerCtx handler.HandlerContext) internaldto.ExecutorOutput {
	logging.GetLogger().Debugln("SubmitQuery() invoked...")
	pb := planbuilder.NewPlanBuilder()
	plan, err := pb.BuildPlanFromContext(handlerCtx)
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
