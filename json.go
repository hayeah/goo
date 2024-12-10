package goo

import (
	"github.com/hayeah/mustache/v2"
	"github.com/tailscale/hujson"
)

// RenderJSON renders a mustache JSON template with the given data.
func RenderJSON(template string, data any) ([]byte, error) {
	out, err := mustache.RenderJSON(template, data)
	if err != nil {
		return nil, err
	}

	return hujson.Minimize(out)
	// return hujson.Standardize(out)
}
