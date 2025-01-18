package internaldto

import (
	"github.com/stackql/any-sdk/pkg/dto"
	"github.com/stackql/psql-wire/pkg/sqldata"
)

type OutputContext struct {
	RuntimeContext dto.RuntimeCtx
	Result         sqldata.ISQLResultStream
}
