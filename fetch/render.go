package fetch

import (
	"bytes"

	"github.com/hayeah/mustache/v2"
)

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
