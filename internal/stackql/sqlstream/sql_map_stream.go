package sqlstream

import (
	"io"

	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/streaming"
)

type SimpleSQLMapStream struct {
	selectCtx *drm.PreparedStatementCtx
	drmCfg    drm.DRMConfig
	sqlEngine sqlengine.SQLEngine
	// No buffering just yet; let us revisit soon
	// store     []map[string]interface{}
}

func NewSimpleSQLMapStream(
	selectCtx *drm.PreparedStatementCtx,
	drmCfg drm.DRMConfig,
	sqlEngine sqlengine.SQLEngine,
) streaming.MapStream {
	return &SimpleSQLMapStream{
		selectCtx: selectCtx,
		drmCfg:    drmCfg,
		sqlEngine: sqlEngine,
	}
}

func (ss *SimpleSQLMapStream) iStackQLReader() {}

func (ss *SimpleSQLMapStream) iStackQLWriter() {}

func (ss *SimpleSQLMapStream) Write(input []map[string]interface{}) error {
	return nil
}

func (ss *SimpleSQLMapStream) SetTCCs(tcc *dto.TxnControlCounters) {
	ss.selectCtx.SetGCCtrlCtrs(tcc)
}

func (ss *SimpleSQLMapStream) Read() ([]map[string]interface{}, error) {
	var rv []map[string]interface{}
	nonControlColumns := ss.selectCtx.GetNonControlColumns()
	r, sqlErr := ss.drmCfg.QueryDML(
		ss.sqlEngine,
		drm.NewPreparedStatementParameterized(ss.selectCtx, nil, true),
	)
	if sqlErr != nil {
		return nil, sqlErr
	}
	i := 0
	var keyArr []string
	var ifArr []interface{}
	for i < len(nonControlColumns) {
		x := nonControlColumns[i]
		y := ss.drmCfg.GetGolangValue(x.GetType())
		ifArr = append(ifArr, y)
		keyArr = append(keyArr, x.Column.GetIdentifier())
		i++
	}
	if r != nil {
		i := 0
		for r.Next() {
			errScan := r.Scan(ifArr...)
			if errScan != nil {
				return nil, errScan
			}
			im := make(map[string]interface{})
			for ord, key := range keyArr {
				val := ifArr[ord]
				ev := ss.drmCfg.ExtractFromGolangValue(val)
				im[key] = ev
			}
			rv = append(rv, im)
			i++
		}
	}
	return rv, io.EOF
}
