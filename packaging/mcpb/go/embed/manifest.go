package embed

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

// platformsJSON is packaging/mcpb/go/embed/platforms.json, rendered by
// packaging/mcpb/scripts/render-platforms.sh from the published
// .mcpb.sha256 release assets. It is the only pin source in this module and
// the same manifest every other StackQL wrapper (npm, PyPI, the other SDKs)
// ships. Render it before building: make go-manifest VERSION=X.Y.Z (from
// packaging/mcpb).
//
//go:embed platforms.json
var platformsJSON []byte

// Manifest is the parsed platforms.json.
type Manifest struct {
	// Version is the stackql release this module is version-locked to
	// (no leading v). The mirror repo is tagged v<Version>.
	Version string `json:"version"`
	// BaseURL is the front door bundles are downloaded from
	// (https://releases.stackql.io/stackql/<version>).
	BaseURL string `json:"baseUrl"`
	// Platforms maps platform key to bundle name and sha256 pin.
	Platforms map[string]Pin `json:"platforms"`
}

// Pin is one platform's published bundle name and sha256.
type Pin struct {
	Bundle string `json:"bundle"`
	SHA256 string `json:"sha256"`
}

var manifest = mustParseManifest(platformsJSON)

func mustParseManifest(raw []byte) Manifest {
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		panic(fmt.Sprintf("stackql mcp: embedded platforms.json is invalid: %v", err))
	}
	if m.Version == "" || m.BaseURL == "" || len(m.Platforms) == 0 {
		panic("stackql mcp: embedded platforms.json is missing version, baseUrl or platforms")
	}
	return m
}

// LoadManifest returns the embedded platforms.json.
func LoadManifest() Manifest { return manifest }

// DefaultVersion is the stackql release this module is version-locked to.
// It namespaces the shared cache and is the version cmd/stackql-mcp-fetch
// downloads.
var DefaultVersion = manifest.Version

// BundlePins maps platform key to the published sha256 of the release .mcpb
// bundle for DefaultVersion.
var BundlePins = func() map[string]string {
	pins := make(map[string]string, len(manifest.Platforms))
	for k, p := range manifest.Platforms {
		pins[k] = p.SHA256
	}
	return pins
}()

// PinFor returns the bundle name and sha256 pin for a platform key.
func PinFor(platformKey string) (Pin, error) {
	p, ok := manifest.Platforms[platformKey]
	if !ok {
		return Pin{}, fmt.Errorf("stackql mcp: no bundle pinned for platform %q", platformKey)
	}
	return p, nil
}

// BundleURL is the download URL for a platform's bundle: <baseUrl>/<bundle>.
func BundleURL(platformKey string) (string, error) {
	p, err := PinFor(platformKey)
	if err != nil {
		return "", err
	}
	return manifest.BaseURL + "/" + p.Bundle, nil
}

// UserAgent identifies this vector and version to the download proxy.
func UserAgent() string {
	return "stackql-mcp-server-go/" + manifest.Version
}
