// Package fetch downloads a stackql MCP release bundle, verifies it against
// the published sha256 pin, and extracts the server binary. It backs the
// cmd/stackql-mcp-fetch go generate tool and the integration tests.
package fetch

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/stackql/stackql-mcp-go/embed"
)

// ReleaseURLBase is the asset download root. Bundles are published as
// assets of the matching stackql/stackql release.
const ReleaseURLBase = "https://github.com/stackql/stackql/releases/download"

func bundleName(platformKey string) string {
	return fmt.Sprintf("stackql-mcp-%s.mcpb", platformKey)
}

func assetURL(version, name string) string {
	return fmt.Sprintf("%s/v%s/%s", ReleaseURLBase, version, name)
}

func get(client *http.Client, url string) ([]byte, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// BundlePin resolves the published sha256 pin for a bundle. Order:
// the in-module pin table (for embed.DefaultVersion), the consolidated
// platforms.json release asset if present, then the per-bundle .sha256
// asset.
func BundlePin(client *http.Client, version, platformKey string) (string, error) {
	if version == embed.DefaultVersion {
		if pin, ok := embed.BundlePins[platformKey]; ok {
			return pin, nil
		}
	}
	if pin, err := pinFromPlatformsJSON(client, version, platformKey); err == nil && pin != "" {
		return pin, nil
	}
	raw, err := get(client, assetURL(version, bundleName(platformKey)+".sha256"))
	if err != nil {
		return "", fmt.Errorf("fetching sha256 pin: %w", err)
	}
	fields := strings.Fields(string(raw))
	if len(fields) == 0 || len(fields[0]) != 64 {
		return "", fmt.Errorf("malformed sha256 asset for %s", bundleName(platformKey))
	}
	return strings.ToLower(fields[0]), nil
}

// pinFromPlatformsJSON tries the consolidated platforms.json asset, which
// the packaging repo plans to publish. Both a flat map and a "platforms"
// wrapper are accepted; missing asset or unknown shape is not an error,
// the caller falls back to the .sha256 asset.
func pinFromPlatformsJSON(client *http.Client, version, platformKey string) (string, error) {
	raw, err := get(client, assetURL(version, "platforms.json"))
	if err != nil {
		return "", err
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", err
	}
	if inner, ok := doc["platforms"]; ok {
		doc = map[string]json.RawMessage{}
		if err := json.Unmarshal(inner, &doc); err != nil {
			return "", err
		}
	}
	entry, ok := doc[platformKey]
	if !ok {
		return "", nil
	}
	var withSha struct {
		SHA256 string `json:"sha256"`
	}
	if err := json.Unmarshal(entry, &withSha); err == nil && withSha.SHA256 != "" {
		return strings.ToLower(withSha.SHA256), nil
	}
	var bare string
	if err := json.Unmarshal(entry, &bare); err == nil && len(bare) == 64 {
		return strings.ToLower(bare), nil
	}
	return "", nil
}

// Result is a verified, extracted server binary.
type Result struct {
	Binary    embed.Binary
	BundleSHA string // the published pin the bundle was verified against
}

// Bundle downloads the .mcpb bundle for version/platformKey, verifies it
// against the published pin, and extracts the server binary named by the
// bundle manifest's entry_point.
func Bundle(client *http.Client, version, platformKey string) (*Result, error) {
	if client == nil {
		client = http.DefaultClient
	}
	pin, err := BundlePin(client, version, platformKey)
	if err != nil {
		return nil, err
	}
	data, err := get(client, assetURL(version, bundleName(platformKey)))
	if err != nil {
		return nil, fmt.Errorf("downloading bundle: %w", err)
	}
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); got != pin {
		return nil, fmt.Errorf("bundle sha256 mismatch for %s: got %s, want %s", bundleName(platformKey), got, pin)
	}
	bin, err := ExtractBinary(data, version, platformKey)
	if err != nil {
		return nil, err
	}
	return &Result{Binary: *bin, BundleSHA: pin}, nil
}

// ExtractBinary pulls the server binary out of verified .mcpb bundle bytes.
func ExtractBinary(bundle []byte, version, platformKey string) (*embed.Binary, error) {
	zr, err := zip.NewReader(bytes.NewReader(bundle), int64(len(bundle)))
	if err != nil {
		return nil, fmt.Errorf("opening bundle zip: %w", err)
	}
	manifestFile, err := zr.Open("manifest.json")
	if err != nil {
		return nil, fmt.Errorf("bundle has no manifest.json: %w", err)
	}
	defer manifestFile.Close()
	var manifest struct {
		Server struct {
			EntryPoint string `json:"entry_point"`
		} `json:"server"`
	}
	if err := json.NewDecoder(manifestFile).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("parsing manifest.json: %w", err)
	}
	entry := manifest.Server.EntryPoint
	if entry == "" {
		return nil, fmt.Errorf("manifest.json has no server.entry_point")
	}
	binFile, err := zr.Open(entry)
	if err != nil {
		return nil, fmt.Errorf("bundle entry_point %q not found: %w", entry, err)
	}
	defer binFile.Close()
	data, err := io.ReadAll(binFile)
	if err != nil {
		return nil, fmt.Errorf("reading entry_point: %w", err)
	}
	sum := sha256.Sum256(data)
	return &embed.Binary{
		Data:        data,
		Version:     version,
		PlatformKey: platformKey,
		SHA256:      hex.EncodeToString(sum[:]),
	}, nil
}
