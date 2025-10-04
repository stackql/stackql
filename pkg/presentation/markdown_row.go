package presentation

import (
	"fmt"
	"sort"
	"strings"
)

type MarkdownRow interface {
	Headers() []string
	Values() []any
	RowString() string
	HeaderString() string
	SeparatorString() string
}

func NewMarkdownRowFromMap(row map[string]interface{}) MarkdownRow {
	var columns []string
	var values []any
	for k := range row {
		columns = append(columns, k)
	}
	sort.Strings(columns)
	for _, k := range columns {
		v := row[k]
		values = append(values, v)
	}
	return &simpleMardownRow{
		columns: columns,
		values:  values,
	}
}

type simpleMardownRow struct {
	columns []string
	values  []any
}

func (s *simpleMardownRow) Headers() []string {
	return s.columns
}

func (s *simpleMardownRow) Values() []any {
	return s.values
}

func (s *simpleMardownRow) RowString() string {
	var sb strings.Builder
	for i := 0; i < len(s.columns); i++ {
		sb.WriteString(fmt.Sprintf("| %v ", s.values[i]))
	}
	sb.WriteString("|")
	return sb.String()
}

func (s *simpleMardownRow) SeparatorString() string {
	var sb strings.Builder
	for range s.columns {
		sb.WriteString("|---")
	}
	sb.WriteString("|")
	return sb.String()
}

func (s *simpleMardownRow) HeaderString() string {
	var sb strings.Builder
	for i := 0; i < len(s.columns); i++ {
		sb.WriteString(fmt.Sprintf("| %s ", s.columns[i]))
	}
	sb.WriteString("|")
	return sb.String()
}
