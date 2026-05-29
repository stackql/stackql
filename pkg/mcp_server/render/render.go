// Package render produces text renderings of tool results for MCP clients.
package render

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

const noResults = "**no results**"

// unwrap returns the underlying scalar for sql.Null* wrappers (and pointers
// to them).  Genuinely nil inputs are preserved.  An invalid (Valid==false)
// wrapper collapses to the empty string, matching the substitution the
// reverse-proxy backend applies during column-type scanning.
//
// Reflection-based to remain agnostic to which Null* shape arrives (string,
// bool, int64, int32, float64, time, byte, or the Go 1.22+ generic
// sql.Null[T]): any struct with a `Valid bool` field plus exactly one other
// exported field is treated as a nullable wrapper.
func unwrap(v any) any {
	if v == nil {
		return nil
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return nil
		}
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Struct {
		return v
	}
	validField := rv.FieldByName("Valid")
	if !validField.IsValid() || validField.Kind() != reflect.Bool {
		return v
	}
	if !validField.Bool() {
		return ""
	}
	for i := 0; i < rv.NumField(); i++ {
		if rv.Type().Field(i).Name != "Valid" {
			return rv.Field(i).Interface()
		}
	}
	return v
}

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
			sb.WriteString(fmt.Sprintf("%s: %v\n", k, unwrap(rec[k])))
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
		sb.WriteString(fmt.Sprintf("| %v ", unwrap(v)))
	}
	sb.WriteString("|")
	return sb.String()
}
