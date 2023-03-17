package psqlwire

import (
	"database/sql/driver"
	"fmt"

	postgreswire "github.com/stackql/psql-wire"

	"github.com/jackc/pgtype"
	"github.com/stackql/psql-wire/pkg/sqldata"
)

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
		return -1, fmt.Errorf("cannot find format code for '%s'", fc)
	}
}
