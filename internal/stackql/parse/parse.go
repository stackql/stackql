package parse

import (
	"errors"
	"fmt"
	"strings"

	"vitess.io/vitess/go/vt/sqlparser"
)

func specialiseParserError(err error, cmd string) error {
	if err != nil {
		if strings.Count(cmd, ".") > 1 {
			return errors.New(
				fmt.Sprintf(
					`Error: Three part object identifiers are not supported, try a USE statement followed by a query with a two part object identifier; Parser Error: %s`,
					err.Error(),
				),
			)
		} else {
			return errors.New(
				fmt.Sprintf(
					`Error:  You have an error in your IQL syntax; Parser Error: %s`,
					err.Error(),
				),
			)
		}
	}
	return err
}

func ParseQuery(cmd string) (sqlparser.Statement, error) {
	statement, err := sqlparser.Parse(cmd)
	return statement, specialiseParserError(err, cmd)
}
