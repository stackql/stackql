package sqlstream

import (
	"fmt"
	"io"

	"github.com/stackql/any-sdk/pkg/logging"
	"github.com/stackql/any-sdk/pkg/streaming"
	"github.com/stackql/any-sdk/public/sqlengine"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/tableinsertioncontainer"
)

type SimpleSQLMapStream struct {
	selectCtx       drm.PreparedStatementCtx
	insertContainer tableinsertioncontainer.TableInsertionContainer
	drmCfg          drm.Config
	sqlEngine       sqlengine.SQLEngine
	// No buffering just yet; let us revisit soon
	staticParams map[string]interface{}
}

func NewSimpleSQLMapStream(
	selectCtx drm.PreparedStatementCtx,
	insertContainer tableinsertioncontainer.TableInsertionContainer,
	drmCfg drm.Config,
	sqlEngine sqlengine.SQLEngine,
	staticParams map[string]interface{},
) streaming.MapStream {
	return &SimpleSQLMapStream{
		selectCtx:       selectCtx,
		insertContainer: insertContainer,
		drmCfg:          drmCfg,
		sqlEngine:       sqlEngine,
		staticParams:    staticParams,
	}
}

func (ss *SimpleSQLMapStream) Write(_ []map[string]interface{}) error {
	return nil
}

func (ss *SimpleSQLMapStream) Read() ([]map[string]interface{}, error) {
	var rv []map[string]interface{}
	nonControlColumns := ss.selectCtx.GetNonControlColumns()
	r, sqlErr := ss.drmCfg.QueryDML(
		ss.sqlEngine,
		drm.NewPreparedStatementParameterized(ss.selectCtx, nil, true),
	)
	if sqlErr != nil {
		logging.GetLogger().Infof("sql map stream query error: %v for query: %s", sqlErr, ss.selectCtx.GetQuery())
		return nil, fmt.Errorf("sql map stream query error: %w", sqlErr)
	}
	if r != nil {
		defer r.Close()
	}
	i := 0
	var keyArr []string
	var ifArr []interface{}
	for i < len(nonControlColumns) {
		x := nonControlColumns[i]
		y := ss.drmCfg.GetGolangValue(x.GetType())
		ifArr = append(ifArr, y)
		keyArr = append(keyArr, x.GetIdentifier())
		i++
	}
	i = 0
	if r != nil {
		for r.Next() {
			errScan := r.Scan(ifArr...)
			if errScan != nil {
				return nil, errScan
			}
			im := make(map[string]interface{})
			for k, v := range ss.staticParams {
				im[k] = v
			}
			for ord, key := range keyArr {
				val := ifArr[ord]
				ev := ss.drmCfg.ExtractFromGolangValue(val)
				im[key] = ev
			}
			rv = append(rv, im)
			logging.GetLogger().Infof(
				"sql map stream query returning row '''%v''' for query: '''%s'''", im, ss.selectCtx.GetQuery())
			i++
		}
	}
	if i == 0 {
		logging.GetLogger().Infof("sql map stream query returned no rows for query: '''%s'''", ss.selectCtx.GetQuery())
	}
	return rv, io.EOF
}
