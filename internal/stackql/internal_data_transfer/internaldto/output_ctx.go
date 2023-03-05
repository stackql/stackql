package internaldto

import (
	"github.com/stackql/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql/internal/stackql/dto"
)

type OutputContext struct {
	RuntimeContext dto.RuntimeCtx
	Result         sqldata.ISQLResultStream
}
