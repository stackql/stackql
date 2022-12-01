package primitive

import (
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/internaldto"
	"github.com/stackql/stackql/internal/stackql/provider"
)

type HTTPRestPrimitive struct {
	Provider      provider.IProvider
	Executor      func(pc IPrimitiveCtx) internaldto.ExecutorOutput
	Preparator    func() drm.PreparedStatementCtx
	TxnControlCtr internaldto.TxnControlCounters
	Inputs        map[int64]internaldto.ExecutorOutput
	InputAliases  map[string]int64
	id            int64
}

func NewHTTPRestPrimitive(provider provider.IProvider, executor func(pc IPrimitiveCtx) internaldto.ExecutorOutput, preparator func() drm.PreparedStatementCtx, txnCtrlCtr internaldto.TxnControlCounters) IPrimitive {
	return &HTTPRestPrimitive{
		Provider:      provider,
		Executor:      executor,
		Preparator:    preparator,
		TxnControlCtr: txnCtrlCtr,
		Inputs:        make(map[int64]internaldto.ExecutorOutput),
		InputAliases:  make(map[string]int64),
	}
}

func (pr *HTTPRestPrimitive) SetTxnId(id int) {
	if pr.TxnControlCtr != nil {
		pr.TxnControlCtr.SetTxnID(id)
	}
}

func (pr *HTTPRestPrimitive) IncidentData(fromId int64, input internaldto.ExecutorOutput) error {
	pr.Inputs[fromId] = input
	return nil
}

func (pr *HTTPRestPrimitive) SetInputAlias(alias string, id int64) error {
	pr.InputAliases[alias] = id
	return nil
}

func (pr *HTTPRestPrimitive) Optimise() error {
	return nil
}

func (pr *HTTPRestPrimitive) GetInputFromAlias(alias string) (internaldto.ExecutorOutput, bool) {
	var rv internaldto.ExecutorOutput
	key, keyExists := pr.InputAliases[alias]
	if !keyExists {
		return rv, false
	}
	input, inputExists := pr.Inputs[key]
	if !inputExists {
		return rv, false
	}
	return input, true
}

func (pr *HTTPRestPrimitive) Execute(pc IPrimitiveCtx) internaldto.ExecutorOutput {
	if pr.Executor != nil {
		op := pr.Executor(pc)
		return op
	}
	return internaldto.NewExecutorOutput(nil, nil, nil, nil, nil)
}

func (pr *HTTPRestPrimitive) ID() int64 {
	return pr.id
}

func (pr *HTTPRestPrimitive) SetExecutor(ex func(pc IPrimitiveCtx) internaldto.ExecutorOutput) error {
	pr.Executor = ex
	return nil
}
