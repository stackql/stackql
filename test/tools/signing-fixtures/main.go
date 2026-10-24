// signing-fixtures generates disposable signed registries for black-box tests.
package main

import (
	"crypto/mldsa"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/stackql/stackql-provider-registry/signing/pqsig"
)

const (
	fixtureDirectoryMode = 0700
	fixtureFileMode      = 0600
)

func main() {
	out := flag.String("out", "", "existing directory for disposable fixture registries")
	flag.Parse()
	if err := generate(*out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate(out string) error {
	if out == "" {
		return fmt.Errorf("-out is required")
	}
	root, err := filepath.Abs(out)
	if err != nil {
		return err
	}
	destination, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer destination.Close()
	source, err := os.OpenRoot(filepath.Join(
		"internal", "stackql", "intrinsic", "testdata", "registry", "fixture", "v0.1.0"))
	if err != nil {
		return err
	}
	defer source.Close()
	key, err := mldsa.GenerateKey(mldsa.MLDSA65())
	if err != nil {
		return err
	}
	for _, variant := range []string{"valid", "tampered-provider", "tampered-service", "unknown", "unsigned", "opt-out"} {
		if writeErr := writeVariant(source, destination, variant, key); writeErr != nil {
			return writeErr
		}
		if configErr := writeConfig(destination, root, variant, key.PublicKey()); configErr != nil {
			return configErr
		}
	}
	return nil
}

func writeVariant(source, destination *os.Root, variant string, key *mldsa.PrivateKey) error {
	return fs.WalkDir(source.FS(), ".", func(file string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		data, err := source.ReadFile(file)
		if err != nil {
			return err
		}
		data, err = variantDocument(data, file, variant, key)
		if err != nil {
			return err
		}
		dest := filepath.Join(variant, "src", "fixture", "v0.1.0", file)
		if mkdirErr := destination.MkdirAll(filepath.Dir(dest), fixtureDirectoryMode); mkdirErr != nil {
			return mkdirErr
		}
		return destination.WriteFile(dest, data, fixtureFileMode)
	})
}

func variantDocument(data []byte, file, variant string, key *mldsa.PrivateKey) ([]byte, error) {
	if variant == "unsigned" || variant == "opt-out" {
		return data, nil
	}
	kid := "robot-test"
	if variant == "unknown" {
		kid = "not-trusted"
	}
	signed, err := pqsig.Sign(data, kid, key)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", file, err)
	}
	if (variant == "tampered-provider" && file == "provider.yaml") ||
		(variant == "tampered-service" && file != "provider.yaml") {
		signed = append(signed, []byte("x-tampered: true\n")...)
	}
	return signed, nil
}

func writeConfig(destination *os.Root, root, variant string, key *mldsa.PublicKey) error {
	target := filepath.Join(root, variant)
	urlPath := filepath.ToSlash(target)
	if !strings.HasPrefix(urlPath, "/") {
		urlPath = "/" + urlPath
	}
	cfg := map[string]any{
		"url":          (&url.URL{Scheme: "file", Path: urlPath}).String(),
		"localDocRoot": target,
		"verifyConfig": pqsig.VerifierConfig{
			NopVerify:  variant == "opt-out",
			PublicKeys: map[string]string{"robot-test": hex.EncodeToString(key.Bytes())},
		},
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	return destination.WriteFile(variant+".json", data, fixtureFileMode)
}
