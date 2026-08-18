package embed

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// DefaultCacheDir returns the shared binary cache root,
// ~/.stackql/mcp-server-bin, the same location the npm and pypi wrappers
// use, so multiple embedders on one machine share a single extraction.
func DefaultCacheDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("stackql mcp: resolving home dir: %w", err)
	}
	return filepath.Join(home, ".stackql", "mcp-server-bin"), nil
}

// cachedBinaryPath returns the canonical extraction path for a binary:
// <cacheDir>/<version>/<platform-key>/stackql[.exe]
func cachedBinaryPath(cacheDir string, b Binary) string {
	return filepath.Join(cacheDir, b.Version, b.PlatformKey, exeName(b.PlatformKey))
}

// EnsureExtracted writes the embedded binary to the shared cache and returns
// its absolute path. If a file with the expected sha256 is already present
// (extracted earlier by this module or by the npm/pypi wrappers) it is
// reused. If cacheDir is empty, DefaultCacheDir is used.
func EnsureExtracted(cacheDir string, b Binary) (string, error) {
	if err := b.validate(); err != nil {
		return "", err
	}
	if cacheDir == "" {
		var err error
		cacheDir, err = DefaultCacheDir()
		if err != nil {
			return "", err
		}
	}
	return installBinary(cachedBinaryPath(cacheDir, b), b.Data, b.SHA256)
}

// installBinary places data at target unless a file with wantSHA is already
// there. Extraction is atomic: the binary is written to a temp file in the
// target directory and renamed into place, so concurrent extractors are
// safe.
func installBinary(target string, data []byte, wantSHA string) (string, error) {
	if ok, err := fileMatchesSHA256(target, wantSHA); err != nil {
		return "", err
	} else if ok {
		return target, nil
	}

	dir := filepath.Dir(target)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("stackql mcp: creating cache dir: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".extract-*")
	if err != nil {
		return "", fmt.Errorf("stackql mcp: creating temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return "", fmt.Errorf("stackql mcp: writing binary: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("stackql mcp: closing temp file: %w", err)
	}
	if err := os.Chmod(tmpName, 0o755); err != nil {
		return "", fmt.Errorf("stackql mcp: chmod binary: %w", err)
	}
	if err := os.Rename(tmpName, target); err != nil {
		// On Windows, renaming over a file another process just placed
		// (or is holding open) fails. If the file that won the race has
		// the right sha, use it.
		if ok, verr := fileMatchesSHA256(target, wantSHA); verr == nil && ok {
			return target, nil
		}
		return "", fmt.Errorf("stackql mcp: installing binary: %w", err)
	}
	return target, nil
}

// fileMatchesSHA256 reports whether path exists and hashes to wantHex.
// A stale or partial file (wrong hash) reports false with no error so the
// caller re-extracts over it.
func fileMatchesSHA256(path, wantHex string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("stackql mcp: opening cached binary: %w", err)
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return false, fmt.Errorf("stackql mcp: hashing cached binary: %w", err)
	}
	return hex.EncodeToString(h.Sum(nil)) == wantHex, nil
}
