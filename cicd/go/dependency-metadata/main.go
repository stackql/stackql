// Command dependency-metadata generates offline attribution for the StackQL binary.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"sort"
	"strings"

	"github.com/google/licensecheck"
	"github.com/stackql/stackql/internal/stackql/dependencies"
)

type module struct {
	Path    string  `json:"Path"`
	Version string  `json:"Version"`
	Dir     string  `json:"Dir"`
	Sum     string  `json:"Sum"`
	Replace *module `json:"Replace"`
}

func main() {
	if err := generate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate() error {
	targets := flag.String("targets", "", "comma-separated GOOS/GOARCH targets; defaults to go env")
	flag.Parse()
	root, err := exec.Command("go", "env", "GOMOD").Output()
	if err != nil {
		return fmt.Errorf("locate Go module: %w", err)
	}
	directory := filepath.Dir(strings.TrimSpace(string(root)))
	if *targets == "" {
		target, targetErr := exec.Command("go", "env", "GOOS", "GOARCH").Output()
		if targetErr != nil {
			return fmt.Errorf("read target platform: %w", targetErr)
		}
		*targets = strings.Join(strings.Fields(string(target)), "/")
	}
	datasets := make(map[string]dependencies.Inventory)
	for _, target := range strings.Split(*targets, ",") {
		inventory, inventoryErr := collectTarget(directory, target)
		if inventoryErr != nil {
			return inventoryErr
		}
		datasets[target] = inventory
	}
	data, err := json.MarshalIndent(datasets, "", "  ")
	if err != nil {
		return fmt.Errorf("encode dependency inventory: %w", err)
	}
	return os.WriteFile(filepath.Join(directory, "internal", "stackql", "dependencies", "metadata.json"),
		append(data, '\n'), 0o600)
}

func collectTarget(directory, target string) (dependencies.Inventory, error) {
	var inventory dependencies.Inventory
	goos, goarch, ok := strings.Cut(target, "/")
	if !ok || goos == "" || goarch == "" {
		return inventory, fmt.Errorf("invalid build target %q", target)
	}
	cmd := exec.Command("go", "list", "-deps", "-json", "./stackql")
	cmd.Dir = directory
	cmd.Stderr = os.Stderr
	for _, variable := range os.Environ() {
		if !strings.HasPrefix(variable, "GOOS=") && !strings.HasPrefix(variable, "GOARCH=") &&
			!strings.HasPrefix(variable, "CGO_ENABLED=") {
			cmd.Env = append(cmd.Env, variable)
		}
	}
	cmd.Env = append(cmd.Env, "GOOS="+goos, "GOARCH="+goarch, "CGO_ENABLED=0")
	output, err := cmd.Output()
	if err != nil {
		return inventory, fmt.Errorf("list StackQL binary packages for %s: %w", target, err)
	}
	modules, err := decodeModules(output)
	if err != nil {
		return inventory, err
	}
	metadata := make([]dependencies.Metadata, 0, len(modules))
	for _, m := range modules {
		effective := m
		if m.Replace != nil {
			effective = m.Replace
		}
		attribution, attributionErr := describe(effective)
		if attributionErr != nil {
			return inventory, attributionErr
		}
		metadata = append(metadata, attribution)
		inventory.Modules = append(inventory.Modules, buildModule(m))
	}
	sort.Slice(metadata, func(i, j int) bool {
		if metadata[i].Name == metadata[j].Name {
			return metadata[i].Version < metadata[j].Version
		}
		return metadata[i].Name < metadata[j].Name
	})
	sort.Slice(inventory.Modules, func(i, j int) bool { return inventory.Modules[i].Path < inventory.Modules[j].Path })
	inventory.Metadata = metadata
	return inventory, nil
}

func decodeModules(output []byte) (map[string]*module, error) {
	decoder := json.NewDecoder(bytes.NewReader(output))
	modules := make(map[string]*module)
	for {
		var pkg struct {
			Module *module `json:"Module"`
		}
		if err := decoder.Decode(&pkg); errors.Is(err, io.EOF) {
			return modules, nil
		} else if err != nil {
			return nil, fmt.Errorf("decode package graph: %w", err)
		}
		if pkg.Module == nil || pkg.Module.Path == "github.com/stackql/stackql" {
			continue
		}
		effective := pkg.Module
		if effective.Replace != nil {
			effective = effective.Replace
		}
		modules[effective.Path+"@"+effective.Version] = pkg.Module
	}
}

func buildModule(m *module) *debug.Module {
	result := &debug.Module{Path: m.Path, Version: m.Version, Sum: m.Sum}
	if m.Replace != nil {
		result.Replace = buildModule(m.Replace)
	}
	return result
}

func describe(m *module) (dependencies.Metadata, error) {
	metadata := dependencies.Metadata{Name: m.Path, Version: m.Version, License: "UNKNOWN"}
	if m.Version == "" {
		return metadata, nil
	}
	metadata.SourceURL, metadata.Description = source(m.Path)
	files, err := os.ReadDir(m.Dir)
	if err != nil {
		return metadata, fmt.Errorf("read module %s: %w", m.Path, err)
	}
	licenses := make(map[string]bool)
	for _, file := range files {
		name := strings.ToUpper(file.Name())
		if file.IsDir() || !(strings.HasPrefix(name, "LICENSE") || strings.HasPrefix(name, "COPYING")) {
			continue
		}
		text, readErr := os.ReadFile(filepath.Join(m.Dir, file.Name()))
		if readErr != nil {
			return metadata, fmt.Errorf("read license for %s: %w", m.Path, readErr)
		}
		coverage := licensecheck.Scan(text)
		for _, match := range coverage.Match {
			if !match.IsURL {
				licenses[match.ID] = true
			}
		}
		if len(licenses) > 0 && metadata.LicenseURL == "" {
			metadata.LicenseURL = licenseURL(m, metadata.SourceURL, file.Name())
		}
	}
	// Multiple detected licences require an explicit expression; do not guess AND/OR.
	if len(licenses) == 1 {
		for license := range licenses {
			metadata.License = license
		}
	}
	return metadata, nil
}

func source(path string) (string, string) {
	parts := strings.Split(path, "/")
	if len(parts) >= 3 && parts[0] == "github.com" {
		return "https://" + strings.Join(parts[:3], "/"), description(path)
	}
	if strings.HasPrefix(path, "golang.org/x/") {
		return "https://go.googlesource.com/" + parts[2], description(path)
	}
	return "", description(path)
}

func description(path string) string {
	switch path {
	case "github.com/stackql/stackql-parser":
		return "StackQL SQL parser, derived from Vitess"
	case "github.com/stackql/any-sdk":
		return "StackQL provider SDK"
	case "github.com/stackql/readline":
		return "StackQL fork of chzyer/readline for interactive line editing"
	default:
		return ""
	}
}

func licenseURL(m *module, repository, filename string) string {
	if repository == "" {
		return ""
	}
	const commitLength = 12
	revision := strings.TrimSuffix(m.Version, "+incompatible")
	if index := strings.LastIndex(revision, "-"); index >= 0 && len(revision[index+1:]) == commitLength {
		revision = revision[index+1:]
	}
	if strings.HasPrefix(repository, "https://github.com/") {
		repoPath := strings.TrimPrefix(repository, "https://")
		suffix := strings.TrimPrefix(m.Path, repoPath)
		parts := strings.Split(strings.Trim(suffix, "/"), "/")
		if len(parts) > 0 && isMajorSuffix(parts[len(parts)-1]) {
			parts = parts[:len(parts)-1]
		}

		subdir := strings.Join(parts, "/")
		if subdir != "" {
			filename = subdir + "/" + filename
			if revision == m.Version {
				revision = subdir + "/" + revision
			}
		}
		return repository + "/blob/" + revision + "/" + filename
	}
	return repository + "/+/" + revision + "/" + filename
}

func isMajorSuffix(part string) bool {
	return len(part) > 1 && part[0] == 'v' && strings.Trim(part[1:], "0123456789") == ""
}
