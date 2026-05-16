// Package buildinfo holds build-time identifiers populated at link time.
// Keeping these in a leaf package lets both the CLI command layer and the
// runtime primitives reference them without creating an import cycle.
//
// The public surface is intentionally minimal: a BuildInfo interface, a
// constructor, and a process-wide accessor. The implementation is a private
// struct that is constructed exactly once, at program start, from cmd/root.go.
package buildinfo

import (
	"fmt"
	"sync"
)

// BuildInfo provides read-only access to build-time identifiers.
type BuildInfo interface {
	GetMajorVersion() string
	GetMinorVersion() string
	GetPatchVersion() string
	GetCommitSHA() string
	GetShortCommitSHA() string
	GetDate() string
	GetPlatform() string
	GetSemVersion() string
}

type buildInfo struct {
	majorVersion   string
	minorVersion   string
	patchVersion   string
	commitSHA      string
	shortCommitSHA string
	date           string
	platform       string
	semVersion     string
}

// NewBuildInfo constructs an immutable BuildInfo from the given identifiers.
// SemVersion is derived from the major/minor/patch trio.
func NewBuildInfo(major, minor, patch, commitSHA, shortCommitSHA, date, platform string) BuildInfo {
	return &buildInfo{
		majorVersion:   major,
		minorVersion:   minor,
		patchVersion:   patch,
		commitSHA:      commitSHA,
		shortCommitSHA: shortCommitSHA,
		date:           date,
		platform:       platform,
		semVersion:     fmt.Sprintf("%s.%s.%s", major, minor, patch),
	}
}

func (b *buildInfo) GetMajorVersion() string   { return b.majorVersion }
func (b *buildInfo) GetMinorVersion() string   { return b.minorVersion }
func (b *buildInfo) GetPatchVersion() string   { return b.patchVersion }
func (b *buildInfo) GetCommitSHA() string      { return b.commitSHA }
func (b *buildInfo) GetShortCommitSHA() string { return b.shortCommitSHA }
func (b *buildInfo) GetDate() string           { return b.date }
func (b *buildInfo) GetPlatform() string       { return b.platform }
func (b *buildInfo) GetSemVersion() string     { return b.semVersion }

//nolint:gochecknoglobals // process-wide singleton; written exactly once via Init.
var (
	singleton = NewBuildInfo("", "", "", "", "", "", "")
	once      sync.Once
)

// Init publishes the process-wide BuildInfo. It is intended to be called once
// from cmd/root.go's init(), after the -ldflags-populated string variables in
// the cmd package have been observed. Subsequent calls are no-ops.
func Init(major, minor, patch, commitSHA, shortCommitSHA, date, platform string) {
	once.Do(func() {
		singleton = NewBuildInfo(major, minor, patch, commitSHA, shortCommitSHA, date, platform)
	})
}

// Get returns the process-wide BuildInfo. Before Init is called it returns an
// instance with empty strings.
func Get() BuildInfo {
	return singleton
}
