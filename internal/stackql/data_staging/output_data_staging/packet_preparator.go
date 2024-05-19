package output_data_staging //nolint:revive,stylecheck // package name is helpful

import (
	"fmt"

	"github.com/lib/pq/oid"
	"github.com/stackql/stackql/internal/stackql/drm"
	"github.com/stackql/stackql/internal/stackql/dto"
	"github.com/stackql/stackql/internal/stackql/logging"
	"github.com/stackql/stackql/internal/stackql/streaming"
	"github.com/stackql/stackql/internal/stackql/typing"
)

type PacketPreparator interface {
	PrepareOutputPacket() (dto.OutputPacket, error)
}

func NewNaivePacketPreparator(
	source Source,
	nonControlColumns []typing.ColumnMetadata,
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
	nonControlColumns []typing.ColumnMetadata
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
		nil,
	), nil
}
