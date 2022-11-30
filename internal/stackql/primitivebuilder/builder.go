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
	"github.com/stackql/stackql/internal/stackql/streaming"
	"github.com/stackql/stackql/internal/stackql/tableinsertioncontainer"
	"github.com/stackql/stackql/internal/stackql/util"

	"github.com/stackql/go-openapistackql/openapistackql"
)

type Builder interface {
	Build() error

	GetRoot() primitivegraph.PrimitiveNode

	GetTail() primitivegraph.PrimitiveNode
}

func prepareGolangResult(
	sqlEngine sqlengine.SQLEngine,
	errWriter io.Writer,
	stmtCtx drm.PreparedStatementParameterized,
	insertContainers []tableinsertioncontainer.TableInsertionContainer,
	nonControlColumns []drm.ColumnMetadata,
	drmCfg drm.DRMConfig,
	stream streaming.MapStream,
) dto.ExecutorOutput {
	r, sqlErr := drmCfg.QueryDML(
		sqlEngine,
		stmtCtx,
	)
	logging.GetLogger().Infoln(fmt.Sprintf("select result = %v, error = %v", r, sqlErr))
	if sqlErr != nil {
		errWriter.Write(
			[]byte(
				fmt.Sprintf("sql SELECT error = %s\n", sqlErr.Error()),
			),
		)
	}
	altKeys, rawRows := drmCfg.ExtractObjectFromSQLRows(r, nonControlColumns, stream)
	var cNames []string
	var cSchemas []*openapistackql.Schema
	for _, v := range nonControlColumns {
		cNames = append(cNames, v.GetColumn().GetIdentifier())
		cSchemas = append(cSchemas, v.GetColumn().GetRepresentativeSchema())
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
	rv := util.PrepareResultSet(dto.NewPrepareResultSetPlusRawAndTypesDTO(nil, altKeys, cNames, cSchemas, rowSort, nil, nil, rawRows))

	if rv.GetSQLResult() == nil {
		var colz []string
		for _, col := range nonControlColumns {
			colz = append(colz, col.GetIdentifier())
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
