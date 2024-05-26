package fetch

import (
	"bytes"

	"github.com/hayeah/mustache/v2"
	"github.com/tailscale/hujson"
)

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
