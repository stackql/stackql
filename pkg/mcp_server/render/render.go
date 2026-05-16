// Package render produces text renderings of tool results for MCP clients.
package render

import (
	"fmt"
	"sort"
	"strings"
)

const noResults = "**no results**"

// RenderTable renders a uniform multi-row result set as a markdown table.
// Column order is stable: the union of keys across all rows, sorted alphabetically.
func RenderTable(rows []map[string]any) string {
	if len(rows) == 0 {
		return noResults
	}
	columns := unionColumns(rows)
	var sb strings.Builder
	sb.WriteString(headerRow(columns))
	sb.WriteString("\n")
	sb.WriteString(separatorRow(len(columns)))
	sb.WriteString("\n")
	for _, row := range rows {
		sb.WriteString(dataRow(columns, row))
		sb.WriteString("\n")
	}
	return sb.String()
}

// RenderKV renders sparse / single-record / mixed-shape results as labelled records.
func RenderKV(title string, records []map[string]any) string {
	var sb strings.Builder
	if title != "" {
		sb.WriteString("# ")
		sb.WriteString(title)
		sb.WriteString("\n\n")
	}
	if len(records) == 0 {
		sb.WriteString(noResults)
		return sb.String()
	}
	for i, rec := range records {
		sb.WriteString(fmt.Sprintf("## Record %d\n\n", i+1))
		keys := sortedKeys(rec)
		for _, k := range keys {
			sb.WriteString(fmt.Sprintf("%s: %v\n", k, rec[k]))
		}
		if i < len(records)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

func unionColumns(rows []map[string]any) []string {
	seen := map[string]struct{}{}
	for _, r := range rows {
		for k := range r {
			seen[k] = struct{}{}
		}
	}
	cols := make([]string, 0, len(seen))
	for k := range seen {
		cols = append(cols, k)
	}
	sort.Strings(cols)
	return cols
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func headerRow(columns []string) string {
	var sb strings.Builder
	for _, c := range columns {
		sb.WriteString(fmt.Sprintf("| %s ", c))
	}
	sb.WriteString("|")
	return sb.String()
}

func separatorRow(n int) string {
	var sb strings.Builder
	for i := 0; i < n; i++ {
		sb.WriteString("|---")
	}
	sb.WriteString("|")
	return sb.String()
}

func dataRow(columns []string, row map[string]any) string {
	var sb strings.Builder
	for _, c := range columns {
		v, ok := row[c]
		if !ok {
			sb.WriteString("|  ")
			continue
		}
		sb.WriteString(fmt.Sprintf("| %v ", v))
	}
	sb.WriteString("|")
	return sb.String()
}
