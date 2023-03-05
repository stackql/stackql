package psqlwire_test

import (
	"context"
	"testing"

	"github.com/lib/pq/oid"
	"github.com/stackql/psql-wire/pkg/sqlbackend"
	"github.com/stackql/psql-wire/pkg/sqldata"
)

func TestMockedStream(t *testing.T) {
	//
}

func mockSQLStream() (sqlbackend.ISQLBackend, error) {
	cols := []sqldata.ISQLColumn{ //nolint:errcheck
		sqldata.NewSQLColumn(
			sqldata.NewSQLTable(0, ""),
			"name",
			0,
			uint32(oid.T_text),
			256,
			0,
			"TextFormat",
		),
		sqldata.NewSQLColumn(
			sqldata.NewSQLTable(0, ""),
			"member",
			0,
			uint32(oid.T_bool),
			1,
			0,
			"TextFormat",
		),
		sqldata.NewSQLColumn(
			sqldata.NewSQLTable(0, ""),
			"age",
			0,
			uint32(oid.T_int4),
			1,
			0,
			"TextFormat",
		),
	}

	rows := []sqldata.ISQLRow{
		sqldata.NewSQLRow([]interface{}{"John", true, 28}),   //nolint:errcheck
		sqldata.NewSQLRow([]interface{}{"Marry", false, 21}), //nolint:errcheck
	}

	sr := sqldata.NewSQLResult(cols, 0, 0, rows)

	sb := sqldata.NewSimpleSQLResultStream(sr)

	qcb := func(context.Context, string) (sqldata.ISQLResultStream, error) {
		return sb, nil
	}

	sqlBackend := sqlbackend.NewSimpleSQLBackend(qcb)

	return sqlBackend, nil
}
