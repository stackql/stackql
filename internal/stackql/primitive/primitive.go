package primitive

import (
	"io"

	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
)

type IPrimitiveCtx interface {
	GetAuthContext(string) (*dto.AuthCtx, error)
	GetWriter() io.Writer
	GetErrWriter() io.Writer
}

type IPrimitive interface {
	Optimise() error

	Execute(IPrimitiveCtx) internaldto.ExecutorOutput

	SetExecutor(func(pc IPrimitiveCtx) internaldto.ExecutorOutput) error

	SetTxnId(int)

	IncidentData(int64, internaldto.ExecutorOutput) error

	SetInputAlias(string, int64) error

	GetInputFromAlias(string) (internaldto.ExecutorOutput, bool)
}
