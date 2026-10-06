// Package dependencies exposes the modules linked into the running executable.
package dependencies

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"regexp"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/stackql/stackql/internal/stackql/iqlutil"
)

//go:generate go run ../../../cicd/go/dependency-metadata
//go:embed metadata*.json
var metadataFS embed.FS //nolint:gochecknoglobals // embed target

// Metadata is build-time attribution, keyed by effective module and exact version.
type Metadata struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	License     string `json:"license"`
	Description string `json:"description,omitempty"`
	SourceURL   string `json:"source_url,omitempty"`
	LicenseURL  string `json:"license_url,omitempty"`
}

// Inventory is the static dataset generated for a target binary.
type Inventory struct {
	Modules  []*debug.Module `json:"modules"`
	Metadata []Metadata      `json:"metadata"`
}

// Columns returns the ordered SHOW DEPENDENCIES headers.
func Columns(extended bool) []string {
	columns := []string{"name", "version", "license", "description"}
	if extended {
		columns = append(columns, "source_url", "license_url", "purl", "checksum", "original_name", "original_version")
	}
	return columns
}

// Rows reconciles embedded attribution with the executable's actual modules.
func Rows(extended bool, pattern *string) (map[string]map[string]interface{}, error) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return nil, fmt.Errorf("SHOW DEPENDENCIES: executable Go build information is unavailable")
	}
	var datasets map[string]Inventory
	data, err := metadataFS.ReadFile("metadata.json")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("SHOW DEPENDENCIES: read embedded inventory: %w", err)
	}
	if err == nil {
		if decodeErr := json.Unmarshal(data, &datasets); decodeErr != nil {
			return nil, fmt.Errorf("SHOW DEPENDENCIES: invalid embedded attribution: %w", decodeErr)
		}
	}
	inventory := datasets[runtime.GOOS+"/"+runtime.GOARCH]
	// Bare go builds can have stale or absent enrichment; the linker inventory remains authoritative.
	if sameModules(info.Deps, inventory.Modules) {
		info = &debug.BuildInfo{Deps: inventory.Modules}
	}
	return rows(info, inventory.Metadata, extended, pattern)
}

func sameModules(actual, generated []*debug.Module) bool {
	if len(actual) != len(generated) {
		return false
	}
	identities := make(map[string]bool, len(actual))
	for _, m := range actual {
		identities[moduleIdentity(m)] = true
	}
	for _, m := range generated {
		if !identities[moduleIdentity(m)] {
			return false
		}
	}
	return true
}

func moduleIdentity(m *debug.Module) string {
	identity := m.Path + "@" + m.Version + ":" + m.Sum
	if m.Replace != nil {
		identity += "=>" + moduleIdentity(m.Replace)
	}
	return identity
}

func rows(
	info *debug.BuildInfo, metadata []Metadata, extended bool, pattern *string,
) (map[string]map[string]interface{}, error) {
	if info == nil {
		return nil, fmt.Errorf("SHOW DEPENDENCIES: executable Go build information is unavailable")
	}
	matcher := likeMatcher(pattern)
	attribution := make(map[string]Metadata, len(metadata))
	for _, m := range metadata {
		attribution[m.Name+"@"+m.Version] = m
	}
	inventory := make(map[string]map[string]interface{})
	for _, requested := range info.Deps {
		effective := requested
		if requested.Replace != nil {
			effective = requested.Replace
		}
		key := effective.Path + "@" + effective.Version
		m := attribution[key]
		row := dependencyRow(requested, effective, m, extended)
		if matches(matcher, effective.Path, originalName(requested), m.Description) {
			inventory[key] = row
		}
	}
	order := make([]string, 0, len(inventory))
	for key := range inventory {
		order = append(order, key)
	}
	sort.Slice(order, func(i, j int) bool {
		leftName, leftVersion, _ := strings.Cut(order[i], "@")
		rightName, rightVersion, _ := strings.Cut(order[j], "@")
		if leftName == rightName {
			return leftVersion < rightVersion
		}
		return leftName < rightName
	})
	result := make(map[string]map[string]interface{}, len(order))
	for i, key := range order {
		result[fmt.Sprintf("%09d", i)] = inventory[key]
	}
	return result, nil
}

func dependencyRow(requested, effective *debug.Module, m Metadata, extended bool) map[string]interface{} {
	license := m.License
	if license == "" {
		license = "UNKNOWN"
	}
	row := map[string]interface{}{
		"name": effective.Path, "version": optional(effective.Version), "license": license,
		"description": optional(m.Description),
	}
	if extended {
		row["source_url"] = optional(m.SourceURL)
		row["license_url"] = optional(m.LicenseURL)
		row["purl"] = packageURL(effective)
		row["checksum"] = optional(effective.Sum)
		row["original_name"] = optional(originalName(requested))
		row["original_version"] = nil
		if requested.Replace != nil {
			row["original_version"] = optional(requested.Version)
		}
	}
	return row
}

func originalName(module *debug.Module) string {
	if module.Replace != nil {
		return module.Path
	}
	return ""
}

func optional(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}

func packageURL(module *debug.Module) interface{} {
	if module.Version == "" || !strings.Contains(module.Path, ".") {
		return nil
	}
	parts := strings.Split(module.Path, "/")
	for i, part := range parts {
		parts[i] = escapePURL(part)
	}
	return "pkg:golang/" + strings.Join(parts, "/") + "@" + escapePURL(module.Version)
}

func escapePURL(value string) string {
	return strings.ReplaceAll(url.QueryEscape(value), "+", "%20")
}

// SQL string escaping is handled by the parser; backslashes here are literal.
func likeMatcher(pattern *string) *regexp.Regexp {
	if pattern == nil {
		return nil
	}
	expression := strings.ReplaceAll(iqlutil.TranslateLikeToRegexPattern(*pattern), "_", ".")
	return regexp.MustCompile("(?is)" + expression)
}

func matches(matcher *regexp.Regexp, fields ...string) bool {
	if matcher == nil {
		return true
	}
	for _, field := range fields {
		if field != "" && matcher.MatchString(field) {
			return true
		}
	}
	return false
}
