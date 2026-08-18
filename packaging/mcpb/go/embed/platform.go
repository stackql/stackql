package embed

import (
	"fmt"
	"runtime"
)

// Platform keys match the stackql-mcpb-packaging release asset names and the
// shared binary cache layout used by the npm and pypi wrappers.
const (
	PlatformLinuxX64        = "linux-x64"
	PlatformLinuxARM64      = "linux-arm64"
	PlatformWindowsX64      = "windows-x64"
	PlatformDarwinUniversal = "darwin-universal"
)

// PlatformKey returns the packaging platform key for the given GOOS/GOARCH
// pair. Both darwin architectures map to the universal binary.
func PlatformKey(goos, goarch string) (string, error) {
	switch goos {
	case "linux":
		switch goarch {
		case "amd64":
			return PlatformLinuxX64, nil
		case "arm64":
			return PlatformLinuxARM64, nil
		}
	case "windows":
		if goarch == "amd64" {
			return PlatformWindowsX64, nil
		}
	case "darwin":
		if goarch == "amd64" || goarch == "arm64" {
			return PlatformDarwinUniversal, nil
		}
	}
	return "", fmt.Errorf("stackql mcp: no published binary for %s/%s", goos, goarch)
}

// CurrentPlatformKey returns the platform key for the running program.
func CurrentPlatformKey() (string, error) {
	return PlatformKey(runtime.GOOS, runtime.GOARCH)
}

// exeName returns the name the extracted server binary uses inside the
// shared cache for the given platform key.
func exeName(platformKey string) string {
	if platformKey == PlatformWindowsX64 {
		return "stackql.exe"
	}
	return "stackql"
}
