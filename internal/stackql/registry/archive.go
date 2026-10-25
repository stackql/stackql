package registry

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"github.com/Masterminds/semver"
)

func latestVersion(versions []string) (string, error) {
	available := make([]*semver.Version, 0, len(versions))
	for _, v := range versions {
		parsed, err := semver.NewVersion(v)
		if err != nil {
			return "", err
		}
		available = append(available, parsed)
	}
	if len(available) == 0 {
		return "", fmt.Errorf("no versions available")
	}
	sort.Sort(semver.Collection(available))
	return available[len(available)-1].Original(), nil
}

func decompressToPath(reader io.ReadCloser, target string) error {
	defer reader.Close()
	compressed, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	defer compressed.Close()
	archive := tar.NewReader(compressed)
	for {
		header, nextErr := archive.Next()
		if nextErr == io.EOF {
			return nil
		}
		if nextErr != nil {
			return nextErr
		}
		if entryErr := extractEntry(archive, header, target); entryErr != nil {
			return entryErr
		}
	}
}

func extractEntry(reader io.Reader, header *tar.Header, target string) error {
	if !filepath.IsLocal(header.Name) {
		return fmt.Errorf("invalid archive path %q", header.Name)
	}
	name := filepath.Join(target, header.Name) // #nosec G305 -- IsLocal rejects absolute paths and parent traversal.
	if header.Typeflag == tar.TypeDir {
		return os.MkdirAll(name, 0755)
	}
	if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
		return fmt.Errorf("unsupported archive entry %q", header.Name)
	}
	if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
		return err
	}
	file, err := os.OpenFile(name, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, copyErr := io.CopyN(file, reader, header.Size)
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

// LatestVersion selects the latest published semantic version from a registry entry.
func LatestVersion(versions []string) (string, error) { return latestVersion(versions) }
