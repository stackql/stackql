package internaldto

import (
	"time"

	"github.com/stackql/psql-wire/pkg/sqldata"
	dto "github.com/stackql/stackql/internal/stackql/runtimeconfig"
)

type OutputContext struct {
	RuntimeContext dto.RuntimeCtx
	Result         sqldata.ISQLResultStream
	// Query and StartTime describe the submitted statement; the otel writer
	// stamps them on every record.
	Query     string
	StartTime time.Time
}
