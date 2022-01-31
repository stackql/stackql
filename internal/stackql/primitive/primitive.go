package primitive

import (
	"io"

	"github.com/stackql/stackql/internal/stackql/dto"
)

type IPrimitiveCtx interface {
	GetAuthContext(string) (*dto.AuthCtx, error)
	GetWriter() io.Writer
	GetErrWriter() io.Writer
}

type IPrimitive interface {
	Optimise() error

	Execute(IPrimitiveCtx) dto.ExecutorOutput

	SetTxnId(int)

	IncidentData(int64, dto.ExecutorOutput) error

	SetInputAlias(string, int64) error
}
