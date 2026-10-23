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

func main() {
	out := flag.String("out", "", "directory for disposable fixture registries")
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
	key, err := mldsa.GenerateKey(mldsa.MLDSA65())
	if err != nil {
		return err
	}
	keys := map[string]string{"robot-test": hex.EncodeToString(key.PublicKey().Bytes())}
	source := filepath.Join("internal", "stackql", "intrinsic", "testdata", "registry", "fixture", "v0.1.0")
	for _, variant := range []string{"valid", "tampered-provider", "tampered-service", "unknown", "unsigned", "opt-out"} {
		target := filepath.Join(root, variant)
		err := filepath.WalkDir(source, func(file string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			relative, err := filepath.Rel(source, file)
			if err != nil {
				return err
			}
			data, err := os.ReadFile(file)
			if err != nil {
				return err
			}
			if variant != "unsigned" && variant != "opt-out" {
				kid := "robot-test"
				if variant == "unknown" {
					kid = "not-trusted"
				}
				data, err = pqsig.Sign(data, kid, key)
				if err != nil {
					return fmt.Errorf("%s: %w", file, err)
				}
			}
			if (variant == "tampered-provider" && relative == "provider.yaml") ||
				(variant == "tampered-service" && relative != "provider.yaml") {
				data = append(data, []byte("x-tampered: true\n")...)
			}
			dest := filepath.Join(target, "src", "fixture", "v0.1.0", relative)
			if err := os.MkdirAll(filepath.Dir(dest), 0700); err != nil {
				return err
			}
			return os.WriteFile(dest, data, 0600)
		})
		if err != nil {
			return err
		}
		urlPath := filepath.ToSlash(target)
		if !strings.HasPrefix(urlPath, "/") {
			urlPath = "/" + urlPath
		}
		cfg := map[string]any{
			"url":          (&url.URL{Scheme: "file", Path: urlPath}).String(),
			"localDocRoot": target,
			"verifyConfig": pqsig.VerifierConfig{NopVerify: variant == "opt-out", PublicKeys: keys},
		}
		data, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(root, variant+".json"), data, 0600); err != nil {
			return err
		}
	}
	return nil
}
