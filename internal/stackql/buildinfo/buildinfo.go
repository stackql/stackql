// Package buildinfo holds build-time identifiers that are populated at link
// time. Keeping these in a leaf package lets both the CLI command layer and
// runtime primitives reference them without creating an import cycle.
package buildinfo

import "fmt"

// BuildInfo provides access to build-time information.
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

// buildInfo is a private struct that implements BuildInfo.
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

// NewBuildInfo creates a new BuildInfo with the provided values.
func NewBuildInfo(major, minor, patch, commitSHA, shortCommitSHA, date, platform string) BuildInfo {
	semVer := fmt.Sprintf("%s.%s.%s", major, minor, patch)
	return &buildInfo{
		majorVersion:   major,
		minorVersion:   minor,
		patchVersion:   patch,
		commitSHA:      commitSHA,
		shortCommitSHA: shortCommitSHA,
		date:           date,
		platform:       platform,
		semVersion:     semVer,
	}
}

// Implement BuildInfo interface.
func (b *buildInfo) GetMajorVersion() string   { return b.majorVersion }
func (b *buildInfo) GetMinorVersion() string   { return b.minorVersion }
func (b *buildInfo) GetPatchVersion() string   { return b.patchVersion }
func (b *buildInfo) GetCommitSHA() string      { return b.commitSHA }
func (b *buildInfo) GetShortCommitSHA() string { return b.shortCommitSHA }
func (b *buildInfo) GetDate() string           { return b.date }
func (b *buildInfo) GetPlatform() string       { return b.platform }
func (b *buildInfo) GetSemVersion() string     { return b.semVersion }

// NewBuildInfoFromLegacy creates a BuildInfo instance from legacy global variables.
func NewBuildInfoFromLegacy() BuildInfo {
	return NewBuildInfo(
		BuildMajorVersion,
		BuildMinorVersion,
		BuildPatchVersion,
		BuildCommitSHA,
		BuildShortCommitSHA,
		BuildDate,
		BuildPlatform,
	)
}

// Legacy global variables for backward compatibility.
// These will be populated by the init() function in cmd/root.go.
//
//nolint:revive,gochecknoglobals // populated by -ldflags at build time
var (
	BuildMajorVersion   string = ""
	BuildMinorVersion   string = ""
	BuildPatchVersion   string = ""
	BuildCommitSHA      string = ""
	BuildShortCommitSHA string = ""
	BuildDate           string = ""
	BuildPlatform       string = ""
)

//nolint:gochecknoglobals // derived from the build-time vars above
var SemVersion = fmt.Sprintf("%s.%s.%s", BuildMajorVersion, BuildMinorVersion, BuildPatchVersion)
