// Package flatfile reads flat maps of scalar configuration values.
package flatfile

import (
	"bytes"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// Format returns the structured format inferred from path, or an empty string for legacy files.
func Format(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".json":
		return "json"
	case ".yaml", ".yml", ".yalm":
		return "yaml"
	case ".toml":
		return "toml"
	default:
		return ""
	}
}

// Load uses the file suffix to select a structured parser or the supplied legacy parser.
func Load(path string, legacy func([]byte) (map[string]string, error), ignore ...string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	format := Format(path)
	if format == "" {
		return legacy(data)
	}
	var decoded any
	switch format {
	case "toml":
		fields := map[string]any{}
		_, err = toml.Decode(string(data), &fields)
		decoded = fields
	case "json":
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		decoded, err = decodeOne(decoder)
	case "yaml":
		decoder := yaml.NewDecoder(bytes.NewReader(data))
		decoded, err = decodeOne(decoder)
	}
	if err != nil {
		if format == "yaml" && errors.Is(err, io.EOF) {
			return map[string]string{}, nil
		}
		// Parser diagnostics can include source values, including credentials.
		return nil, fmt.Errorf("invalid %s configuration", format)
	}
	fields, ok := decoded.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("%s configuration must be a mapping with string keys", format)
	}
	values := make(map[string]string, len(fields))
	for key, value := range fields {
		if slices.Contains(ignore, key) {
			continue
		}
		text, scalarErr := scalarText(value)
		if scalarErr != nil {
			return nil, fmt.Errorf("configuration key %q: %w", key, scalarErr)
		}
		values[key] = text
	}
	return values, nil
}

func decodeOne(decoder interface{ Decode(any) error }) (any, error) {
	var value any
	if err := decoder.Decode(&value); err != nil {
		return nil, err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("expected one document")
	}
	return value, nil
}

func scalarText(value any) (string, error) {
	switch v := value.(type) {
	case nil:
		return "null", nil
	case string:
		return v, nil
	case bool, int, int64, uint64, float64, json.Number:
		return fmt.Sprint(v), nil
	case encoding.TextMarshaler:
		text, err := v.MarshalText()
		if err != nil {
			return "", fmt.Errorf("invalid scalar value")
		}
		return string(text), nil
	default:
		return "", fmt.Errorf("expected a scalar value")
	}
}
