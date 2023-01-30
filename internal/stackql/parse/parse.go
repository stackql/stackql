package parse

import (
	"fmt"

	"vitess.io/vitess/go/vt/sqlparser"
)

func specialiseParserError(err error, cmd string) error {
	if err != nil {
		return fmt.Errorf(
			`error:  You have an error in your stackql syntax; parser error: %s`,
			err.Error(),
		)
	}
	return err
}

func ParseQuery(cmd string) (sqlparser.Statement, error) {
	statement, err := sqlparser.Parse(cmd)
	return statement, specialiseParserError(err, cmd)
}
