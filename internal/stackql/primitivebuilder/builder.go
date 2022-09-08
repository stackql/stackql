package primitivebuilder

import (
	"fmt"
	"io"
	"sort"
	"strconv"

	"github.com/jeroenrinzema/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/primitivegraph"
	"github.com/stackql/stackql/internal/stackql/sqlengine"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/go-openapistackql/openapistackql"

	"vitess.io/vitess/go/sqltypes"
	querypb "vitess.io/vitess/go/vt/proto/query"
)

type Builder interface {
	Build() error

	GetRoot() primitivegraph.PrimitiveNode

	GetTail() primitivegraph.PrimitiveNode
}

func prepareGolangResult(sqlEngine sqlengine.SQLEngine, errWriter io.Writer, stmtCtx drm.PreparedStatementParameterized, nonControlColumns []drm.ColumnMetadata, drmCfg drm.DRMConfig) dto.ExecutorOutput {
	r, sqlErr := drmCfg.QueryDML(
		sqlEngine,
		stmtCtx,
	)
	logging.GetLogger().Infoln(fmt.Sprintf("select result = %v, error = %v", r, sqlErr))
	if sqlErr != nil {
		errWriter.Write(
			[]byte(
				fmt.Sprintf("sql error = %s\n", sqlErr.Error()),
			),
		)
	}
	altKeys := make(map[string]map[string]interface{})
	rawRows := make(map[int]map[int]interface{})
	var ks []int
	i := 0
	var keyArr []string
	var ifArr []interface{}
	for i < len(nonControlColumns) {
		x := nonControlColumns[i]
		y := drmCfg.GetGolangValue(x.GetType())
		ifArr = append(ifArr, y)
		keyArr = append(keyArr, x.Column.GetIdentifier())
		i++
	}
	if r != nil {
		i := 0
		for r.Next() {
			errScan := r.Scan(ifArr...)
			if errScan != nil {
				logging.GetLogger().Infoln(fmt.Sprintf("%v", errScan))
			}
			for ord, val := range ifArr {
				logging.GetLogger().Infoln(fmt.Sprintf("col #%d '%s':  %v  type: %T", ord, nonControlColumns[ord].GetName(), val, val))
			}
			im := make(map[string]interface{})
			imRaw := make(map[int]interface{})
			for ord, key := range keyArr {
				val := ifArr[ord]
				ev := drmCfg.ExtractFromGolangValue(val)
				im[key] = ev
				imRaw[ord] = ev
			}
			altKeys[strconv.Itoa(i)] = im
			rawRows[i] = imRaw
			ks = append(ks, i)
			i++
		}

		for ord := range ks {
			val := altKeys[strconv.Itoa(ord)]
			logging.GetLogger().Infoln(fmt.Sprintf("row #%d:  %v  type: %T", ord, val, val))
		}
	}
	var cNames []string
	for _, v := range nonControlColumns {
		cNames = append(cNames, v.Column.GetIdentifier())
	}
	rowSort := func(m map[string]map[string]interface{}) []string {
		var arr []int
		for k, _ := range m {
			ord, _ := strconv.Atoi(k)
			arr = append(arr, ord)
		}
		sort.Ints(arr)
		var rv []string
		for _, v := range arr {
			rv = append(rv, strconv.Itoa(v))
		}
		return rv
	}
	rv := util.PrepareResultSet(dto.NewPrepareResultSetPlusRawDTO(nil, altKeys, cNames, rowSort, nil, nil, rawRows))
	if rv.GetSQLResult() == nil {

		resVal := &sqltypes.Result{
			Fields: make([]*querypb.Field, len(nonControlColumns)),
		}

		var colz []string
		for _, col := range nonControlColumns {
			colz = append(colz, col.GetIdentifier())
		}

		for f := range resVal.Fields {
			resVal.Fields[f] = &querypb.Field{
				Name: cNames[f],
			}
		}
		rv.GetSQLResult = func() sqldata.ISQLResultStream { return util.GetHeaderOnlyResultStream(colz) }
	}
	return rv
}

func castItemsArray(iArr interface{}) ([]map[string]interface{}, error) {
	switch iArr := iArr.(type) {
	case []map[string]interface{}:
		return iArr, nil
	case []interface{}:
		var rv []map[string]interface{}
		for i := range iArr {
			item, ok := iArr[i].(map[string]interface{})
			if !ok {
				if iArr[i] != nil {
					item = map[string]interface{}{openapistackql.AnonymousColumnName: iArr[i]}
				} else {
					item = nil
				}
			}
			rv = append(rv, item)
		}
		return rv, nil
	default:
		return nil, fmt.Errorf("cannot accept items array of type = '%T'", iArr)
	}
}
