package internaldto

import (
	"io"

	"github.com/stackql/any-sdk/pkg/dto"
)

var (
	_ BasicPrimitiveContext = &standardBasicPrimitiveContext{}
)

type BasicPrimitiveContext interface {
	GetAuthContext(prov string) (*dto.AuthCtx, error)
	GetErrWriter() io.Writer
	GetWriter() io.Writer
}

type standardBasicPrimitiveContext struct {
	authCtx   func(string) (*dto.AuthCtx, error)
	writer    io.Writer
	errWriter io.Writer
}

func NewBasicPrimitiveContext(
	authCtx func(string) (*dto.AuthCtx, error),
	writer io.Writer,
	errWriter io.Writer,
) BasicPrimitiveContext {
	return &standardBasicPrimitiveContext{
		authCtx:   authCtx,
		writer:    writer,
		errWriter: errWriter,
	}
}

func (bpp *standardBasicPrimitiveContext) GetAuthContext(prov string) (*dto.AuthCtx, error) {
	return bpp.authCtx(prov)
}

func (bpp *standardBasicPrimitiveContext) GetWriter() io.Writer {
	return bpp.writer
}

func (bpp *standardBasicPrimitiveContext) GetErrWriter() io.Writer {
	return bpp.errWriter
}
