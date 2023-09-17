package output_data_staging //nolint:revive,stylecheck // package name is helpful

import (
	"database/sql"

	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/sqlmachinery"
)

type Source interface {
	SourceSQLRows() (*sql.Rows, error)
}

func NewNaiveSource(
	querier sqlmachinery.Querier,
	stmtCtx drm.PreparedStatementParameterized,
	drmCfg drm.Config,
) Source {
	return &naiveSource{
		querier: querier,
		stmtCtx: stmtCtx,
		drmCfg:  drmCfg,
	}
}

type naiveSource struct {
	querier sqlmachinery.Querier
	stmtCtx drm.PreparedStatementParameterized
	drmCfg  drm.Config
}

func (st *naiveSource) SourceSQLRows() (*sql.Rows, error) {
	r, sqlErr := st.drmCfg.QueryDML(
		st.querier,
		st.stmtCtx,
	)
	return r, sqlErr
}
