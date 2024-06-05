package fetch

import (
	"bytes"

	"github.com/hayeah/mustache/v2"
	"github.com/tailscale/hujson"
)

// RenderJSON renders a mustache JSON template with the given data.
func RenderJSON(template string, data any) ([]byte, error) {
	t, err := mustache.JSONTemplate(template)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer

	err = t.Frender(&buf, data)
	if err != nil {
		return nil, err
	}

	// Standardize the JSON with hujson to allow trailing commas.
	return hujson.Standardize(buf.Bytes())
}

// RenderJSON renders a mustache URL template with the given data.
func RenderURLPath(path string, data interface{}) (string, error) {
	// FIXME: should escape URL...
	// url.PathEscape(path)
	template, err := mustache.New().WithEscapeMode(mustache.Raw).CompileString(path)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer

	err = template.Frender(&buf, data)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
