package embed

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// Binary describes a stackql server binary carried inside the calling
// program, typically via go:embed. The fetch tool
// (cmd/stackql-mcp-fetch) downloads the signed release bundle, verifies it
// against the published sha256 pin, and generates a file that returns a
// populated Binary.
type Binary struct {
	// Data is the raw server binary for the target platform.
	Data []byte
	// Version is the stackql release version, without the leading v
	// (for example "0.10.601"). It namespaces the shared cache.
	Version string
	// PlatformKey is one of the packaging platform keys
	// (linux-x64, linux-arm64, windows-x64, darwin-universal).
	PlatformKey string
	// SHA256 is the lowercase hex sha256 of Data, recorded at generate
	// time from the pin-verified bundle. Extraction re-verifies it.
	SHA256 string
}

func (b Binary) validate() error {
	if len(b.Data) == 0 {
		return fmt.Errorf("stackql mcp: Binary.Data is empty (run go generate to fetch the server binary)")
	}
	if b.Version == "" {
		return fmt.Errorf("stackql mcp: Binary.Version is required")
	}
	switch b.PlatformKey {
	case PlatformLinuxX64, PlatformLinuxARM64, PlatformWindowsX64, PlatformDarwinUniversal:
	default:
		return fmt.Errorf("stackql mcp: unknown platform key %q", b.PlatformKey)
	}
	if b.SHA256 == "" {
		return fmt.Errorf("stackql mcp: Binary.SHA256 is required")
	}
	sum := sha256.Sum256(b.Data)
	if got := hex.EncodeToString(sum[:]); got != b.SHA256 {
		return fmt.Errorf("stackql mcp: embedded binary sha256 mismatch: got %s, want %s", got, b.SHA256)
	}
	return nil
}
