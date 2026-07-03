// Package render produces text renderings of tool results for MCP clients.
package render

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
)

const noResults = "**no results**"

// Text content render formats (issue #669).  Markdown is the historical
// default; JSON renders the same DTO carried in structuredContent as compact
// JSON so text-only MCP consumers get a machine-readable payload.
const (
	FormatMarkdown = "markdown"
	FormatJSON     = "json"
)

// IsLegalFormat reports whether the supplied format name is supported.
// The empty string is legal and means "use the default".
func IsLegalFormat(format string) bool {
	switch format {
	case "", FormatMarkdown, FormatJSON:
		return true
	default:
		return false
	}
}

// JSONValue renders any value as compact JSON for text content.  Row sets
// pass through UnwrapRows first so database/sql nullable wrappers serialise
// as their scalar payloads rather than {Valid, ...} envelopes.
func JSONValue(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf(`{"error":%q}`, fmt.Sprintf("failed to marshal: %v", err))
	}
	return string(b)
}

// UnwrapRows returns a copy of `rows` with database/sql nullable wrappers
// collapsed to their scalar payloads (see unwrap).
func UnwrapRows(rows []map[string]any) []map[string]any {
	clean := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		cleanRow := make(map[string]any, len(row))
		for k, v := range row {
			cleanRow[k] = unwrap(v)
		}
		clean = append(clean, cleanRow)
	}
	return clean
}

// unwrap normalises database/sql nullable wrappers (sql.NullString, NullBool,
// NullInt64, NullInt32, NullFloat64, NullByte, NullTime, the generic sql.Null[T])
// down to their scalar payload before formatting.  Anything else is returned
// unchanged.  Invalid wrappers collapse to "" so cells render empty rather than
// as the typed zero value.
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
	return firstNonValidField(rv)
}

// firstNonValidField returns the first exported struct field whose name is not
// "Valid".  Split out of unwrap to keep gocognit complexity low.
func firstNonValidField(rv reflect.Value) any {
	for i := 0; i < rv.NumField(); i++ {
		if rv.Type().Field(i).Name != "Valid" {
			return rv.Field(i).Interface()
		}
	}
	return rv.Interface()
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
