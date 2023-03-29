package output_data_staging //nolint:revive,stylecheck // package name is helpful

import (
	"database/sql"
	"fmt"
	"sort"
	"strconv"

	"github.com/lib/pq/oid"
	"github.com/stackql/psql-wire/pkg/sqldata"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/internal_data_transfer/internaldto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/sqlmachinery"
	"github.com/stackql/stackql/internal/stackql/streaming"
	"github.com/stackql/stackql/internal/stackql/util"
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

type PacketPreparator interface {
	PrepareOutputPacket() (dto.OutputPacket, error)
}

func NewNaivePacketPreparator(
	source Source,
	nonControlColumns []internaldto.ColumnMetadata,
	stream streaming.MapStream,
	drmCfg drm.Config,
) PacketPreparator {
	return &naivePacketPreparator{
		source:            source,
		nonControlColumns: nonControlColumns,
		stream:            stream,
		drmCfg:            drmCfg,
	}
}

type naivePacketPreparator struct {
	source            Source
	nonControlColumns []internaldto.ColumnMetadata
	stream            streaming.MapStream
	drmCfg            drm.Config
}

func (st *naivePacketPreparator) PrepareOutputPacket() (dto.OutputPacket, error) {
	//nolint:rowserrcheck // TODO: fix this
	r, err := st.source.SourceSQLRows()
	logging.GetLogger().Infoln(fmt.Sprintf("select result = %v, error = %v", r, err))
	if err != nil {
		return nil, err
	}
	rowDicts, rawRows := st.drmCfg.ExtractObjectFromSQLRows(r, st.nonControlColumns, st.stream)
	var cNames []string
	var colOIDs []oid.Oid
	for _, v := range st.nonControlColumns {
		cNames = append(cNames, v.GetIdentifier())
		colOIDs = append(colOIDs, v.GetColumnOID())
	}
	return dto.NewStandardOutputPacket(
		rowDicts,
		rawRows,
		cNames,
		colOIDs,
	), nil
}

type Outputter interface {
	OutputExecutorResult() internaldto.ExecutorOutput
}

func NewNaiveOutputter(packetPreparator PacketPreparator, nonControlColumns []internaldto.ColumnMetadata) Outputter {
	return &naiveOutputter{
		packetPreparator:  packetPreparator,
		nonControlColumns: nonControlColumns,
	}
}

type naiveOutputter struct {
	packetPreparator  PacketPreparator
	nonControlColumns []internaldto.ColumnMetadata
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
		),
	)

	if rv.GetSQLResult() == nil {
		var colz []string
		for _, col := range st.nonControlColumns {
			colz = append(colz, col.GetIdentifier())
		}
		rv.SetSQLResultFn(
			func() sqldata.ISQLResultStream { return util.GetHeaderOnlyResultStream(colz) },
		)
	}
	return rv
}
