package typing

import (
	"fmt"
	"strings"

	"github.com/lib/pq/oid"
	"github.com/stackql/stackql-parser/go/vt/sqlparser"
)

var (
	_ RelationalColumn = &standardRelationalColumn{}
)

type RelationalColumn interface {
	CanonicalSelectionString() string
	DelimitedSelectionString(string) string
	GetAlias() string
	GetDecorated() string
	GetName() string
	GetOID() (oid.Oid, bool)
	GetQualifier() string
	GetType() string
	GetWidth() int
	WithAlias(string) RelationalColumn
	WithDecorated(string) RelationalColumn
	WithUnquote(bool) RelationalColumn
	WithParserNode(sqlparser.SQLNode) RelationalColumn
	WithQualifier(string) RelationalColumn
	WithWidth(int) RelationalColumn
	WithOID(oid.Oid) RelationalColumn
}

func NewRelationalColumn(colName string, colType string) RelationalColumn {
	return &standardRelationalColumn{
		colType: colType,
		colName: colName,
	}
}

type standardRelationalColumn struct {
	alias         string
	colType       string
	colName       string
	decorated     string
	qualifier     string
	width         int
	oID           *oid.Oid
	sqlParserNode sqlparser.SQLNode
	unquote       bool
}

func (rc *standardRelationalColumn) WithUnquote(unquote bool) RelationalColumn {
	rc.unquote = unquote
	return rc
}

func (rc *standardRelationalColumn) CanonicalSelectionString() string {
	if rc.decorated != "" {
		// if !strings.ContainsAny(rc.decorated, " '`\t\n\"().") {
		// 	return fmt.Sprintf(`"%s" `, rc.decorated)
		// }
		return fmt.Sprintf("%s ", rc.decorated)
	}
	var colStringBuilder strings.Builder
	if rc.qualifier != "" {
		colStringBuilder.WriteString(fmt.Sprintf(`"%s"."%s" `, rc.qualifier, rc.colName))
	} else {
		colStringBuilder.WriteString(fmt.Sprintf(`"%s" `, rc.colName))
	}
	if rc.alias != "" {
		colStringBuilder.WriteString(fmt.Sprintf(` AS "%s"`, rc.alias))
	}
	return colStringBuilder.String()
}

//nolint:gocritic // acceptable
func (rc *standardRelationalColumn) DelimitedSelectionString(delim string) string {
	aliasDelim := delim
	if rc.unquote {
		delim = ""
	}
	switch node := rc.sqlParserNode.(type) {
	case *sqlparser.AliasedExpr:
		switch node.Expr.(type) {
		case *sqlparser.FuncExpr:
			delim = ""
		}
	}
	if rc.decorated != "" {
		// if !strings.ContainsAny(rc.decorated, " '`\t\n\"().") {
		// 	return fmt.Sprintf(`"%s" `, rc.decorated)
		// }
		if rc.decorated == rc.colName {
			return fmt.Sprintf("%s%s%s ", delim, rc.colName, delim)
		}
		if rc.decorated == rc.qualifier+"."+rc.colName {
			return fmt.Sprintf("%s%s%s.%s%s%s ", delim, rc.qualifier, delim, delim, rc.colName, delim)
		}
		return fmt.Sprintf("%s ", rc.decorated)
	}
	var colStringBuilder strings.Builder
	if rc.qualifier != "" {
		colStringBuilder.WriteString(fmt.Sprintf(`%s%s%s.%s%s%s `, delim, rc.qualifier, delim, delim, rc.colName, delim))
	} else {
		colStringBuilder.WriteString(fmt.Sprintf(`%s%s%s `, delim, rc.colName, delim))
	}

	if rc.alias != "" {
		colStringBuilder.WriteString(fmt.Sprintf(` AS %s%s%s`, aliasDelim, rc.alias, aliasDelim))
	}
	return colStringBuilder.String()
}

func (rc *standardRelationalColumn) GetName() string {
	return rc.colName
}

func (rc *standardRelationalColumn) GetQualifier() string {
	return rc.qualifier
}

func (rc *standardRelationalColumn) GetType() string {
	return rc.colType
}

func (rc *standardRelationalColumn) GetWidth() int {
	return rc.width
}

func (rc *standardRelationalColumn) GetAlias() string {
	return rc.alias
}

func (rc *standardRelationalColumn) GetDecorated() string {
	return rc.decorated
}

func (rc *standardRelationalColumn) WithDecorated(decorated string) RelationalColumn {
	rc.decorated = decorated
	return rc
}

func (rc *standardRelationalColumn) WithParserNode(sqlNode sqlparser.SQLNode) RelationalColumn {
	rc.sqlParserNode = sqlNode
	return rc
}

func (rc *standardRelationalColumn) WithQualifier(qualifier string) RelationalColumn {
	rc.qualifier = qualifier
	return rc
}

func (rc *standardRelationalColumn) WithAlias(alias string) RelationalColumn {
	rc.alias = alias
	return rc
}

func (rc *standardRelationalColumn) WithWidth(width int) RelationalColumn {
	rc.width = width
	return rc
}

func (rc *standardRelationalColumn) WithOID(oID oid.Oid) RelationalColumn {
	op := &oID
	rc.oID = op
	return rc
}

func (rc *standardRelationalColumn) GetOID() (oid.Oid, bool) {
	var defaultRv oid.Oid
	if rc.oID != nil {
		return *rc.oID, true
	}
	return defaultRv, false
}
