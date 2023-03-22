package psqlwire

import (
	"bytes"
	"database/sql/driver"
	"fmt"
	"strconv"

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

// TODO: remove this hack once correct type system comes in
func shimNumericElement(rowElement interface{}) interface{} {
	switch re := rowElement.(type) { //nolint:gocritic // acceptable
	case []byte:
		f, err := strconv.ParseFloat(string(re), 64)
		if err == nil {
			return f
		}
	}
	return rowElement
}

// TODO: remove this hack once correct type system comes in
// Acknowledgement: This is from the MIT-licensed:
//
//	https://github.com/jackc/pgx/blob/9ae852eb583d2dced83b1d2ffe1c8803dda2c92e/pgtype/numeric.go#L256
//
//nolint:gocritic // this is a hack
func shimNumericTextBytes(n *pgtype.Numeric) []byte {
	intStr := n.Int.String()

	buf := &bytes.Buffer{}

	if len(intStr) > 0 && intStr[:1] == "-" {
		intStr = intStr[1:]
		buf.WriteByte('-')
	}

	exp := int(n.Exp)
	if exp > 0 {
		buf.WriteString(intStr)
		for i := 0; i < exp; i++ {
			buf.WriteByte('0')
		}
	} else if exp < 0 {
		if len(intStr) <= -exp {
			buf.WriteString("0.")
			leadingZeros := -exp - len(intStr)
			for i := 0; i < leadingZeros; i++ {
				buf.WriteByte('0')
			}
			buf.WriteString(intStr)
		} else if len(intStr) > -exp {
			dpPos := len(intStr) + exp
			buf.WriteString(intStr[:dpPos])
			buf.WriteByte('.')
			buf.WriteString(intStr[dpPos:])
		}
	} else {
		buf.WriteString(intStr)
	}

	return buf.Bytes()
}

// end hack

func ExtractRowElement(column sqldata.ISQLColumn, src interface{}, ci *pgtype.ConnInfo) ([]byte, error) {
	typed, has := ci.DataTypeForOID(column.GetObjectID())
	if !has {
		return nil, fmt.Errorf("unknown data type: %T", column)
	}

	processedElement := processRowElement(src)
	// TODO: retire this hack once correct type system comes in
	if typed.Name == "numeric" {
		processedElement = shimNumericElement(src)
	}
	// end hack
	err := typed.Value.Set(processedElement)
	if err != nil {
		return nil, err
	}

	fc, err := getFormatCode(column.GetFormat())
	if err != nil {
		return nil, err
	}
	// TODO: retire this hack once correct type system comes in
	switch t := typed.Value.(type) { //nolint:gocritic // acceptable
	case *pgtype.Numeric:
		b := shimNumericTextBytes(t)
		return b, nil
	}
	// end hack
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
