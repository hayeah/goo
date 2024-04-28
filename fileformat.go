package goo

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pelletier/go-toml/v2"
)

var (
	JSONFormat = "json"
	YAMLFormat = "yaml"
	TOMLFormat = "toml"
)

func PrintJSON(o interface{}) error {
	return Encode(os.Stdout, JSONFormat, o)
}

func EncodeFile(file string, o interface{}) error {
	ext := strings.ToLower(filepath.Ext(file))

	w, err := os.Create(file)
	if err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	defer w.Close()

	// .toml -> "toml"
	format := strings.TrimPrefix(ext, ".")

	return Encode(w, format, o)

}

func Encode(w io.Writer, format string, o interface{}) error {
	switch format {
	case "toml":
		err := toml.NewEncoder(w).Encode(o)
		if err != nil {
			return fmt.Errorf("encode toml: %w", err)
		}
	case "yaml":
		data, err := yaml.Marshal(o)
		if err != nil {
			return fmt.Errorf("encode yaml: %w", err)
		}

		_, err = w.Write(data)
		if err != nil {
			return fmt.Errorf("encode yaml: %w", err)
		}
	case "json":
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		err := enc.Encode(o)
		if err != nil {
			return fmt.Errorf("encode json: %w", err)
		}
	default:
		return fmt.Errorf("unsupported config file format: %s", format)
	}

	return nil
}

func Decode(r io.Reader, format string, o interface{}) error {
	switch format {
	case "toml":
		err := toml.NewDecoder(r).Decode(o)
		if err != nil {
			return fmt.Errorf("decode toml: %w", err)
		}
	case "yaml":
		data, err := io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("decode yaml: %w", err)
		}

		err = yaml.Unmarshal(data, o)
		if err != nil {
			return fmt.Errorf("decode yaml: %w", err)
		}
	case "json":
		err := json.NewDecoder(r).Decode(o)
		if err != nil {
			return fmt.Errorf("decode json: %w", err)
		}
	default:
		return fmt.Errorf("unsupported config file format: %s", format)
	}

	return nil
}

func DecodeFile(file string, o interface{}) error {
	ext := strings.ToLower(filepath.Ext(file))

	r, err := os.Open(file)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	defer r.Close()

	// .toml -> "toml"
	format := strings.TrimPrefix(ext, ".")

	return Decode(r, format, o)
}
