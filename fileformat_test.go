package goo

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecodeURL(t *testing.T) {
	assert := assert.New(t)

	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "test*.json")
	assert.NoError(err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(`{"key": "value"}`)
	assert.NoError(err)
	defer tmpFile.Close()

	_, err = tmpFile.Seek(0, 0)
	assert.NoError(err)

	// Create a test HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"key": "value"}`))
	}))
	defer server.Close()

	server.URL = server.URL + "/file.json"

	// Test cases
	testCases := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "local file",
			url:     tmpFile.Name(),
			wantErr: false,
		},
		{
			name:    "http URL",
			url:     server.URL,
			wantErr: false,
		},
		{
			name:    "unknown protocol",
			url:     "ftp://example.com/file.json",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var result map[string]interface{}
			err := DecodeURL(tc.url, &result)

			if tc.wantErr {
				assert.Error(err, tc.name)
			} else {
				assert.NoError(err, tc.name)

				assert.Equal(map[string]interface{}{"key": "value"}, result, tc.name)
			}
		})
	}
}
