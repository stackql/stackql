// Package sqlsplit splits SQL text into statements the way the execution
// layer does, so callers reason about the same statements that will run.
package sqlsplit

import (
	"strings"

	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

// Statements returns the trimmed, non-blank statements in a payload.
func Statements(sql string) []string {
	// Error ignored as in the execution layer, which runs whatever pieces come back.
	pieces, _ := sqlparser.SplitStatementToPieces(sql)
	rv := make([]string, 0, len(pieces))
	for _, piece := range pieces {
		if trimmed := strings.TrimSpace(piece); trimmed != "" {
			rv = append(rv, trimmed)
		}
	}
	return rv
}
