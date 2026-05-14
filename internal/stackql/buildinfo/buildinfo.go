// Package buildinfo holds build-time identifiers that are populated at link
// time. Keeping these in a leaf package lets both the CLI command layer and
// runtime primitives reference them without creating an import cycle.
package buildinfo

import "fmt"

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
