package embed

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// FetchResult is a verified, extracted server binary.
type FetchResult struct {
	Binary    Binary
	BundleSHA string // the published pin the bundle was verified against
}

// FetchBundle downloads the .mcpb bundle for platformKey at the module's
// pinned version from the manifest's baseUrl (the releases.stackql.io
// front door), verifies it against the manifest's sha256 pin, and extracts
// the server binary named by the bundle manifest's entry_point. There is no
// version override: the module is version-locked to DefaultVersion.
func FetchBundle(client *http.Client, platformKey string) (*FetchResult, error) {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Minute}
	}
	pin, err := PinFor(platformKey)
	if err != nil {
		return nil, err
	}
	url := manifest.BaseURL + "/" + pin.Bundle
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent())
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("stackql mcp: downloading %s: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("stackql mcp: GET %s: %s", url, resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("stackql mcp: downloading %s: %w", url, err)
	}
	sum := sha256.Sum256(data)
	if got := hex.EncodeToString(sum[:]); got != pin.SHA256 {
		return nil, fmt.Errorf("stackql mcp: bundle sha256 mismatch for %s: got %s, want %s", pin.Bundle, got, pin.SHA256)
	}
	bin, err := ExtractBinary(data, platformKey)
	if err != nil {
		return nil, err
	}
	return &FetchResult{Binary: *bin, BundleSHA: pin.SHA256}, nil
}

// ExtractBinary pulls the server binary out of .mcpb bundle bytes (a zip
// whose manifest.json names server.entry_point). The returned Binary is
// versioned as DefaultVersion.
func ExtractBinary(bundle []byte, platformKey string) (*Binary, error) {
	zr, err := zip.NewReader(bytes.NewReader(bundle), int64(len(bundle)))
	if err != nil {
		return nil, fmt.Errorf("stackql mcp: opening bundle zip: %w", err)
	}
	manifestFile, err := zr.Open("manifest.json")
	if err != nil {
		return nil, fmt.Errorf("stackql mcp: bundle has no manifest.json: %w", err)
	}
	defer manifestFile.Close()
	var bm struct {
		Server struct {
			EntryPoint string `json:"entry_point"`
		} `json:"server"`
	}
	if err := json.NewDecoder(manifestFile).Decode(&bm); err != nil {
		return nil, fmt.Errorf("stackql mcp: parsing manifest.json: %w", err)
	}
	entry := bm.Server.EntryPoint
	if entry == "" {
		return nil, fmt.Errorf("stackql mcp: manifest.json has no server.entry_point")
	}
	binFile, err := zr.Open(entry)
	if err != nil {
		return nil, fmt.Errorf("stackql mcp: bundle entry_point %q not found: %w", entry, err)
	}
	defer binFile.Close()
	data, err := io.ReadAll(binFile)
	if err != nil {
		return nil, fmt.Errorf("stackql mcp: reading entry_point: %w", err)
	}
	sum := sha256.Sum256(data)
	return &Binary{
		Data:        data,
		Version:     manifest.Version,
		PlatformKey: platformKey,
		SHA256:      hex.EncodeToString(sum[:]),
	}, nil
}
