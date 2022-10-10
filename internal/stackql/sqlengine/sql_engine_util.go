package sqlengine

import (
	"database/sql"
	"fmt"
	"strings"
)

func singleColRowsToString(rows *sql.Rows) (string, error) {
	var acc []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return "", fmt.Errorf("could not stringify sql rows: %v", err)
		}
		acc = append(acc, s)
	}
	return strings.Join(acc, " "), nil
}
