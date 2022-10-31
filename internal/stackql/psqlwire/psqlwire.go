package psqlwire

import (
	"context"
	"database/sql/driver"
	"fmt"

	postgreswire "github.com/jeroenrinzema/psql-wire"
	"github.com/sirupsen/logrus"

	"github.com/jackc/pgtype"
	"github.com/jeroenrinzema/psql-wire/pkg/sqlbackend"
	"github.com/jeroenrinzema/psql-wire/pkg/sqldata"
	"github.com/lib/pq/oid"
)

func MakeSQLStream() (sqlbackend.ISQLBackend, error) {
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

func makePGServer(sqlBackend sqlbackend.ISQLBackend) (*postgreswire.Server, error) {
	return postgreswire.NewServer(postgreswire.SQLBackend(sqlBackend), postgreswire.Logger(logrus.StandardLogger()))
}

func processRowElement(rowElement interface{}) interface{} {
	switch re := rowElement.(type) {
	case driver.Valuer:
		v, _ := re.Value()
		return v
	default:
		return re
	}
}

func ExtractRowElement(column sqldata.ISQLColumn, src interface{}, ci *pgtype.ConnInfo) ([]byte, error) {
	typed, has := ci.DataTypeForOID(column.GetObjectID())
	if !has {
		return nil, fmt.Errorf("unknown data type: %T", column)
	}

	processedElement := processRowElement(src)
	err := typed.Value.Set(processedElement)
	if err != nil {
		return nil, err
	}

	fc, err := getFormatCode(column.GetFormat())
	if err != nil {
		return nil, err
	}
	encoder := fc.Encoder(typed)
	bb, err := encoder(ci, nil)
	if err != nil {
		return nil, err
	}
	return bb, nil
}

func getFormatCode(fc string) (postgreswire.FormatCode, error) {
	switch fc {
	case "TextFormat":
		return postgreswire.TextFormat, nil
	case "":
		return postgreswire.BinaryFormat, nil
	default:
		return 3, fmt.Errorf("cannot find format code for '%s'", fc)
	}
}
