package output_data_staging //nolint:revive,stylecheck // package name is helpful

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/stackql/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/typing"
	"github.com/stackql/stackql/internal/stackql/util"
)

type Outputter interface {
	OutputExecutorResult() internaldto.ExecutorOutput
}

func NewNaiveOutputter(
	packetPreparator PacketPreparator,
	nonControlColumns []typing.ColumnMetadata,
	typCfg typing.Config,
) Outputter {
	return &naiveOutputter{
		packetPreparator:  packetPreparator,
		nonControlColumns: nonControlColumns,
		typCfg:            typCfg,
	}
}

type naiveOutputter struct {
	packetPreparator  PacketPreparator
	nonControlColumns []typing.ColumnMetadata
	typCfg            typing.Config
}

func (st *naiveOutputter) OutputExecutorResult() internaldto.ExecutorOutput {
	pkt, err := st.packetPreparator.PrepareOutputPacket()
	if err != nil {
		return internaldto.NewErroneousExecutorOutput(fmt.Errorf("sql packet preparation error: %w", err))
	}
	rows := pkt.GetRows()
	rawRows := pkt.GetRawRows()
	cNames := pkt.GetColumnNames()
	colOIDs := pkt.GetColumnOIDs()

	rowSort := func(m map[string]map[string]interface{}) []string {
		var arr []int
		for k := range m {
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
	rv := util.PrepareResultSet(
		internaldto.NewPrepareResultSetPlusRawAndTypesDTO(
			nil,
			rows,
			cNames,
			colOIDs,
			rowSort,
			nil,
			nil,
			rawRows,
			st.typCfg,
		),
	)

	if rv.GetSQLResult() == nil {
		var colz []string
		for _, col := range st.nonControlColumns {
			colz = append(colz, col.GetIdentifier())
		}
		rv.SetSQLResultFn(
			func() sqldata.ISQLResultStream { return util.GetHeaderOnlyResultStream(colz, st.typCfg) },
		)
	}
	return rv
}
