package internaldto

import (
	"io"

	"github.com/stackql/stackql/internal/stackql/dto"
)

type BasicPrimitiveContext struct {
	body      map[string]interface{}
	authCtx   func(string) (*dto.AuthCtx, error)
	writer    io.Writer
	errWriter io.Writer
}

func NewBasicPrimitiveContext(authCtx func(string) (*dto.AuthCtx, error), writer io.Writer, errWriter io.Writer) *BasicPrimitiveContext {
	return &BasicPrimitiveContext{
		authCtx:   authCtx,
		writer:    writer,
		errWriter: errWriter,
	}
}

func (bpp *BasicPrimitiveContext) GetAuthContext(prov string) (*dto.AuthCtx, error) {
	return bpp.authCtx(prov)
}

func (bpp *BasicPrimitiveContext) GetWriter() io.Writer {
	return bpp.writer
}

func (bpp *BasicPrimitiveContext) GetErrWriter() io.Writer {
	return bpp.errWriter
}
