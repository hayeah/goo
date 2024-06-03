package goo

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pelletier/go-toml/v2"
	"github.com/tailscale/hujson"
)

var (
	JSONFormat  = "json"
	JSONCFormat = "jsonc"
	YAMLFormat  = "yaml"
	TOMLFormat  = "toml"
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

// Decode unmarshals data from the reader according to the format into the object.
// format is one of "toml", "yaml", "json", "jsonc".
func Decode(r io.Reader, format string, o interface{}) error {
	switch format {
	case TOMLFormat:
		err := toml.NewDecoder(r).Decode(o)
		if err != nil {
			return fmt.Errorf("decode toml: %w", err)
		}
	case YAMLFormat:
		data, err := io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("decode yaml: %w", err)
		}

		err = yaml.Unmarshal(data, o)
		if err != nil {
			return fmt.Errorf("decode yaml: %w", err)
		}
	case JSONFormat, JSONCFormat:
		data, err := io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("decode json: %w", err)
		}

		data, err = hujson.Standardize(data)
		if err != nil {
			return fmt.Errorf("decode json: %w", err)
		}

		err = json.Unmarshal(data, o)
		if err != nil {
			return fmt.Errorf("decode json: %w", err)
		}
	default:
		return fmt.Errorf("unsupported decode format: %s", format)
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

// DecodeURL parses the data URL and decodes the content into the provided object.
func DecodeURL(dataurl string, o interface{}) error {
	var scheme string
	parsedURL, err := url.Parse(dataurl)
	if err != nil {
		// if invalid url, just treat it as file path
		err = nil
	} else {
		scheme = parsedURL.Scheme
	}

	var r io.ReadCloser

	switch scheme {
	case "http", "https":
		resp, err := http.Get(dataurl)
		if err != nil {
			return fmt.Errorf("decodeURL: http get: %w", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("decodeURL: http get: non-200 status code: %d", resp.StatusCode)
		}
		r = resp.Body
	case "":
		r, err = os.Open(dataurl)
		if err != nil {
			return fmt.Errorf("decodeURL: open file: %w", err)
		}
		defer r.Close()
	default:
		return fmt.Errorf("decodeURL: unknown protocol: %s", parsedURL.Scheme)
	}

	ext := strings.ToLower(filepath.Ext(parsedURL.Path))
	format := strings.TrimPrefix(ext, ".")

	return Decode(r, format, o)
}
