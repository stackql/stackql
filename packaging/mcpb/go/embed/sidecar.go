package embed

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// Env overrides shared with the npm/PyPI wrappers and the other SDKs.
const (
	// EnvBin names an existing stackql binary to run as-is (skips acquisition).
	EnvBin = "STACKQL_MCP_BIN"
	// EnvBundle names a local .mcpb to extract instead of downloading. No pin
	// check: the override is explicit operator intent (CI, custom builds).
	EnvBundle = "STACKQL_MCP_BUNDLE"
)

// ResolveBinary returns a runnable server binary path without an embedded
// Binary (the sidecar path). Order:
//
//  1. $STACKQL_MCP_BIN
//  2. $STACKQL_MCP_BUNDLE, extracted under <cache>/custom/<sha256[:16]>/
//  3. the shared cache <cache>/<version>/<platform-key>/stackql[.exe]
//  4. download from platforms.json baseUrl, verify the pin, extract
//
// If cacheDir is empty, DefaultCacheDir is used.
func ResolveBinary(cacheDir string) (string, error) {
	if path, ok, err := envOverride(cacheDir); ok || err != nil {
		return path, err
	}
	if cacheDir == "" {
		var err error
		if cacheDir, err = DefaultCacheDir(); err != nil {
			return "", err
		}
	}
	key, err := CurrentPlatformKey()
	if err != nil {
		return "", err
	}
	target := filepath.Join(cacheDir, manifest.Version, key, exeName(key))
	if _, err := os.Stat(target); err == nil {
		return target, nil
	}
	pin, err := PinFor(key)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(os.Stderr, "stackql-mcp: downloading %s (first run only) ...\n", pin.Bundle)
	res, err := FetchBundle(nil, key)
	if err != nil {
		return "", err
	}
	path, err := EnsureExtracted(cacheDir, res.Binary)
	if err != nil {
		return "", err
	}
	fmt.Fprintf(os.Stderr, "stackql-mcp: installed %s\n", path)
	return path, nil
}

// envOverride applies STACKQL_MCP_BIN / STACKQL_MCP_BUNDLE. ok reports
// whether an override was set.
func envOverride(cacheDir string) (path string, ok bool, err error) {
	if bin := os.Getenv(EnvBin); bin != "" {
		if _, err := os.Stat(bin); err != nil {
			return "", true, fmt.Errorf("stackql mcp: %s points to %s: %w", EnvBin, bin, err)
		}
		return bin, true, nil
	}
	bundlePath := os.Getenv(EnvBundle)
	if bundlePath == "" {
		return "", false, nil
	}
	data, err := os.ReadFile(bundlePath)
	if err != nil {
		return "", true, fmt.Errorf("stackql mcp: reading %s: %w", EnvBundle, err)
	}
	key, err := CurrentPlatformKey()
	if err != nil {
		return "", true, err
	}
	bin, err := ExtractBinary(data, key)
	if err != nil {
		return "", true, err
	}
	if cacheDir == "" {
		if cacheDir, err = DefaultCacheDir(); err != nil {
			return "", true, err
		}
	}
	sum := sha256.Sum256(data)
	slot := filepath.Join(cacheDir, "custom", hex.EncodeToString(sum[:])[:16], exeName(key))
	path, err = installBinary(slot, bin.Data, bin.SHA256)
	return path, true, err
}
