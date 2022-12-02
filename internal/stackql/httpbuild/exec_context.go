package httpbuild

import (
	"github.com/stackql/go-openapistackql/openapistackql"
	"github.com/stackql/stackql/internal/stackql/internaldto"
)

var (
	_ ExecContext = &standardExecContext{}
)

type ExecContext interface {
	GetExecPayload() internaldto.ExecPayload
	GetResource() *openapistackql.Resource
}

type standardExecContext struct {
	execPayload internaldto.ExecPayload
	resource    *openapistackql.Resource
}

func (ec *standardExecContext) GetExecPayload() internaldto.ExecPayload {
	return ec.execPayload
}

func (ec *standardExecContext) GetResource() *openapistackql.Resource {
	return ec.resource
}

func NewExecContext(payload internaldto.ExecPayload, rsc *openapistackql.Resource) ExecContext {
	return &standardExecContext{
		execPayload: payload,
		resource:    rsc,
	}
}
