package parserutil

import (
	"github.com/stackql/any-sdk/public/formulation"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

type ColumnHandle struct {
	Alias           string
	Expr            sqlparser.Expr
	Name            string
	Qualifier       string
	DecoratedColumn string
	IsColumn        bool
	IsAggregateExpr bool
	Type            sqlparser.ValType
	Val             *sqlparser.SQLVal
}

// WithDisplayName re-keys a wire-spelled column reference to the tabulation
// display name the backend column carries (snake_case_aliases, any-sdk #131),
// keeping the original spelling as the output alias. No-op when the names agree.
func (ch ColumnHandle) WithDisplayName(tabulation formulation.Tabulation) ColumnHandle {
	if tabulation == nil || !ch.IsColumn {
		return ch
	}
	for _, tc := range tabulation.GetColumns() {
		if tc.GetWireName() == ch.Name && tc.GetName() != ch.Name {
			if ch.Alias == "" {
				ch.Alias = ch.Name
			}
			ch.Name = tc.GetName()
			ch.DecoratedColumn = ""
			return ch
		}
	}
	return ch
}
